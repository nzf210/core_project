#!/bin/bash
# Simple concurrent WA session load test using curl
# Usage: ./wa-concurrent-load.sh <num_tenants> <messages_per_tenant> <base_url>

set -e

NUM_TENANTS=${1:-10}
MESSAGES_PER_TENANT=${2:-5}
BASE_URL=${3:-http://localhost:8202}

RESULTS_DIR="results/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo "=== WA Gateway Concurrent Load Test ==="
echo "Tenants: $NUM_TENANTS"
echo "Messages per tenant: $MESSAGES_PER_TENANT"
echo "Base URL: $BASE_URL"
echo "Results: $RESULTS_DIR"
echo ""

# Function to send message and measure time
send_message() {
    local tenant_id=$1
    local message_num=$2
    local start=$(date +%s%N)

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/wa/send" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "tenant_id=test-tenant-${tenant_id}" \
        -d "target=628123456789@s.whatsapp.net" \
        -d "message=Load test message ${message_num} from tenant ${tenant_id}" \
        2>&1)

    local end=$(date +%s%N)
    local duration_ms=$(( (end - start) / 1000000 ))
    local http_code=$(echo "$response" | tail -n1)

    echo "${tenant_id},${message_num},${http_code},${duration_ms}" >> "$RESULTS_DIR/raw_results.csv"

    if [ "$http_code" == "200" ]; then
        echo "✓ Tenant $tenant_id msg $message_num: ${duration_ms}ms"
    else
        echo "✗ Tenant $tenant_id msg $message_num: HTTP $http_code (${duration_ms}ms)"
    fi
}

# Initialize results file
echo "tenant_id,message_num,http_code,duration_ms" > "$RESULTS_DIR/raw_results.csv"

# Main load test loop
echo "Starting load test..."
start_time=$(date +%s)

for tenant in $(seq 1 $NUM_TENANTS); do
    for msg in $(seq 1 $MESSAGES_PER_TENANT); do
        send_message $tenant $msg &
    done
done

# Wait for all background jobs
wait

end_time=$(date +%s)
total_duration=$((end_time - start_time))

# Generate summary report
total_requests=$((NUM_TENANTS * MESSAGES_PER_TENANT))
success_count=$(grep -c ",200," "$RESULTS_DIR/raw_results.csv" || echo "0")
fail_count=$((total_requests - success_count))

# Calculate latency percentiles
cat "$RESULTS_DIR/raw_results.csv" | tail -n +2 | cut -d',' -f4 | sort -n > "$RESULTS_DIR/latencies.txt"
p50=$(awk 'NR==int((NR+1)*0.5)' "$RESULTS_DIR/latencies.txt")
p95=$(awk 'NR==int((NR+1)*0.95)' "$RESULTS_DIR/latencies.txt")
p99=$(awk 'NR==int((NR+1)*0.99)' "$RESULTS_DIR/latencies.txt")

# Write summary report
cat > "$RESULTS_DIR/summary.txt" <<EOF
=== WA Gateway Load Test Summary ===
Date: $(date)

Configuration:
- Concurrent tenants: $NUM_TENANTS
- Messages per tenant: $MESSAGES_PER_TENANT
- Total requests: $total_requests
- Base URL: $BASE_URL

Results:
- Total duration: ${total_duration}s
- Throughput: $(echo "scale=2; $total_requests / $total_duration" | bc) req/s
- Success: $success_count ($((success_count * 100 / total_requests))%)
- Failed: $fail_count ($((fail_count * 100 / total_requests))%)

Latency:
- p50: ${p50}ms
- p95: ${p95}ms
- p99: ${p99}ms

HTTP Status Codes:
$(tail -n +2 "$RESULTS_DIR/raw_results.csv" | cut -d',' -f3 | sort | uniq -c | awk '{print "  " $2 ": " $1 " requests"}')

Raw data: $RESULTS_DIR/raw_results.csv
EOF

cat "$RESULTS_DIR/summary.txt"

echo ""
echo "Load test complete. Results saved to: $RESULTS_DIR"
