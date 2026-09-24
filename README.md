# Pulse — API Monitoring & Incident Platform

> Ekosistem: [api](https://github.com/RakhaYandra/pulse) · [web](https://github.com/RakhaYandra/pulse-web)

Go, Gin, PostgreSQL, Redis, Docker. Full Clean Architecture.

Self-hosted platform that health-checks APIs on a schedule, stores results,
opens incidents on configurable failure thresholds, resolves on recovery, and
notifies via Telegram. Monitoring runs in background workers — never in the
dashboard request lifecycle.

## Quickstart

```bash
cp .env.example .env   # fill TELEGRAM_* to enable notifications (optional)
docker compose up -d --build
curl localhost:8080/health
# dashboard: https://github.com/RakhaYandra/pulse-web
```

Demo login: `demo@pulse.local / demo1234` (register your own for isolation).

## Verification (all green, 2026-09-24)

| Check | Result |
|---|---|
| Engine E2E (real) | 404-monitor → OPEN after 2 fails → RESOLVED after recovery, dashboard correct |
| Newman | 20/20 (`qa/collection.json`) |
| Go unit | 4/4 (`checker`: UP/DOWN/TIMEOUT/ERROR) |
| Playwright e2e | 5/5 (`qa/e2e`) |
| Avg API latency (Newman) | ~10ms local |

## Key engineering points

1. API layer separated from execution via Redis-backed workers.
2. Retry (≤3 attempts) collapses into one check — uptime math stays correct.
3. Incident state machine: fail streak → OPEN, success streak → RESOLVED.
4. Failure isolation: per-job `recover()`, one bad monitor can't kill workers.
5. Queue holds no business state — loss only delays, never corrupts.

## Layout

`backend/` (Go, Clean Architecture: domain/usecase/infrastructure/delivery)
`qa/` (Newman + Playwright) · `docs/` (BRD, PRD, Architecture, ADRs)

## Future (post-MVP)

Concurrency benchmark 100→1000 monitors, `/metrics` + Prometheus/Grafana,
MTTR/SLA reports, self-monitoring, single-VPS deploy.
