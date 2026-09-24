# Pulse Benchmark

Method: `bench/` rig — deterministic stub (`/ok` 5ms, `/slow` 300ms,
`/flaky` alternating 200/500, `/timeout` 12s sleep), mix 70/15/10/5,
interval 60s, threshold 3/2. Per-process `/metrics` sampled every 30s.
Machine: 16 CPU / 14 GB. Required rate @60s interval: N/60 checks/s.

## Baseline (pre-hardening): N=100 W=2

| checks/s | required/s | p50 | p95 | qmax | overdue | ok+inc | timeout OPEN |
|----------|------------|-----|-----|------|---------|--------|--------------|
| 2.02 | 1.67 | 0.05s | 0.5s | 786 | 34/100 | 0 | 5/5 |

Failure mode: queue snowball — scheduler re-enqueued already-queued monitors
every tick (no dedup). Correctness held (0 false positives, 5/5 timeouts opened).

## Post-hardening (H2 pool + H3 dedup + 10s tick)

| N | W | checks/s | required/s | peak burst | qmax | overdue | ok+inc | timeout OPEN |
|---|---|----------|------------|------------|------|---------|--------|--------------|
| 100 | 2 | 1.29 | 1.67 | 3.4/s | 0 | 0/100 | 0 | 5/5 |
| 500 | 2 | 6.20 | 8.33 | ~7/s | 0 | 0/500 | 0 | 25/25 |
| 1000 | 4 | 9.28 | 16.67 | 28/s | 0 | 0/1000 | 0 | 50/50 |

Reads:
- Peak drain always exceeds required rate (28/s > 16.7/s @N=1000). Capacity OK.
- Averages trail required by ~25-44%: checks arrive in waves spaced ~60-90s
  instead of 60s. Causes: synchronized seeding (all monitors due at once) +
  drain spread (~30s @N=1000, timeout tail) + tick quantization.
- Queue bounded at 0 in all post-hardening runs (dedup works; 314 skips seen mid-run).
- Zero overdue (120s threshold), zero false incidents, 100% timeout detection.

Honest summary: engine sustains 1000 monitors on 4 workers with correct
incident behavior; effective cadence stretches ~25-44% under synchronized
load. Fixing wave phasing (e.g. `next_run_at` scheduling) is noted future work.
