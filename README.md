# Pulse — API Monitoring & Incident Platform

> Ecosystem: [api](https://github.com/RakhaYandra/pulse) (this repo) · [web](https://github.com/RakhaYandra/pulse-web) · [docs](https://github.com/RakhaYandra/pulse-docs/releases) · [data](https://github.com/RakhaYandra/pulse-data)

Go, Gin, PostgreSQL, Redis, Docker. Full Clean Architecture.

Self-hosted platform for developers and small teams who own several APIs:
register endpoints, health-check them on a schedule, store every result,
open incidents on configurable failure thresholds, resolve on recovery, and
get notified via Telegram. Monitoring runs in background workers — never in
the dashboard request lifecycle.

## How it works

```
dashboard (pulse-web)
        │  JSON contract (Newman-guarded)
        ▼
Go/Gin API ──┬──► PostgreSQL (users, monitors, checks, incidents)
             └──► Redis (pulse:jobs) ──► Worker 1..N ──► target APIs
                                                      │ incidents ──► Telegram
```

1. You create a monitor: `GET https://api.example.com/health`, every 60s,
   timeout 5s, failure threshold 3, recovery threshold 2.
2. The scheduler finds due monitors each tick and enqueues jobs (deduped —
   one outstanding job per monitor).
3. Workers execute the HTTP check (≤3 attempts collapse into one result),
   store it, and evaluate the streak.
4. 3 consecutive failures → incident OPEN (+ Telegram). 2 consecutive
   successes → RESOLVED (+ Telegram).

Example lifecycle:

```
09:00  200  180ms  ✓
09:01  200  175ms  ✓
09:02  500  420ms  ✗
09:03  500  390ms  ✗
09:04  timeout      ✗   → INCIDENT OPENED
09:05  200  190ms  ✓
09:06  200  185ms  ✓   → INCIDENT RESOLVED
```

Key engineering points:

1. API layer separated from execution via Redis-backed workers.
2. Retry (≤3 attempts) collapses into one check — uptime math stays correct.
3. Incident state machine: fail streak → OPEN, success streak → RESOLVED.
4. Failure isolation: per-job `recover()` + concurrency pool; one bad monitor
   can't kill workers.
5. Queue holds no business state — loss only delays, never corrupts.
6. Schedule-driven cadence (`next_run_at`): due time advances from the
   scheduled moment, not from check completion — no drift under load.

## Quickstart

```bash
cp .env.example .env   # set JWT_SECRET (required — boot fails without it)
docker compose up -d --build
curl localhost:8080/health
```

Register your own account (there are no seeded credentials):

```bash
curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"pick-a-strong-password","name":"You"}'
```

Create your first monitor (save the token as `$T`):

```bash
curl -s -X POST localhost:8080/api/v1/monitors \
  -H "Authorization: Bearer $T" -H 'Content-Type: application/json' \
  -d '{"name":"My API","url":"https://api.example.com/health","interval_seconds":300}'
```

Dashboard: [pulse-web](https://github.com/RakhaYandra/pulse-web) (`npm run dev`).

## Run modes & services

One Go binary, three modes (`-mode=` flag). Every mode serves its own
`/metrics` (counters are process-local).

| Service | Mode | Purpose |
|---|---|---|
| `api` | `api` | REST + auth + reads + `/metrics` |
| `scheduler` | `scheduler` | Due-scan → enqueue (deduped) |
| `worker-1..4` | `worker` | Pop jobs → check → store → incident eval → notify |

Workers run a semaphore pool (`WORKER_CONCURRENCY`, default NumCPU);
explicit `worker-1..4` services (not replicas) so each `/metrics` endpoint
is scrapable at a fixed port.

## API contract

Base `http://localhost:8080/api/v1`. Auth: `Authorization: Bearer <JWT>`.
Errors: `{"error": "<message>"}` with 400/401/404/409/429/500.
Success: `{"data": ...}`.

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/health` | no | liveness |
| GET | `/api/v1/health` | no | db + redis status |
| POST | `/api/v1/auth/register` | no | 10/min/IP, 409 on dup email |
| POST | `/api/v1/auth/login` | no | 10/min/IP, JWT 24h |
| GET | `/api/v1/auth/me` | yes | current user |
| GET | `/api/v1/monitors` | yes | own monitors |
| POST | `/api/v1/monitors` | yes | validate URL, interval ≥60s |
| GET | `/api/v1/monitors/:id` | yes | 404 if not owned |
| PATCH | `/api/v1/monitors/:id` | yes | true PATCH — omitted fields kept |
| DELETE | `/api/v1/monitors/:id` | yes | cascades checks + incidents |
| POST | `/api/v1/monitors/:id/pause` | yes | scheduler skips paused |
| POST | `/api/v1/monitors/:id/resume` | yes | — |
| GET | `/api/v1/monitors/:id/checks?limit=20` | yes | newest first, max 200 |
| GET | `/api/v1/monitors/:id/incidents` | yes | per-monitor history |
| GET | `/api/v1/incidents` | yes | all own incidents, newest first |
| GET | `/api/v1/dashboard/summary` | yes | totals, up/down, active, 24h uptime |
| GET | `/api/v1/reports/reliability?days=30` | yes | per-monitor MTTR, uptime, 1-90d |

Full machine-readable contract: [`qa/collection.json`](qa/collection.json)
(22 requests, Newman-gated).

## Configuration

| Variable | Default | Required | Purpose |
|---|---|---|---|
| `JWT_SECRET` | — | **yes** | Signing secret; boot fails without it |
| `DATABASE_URL` | — | yes (compose sets) | Postgres DSN |
| `REDIS_ADDR` | `localhost:6379` | no | Redis address |
| `PORT` | `8080` | no | API listen port |
| `METRICS_PORT` | `9100` | no | Per-process `/metrics` port |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | no | `*` only if set explicitly |
| `PULSE_ALLOW_HOSTS` | empty (strict) | no | SSRF-exempt hostnames (bench `stub`) |
| `TRUST_PROXY` | unset | no | `=1` trusts `X-Forwarded-For` for rate limit IP |
| `SCHED_TICK_SECONDS` | `10` | no | Scheduler granularity (1-60) |
| `WORKER_CONCURRENCY` | NumCPU | no | In-flight jobs per worker |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | empty (disabled) | no | Notify on OPEN/RESOLVED only |
| `LOG_LEVEL` | `info` | no | Log verbosity |
| `GF_SECURITY_ADMIN_PASSWORD` | `pulse` | no | Grafana dev login (observability) |

## Ports

| Port | Service |
|---|---|
| 8080 | API |
| 9101 / 9102–04,06 / 9105 | metrics: api / worker-1..4 / scheduler |
| 5433 → 5432 | PostgreSQL |
| 6380 → 6379 | Redis |
| 9090 / 3000 | Prometheus / Grafana (observability profile) |
| 8099 | Bench stub (`--profile bench` only) |

## Security

- **SSRF**: literal private/loopback/link-local hosts rejected at validation
  (`domain`); resolved IPs re-checked at dial time (TOCTOU-safe); redirects
  never followed; `PULSE_ALLOW_HOSTS` is the only exemption. See ADR-008.
- **Rate limiting** (Redis, shared across replicas, fail-open): auth 10/min/IP,
  API 100/min/IP, 429 + `Retry-After`.
- **Auth**: bcrypt passwords (min 8), JWT 24h, per-user isolation on every
  query (`WHERE user_id`).
- **Defaults**: no secret defaults in code; `*` CORS and default passwords
  only via explicit env. Dev compose values are documented insecure.

## Verification

| Layer | Command | Result |
|---|---|---|
| Go unit | `go test ./...` (in `backend/`) | domain + usecase (fake repos) + checker + SSRF + rate limit |
| API contract | `npx newman run qa/collection.json --env-var baseUrl=http://localhost:8080` | 22/22 |
| E2E (black-box) | `npm test --prefix e2e` (in `pulse-web`) | 7/7 |
| Engine | register → 404 monitor → OPEN → fix URL → RESOLVED | verified live |

## Benchmark (measured, [`docs/BENCHMARK.md`](https://github.com/RakhaYandra/pulse-docs/blob/main/BENCHMARK.md))

Deterministic stub (70% ok / 15% slow / 10% flaky / 5% timeout), 60s interval,
16 CPU / 14 GB. Post-hardening (dedup + worker pool + 10s tick + `next_run_at`):

| Monitors | Workers | checks/s | peak burst | queue max | overdue | false incidents | timeouts detected |
|---|---|---|---|---|---|---|---|
| 100 | 2 | 1.29 | 3.4/s | 0 | 0 | 0 | 5/5 |
| 500 | 2 | 7.48 | ~8/s | 0 | 0 | 0 | 25/25 |
| 1000 | 4 | 9.28 | 28/s | 0 | 0 | 0 | 50/50 |

Honest note: peak drain exceeds required rate at every scale; after
`next_run_at`, wave spacing settled to ~60s steady-state (cold start aside).
Queue stays bounded; incident behavior exact.

Run your own: `./bench/run.sh [N=100] [W=2] [D=600]` — with pre-flight
gates (stub alive, workers allowlisted, checks flowing) and validity exit
codes. Clean up after: `bench/clean.sql`.

## Observability

Per-process `/metrics`, Prometheus + Grafana + 2 alerts
(`PulseQueueBacklog`, `PulseProcessErrors`) via `docker-compose.observability.yml`:

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d prometheus grafana
# Grafana http://localhost:3000 (admin / $GF_SECURITY_ADMIN_PASSWORD), dashboard "Pulse"
```

![Grafana dashboard](https://raw.githubusercontent.com/RakhaYandra/pulse-docs/main/grafana.png)

## Operations

- Backup/restore: [`ops/BACKUP.md`](ops/BACKUP.md) (pg_dump, volume snapshots).
- Bench cleanup: `bench/clean.sql` (deletes `bench-%` monitors + bench users).
- After editing compose env: use `up -d` (recreates), never bare `start` —
  stale containers keep old env (see [POSTMORTEM-002](https://github.com/RakhaYandra/pulse-docs/blob/main/POSTMORTEM-002-bench-ssrf.md)).
- Restart policies + Redis AOF are on; Postgres data persists in `pgdata`.

## Layout & docs

```
pulse/
├── backend/            # Go, Clean Architecture: domain/usecase/infrastructure/delivery
├── bench/              # stub + seed.sql + run.sh + clean.sql + results/*.json
├── observability/      # prometheus.yml, alerts.yml, grafana provisioning + dashboard
├── ops/                # BACKUP.md
├── qa/                 # collection.json (Newman contract)
└── docs/ → [pulse-docs](https://github.com/RakhaYandra/pulse-docs)  # split out
```

Docs live in [pulse-docs](https://github.com/RakhaYandra/pulse-docs):
[ARCHITECTURE](https://github.com/RakhaYandra/pulse-docs/blob/main/ARCHITECTURE.md) · [BENCHMARK](https://github.com/RakhaYandra/pulse-docs/blob/main/BENCHMARK.md) ·
[ADR-001](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-001-monolith.md) [002](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-002-redis-queue.md)
[003](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-003-retry-vs-check.md) [004](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-004-split.md)
[005](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-005-full-ca.md) [006](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-006-metrics.md)
[007](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-007-hardening.md) [008](https://github.com/RakhaYandra/pulse-docs/blob/main/ADR-008-security.md) ·
[POSTMORTEM-001](https://github.com/RakhaYandra/pulse-docs/blob/main/POSTMORTEM-001-jwt-workers.md)
[002](https://github.com/RakhaYandra/pulse-docs/blob/main/POSTMORTEM-002-bench-ssrf.md)

## Roadmap

Done: MVP engine, full CA both repos, SSRF + rate limiting, metrics +
benchmark + hardening, observability, backup runbook, incident click-through,
MTTR/SLA reports, self-monitoring.
Frozen (owner decision): live Telegram verification (needs token), single-VPS
deploy, porto/CV entry.
Open: residual cadence stretch (see BENCHMARK.md), per-replica rate limit,
`pulse-data` analytics pipeline.
Non-goals: HA Postgres/Redis, multi-region, teams/billing.
