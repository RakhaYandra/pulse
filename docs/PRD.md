# Pulse — PRD (Product Requirements Document)

## FR-01 Auth
Register/login (bcrypt, JWT 24h), `GET /me`. Users see only their monitors.

## FR-02 Monitor CRUD
Fields: name, url (http/https, no localhost), method GET (MVP), interval ≥60s,
timeout 1..60s and < interval, failure/recovery thresholds ≥1.
Pause/resume toggles `is_active` (scheduler skips paused).

## FR-03 Check execution
Scheduler (15s tick) enqueues due monitors to Redis `pulse:jobs`.
Workers BLPop jobs, run HTTP GET with per-check retry (≤3 attempts, 1s/2s
backoff) recorded as ONE check row: UP | DOWN | TIMEOUT | ERROR + code + ms.

## FR-04 Incident lifecycle
Trailing streak evaluated per check: fail streak ≥ failure_threshold → OPEN;
while OPEN, success streak ≥ recovery_threshold → RESOLVED.
Transitions trigger Telegram (OPEN/RESOLVED only).

## FR-05 Dashboard & history
Summary: totals, up/down, active incidents, 24h uptime.
Monitor detail: status, uptime, avg response, SVG response-time chart,
20 recent checks, incident history.

## NFR
- Failure isolation: one monitor's failure never stops workers/others (recover() per job).
- API p95 local < 100ms (Newman avg ~10ms measured).
- Secrets only via env; Telegram no-op when unconfigured.
