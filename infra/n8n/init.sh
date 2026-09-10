#!/bin/bash
# N8N Init Script — Queue Mode Support
# Import semua workflow di /workflows/*.json ke n8n.
# Tandai .workflow_imported supaya idempotent (tidak re-import setiap restart).
# Mendukung Queue Mode: main (UI + webhook) dan worker (execution).

set -e

# Tunggu PostgreSQL & Redis siap
sleep 5

IMPORT_DIR="/workflows"
MARKER="/home/node/.n8n/.workflow_imported"

# Deteksi apakah ini worker atau main instance
# Worker dipanggil dengan command "worker", main tanpa argument
IS_WORKER=false
for arg in "$@"; do
  if [[ "$arg" = "worker" ]]; then
    IS_WORKER=true
    break
  fi
done

if [[ "$IS_WORKER" = "true" ]]; then
  echo "=== N8N Worker Mode ==="
  echo "Starting n8n worker..."
  echo "Queue: Redis ${QUEUE_BULL_REDIS_HOST:-redis}:${QUEUE_BULL_REDIS_PORT:-6379} DB:${QUEUE_BULL_REDIS_DB:-2}"
  echo "Concurrency: ${N8N_CONCURRENCY_PRODUCTION_LIMIT:-10}"
  exec n8n worker
else
  echo "=== N8N Main Mode (Queue) ==="
  echo "Execution Mode: ${EXECUTIONS_MODE:-regular}"

  # Import workflows hanya di main instance
  HASH_FILE="/home/node/.n8n/.workflows_hash"
  CURRENT_HASH=$(cat "$IMPORT_DIR"/*.json 2>/dev/null | md5sum | awk '{print $1}')
  STORED_HASH=""
  [[ -f "$HASH_FILE" ]] && STORED_HASH=$(cat "$HASH_FILE" 2>/dev/null)

  if [[ ! -f "$MARKER" || "$CURRENT_HASH" != "$STORED_HASH" ]]; then
    echo "Importing/updating n8n workflows from $IMPORT_DIR (hash: ${CURRENT_HASH:-new}) ..."
    for wf in "$IMPORT_DIR"/*.json; do
      [[ -e "$wf" ]] || continue
      name=$(basename "$wf")
      echo "  → $name"
      n8n import:workflow --input="$wf" --activeState=fromJson || echo "    failed: $name"
    done
    touch "$MARKER"
    [[ -n "$CURRENT_HASH" ]] && echo "$CURRENT_HASH" > "$HASH_FILE"
    echo "All workflows imported and activated."
  else
    echo "Workflows up to date (hash: $CURRENT_HASH). Skipping re-import."
  fi

  echo "Starting n8n main..."
  exec n8n start
fi
