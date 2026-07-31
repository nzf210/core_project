# PgBouncer Implementation Guide — WCH Platform

**Status:** Ready for Implementation  
**Last Updated:** 2026-07-31

## 📊 Problem Statement

WCH Platform memiliki **12 backend services** yang masing-masing membuat connection pool langsung ke PostgreSQL:

```
Current State (WITHOUT PgBouncer):
├─ 12 services × default pool (10-20 conns) = 120-240 connections
├─ PostgreSQL max_connections = 100 (default)
├─ ❌ Connection exhaustion under load
├─ ❌ No connection reuse across services
└─ ❌ Idle connections waste resources
```

## ✅ Solution: PgBouncer Connection Pooler

```
Target Architecture (WITH PgBouncer):
┌─────────────────────────────────────────────────────────────┐
│  12 Backend Services (Go)                                   │
│  Each: pool_size=5 → Total 60 app connections              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
         ┌───────────────┐
         │   PgBouncer   │  Pool Mode: Transaction
         │   Port: 6432  │  Max client: 100
         └───────┬───────┘  Pool size: 25 per database
                 │          (60% connection reduction)
                 ▼
         ┌───────────────┐
         │  PostgreSQL   │  max_connections: 50
         │   Port: 5432  │  (vs 100 sebelumnya)
         └───────────────┘
```

## 🎯 Benefits

1. **60% Connection Reduction** — 60 app connections → 25 DB connections
2. **Transaction Pooling** — Connection released immediately post-commit
3. **Lower Latency** — No TCP handshake per query (~3-5ms saved)
4. **Resource Efficiency** — PostgreSQL uses 50% fewer backend processes
5. **Horizontal Scaling Ready** — Add instances without exhausting DB pool

## 📦 Files Created

```
infra/pgbouncer/
├── pgbouncer.ini           # PgBouncer configuration
├── userlist.txt.example    # Auth template (generate hash)
└── Dockerfile              # PgBouncer container image

scripts/
└── generate-pgbouncer-userlist.sh  # Hash generator script

docker-compose.yml          # Updated with pgbouncer service
```

## 🚀 Deployment Steps

### Step 1: Generate PgBouncer User Hash

```bash
# Generate MD5 hash for PgBouncer auth
cd /home/syahril/dev/core_project

# Generate userlist.txt (replace with your DB password)
./scripts/generate-pgbouncer-userlist.sh wch_admin "YOUR_DB_PASSWORD" infra/pgbouncer/userlist.txt

# Output example:
# ✅ Generated userlist.txt at infra/pgbouncer/userlist.txt
#    Username: wch_admin
#    Hash: md5abc123def456...
```

### Step 2: Update Environment Variables

Edit `.env`:

```bash
# OLD (direct PostgreSQL connection)
DB_HOST=127.0.0.1
DB_PORT=5433

# NEW (via PgBouncer)
DB_HOST=127.0.0.1
DB_PORT=6432            # PgBouncer port

# Keep PostgreSQL credentials same
DB_USER=wch_admin
DB_PASSWORD=your_password
DB_NAME=wch_platform
```

### Step 3: Start PgBouncer

```bash
# Start PostgreSQL + PgBouncer
docker compose up -d postgres pgbouncer

# Check PgBouncer logs
docker compose logs -f pgbouncer

# Expected output:
# LOG kernel file descriptor limit: 1048576 (hard: 1048576)
# LOG listening on 0.0.0.0:6432
# LOG process up: PgBouncer 1.21.0
```

### Step 4: Verify PgBouncer Connection

```bash
# Test connection via PgBouncer
psql -h 127.0.0.1 -p 6432 -U wch_admin -d wch_platform -c "SELECT version();"

# Should return PostgreSQL version (connected via PgBouncer)

# Check PgBouncer stats (admin database)
psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer -c "SHOW POOLS;"

# Expected output:
#  database     | user      | cl_active | cl_waiting | sv_active | sv_idle
# -------------+-----------+-----------+------------+-----------+---------
#  wch_platform | wch_admin |         0 |          0 |         0 |       5
#  pgbouncer    | pgbouncer |         1 |          0 |         0 |       0
```

### Step 5: Update Backend Services (Optional Optimization)

Untuk production, tambahkan pool configuration di setiap service:

**Example: `services/auth-service/db.go`**

```go
func initDB(cfg *config.Config) error {
    dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
        cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

    // Parse connection string
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return fmt.Errorf("unable to parse DSN: %w", err)
    }

    // Configure pool (smaller pools via PgBouncer)
    config.MaxConns = 5                    // Down from default 10-20
    config.MinConns = 2                    // Keep 2 connections ready
    config.MaxConnLifetime = 1 * time.Hour // Recycle connections
    config.MaxConnIdleTime = 10 * time.Minute

    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return fmt.Errorf("unable to connect to database: %w", err)
    }

    if err := pool.Ping(context.Background()); err != nil {
        return fmt.Errorf("unable to ping database: %w", err)
    }

    DB = pool
    slog.Info("✅ Connected to PostgreSQL via PgBouncer")
    return nil
}
```

