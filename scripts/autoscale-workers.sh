#!/bin/bash
# Auto-scale umkm-automation workers based on RabbitMQ queue depth
# Usage: ./scripts/autoscale-workers.sh [--min-workers 1] [--max-workers 10] [--threshold 100]

set -euo pipefail

# Configuration
MIN_WORKERS=${MIN_WORKERS:-1}
MAX_WORKERS=${MAX_WORKERS:-10}
QUEUE_THRESHOLD=${QUEUE_THRESHOLD:-100}  # jobs per worker
RABBITMQ_URL=${RABBITMQ_URL:-"http://localhost:10673"}
RABBITMQ_USER=${RABBITMQ_USER:-"wch_admin"}
RABBITMQ_PASSWORD=${RABBITMQ_PASSWORD:-"rabbitmq_pass"}
CHECK_INTERVAL=${CHECK_INTERVAL:-30}  # seconds

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --min-workers) MIN_WORKERS="$2"; shift 2 ;;
    --max-workers) MAX_WORKERS="$2"; shift 2 ;;
    --threshold) QUEUE_THRESHOLD="$2"; shift 2 ;;
    --interval) CHECK_INTERVAL="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

log() {
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

get_queue_depth() {
  # Get total messages across all queues
  curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASSWORD}" \
    "${RABBITMQ_URL}/api/queues" | \
    jq '[.[] | select(.name | startswith("accounting.") or startswith("voucher.") or startswith("chatbot.")) | .messages] | add // 0'
}

get_current_workers() {
  docker ps --filter "name=umkm-automation" --format "{{.Names}}" | wc -l
}

scale_workers() {
  local target=$1
  local current=$(get_current_workers)

  if [ "$target" -eq "$current" ]; then
    log "Workers already at target: $target"
    return
  fi

  log "Scaling workers: $current → $target"

  # Determine compose file based on environment
  if [ -f "docker-compose.staging.yml" ] && [ "${ENV:-dev}" = "staging" ]; then
    COMPOSE_PROJECT_NAME=wch-stg docker compose \
      -f docker-compose.yml -f docker-compose.staging.yml \
      up -d --scale umkm-automation="$target" --no-recreate
  else
    docker compose up -d --scale umkm-automation="$target" --no-recreate
  fi

  log "Scaled to $target workers"
}

calculate_target_workers() {
  local queue_depth=$1
  local current_workers=$2

  # Calculate ideal workers: queue_depth / threshold, minimum MIN_WORKERS
  local ideal=$(( (queue_depth + QUEUE_THRESHOLD - 1) / QUEUE_THRESHOLD ))

  # Clamp between MIN and MAX
  if [ "$ideal" -lt "$MIN_WORKERS" ]; then
    echo "$MIN_WORKERS"
  elif [ "$ideal" -gt "$MAX_WORKERS" ]; then
    echo "$MAX_WORKERS"
  else
    echo "$ideal"
  fi
}

log "=== Worker Auto-Scaler Started ==="
log "Config: MIN=$MIN_WORKERS MAX=$MAX_WORKERS THRESHOLD=$QUEUE_THRESHOLD jobs/worker"
log "Check interval: ${CHECK_INTERVAL}s"

while true; do
  queue_depth=$(get_queue_depth)
  current_workers=$(get_current_workers)
  target_workers=$(calculate_target_workers "$queue_depth" "$current_workers")

  log "Queue depth: $queue_depth jobs | Workers: $current_workers → target: $target_workers"

  # Scale if target differs from current
  if [ "$target_workers" -ne "$current_workers" ]; then
    scale_workers "$target_workers"
  fi

  sleep "$CHECK_INTERVAL"
done
