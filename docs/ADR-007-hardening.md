# ADR-007 — Hardening from benchmark data

Status: accepted (2026-09-25). Evidence: `docs/BENCHMARK.md`.

## H1 shared HTTP client — SKIPPED
Hypothesis (connection churn) not confirmed: p95 process 0.5s dominated by
target latency + timeout tail, not handshake cost. Not implemented.

## H3 enqueue dedup — IMPLEMENTED
Trigger: baseline N=100 W=2 → queue 786, overdue 34/100 (re-enqueue of
already-queued monitors every tick). Lua claim (`pulse:queued:<id>`, TTL
3x interval) + worker `Release` on completion; TTL is crash safety net.
Result: qmax 0 across all post-hardening runs.

## H2 worker pool — IMPLEMENTED
Trigger: inflight saturated 2/2, timeout monitors (~18s each) monopolizing
sequential workers. Semaphore pool per container (`WORKER_CONCURRENCY`,
default NumCPU), shutdown-aware acquire, per-job recover kept.
Result: peak drain 28/s @N=1000 W=4 (required 16.7/s).

## Scheduler tick 15s → 10s (+ `SCHED_TICK_SECONDS`)
Trigger: wave spacing ~90s vs 60s interval (tick quantization + drain spread
from synchronized seeding). Cheap indexed scan, halved granularity.
Residual ~25-44% cadence stretch documented honestly; `next_run_at`
scheduling noted as future work, not justified now.
