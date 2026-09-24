# ADR-001 — Modular monolith, not microservices

Status: accepted. Context: solo portfolio project, one deployable.
Decision: single Go module/binary with `-mode=api|worker|scheduler`.
Consequence: no network boundaries to mock; scale out later by running more
worker containers against the same Redis queue. Split services only when a
component needs independent deploy/scale (not now).
