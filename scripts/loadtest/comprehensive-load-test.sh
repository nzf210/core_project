#!/bin/bash
# Comprehensive Load Test - WCH Platform
# Tests: HTTP endpoints, RabbitMQ queue capacity, Database connection pool
# Usage: ./scripts/loadtest/comprehensive-load-test.sh [environment]

set -euo pipefail

ENV=${1:-dev}
RESULTS_DIR="results/$(date +'%Y%m%d_%H%M%S')"
mkdir -p "$RESULTS_DIR"

# Configuration based on environment
case $ENV in
  dev)
    BASE_URL="http://localhost:8000"
    RABBITMQ_UI="http://localhost:10673"
    ;;
  staging)
    BASE_URL="http://157.15.40.27:21000"
    RABBITMQ_UI="http://157.15.40.27:20673"
    ;;
  *)
    echo "Unknown environment: $ENV"
    exit 1
    ;;
esac

log() {
  echo "[$(date +'%H:%M:%S')] $*" | tee -a "$RESULTS_DIR/test.log"
}

log "=== WCH Platform Comprehensive Load Test ==="
log "Environment: $ENV"
log "Base URL: $BASE_URL"
log "Results: $RESULTS_DIR"

# ========================================
# Test 1: Baseline API Load (1K requests)
# ========================================
log ""
log "[Test 1/5] Baseline API Load - 1000 requests, 50 concurrent"

cat > "$RESULTS_DIR/baseline-test.js" <<'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up
    { duration: '2m', target: 50 },    // Hold
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<3000'],  // 95% < 3s
    'errors': ['rate<0.05'],              // Error < 5%
  },
};

export default function () {
  const res = http.get(__ENV.BASE_URL + '/health');

  check(res, {
    'status 200': (r) => r.status === 200,
  }) || errorRate.add(1);

  sleep(1);
}
EOF

if command -v k6 &>/dev/null; then
  k6 run -e BASE_URL="$BASE_URL" \
    --summary-export="$RESULTS_DIR/baseline-summary.json" \
    "$RESULTS_DIR/baseline-test.js" 2>&1 | tee "$RESULTS_DIR/baseline-output.txt"
else
  log "⚠️  k6 not installed - skipping baseline test"
  log "   Install: brew install k6 (macOS) or apt-get install k6 (Ubuntu)"
fi

# ========================================
# Test 2: RabbitMQ Queue Capacity
# ========================================
log ""
log "[Test 2/5] RabbitMQ Queue Capacity - Publish 10K jobs"

cat > "$RESULTS_DIR/queue-test.sh" <<'QEOF'
#!/bin/bash
JOBS=10000
BATCH=100
RABBITMQ_URL=${1:-amqp://wch_admin:rabbitmq_pass@localhost:10672/}

echo "Publishing $JOBS jobs to RabbitMQ in batches of $BATCH..."

for ((i=0; i<JOBS; i+=BATCH)); do
  for ((j=0; j<BATCH; j++)); do
    curl -s -u "wch_admin:rabbitmq_pass" \
      -H "content-type:application/json" \
      -X POST "http://localhost:10673/api/exchanges/%2f/amq.default/publish" \
      -d "{\"properties\":{},\"routing_key\":\"test.queue\",\"payload\":\"test-job-$((i+j))\",\"payload_encoding\":\"string\"}" \
      >/dev/null &
  done
  wait
  echo -ne "\rProgress: $((i+BATCH))/$JOBS jobs"
done

echo ""
echo "✓ Published $JOBS jobs"

# Check queue depth
DEPTH=$(curl -s -u "wch_admin:rabbitmq_pass" http://localhost:10673/api/queues/%2f/test.queue | jq '.messages')
echo "Queue depth: $DEPTH messages"
QEOF

chmod +x "$RESULTS_DIR/queue-test.sh"

if [[ $ENV == "dev" ]]; then
  bash "$RESULTS_DIR/queue-test.sh" 2>&1 | tee "$RESULTS_DIR/queue-output.txt"
else
  log "⚠️  Queue test only runs in dev environment"
fi

# ========================================
# Test 3: Database Connection Pool Stress
# ========================================
log ""
log "[Test 3/5] Database Connection Pool - 500 concurrent queries"

cat > "$RESULTS_DIR/db-test.js" <<'EOF'
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '20s', target: 500 },   // Spike to 500
    { duration: '30s', target: 500 },   // Hold
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  // Query endpoint that hits database
  const res = http.get(__ENV.BASE_URL + '/api/umkm/settings', {
    headers: {
      'Authorization': 'Bearer test-token',
      'X-Tenant-ID': 'test-tenant-1',
    },
  });

  check(res, {
    'not 5xx': (r) => r.status < 500,
  });
}
EOF

