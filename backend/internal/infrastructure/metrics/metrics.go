package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ChecksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_checks_total",
		Help: "Completed monitoring checks by result status.",
	}, []string{"status"})

	CheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "monitor_check_duration_seconds",
		Help:    "HTTP check duration (last attempt) in seconds.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	})

	ProcessDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "monitor_process_duration_seconds",
		Help:    "Full ProcessCheck duration (check + DB + incident eval).",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	})

	JobsInflight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "worker_jobs_inflight",
		Help: "Jobs currently being processed across workers.",
	})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Pending jobs in the Redis queue (LLEN polled).",
	})

	ProcessErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "process_check_errors_total",
		Help: "ProcessCheck failures by stage.",
	}, []string{"stage"})

	SchedulerDue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_due_found_total",
		Help: "Monitors found due across scheduler scans.",
	})

	SchedulerEnqueued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_enqueued_total",
		Help: "Jobs successfully enqueued.",
	})

	SchedulerSkipped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "scheduler_skipped_total",
		Help: "Due monitors skipped — already queued (dedup).",
	})

	IncidentsOpened = promauto.NewCounter(prometheus.CounterOpts{
		Name: "incidents_opened_total",
		Help: "Incidents opened by workers.",
	})

	IncidentsResolved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "incidents_resolved_total",
		Help: "Incidents resolved by workers.",
	})
)
