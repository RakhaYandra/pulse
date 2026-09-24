# Pulse Benchmark

Method: `bench/` rig — deterministic stub (`/ok` 5ms, `/slow` 300ms,
`/flaky` alternating 200/500, `/timeout` 12s sleep), mix 70/15/10/5,
interval 60s, threshold 3/2. Metrics sampled every 30s per worker process.
Machine: 16 CPU / 14 GB. Required rate @60s interval: N/60 checks/s.

## Baseline (pre-hardening)

| N | W | checks/s | required/s | p50 | p95 | qmax | overdue | ok+inc | timeout OPEN |
|---|---|----------|------------|-----|-----|------|---------|--------|--------------|
| 100 | 2 | 2.02 | 1.67 | 0.05s | 0.5s | 786 | 34/100 | 0 | 5/5 |

Reads: throughput meets required rate, but the queue snowballs — scheduler
re-enqueues already-queued monitors every 15s tick (no dedup), so backlog
grows unboundedly and 34% monitors starve. Correctness holds (0 false
positives on ok monitors, all 5 timeout monitors opened).

Failure mode confirmed → H3 (enqueue dedup) first, then H2 (worker pool).
NOTE: scheduler counters (`scheduler_enqueued_total`) were not sampled in
this run (worker ports only) — fixed in rig after baseline.
