# Load Testing Guide — WA Gateway Concurrent Sessions

**Date:** 2026-08-17  
**Scope:** P2-9 — Load testing concurrent WhatsApp sessions  
**Tools:** Bash script (simple), k6 (advanced)  
**Target:** `services/wa-gateway` (Port 8202)

---

## Overview

Load testing validates that the WA gateway can handle concurrent message sends from multiple tenants under realistic and peak load conditions. This guide provides two approaches:

1. **Bash script** — Simple, no dependencies, good for quick tests
2. **k6** — Advanced, detailed metrics, scenarios, CI-friendly

---

## Quick Start

### Option 1: Simple Bash Script

```bash
cd scripts/loadtest

# Basic test: 10 tenants, 5 messages each
./wa-concurrent-load.sh 10 5

# Stress test: 50 tenants, 10 messages each
./wa-concurrent-load.sh 50 10

# Custom endpoint
./wa-concurrent-load.sh 20 5 http://staging-server:8202
```

**Output:**
```
=== WA Gateway Concurrent Load Test ===
Tenants: 10
Messages per tenant: 5
Total requests: 50
Results: results/20260817_143022

✓ Tenant 1 msg 1: 234ms
✓ Tenant 2 msg 1: 189ms
...

=== Summary ===
- Throughput: 8.33 req/s
- Success: 48 (96%)
- Failed: 2 (4%)
- p50: 245ms
- p95: 1203ms
- p99: 3456ms
```

### Option 2: k6 Advanced Testing

**Prerequisites:**
```bash
# Install k6 (Ubuntu/Debian)
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Or via Homebrew (macOS)
brew install k6
```

**Run tests:**
```bash
cd scripts/loadtest

# Stress test (gradual ramp-up)
k6 run wa-load-test.js

# Custom base URL
k6 run -e BASE_URL=http://staging:8202 wa-load-test.js

# Spike test only
k6 run --scenarios spike_test wa-load-test.js

# With detailed output
k6 run --out json=results.json wa-load-test.js
```

---

## Test Scenarios

### Scenario 1: Baseline Load (10 concurrent tenants)

**Goal:** Verify normal operation under expected load.

**Bash:**
```bash
./wa-concurrent-load.sh 10 5
```

**k6:** (Included in default `wa-load-test.js`)
```javascript
stages: [
  { duration: '2m', target: 10 },  // Ramp up
  { duration: '5m', target: 10 },  // Hold
]
```

**Expected results:**
- Success rate: >95%
- p95 latency: <3s
- Rate limit errors: 0 (rate limiter allows 5 msg/min per tenant)

---

### Scenario 2: Stress Test (50 concurrent tenants)

**Goal:** Find breaking point and verify graceful degradation.

**Bash:**
```bash
./wa-concurrent-load.sh 50 10
```

**k6:**
```javascript
stages: [
  { duration: '2m', target: 20 },
  { duration: '5m', target: 20 },
  { duration: '2m', target: 50 },  // Ramp to stress level
  { duration: '5m', target: 50 },  // Hold stress
]
```

**Expected results:**
- Success rate: >90% (some rate limiting expected)
- p95 latency: <5s
- Rate limit errors: Present (429 responses due to 5 msg/min limit per tenant)

---

### Scenario 3: Spike Test (sudden load burst)

**Goal:** Test resilience to sudden traffic spikes.

**k6 only:**
```javascript
stages: [
  { duration: '30s', target: 5 },
  { duration: '1m', target: 50 },   // Sudden spike
  { duration: '2m', target: 50 },   // Hold spike
  { duration: '30s', target: 5 },   // Drop back
]
```

**Expected results:**
- No crashes or panics
- Circuit breaker activates if Cloud API fails
- Rate limiter prevents whatsmeow overload

---

### Scenario 4: Endurance Test (sustained load)

**Goal:** Detect memory leaks, connection pool exhaustion.

**k6:**
```bash
k6 run --vus 20 --duration 30m wa-load-test.js
```