**Apply same pattern to:**
- `services/billing-service/db.go`
- `services/ai-gateway/db.go`
- `apps/umkm/accounting/main.go` (DB init)
- `apps/campaign/api/repository/db.go`
- All other services with `pgxpool.New()`

### Step 6: Restart All Services

```bash
# Native dev (hot-reload)
make stop-all
make dev-all

# Docker
docker compose down
docker compose up -d
```

### Step 7: Monitoring & Verification

```bash
# Monitor PgBouncer stats
watch -n 2 'psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer -c "SHOW POOLS;"'

# Monitor PostgreSQL connections
watch -n 2 'psql -h 127.0.0.1 -p 5433 -U wch_admin -d wch_platform -c "SELECT count(*) FROM pg_stat_activity WHERE state = '\''active'\'';"'

# Check PgBouncer metrics (Prometheus exporter available)
curl http://localhost:6432/metrics  # If metrics enabled
```

## 🔧 PgBouncer Configuration Explained

### Pool Modes

| Mode | Best For | Connection Lifecycle | Limitations |
|:-----|:---------|:---------------------|:------------|
| **transaction** ✅ | OLTP apps (WCH) | Released after COMMIT/ROLLBACK | No session state (temp tables, SET vars) |
| **session** | Apps with session state | Released on disconnect | More connections needed |
| **statement** | Simple queries only | Released after each query | NO transactions, NO prepared statements |

**WCH Platform uses `transaction` mode** — optimal untuk REST API pattern.

### Key Parameters

```ini
# Connection Limits
max_client_conn = 100          # Max connections from all apps
default_pool_size = 25         # Connections per database to PostgreSQL
min_pool_size = 5              # Always keep 5 ready (latency optimization)
reserve_pool_size = 5          # Emergency pool jika default full
reserve_pool_timeout = 3       # Wait 3s for reserved pool

# Lifecycle
server_lifetime = 3600         # Recycle connection after 1 hour
server_idle_timeout = 600      # Close idle connection after 10 min
server_check_delay = 30        # Health check interval

# Cleanup
server_reset_query = DISCARD ALL  # Reset session state on return to pool
```

### Auth Methods

| Method | Security | Performance | Use Case |
|:-------|:---------|:------------|:---------|
| `md5` ✅ | Medium | Fast | WCH Platform (current) |
| `scram-sha-256` | High | Fast | PostgreSQL 14+ with SCRAM |
| `cert` | Highest | Fast | TLS client certificates |
| `plain` | ❌ Low | Fast | Dev only (NOT production) |

## 📊 Expected Performance Improvements

### Before PgBouncer (Direct Connection)

```
Scenario: 100 concurrent requests
├─ Each request: New TCP handshake (3-5ms overhead)
├─ Connection pool exhaustion under spike
├─ PostgreSQL: 100+ active backends
└─ Response time p95: 150ms
```

### After PgBouncer

```
Scenario: 100 concurrent requests
├─ No TCP handshake (reuse pooled connections)
├─ Connection queueing (no exhaustion)
├─ PostgreSQL: 25 active backends (stable)
└─ Response time p95: 120ms (~20% improvement)
```

### Load Test Results (Expected)

| Metric | Before | After | Improvement |
|:-------|:-------|:------|:------------|
| Active DB Connections | 80-120 | 20-30 | 70% ↓ |
| Connection Latency | 3-5ms | <1ms | 75% ↓ |
| Query Throughput | 1000 qps | 1400 qps | 40% ↑ |
| PostgreSQL CPU | 60% | 40% | 33% ↓ |

## 🐛 Troubleshooting

### Error: "No such database: pgbouncer"

**Cause:** Trying to query admin database without proper setup.

**Solution:**
```bash
# Admin database is special (lowercase)
psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer -c "SHOW STATS;"
```

### Error: "Authentication failed"

**Cause:** MD5 hash mismatch in `userlist.txt`.

**Solution:**
```bash
# Regenerate hash
./scripts/generate-pgbouncer-userlist.sh wch_admin "YOUR_PASSWORD" infra/pgbouncer/userlist.txt

# Restart PgBouncer
docker compose restart pgbouncer
```

### Error: "No more connections allowed"

**Cause:** `max_client_conn` exceeded.

**Solution:**
```bash
# Check active connections
psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer -c "SHOW CLIENTS;"

# Increase limit in pgbouncer.ini
max_client_conn = 200  # Up from 100

docker compose restart pgbouncer
```

### Error: "Server connection lost" or random disconnects

**Cause:** `server_lifetime` too aggressive, or PostgreSQL `idle_in_transaction_session_timeout`.

**Solution:**
```ini
# pgbouncer.ini
server_lifetime = 7200         # Increase to 2 hours
server_idle_timeout = 1200     # Increase to 20 min
```

### High `cl_waiting` (clients waiting for connection)

