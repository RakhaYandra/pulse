# ADR-005 — Full Clean Architecture (not lite)

Status: accepted (2026-09-25). Amends ADR-001 (modular monolith stands;
layering goes full CA instead of lite).

Context: MVP handlers held `*sql.DB` with SQL inline; entities carried JSON
tags (transport leaking inward); incident logic untestable without Postgres.

Decision: `domain/` (entities w/o json tags, typed statuses, validation,
domain errors, pure `EvalStreak`) → `usecase/` (ports owned here, per-use-case
DTOs, services return domain errors) → `infrastructure/` (postgres repos +
row mappers, bcrypt/JWT, Redis queue, HTTP checker, Telegram formatter) →
`delivery/` (Gin handlers thin, request/response DTOs, single error_map,
worker/scheduler runners). `cmd/pulse` is the composition root.

Rules enforced by grep: domain = stdlib only; usecase = domain + stdlib +
uuid (deterministic util, no I/O); inner layers never import delivery/infra.

Consequences:
- Incident streak logic unit-tested with fake repos (no DB).
- Queue payload simplified to plain monitorID (was JSON); queue is
  ephemeral so no migration needed.
- PATCH full-replace quirk (omitted thresholds reset to defaults) preserved
  as documented behavior, not silently changed.
- API JSON contract byte-identical (Newman 20/20 before/after).