**Monitor during test:**
```bash
# Memory usage
docker stats wch-wa-gateway

# Connection count
ss -tn | grep :8202 | wc -l

# Logs
docker logs -f wch-wa-gateway
```

---

## Metrics Explained

### Bash Script Metrics

| Metric | Description | Healthy Range |
|:-------|:------------|:--------------|
| **Throughput** | Requests per second | 5-20 req/s (depends on rate limiter) |
| **Success rate** | % of 200 responses | >95% |
| **p50 (median)** | 50th percentile latency | <500ms |
| **p95** | 95th percentile latency | <3s |
| **p99** | 99th percentile latency | <5s |

### k6 Metrics

```
=== Load Test Summary ===

HTTP Requests:
  Total: 1234
  Failed: 12 (0.97%)
  Duration (p95): 2345.67ms
  Duration (p99): 4567.89ms

WA Gateway Metrics:
  Error Rate: 1.23%
  Rate Limit Errors: 5
  Connection Errors: 0
  Message Duration (avg): 567.89ms
  Message Duration (p95): 2345.67ms

Virtual Users:
  Max: 50
  Concurrent (avg): 32.45
```

**Key metrics to watch:**
- `http_req_failed` — Should be <10%
- `rate_limit_errors` — Expected under stress, should not dominate
- `connection_errors` — Should be 0 (401/503 indicates session issues)

---

## Interpreting Results

### Good Results ✅

```
Throughput: 12.5 req/s
Success: 95%
p95: 1.2s
Rate Limit Errors: 3
```

**Interpretation:** System handling load well. Few rate limits (expected for whatsmeow). Low latency.

---

### Degraded Performance ⚠️

```
Throughput: 8.2 req/s
Success: 85%
p95: 4.8s
Rate Limit Errors: 45
Connection Errors: 5
```

**Interpretation:**
- High rate limit errors: Too many tenants hitting 5 msg/min limit
- Connection errors: Some whatsmeow sessions disconnected
- Latency spike: Reconnection backoff or Cloud API fallback

**Actions:**
1. Check if Cloud API is active for transactional messages
2. Verify reconnection backoff working (prevents ban)
3. Monitor PostgreSQL connection pool

---

### System Failure ❌

```
Throughput: 2.1 req/s
Success: 45%
p95: 12.3s
Rate Limit Errors: 120
Connection Errors: 89
```

**Interpretation:** System overloaded or misconfigured.

**Debug:**
```bash
# Check service health
docker ps | grep wa-gateway

# Check logs for panics/errors
docker logs wch-wa-gateway --tail 100

# Check database connections
docker exec wch-postgres psql -U wch_admin -d wch_platform -c "SELECT count(*) FROM pg_stat_activity;"

# Check Redis connections
docker exec wch-redis redis-cli INFO clients
```

---

## Rate Limiting Behavior

### whatsmeow Rate Limiter

**Implementation:** `services/wa-gateway/rate_limiter.go`

```go
// Token bucket: 5 messages per minute per tenant
rateLimiter := NewTokenBucket(5, time.Minute)
```

**Expected behavior:**
- First 5 messages: ✅ Sent immediately
- 6th message within 1 minute: ❌ 429 "Rate limit exceeded"
- After 1 minute: ✅ Bucket refills, messages allowed again

**Load test impact:**
```bash
# 10 tenants, 5 messages each = 50 total
# All should succeed (each tenant sends exactly 5)
./wa-concurrent-load.sh 10 5
# Expected: 100% success

# 10 tenants, 10 messages each = 100 total
# 50 will hit rate limit (each tenant's 6th-10th message)
./wa-concurrent-load.sh 10 10
# Expected: ~50% success, 50% rate limited
```

---

## Cloud API vs whatsmeow Routing

**Routing logic:** `services/wa-gateway/cloud_routing.go`

Messages route based on:
1. `X-Message-Type` header (`otp`, `invoice`, `broadcast` → Cloud API)
2. `X-Source` header (`auth-service`, `billing-service` → Cloud API)
3. Tenant preference (`wa_provider_preference` DB field)

