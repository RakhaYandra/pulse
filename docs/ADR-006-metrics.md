# ADR-006 — Prometheus metrics via client_golang

Status: accepted (2026-09-25).

Context: benchmark needed engine visibility (throughput, latency, queue,
scheduler behavior, incidents). Options: hand-rolled `/metrics` text,
`client_golang`, OpenTelemetry.

Decision: `client_golang` with package `infrastructure/metrics`.
Series (deliberately small):
`monitor_checks_total{status}`, `monitor_check_duration_seconds`,
`monitor_process_duration_seconds`, `worker_jobs_inflight`, `queue_depth`
(LLEN polled 5s), `process_check_errors_total{stage}`,
`scheduler_due_found_total`, `scheduler_enqueued_total`,
`scheduler_skipped_total`, `incidents_opened/resolved_total`.

Key design point: counters are process-local, so EVERY mode (api, worker,
scheduler) serves its own `/metrics` (`METRICS_PORT`, default 9100).
Compose pins fixed ports (api 9101, workers 9102-9103/9104/9106, scheduler
9105) and uses explicit `worker-1..4` services instead of replicas so each
process is scrapable. Rejected pushgateway (staleness, extra component).

Consequence: `usecase.JobQueue` gained `Depth()` (metrics-only port method);
`ProcessCheck` returns `Outcome{Result, Transition}` so delivery can observe
without re-querying.
