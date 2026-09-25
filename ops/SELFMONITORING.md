# Pulse Self-Monitoring (dogfooding)

Pulse watches its own API. One monitor, no automation, no secrets in repo.

## Setup (once)

1. Allowlist the internal API hostname (SSRF filtering blocks Docker-internal
   DNS by default). Local `.env` (never committed):
   ```
   PULSE_ALLOW_HOSTS=api
   ```
   Recreate workers so the env applies (`up -d` recreates; bare `start`
   keeps stale env — see POSTMORTEM-002):
   ```bash
   docker compose up -d worker-1 worker-2
   ```
2. Create the monitor via API on any ops account (manual step, no seed):
   - Name: `Pulse Self`, URL: `http://api:8080/health`
   - Interval 60s, timeout 5s, failure threshold 2, recovery threshold 1.

## Why it works

- The check hits the real `/health` (db + redis status) from a worker,
  same path as any user monitor.
- API down → ERROR streak → incident OPEN (Telegram if configured).
- API back → RESOLVED. The existing `PulseQueueBacklog` /
  `PulseProcessErrors` alerts cover the queue side.

## Verified

Stop api 2 min → incident OPEN; restart → RESOLVED. (Procedure, not fixture:
re-verify after engine changes.)