**Load test implications:**
- **No headers** = whatsmeow (5 msg/min limit applies)
- **With transactional headers** = Cloud API (no rate limit, but costs $$)

**Test Cloud API routing:**
```bash
# Modify wa-concurrent-load.sh to add header:
curl -X POST "$BASE_URL/api/wa/send" \
  -H "X-Message-Type: broadcast" \
  -d "tenant_id=..." -d "target=..." -d "message=..."
```

---

## Monitoring During Load Tests

### 1. Real-Time Metrics (Prometheus + Grafana)

```bash
# Start observability stack
docker compose up -d grafana prometheus

# Open Grafana
open http://localhost:3001
```

**Dashboards to watch:**
- WA Gateway message throughput
- Rate limit hits (`wa_rate_limited_total`)
- Fallback rate (`wa_fallback_total` — Cloud API → whatsmeow)
- HTTP request duration (`http_request_duration_ms`)

### 2. Live Logs

```bash
# Follow wa-gateway logs
docker logs -f wch-wa-gateway | grep -E "rate limit|fallback|error"

# Follow wa-cloud-api logs
docker logs -f wch-wa-cloud-api
```

### 3. Database Load

```bash
# Active connections
docker exec wch-postgres psql -U wch_admin -d wch_platform -c \
  "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"

# Slow queries (>1s)
docker exec wch-postgres psql -U wch_admin -d wch_platform -c \
  "SELECT pid, usename, state, query_start, query FROM pg_stat_activity WHERE state != 'idle' AND query_start < NOW() - INTERVAL '1 second';"
```

### 4. Redis Load

```bash
# Connection count
docker exec wch-redis redis-cli INFO clients | grep connected_clients

# Memory usage
docker exec wch-redis redis-cli INFO memory | grep used_memory_human

# Key count (session locks)
docker exec wch-redis redis-cli DBSIZE
```

---

## Common Issues & Fixes

### Issue 1: High Rate Limit Errors (>50%)

**Symptom:**
```
Rate Limit Errors: 245
Success: 45%
```

**Cause:** Sending >5 messages/minute per tenant via whatsmeow.

**Solutions:**
1. **Use Cloud API for high-volume:**
   ```bash
   # Set tenant preference to cloud_api
   psql -c "UPDATE tenant_chatbot_configs SET wa_provider_preference = 'cloud_api' WHERE tenant_id = '...';"
   ```

2. **Increase rate limit** (for testing only):
   ```go
   // services/wa-gateway/rate_limiter.go
   - rateLimiter := NewTokenBucket(5, time.Minute)
   + rateLimiter := NewTokenBucket(20, time.Minute)  // Test only!
   ```

---

### Issue 2: Connection Errors (401/503)

**Symptom:**
```
Connection Errors: 89
HTTP 401: "Not connected to WhatsApp"
```

**Cause:** Whatsmeow sessions expired or not initialized.

**Debug:**
```bash
# Check session status
docker exec wch-postgres psql -U wch_admin -d wch_platform -c \
  "SELECT tenant_id, status FROM wa_sessions;"

# Check QR codes scanned
docker exec wch-postgres psql -U wch_admin -d wch_platform -c \
  "SELECT tenant_id, jid FROM wa_tenant_sessions WHERE jid IS NOT NULL;"
```

**Fix:**
```bash
# Tenants must scan QR code first
curl http://localhost:8202/api/wa/qr?tenant_id=test-tenant-1
# Scan the returned QR code with WhatsApp mobile app
```

---

### Issue 3: Slow Response Times (p95 >10s)

**Symptom:**
```
p95: 12345ms
p99: 23456ms
```

**Possible causes:**
1. **Reconnection backoff** — Sessions disconnecting during test
2. **Database slow** — N+1 queries or missing indexes
3. **Cloud API timeout** — Fallback taking too long

**Debug:**
1. Check reconnection logs:
   ```bash
   docker logs wch-wa-gateway | grep "reconnect"
   ```

