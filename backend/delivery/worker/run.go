package worker

import (
	"context"
	"time"

	"github.com/RakhaYandra/pulse/pkg/logger"
	"github.com/RakhaYandra/pulse/usecase"
)

type Runner struct {
	Log        *logger.Logger
	Monitoring usecase.MonitoringService
	Queue      usecase.JobQueue
	Notifier   usecase.Notifier
}

func (r Runner) Run(ctx context.Context) {
	r.Log.Info("worker waiting for jobs")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		monitorID, err := r.Queue.Dequeue(ctx, 30*time.Second)
		if err != nil {
			r.Log.Error("queue pop failed", "err", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if monitorID == "" {
			continue
		}
		// One monitor failing must never kill the worker or other monitors.
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.Log.Error("job panic recovered", "monitor", monitorID)
				}
			}()
			tr, err := r.Monitoring.ProcessCheck(ctx, monitorID)
			if err != nil {
				r.Log.Error("process check failed", "monitor", monitorID, "err", err)
				return
			}
			if tr != nil {
				r.Log.Info("incident transition", "monitor", monitorID, "type", string(tr.Type))
				r.Notifier.Notify(*tr)
			}
		}()
	}
}
