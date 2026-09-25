# Pulse Backup & Restore

Single-node scope: Postgres volume + Redis AOF. No replicas by design.

## Postgres (source of truth)

```bash
# backup
docker compose exec -T postgres pg_dump -U pulse -d pulse > backup-$(date +%F).sql
# restore (stop writers first)
docker compose stop api scheduler worker-1 worker-2 worker-3 worker-4
cat backup-YYYY-MM-DD.sql | docker compose exec -T postgres psql -U pulse -d pulse
docker compose start api scheduler worker-1 worker-2
```

## Redis (ephemeral by design)

Jobs + dedup claims re-derive from Postgres on next scheduler tick, so a
queue loss only delays checks. AOF (`--appendonly yes` + `redisdata` volume)
preserves them across restarts anyway:

```bash
docker compose exec -T redis redis-cli LASTSAVE
```

## Volumes

```bash
docker volume ls | grep pulse
docker run --rm -v pulse_pgdata:/src -v "$PWD":/dst alpine tar -czf /dst/pgdata.tgz -C /src .
```

## What is NOT covered

Point-in-time recovery, off-site copies, PG replicas, Redis Sentinel —
all out of scope for self-hosted single node (see docs).
