# Postmortem 001 — Fail-fast JWT killed workers/scheduler

Date: 2026-09-25. Severity: high (all background processing stopped).
Blameless: the failure was systemic, not individual.

## Timeline

1. `JWT_SECRET` default removed for production safety (fail-fast in `main`).
2. `docker compose up -d --build` recreated api, worker-1/2, scheduler.
3. api healthy; worker-1/2 and scheduler exited(1): `JWT_SECRET is required`.
4. Monitors stuck UNKNOWN, queue silent; noticed via missing checks.

## Root cause

`jwtSecret()` was called unconditionally in the composition root, but only
the `api` service carries `JWT_SECRET` in compose — workers/scheduler never
needed the signing secret (old default masked this). Fail-fast turned a
latent wiring assumption into a fleet-wide kill.

## Fix

JWT required in `api` mode only (least privilege); workers/scheduler boot
without the secret. Verified: all services running, checks flowing.

## Action items

- [x] Mode-scoped secret requirement (this fix).
- [ ] Startup config log per mode (POSTMORTEM-002 action, shared).
