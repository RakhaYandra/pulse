# ADR-004 — Split frontend to pulse-web

Status: accepted (2026-09-25).

Context: MVP shipped as monorepo (`backend/ + frontend/`). Portfolio pattern
(Shiftbase/LifeOS/KosManager) is api/web split; frontend now has its own
Dockerfile + release cadence.

Decision: 2 repos — `pulse` (Go engine + qa + docs + compose) and `pulse-web`
(React, `VITE_API_URL` build arg). No 6-repo split: data/ops stay lite inside
`pulse`. Frontend history preserved via `git subtree split -P frontend`.

Consequence: API contract is the boundary (Newman 20/20 guards it); e2e
`BASE_URL` env lets QA run against either repo's frontend.
