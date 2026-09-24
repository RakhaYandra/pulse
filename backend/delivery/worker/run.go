package worker

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/RakhaYandra/pulse/infrastructure/metrics"
	"github.com/RakhaYandra/pulse/pkg/logger"
	"github.com/RakhaYandra/pulse/usecase"
)

type Runner struct {
	Log        *logger.Logger
	Monitoring usecase.MonitoringService
	Queue      usecase.JobQueue
	Notifier   usecase.Notifier
}

// concurrency caps in-flight jobs per container. Sequential (1) preserves
// the original behavior; I/O-bound checks scale with higher values.
func concurrency() int {
	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return runtime.NumCPU()
}

func (r Runner) Run(ctx context.Context) {
	sem := make(chan struct{}, concurrency())
	r.Log.Info("worker waiting for jobs")
	depthTick := time.NewTicker(5 * time.Second)
	defer depthTick.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-depthTick.C:
				if n, err := r.Queue.Depth(ctx); err == nil {
					metrics.QueueDepth.Set(float64(n))
				}
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		monitorID, err := r.Queue.Dequeue(ctx, 30*time.Second)
		if err != nil {
			metrics.ProcessErrors.WithLabelValues("dequeue").Inc()
			r.Log.Error("queue pop failed", "err", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if monitorID == "" {
			continue
		}
		// Acquire a pool slot before handing off; shutdown-aware.
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		go r.process(ctx, sem, monitorID)
	}
}

func (r Runner) process(ctx context.Context, sem chan struct{}, monitorID string) {
	defer func() { <-sem }()
	// One monitor failing must never kill the worker or other monitors.
	defer func() {
		if rec := recover(); rec != nil {
			metrics.ProcessErrors.WithLabelValues("panic").Inc()
			r.Log.Error("job panic recovered", "monitor", monitorID)
		}
	}()
	metrics.JobsInflight.Inc()
	defer metrics.JobsInflight.Dec()
	start := time.Now()
	oc, err := r.Monitoring.ProcessCheck(ctx, monitorID)
	// Release the dedup claim: this monitor may be re-queued on
	// the next due scan. TTL remains as crash safety net.
	if relErr := r.Queue.Release(ctx, monitorID); relErr != nil {
		r.Log.Error("claim release failed", "monitor", monitorID, "err", relErr)
	}
	metrics.ProcessDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ProcessErrors.WithLabelValues("process").Inc()
		r.Log.Error("process check failed", "monitor", monitorID, "err", err)
		return
	}
	metrics.ChecksTotal.WithLabelValues(string(oc.Result.Status)).Inc()
	metrics.CheckDuration.Observe(float64(oc.Result.ResponseTimeMs) / 1000)
	if tr := oc.Transition; tr != nil {
		r.Log.Info("incident transition", "monitor", monitorID, "type", string(tr.Type))
		if tr.Type == domain.TransitionOpened {
			metrics.IncidentsOpened.Inc()
		} else {
			metrics.IncidentsResolved.Inc()
		}
		r.Notifier.Notify(*tr)
	}
}
