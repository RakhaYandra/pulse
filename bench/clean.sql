-- Bench cleanup: remove all seeded bench monitors + their incidents/checks
-- (cascades via FK), bench users, and flush the job queue claim keys.
-- Run: docker compose exec -T postgres psql -U pulse -d pulse -f - < bench/clean.sql
-- Then flush redis: docker compose exec -T redis redis-cli FLUSHDB
DELETE FROM monitors WHERE name LIKE 'bench-%';
DELETE FROM users WHERE email LIKE 'bench%@pulse.local';
