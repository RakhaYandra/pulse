# Pulse — Architecture

Modular monolith (one Go binary, 3 run modes) + 2 workers + Postgres + Redis.

```
React ──► Go/Gin API ──┬──► PostgreSQL (users, monitors, checks, incidents)
                       └──► Redis (pulse:jobs) ──► Worker 1..N ──► target APIs
                                                        │ incidents ──► Telegram
```

Component responsibilities: API = auth + CRUD + reads; Scheduler = due-scan +
enqueue only (never HTTP); Worker = HTTP + retry + store + incident eval;
Postgres = persistent state; Redis = job queue only (no business state).

ADRs: ADR-001 monolith over microservices; ADR-002 Redis as queue;
ADR-003 retry-attempt vs check distinction.