if command -v k6 &>/dev/null; then
  k6 run -e BASE_URL="$BASE_URL" \
    --summary-export="$RESULTS_DIR/db-summary.json" \
    "$RESULTS_DIR/db-test.js" 2>&1 | tee "$RESULTS_DIR/db-output.txt"
else
  log "⚠️  Skipping DB test (k6 not installed)"
fi

# ========================================
# Test 4: Multi-Tenant Concurrent Load
# ========================================
log ""
log "[Test 4/5] Multi-Tenant Load - 100 tenants, 10 requests each"

if [[ -f scripts/loadtest/wa-concurrent-load.sh ]]; then
  bash scripts/loadtest/wa-concurrent-load.sh 100 10 "$BASE_URL" \
    2>&1 | tee "$RESULTS_DIR/multitenant-output.txt"
else
  log "⚠️  wa-concurrent-load.sh not found - skipping"
fi

# ========================================
# Test 5: System Metrics Collection
# ========================================
log ""
log "[Test 5/5] Collecting system metrics..."

# RabbitMQ stats
if command -v curl &>/dev/null; then
  curl -s -u "wch_admin:rabbitmq_pass" \
    "http://localhost:10673/api/overview" \
    > "$RESULTS_DIR/rabbitmq-overview.json" 2>/dev/null || log "⚠️  Could not fetch RabbitMQ stats"
fi

# Docker stats snapshot
if command -v docker &>/dev/null; then
  docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
    > "$RESULTS_DIR/docker-stats.txt" 2>/dev/null || log "⚠️  Could not fetch Docker stats"
fi

# Worker count
WORKERS=$(docker ps --filter "name=umkm-automation" --format "{{.Names}}" | wc -l)
echo "Active workers: $WORKERS" >> "$RESULTS_DIR/test.log"

# ========================================
# Summary Report
# ========================================
log ""
log "=== Test Complete ==="
log "Results saved to: $RESULTS_DIR"

cat > "$RESULTS_DIR/REPORT.md" <<EOF
# Load Test Report — WCH Platform

**Date:** $(date +'%Y-%m-%d %H:%M:%S')
**Environment:** $ENV
**Base URL:** $BASE_URL

## Test Results

### 1. Baseline API Load (1000 requests)
- Target: 50 concurrent users
- Duration: 3 minutes
- See: \`baseline-output.txt\`

### 2. RabbitMQ Queue Capacity
- Published: 10,000 jobs
- See: \`queue-output.txt\`

### 3. Database Connection Pool
- Peak load: 500 concurrent queries
- See: \`db-output.txt\`

### 4. Multi-Tenant Load
- Tenants: 100
- Requests per tenant: 10
- See: \`multitenant-output.txt\`

### 5. System Metrics
- RabbitMQ overview: \`rabbitmq-overview.json\`
- Docker stats: \`docker-stats.txt\`
- Active workers: $WORKERS

## Key Metrics

**From baseline test:**
$(if [[ -f "$RESULTS_DIR/baseline-summary.json" ]]; then
  jq -r '"- Requests: \(.metrics.http_reqs.values.count)\n- Success rate: \(.metrics.http_req_failed.values.rate * 100 | floor)%\n- p95 latency: \(.metrics.http_req_duration.values."p(95)") ms"' "$RESULTS_DIR/baseline-summary.json" 2>/dev/null || echo "- Parse failed"
else
  echo "- k6 not installed"
fi)

## Recommendations

- If p95 latency > 3000ms: Tune database queries or add caching
- If error rate > 5%: Check logs for bottlenecks
- If queue depth keeps growing: Scale workers with \`docker compose up -d --scale umkm-automation=5\`

## Next Steps

1. Review Grafana dashboard during tests: http://localhost:13001
2. Check RabbitMQ Management UI: $RABBITMQ_UI
3. Run autoscaler: \`./scripts/autoscale-workers.sh --max-workers 10\`
EOF

log ""
cat "$RESULTS_DIR/REPORT.md"
