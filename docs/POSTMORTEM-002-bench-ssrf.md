# Postmortem 002 — Invalid bench run (SSRF allowlist never applied)

Date: 2026-09-25. Severity: medium (wasted run, misleading data caught before publishing).
Blameless: the rig lacked guards, not the operator.

## Timeline

1. SSRF allowlist (`PULSE_ALLOW_HOSTS`) added to compose for workers.
2. Bench re-ran with `docker compose stop/start` — `start` does NOT recreate
   containers, so the new env never applied to running workers.
3. All 500 bench checks failed: `host resolves to blocked address: stub`.
4. Result: 0.09/s, qmax 860, 346 false incidents on ok monitors.
5. Anomaly spotted in analysis (fail≈1.0 on ok mix is impossible); run
   discarded, rig fixed, valid re-run: 7.48/s, qmax 0, 0 false incidents.

## Root cause (two layers)

- (a) Stale containers: `start` resumes old config; only `up -d` recreates.
  Fixed: run.sh uses `up -d` for workers.
- (b) No pre-flight: rig sampled 10 minutes blindly with a dead stub path and
  zero early validation. Nothing checked that checks were actually succeeding.

## Fix

- `PULSE_ALLOW_HOSTS=stub` exported by run.sh + workers recreated via `up -d`.
- Pre-flight gates: stub 200, worker env inspection, early checks delta.
- Analysis exits non-zero on invalid runs (`ok_inc`, `qmax` bounds).

## Action items

- [x] run.sh pre-flight + validity exit codes.
- [ ] Startup config log per mode (shared with POSTMORTEM-001).
