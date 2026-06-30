#!/bin/bash
# =============================================================================
# WCH Platform — Native Development (Hot-Reload)
#
# Prerequisites:
#   1. Docker: postgres + redis running
#   2. ~/go/bin/air installed (go install github.com/air-verse/air@latest)
#
# Usage:
#   ./scripts/dev-native.sh           Start everything hot-reload
#   ./scripts/dev-native.sh --stop    Stop all
#   ./scripts/dev-native.sh auth     Start only auth-service
#   ./scripts/dev-native.sh gateway   Start only api-gateway
# =============================================================================


# Cleanup ALL processes on expected ports before start (prevents stale processes)
cleanup_ports() {
  echo "🧹 Cleaning stale processes..."
  for port in 8000 8001 8002 8003 8005 8006 8201 8202 8210 9001 9002 3201 3301 3401; do
    pids=$(lsof -ti:$port 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
      for pid in $pids; do
        kill "$pid" 2>/dev/null && echo "  ✓ killed $port (PID $pid)" || echo "  ✗ failed kill $port"
      done
    fi
  done
  echo "  done cleaning."
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$ROOT_DIR/logs"
RUN_DIR="$ROOT_DIR/run"
AIR="$HOME/go/bin/air"

mkdir -p "$LOG_DIR" "$RUN_DIR"

# Each service: name:working_dir:log_name:env_relative
# NOTE: umkm-business hardcoded port 9001 (not 9005) per source code
SERVICES=(
  "api-gateway:services/api-gateway:api-gateway:../../.env"
  "auth-service:services/auth-service:auth-service:../../.env"
  "ai-gateway:services/ai-gateway:ai-gateway:../../.env"
  "billing-service:services/billing-service:billing-service:../../.env"
  "notification-service:services/notification-service:notification-service:../../.env"
  "wa-gateway:services/wa-gateway:wa-gateway:../../.env"
  "umkm-accounting:apps/umkm/accounting:accounting:../../../.env"
  "umkm-chatbot:apps/umkm/chatbot:chatbot:../../../.env"
  "umkm-business:apps/umkm/business:business:../../../.env"
  "umkm-automation:apps/umkm/automation:automation:../../../.env"
  "campaign-api:apps/campaign/api:campaign-api:../../../.env"
)

stop_all() {
  echo "Stopping all dev services..."
  for pair in "${SERVICES[@]}"; do
    IFS=':' read -r svc dir logfile _ <<< "$pair"
    pidfile="$RUN_DIR/dev-$svc.pid"
    [[ -f "$pidfile" ]] && kill "$(cat "$pidfile")" 2>/dev/null && echo "  ✓ $svc" || true
    rm -f "$pidfile"
  done
  for fe in umkm-web campaign-web superadmin-web; do
    pidfile="$RUN_DIR/dev-$fe.pid"
    [[ -f "$pidfile" ]] && kill "$(cat "$pidfile")" 2>/dev/null && echo "  ✓ $fe" || true
    rm -f "$pidfile"
  done
  echo "Done."
}

start_service() {
  local svc="$1"
  local dir="$2"
  local logname="$3"
  local envfile="$4"
  local pidfile="$RUN_DIR/dev-$svc.pid"
  local logfile="$LOG_DIR/dev-$logname.log"

  [[ -f "$pidfile" ]] && kill "$(cat "$pidfile")" 2>/dev/null || true

  echo "🔄 $svc (hot-reload)..."
  # Go config.LoadConfig() loads .env itself via godotenv (searches ., ../../, ../../../)
  nohup sh -c "cd $ROOT_DIR/$dir && exec $AIR" > "$logfile" 2>&1 &
  echo $! > "$pidfile"
  echo "  → $logfile"
}

start_all() {
  cleanup_ports
  echo "🚀 WCH Platform — native dev mode (hot-reload)"
  echo "   Postgres + Redis harus jalan dulu (Docker)."
  echo ""

  for pair in "${SERVICES[@]}"; do
    IFS=':' read -r svc dir logname envfile <<< "$pair"
    start_service "$svc" "$dir" "$logname" "$envfile"
  done

  echo ""
  echo "🔄 Frontend (Vite hot-reload)..."
  for entry in "umkm-web:3201" "campaign-web:3301" "superadmin-web:3401"; do
    IFS=':' read -r name port <<< "$entry"
    local logfile="$LOG_DIR/dev-$name.log"
    local pidfile="$RUN_DIR/dev-$name.pid"
    [[ -f "$pidfile" ]] && kill "$(cat "$pidfile")" 2>/dev/null || true
    nohup sh -c "cd $ROOT_DIR/frontend/$name && npm run dev -- --port $port" > "$logfile" 2>&1 &
    echo $! > "$pidfile"
    echo "  ✓ $name on :$port"
  done

  echo ""
  echo "✅ Semua service jalan!"
  echo "   umkm:    http://localhost:3201"
  echo "   campaign: http://localhost:3301"
  echo "   admin:   http://localhost:3401"
  echo ""
  echo "   Logs:  tail -f $LOG_DIR/dev-*.log"
  echo "   Stop:  $0 --stop"
}

case "${1:-}" in
  --stop) stop_all ;;
  gateway|auth|ai|billing|notification|wa-gateway|accounting|chatbot|business|automation|campaign)
    svc_id="${1}"
    for pair in "${SERVICES[@]}"; do
      IFS=':' read -r svc dir logname envfile <<< "$pair"
      [[ "$svc" == "$svc_id" ]] && start_service "$svc" "$dir" "$logname" "$envfile" && break
    done
    ;;
  *) start_all ;;
esac