**Cause:** Pool size too small for load.

**Solution:**
```ini
# pgbouncer.ini
default_pool_size = 40         # Up from 25
reserve_pool_size = 10         # Up from 5
```

## 🔐 Security Best Practices

### 1. Firewall Rules

```bash
# PostgreSQL should NOT be exposed directly
# Only PgBouncer should be accessible

# UFW example (production)
sudo ufw allow 6432/tcp comment 'PgBouncer'
sudo ufw deny 5432/tcp  comment 'PostgreSQL - internal only'
```

### 2. TLS Encryption

**Enable TLS between PgBouncer ↔ PostgreSQL:**

```ini
# pgbouncer.ini
[databases]
wch_platform = host=postgres port=5432 dbname=wch_platform sslmode=require

[pgbouncer]
client_tls_sslmode = require
client_tls_cert_file = /etc/pgbouncer/server.crt
client_tls_key_file = /etc/pgbouncer/server.key
```

### 3. Auth File Permissions

```bash
# userlist.txt must be readable only by PgBouncer
chmod 600 infra/pgbouncer/userlist.txt

# Owner: root or pgbouncer user
chown pgbouncer:pgbouncer infra/pgbouncer/userlist.txt  # Production
```

### 4. Separate User for PgBouncer

```sql
-- Create dedicated DB user (lower privileges)
CREATE USER pgbouncer_user WITH PASSWORD 'strong_password';
GRANT CONNECT ON DATABASE wch_platform TO pgbouncer_user;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO pgbouncer_user;
-- NO DELETE, TRUNCATE, DROP permissions
```

## 📈 Monitoring & Metrics

### PgBouncer Admin Commands

```sql
-- Connect to admin database
psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer

-- Show pools
SHOW POOLS;

-- Show client connections
SHOW CLIENTS;

-- Show server connections (to PostgreSQL)
SHOW SERVERS;

-- Show statistics
SHOW STATS;

-- Show configuration
SHOW CONFIG;

-- Reload config without restart
RELOAD;

-- Pause all connections (maintenance)
PAUSE;

-- Resume
RESUME;
```

### Grafana Dashboard

Add PgBouncer metrics to Grafana:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'pgbouncer'
    static_configs:
      - targets: ['pgbouncer:9127']  # pgbouncer_exporter
```

**Key Metrics:**
- `pgbouncer_pools_cl_active` — Active client connections
- `pgbouncer_pools_cl_waiting` — Clients waiting for connection
- `pgbouncer_pools_sv_active` — Active server connections
- `pgbouncer_pools_sv_idle` — Idle server connections
- `pgbouncer_stats_queries_total` — Query throughput

### Alert Rules

```yaml
# infra/observability/prometheus/alerts.yml
- alert: PgBouncerHighWaiting
  expr: pgbouncer_pools_cl_waiting > 10
  for: 2m
  annotations:
    summary: "High client wait queue ({{ $value }} clients waiting)"

- alert: PgBouncerPoolExhaustion
  expr: pgbouncer_pools_sv_idle == 0
  for: 1m
  annotations:
    summary: "PgBouncer pool exhausted (no idle connections)"
```

## 🔄 Migration Checklist

- [ ] Generate `userlist.txt` with correct hash
- [ ] Update `.env` — `DB_HOST=127.0.0.1`, `DB_PORT=6432`
- [ ] Start PgBouncer: `docker compose up -d pgbouncer`
- [ ] Verify connection: `psql -h 127.0.0.1 -p 6432 -U wch_admin -d wch_platform`
- [ ] Check PgBouncer stats: `SHOW POOLS;`
- [ ] Update backend services with smaller pool sizes (MaxConns=5)
- [ ] Restart all services: `make stop-all && make dev-all`
- [ ] Monitor for 24 hours (check `cl_waiting` and query latency)
- [ ] Load test: `hey -n 10000 -c 100 http://localhost:8000/health`
- [ ] Compare before/after PostgreSQL connection count
- [ ] Update production deployment docs (`docs/DEPLOYMENT_RUNBOOK.md`)

## 📚 References

- [PgBouncer Official Docs](https://www.pgbouncer.org/usage.html)
- [PostgreSQL Connection Pooling](https://wiki.postgresql.org/wiki/Number_Of_Database_Connections)
- [pgx Connection Pool Best Practices](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx#connection-pool)

## 🆘 Support

**Production Issues:**
- Monitor: `docker compose logs -f pgbouncer`
- Admin console: `psql -h 127.0.0.1 -p 6432 -U wch_admin -d pgbouncer`
- Emergency bypass: Change `DB_PORT=5433` to connect directly to PostgreSQL

**Rollback Plan:**
```bash
# Stop PgBouncer
docker compose stop pgbouncer

# Revert .env
DB_PORT=5433  # Direct PostgreSQL

# Restart services
make stop-all && make dev-all
```

---

**Last Updated:** 2026-07-31  
**Owner:** DevOps Team  
**Status:** Ready for Implementation
