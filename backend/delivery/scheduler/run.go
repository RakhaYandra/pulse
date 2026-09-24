package scheduler

import (
	"context"
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
	tick := time.NewTicker(15 * time.Second)
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

func (r Runner) enqueue(ctx context.Context) {
	due, err := r.Monitors.Due(ctx)
	if err != nil {
		r.Log.Error("due scan failed", "err", err)
		return
	}
	metrics.SchedulerDue.Add(float64(len(due)))
	n := 0
	for _, m := range due {
		if err := r.Queue.Enqueue(ctx, m.ID); err != nil {
			r.Log.Error("enqueue failed", "err", err)
			continue
		}
		n++
	}
	metrics.SchedulerEnqueued.Add(float64(n))
	if n > 0 {
		r.Log.Info("enqueued jobs", "count", strconv.Itoa(n))
	}
}