2. Check database performance:
   ```bash
   # Enable slow query log (threshold: 500ms)
   docker exec wch-postgres psql -U wch_admin -c \
     "ALTER SYSTEM SET log_min_duration_statement = 500;"
   docker restart wch-postgres
   ```

3. Check Cloud API fallback rate:
   ```bash
   # Prometheus query
   rate(wa_fallback_total[5m])
   ```

---

### Issue 4: Memory Leak (OOM after long test)

**Symptom:**
```bash
docker stats wch-wa-gateway
# CONTAINER   MEM USAGE / LIMIT     MEM %
# wa-gateway  1.8GiB / 2GiB         90%   # Growing over time
```

**Debug:**
```bash
# Enable Go memory profiler
docker exec wch-wa-gateway curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check for goroutine leaks
docker exec wch-wa-gateway curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

**Common causes:**
- Goroutine leak in reconnection logic
- Unbounded `reconnectAttempts` map (see `docs/WA_RECONNECT_PERFORMANCE_PROFILE.md` for fix)

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2am UTC
  workflow_dispatch:     # Manual trigger

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start services
        run: docker compose up -d postgres redis wa-gateway
      
      - name: Wait for services
        run: sleep 30
      
      - name: Run load test
        run: |
          cd scripts/loadtest
          chmod +x wa-concurrent-load.sh
          ./wa-concurrent-load.sh 20 5 http://localhost:8202
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: scripts/loadtest/results/
      
      - name: Check thresholds
        run: |
          SUCCESS_RATE=$(tail -n 20 scripts/loadtest/results/*/summary.txt | grep "Success:" | awk '{print $2}' | sed 's/%//')
          if [ "$SUCCESS_RATE" -lt 90 ]; then
            echo "Load test failed: success rate $SUCCESS_RATE% < 90%"
            exit 1
          fi
```

---

## Best Practices

### 1. Realistic Load Profiles

**❌ Bad:**
```bash
# Unrealistic: all tenants send at exact same time
for i in {1..100}; do
  curl ... &
done
wait
```

**✅ Good:**
```bash
# Realistic: stagger requests, add think time
for i in {1..100}; do
  curl ... &
  sleep 0.$((RANDOM % 10))  # Random 0-1s delay
done
```

### 2. Warm-Up Period

Start with low load to warm up:
- Database connection pool
- Redis connection pool
- HTTP keep-alive connections
- LLM/Cloud API connections

**k6 example:**
```javascript
stages: [
  { duration: '1m', target: 5 },   // Warm-up
  { duration: '5m', target: 50 },  // Actual test
]
```

### 3. Monitor System Resources

Track during test:
- CPU usage (`docker stats`)
- Memory usage
- Network I/O
- Database connections
- Redis memory

### 4. Clean Up Between Tests

```bash
# Reset rate limiter state
docker exec wch-redis redis-cli FLUSHDB

# Reset test tenant sessions
docker exec wch-postgres psql -U wch_admin -d wch_platform -c \
  "DELETE FROM wa_sessions WHERE tenant_id LIKE 'test-tenant-%';"
```

---

## Summary

**Created:**
- `scripts/loadtest/wa-concurrent-load.sh` — Simple bash load test (no dependencies)
- `scripts/loadtest/wa-load-test.js` — Advanced k6 load test with scenarios

**Test Scenarios:**
1. Baseline (10 tenants, 5 msg/tenant)
2. Stress (50 tenants, 10 msg/tenant)
3. Spike (5→50→5 tenants)
4. Endurance (20 tenants, 30 minutes)

**Key Metrics:**
- Throughput (req/s)
- Success rate (%)
- Latency (p50, p95, p99)
- Rate limit errors
- Connection errors

**Run Tests:**
```bash
# Simple test
cd scripts/loadtest
./wa-concurrent-load.sh 10 5

# Advanced test (requires k6)
k6 run wa-load-test.js
```

**Next Steps:**
1. Run baseline test to establish performance baseline
2. Run stress test to find breaking point
3. Set up CI/CD for daily load tests
4. Monitor Grafana dashboards during tests
5. Tune rate limiter and connection pools based on results
