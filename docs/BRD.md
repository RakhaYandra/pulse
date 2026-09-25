# Pulse — BRD (Business Requirements Document)

## 1. Problem
Developers/small teams own several APIs but check health manually or rely on
infra-provider monitoring. Downtime is discovered late, via users.

## 2. Objective
A simple self-hosted platform to register API endpoints, health-check them
periodically, store results, detect repeated failures as incidents, and show
reliability history.

## 3. Stakeholders
- Primary: individual developers, small teams (own REST APIs, side projects)
- Secondary: QA (staging/test env monitoring)

## 4. Scope (MVP)
IN: auth, monitor CRUD + pause/resume, scheduled HTTP checks, check history,
incident open/resolve on thresholds, Telegram notify on transitions, dashboard.
OUT: teams/billing, SMS/email, multi-region, K8s/Terraform, complex RBAC.

## 5. Success criteria
- Monitor with failure_threshold=3 opens an incident after 3 consecutive failures.
- Recovery_threshold successes resolve it; Telegram received on both transitions.
- Newman 20/20, unit 4/4, e2e 5/5.
