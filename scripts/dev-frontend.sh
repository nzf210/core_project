#!/bin/bash
# =============================================================================
# WCH Platform — Frontend Development Script
#
# Menjalankan semua frontend (umkm-web, campaign-web, superadmin-web) sekaligus
# dengan auto-kill port yang bentrok dan fixed port (tidak lompat).
#
# Usage:
#   ./scripts/dev-frontend.sh           Start all frontends
#   ./scripts/dev-frontend.sh --stop    Stop all frontends
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
FRONTEND_DIR="$ROOT_DIR/frontend"
LOG_DIR="$ROOT_DIR/logs"
RUN_DIR="$ROOT_DIR/run"

mkdir -p "$LOG_DIR" "$RUN_DIR"

# Frontend definitions: name:dir:port
FRONTENDS=(
  "umkm-web:umkm-web:3201"
  "campaign-web:campaign-web:3301"
  "superadmin-web:superadmin-web:3401"
)

# Kill process on specific port
kill_port() {
  local port=$1
  local pids=$(lsof -ti:$port 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    for pid in $pids; do
      kill -9 "$pid" 2>/dev/null && echo "  ✓ Killed process on port $port (PID $pid)" || true
    done
  fi
}

# Cleanup all frontend ports
cleanup_ports() {
  echo "🧹 Cleaning up frontend ports..."
  for entry in "${FRONTENDS[@]}"; do
    IFS=':' read -r name dir port <<< "$entry"
    kill_port "$port"
  done
  echo "  Done."
}

# Stop all frontends
stop_all() {
  echo "🛑 Stopping all frontends..."
  cleanup_ports

  # Kill by PID file
  for entry in "${FRONTENDS[@]}"; do
    IFS=':' read -r name dir port <<< "$entry"
    pid_file="$RUN_DIR/${name}.pid"
    if [[ -f "$pid_file" ]]; then
      pid=$(cat "$pid_file")
      if ps -p "$pid" > /dev/null 2>&1; then
        kill "$pid" 2>/dev/null && echo "  ✓ Stopped $name (PID $pid)" || true
      fi
      rm -f "$pid_file"
    fi
  done

  echo "✅ All frontends stopped."
  exit 0
}

# Start all frontends
start_all() {
  echo "🚀 Starting all frontends..."

  # Cleanup ports first
  cleanup_ports

  for entry in "${FRONTENDS[@]}"; do
    IFS=':' read -r name dir port <<< "$entry"
    frontend_path="$FRONTEND_DIR/$dir"
    log_file="$LOG_DIR/${name}.log"
    pid_file="$RUN_DIR/${name}.pid"

    if [[ ! -d "$frontend_path" ]]; then
      echo "  ⚠️  Skip $name — directory not found: $frontend_path"
      continue
    fi

    if [[ ! -d "$frontend_path/node_modules" ]]; then
      echo "  📦 Installing dependencies for $name..."
      (cd "$frontend_path" && npm install --ignore-scripts > "$log_file" 2>&1)
    fi

    echo "  ▶️  Starting $name on port $port..."
    (
      cd "$frontend_path"
      npm run dev > "$log_file" 2>&1 &
      echo $! > "$pid_file"
    )

    echo "     Log: $log_file"
    echo "     PID: $(cat "$pid_file")"
  done

  echo ""
  echo "✅ All frontends started!"
  echo ""
  echo "Access URLs:"
  echo "  • UMKM Web:       http://localhost:3201"
  echo "  • Campaign Web:   http://localhost:3301"
  echo "  • Superadmin Web: http://localhost:3401"
  echo ""
  echo "Logs:"
  echo "  tail -f $LOG_DIR/umkm-web.log"
  echo "  tail -f $LOG_DIR/campaign-web.log"
  echo "  tail -f $LOG_DIR/superadmin-web.log"
  echo ""
  echo "Stop all: ./scripts/dev-frontend.sh --stop"
}

# Main
case "${1:-}" in
  --stop)
    stop_all
    ;;
  *)
    start_all
    ;;
esac
