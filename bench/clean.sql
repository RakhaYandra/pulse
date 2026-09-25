-- Bench + e2e cleanup: remove seeded/test monitors + incidents/checks
-- (cascades via FK), bench users, and flush the job queue claim keys.
-- Run: docker compose exec -T postgres psql -U pulse -d pulse -f - < bench/clean.sql
-- Then flush redis: docker compose exec -T redis redis-cli FLUSHDB
-- Schedule: after every bench run and e2e batch (both seed throwaway data).
DELETE FROM monitors WHERE name LIKE 'bench-%' OR name LIKE 'E2E %';
DELETE FROM users WHERE email LIKE 'bench%@pulse.local' OR email LIKE 'e2e%@pulse.local' OR email LIKE 'qa[0-9]%@pulse.local';
