package scheduler

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/RakhaYandra/pulse/infrastructure/metrics"
	"github.com/RakhaYandra/pulse/pkg/logger"
	"github.com/RakhaYandra/pulse/usecase"
)

type Runner struct {
	Log      *logger.Logger
	Monitors usecase.MonitorService
	Queue    usecase.JobQueue
}

func (r Runner) Run(ctx context.Context) {
	tick := time.NewTicker(tickInterval())
	defer tick.Stop()
	for {
		r.enqueue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// tickInterval bounds schedule granularity. Lower = tighter cadence at the
// cost of more due-scans (cheap indexed query). Benchmarked at 10s.
func tickInterval() time.Duration {
	if v := os.Getenv("SCHED_TICK_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 60 {
			return time.Duration(n) * time.Second
		}
	}
	return 10 * time.Second
}

func (r Runner) enqueue(ctx context.Context) {
	due, err := r.Monitors.Due(ctx)
	if err != nil {
		r.Log.Error("due scan failed", "err", err)
		return
	}
	metrics.SchedulerDue.Add(float64(len(due)))
	n, skipped := 0, 0
	for _, m := range due {
		// Claim TTL 3x interval: crash safety net. Normal path releases
		// the claim when the worker finishes the job.
		queued, err := r.Queue.Enqueue(ctx, m.ID, 3*time.Duration(m.IntervalSeconds)*time.Second)
		if err != nil {
			r.Log.Error("enqueue failed", "err", err)
			continue
		}
		if !queued {
			skipped++
			continue
		}
		n++
	}
	metrics.SchedulerEnqueued.Add(float64(n))
	metrics.SchedulerSkipped.Add(float64(skipped))
	if n > 0 {
		r.Log.Info("enqueued jobs", "count", strconv.Itoa(n), "skipped", strconv.Itoa(skipped))
	}
}
