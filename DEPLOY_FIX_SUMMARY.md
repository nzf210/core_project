# Deployment Fix Summary

## Problem
GitHub Actions deployment ke VPS staging gagal dengan error:
```
Container wch-stg-pgbouncer Error dependency pgbouncer failed to start
Container wch-stg-redis Error dependency redis failed to start
dependency failed to start: container wch-stg-pgbouncer is unhealthy
```

## Root Cause
**Race condition**: Workflow langsung start pgbouncer tanpa memastikan postgres dan redis sudah healthy dulu.

PgBouncer depends on:
- `postgres` (service_healthy condition)
- `redis` (untuk session cache)

Tapi workflow original (line 254-260) langsung `up -d pgbouncer` tanpa wait loop untuk ensure dependencies ready.

## Fix Applied
Modified `.github/workflows/deploy.yml` (line 254-292):

**Before:**
```yaml
docker compose pull pgbouncer
docker compose up -d pgbouncer
# wait pgbouncer (tapi postgres/redis belum guaranteed ready)
```

**After:**
```yaml
# 1. Start postgres + redis first
docker compose up -d postgres redis

# 2. Wait postgres healthy (30 retries × 5s)
for i in $(seq 1 30); do
  docker exec wch-stg-postgres pg_isready ...
done

# 3. Wait redis healthy (12 retries × 3s)
for i in $(seq 1 12); do
  docker exec wch-stg-redis redis-cli ping ...
done

# 4. Baru start pgbouncer
docker compose up -d pgbouncer

# 5. Wait pgbouncer ready
for i in $(seq 1 12); do
  docker exec wch-stg-pgbouncer nc -z 127.0.0.1 6432 ...
done
```

## Testing
1. Commit fix:
```bash
git add .github/workflows/deploy.yml
git commit -m "fix: ensure postgres and redis healthy before starting pgbouncer"
git push origin main
```

2. Trigger deployment:
```bash
git tag stg-be-v1.0.1
git push origin stg-be-v1.0.1
```

3. Monitor GitHub Actions:
https://github.com/nzf210/core_project/actions

## Expected Result
- Postgres start → healthy
- Redis start → healthy  
- PgBouncer start → healthy (no more dependency errors)
- Backend services deploy successfully

## Rollback Plan
Jika deployment masih gagal:
1. SSH ke VPS: `ssh -p 3209 deploy@157.15.40.27`
2. Check logs:
```bash
cd /opt/wch-staging
docker compose -f docker-compose.yml -f docker-compose.staging.yml logs postgres
docker compose -f docker-compose.yml -f docker-compose.staging.yml logs redis
docker compose -f docker-compose.yml -f docker-compose.staging.yml logs pgbouncer
```
3. Manual recovery:
```bash
COMPOSE_PROJECT_NAME=wch-stg docker compose \
  -f docker-compose.yml -f docker-compose.staging.yml \
  up -d postgres redis

# Wait sampai healthy
docker ps

# Baru start pgbouncer
COMPOSE_PROJECT_NAME=wch-stg docker compose \
  -f docker-compose.yml -f docker-compose.staging.yml \
  up -d pgbouncer
```

## Files Modified
- `.github/workflows/deploy.yml` (line 254-292)
- `.claude/settings.local.json` (fix Redis permission patterns — sudah done)

## Related Docs
- `docs/DEPLOY_SSH_TIMEOUT_FIX.md` — SSH connectivity troubleshooting
- `docs/DEPLOYMENT_RUNBOOK.md` — Production deployment procedures
- `docs/PGBOUNCER_STATUS.md` — PgBouncer configuration reference
