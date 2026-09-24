# Pulse — Architecture

Full Clean Architecture (ADR-005), one Go binary with 3 run modes.

```
pulse-web (React)
   │  JSON contract (Newman-guarded)
   ▼
delivery/http ──► usecase ◄── delivery/worker, delivery/scheduler
                      │  ▲
                      ▼  │ ports owned by usecase
               infrastructure
               (postgres, security, httpcheck, queue, notify)
                      │
                      ▼
        PostgreSQL · Redis (pulse:jobs) · target APIs · Telegram
```

Dependency rule: delivery → infrastructure → usecase → domain.
Domain: stdlib only. Usecase: domain + stdlib + uuid.
Inner layers never import delivery/infrastructure (grep-verified).

Use-case flow (ProcessCheck): ActiveMonitor → Checker.Check → CheckRepo.Append
→ RecordStatus → RecentStatuses → EvalStreak → Open/Resolve → Transition.
Delivery notifies only on non-nil Transition. One bad monitor can't kill
workers (per-job recover in runner).

Repos: [pulse](https://github.com/RakhaYandra/pulse) (this) ·
[pulse-web](https://github.com/RakhaYandra/pulse-web) (dashboard).
