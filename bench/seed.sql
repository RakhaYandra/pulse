-- Benchmark seeder. Run with psql against the pulse database.
-- Usage: N=100 MIX_OK=70 MIX_SLOW=15 MIX_FLAKY=10 MIX_TIMEOUT=5 psql ... -f seed.sql
-- Simpler: \set N 100 then fixed 70/15/10/5 mix below.
--
-- Requires a bench user row; create one via API register first, then:
--   UPDATE bench_seed SET user_id = '<uuid>' — or pass PSQL var USER_ID.

\if :{?USER_ID}
\else
\echo 'Pass -v USER_ID=<uuid> from a registered user'
\endif

WITH gen AS (
  SELECT
    g AS i,
    CASE
      WHEN g % 100 < 70 THEN 'http://stub:8099/ok'
      WHEN g % 100 < 85 THEN 'http://stub:8099/slow'
      WHEN g % 100 < 95 THEN 'http://stub:8099/flaky'
      ELSE 'http://stub:8099/timeout'
    END AS url,
    CASE
      WHEN g % 100 < 70 THEN 'bench-ok-' || g
      WHEN g % 100 < 85 THEN 'bench-slow-' || g
      WHEN g % 100 < 95 THEN 'bench-flaky-' || g
      ELSE 'bench-timeout-' || g
    END AS nm
  FROM generate_series(1, :N) g
)
INSERT INTO monitors(id, user_id, name, url, method, interval_seconds, timeout_seconds,
  failure_threshold, recovery_threshold, status, is_active)
SELECT gen_random_uuid(), :'USER_ID', nm, url, 'GET', 60, 5, 3, 2, 'UNKNOWN', TRUE
FROM gen;
