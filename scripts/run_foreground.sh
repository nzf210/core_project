#!/bin/bash
# =============================================================================
# WCH Platform — run_foreground.sh
# =============================================================================
# Menjalankan semua service di background dan menunggu.
# Log disimpan di logs/, PID di run/
# Tekan Ctrl+C untuk mematikan semua service sekaligus.
#
# Cara pakai:
#   bash scripts/run_foreground.sh
# atau (dengan make):
#   make start-all  (rekomendasi)
# =============================================================================

set -e

# Pastikan script dijalankan dari root project
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Buat direktori output jika belum ada
mkdir -p logs run

echo ""
echo "  WCH Platform — Starting All Services"
echo "  ======================================"
echo "  Log files → logs/"
echo "  PID files → run/"
echo ""

# Hentikan service yang sudah berjalan dulu
make stop-all 2>/dev/null || true
sleep 2

# ============================================================
# Shared Services
# ============================================================
echo "▶ Starting shared services..."
nohup go run ./services/api-gateway          > logs/api-gateway.log          2>&1 & echo $! > run/api-gateway.pid
nohup go run ./services/auth-service         > logs/auth.log                  2>&1 & echo $! > run/auth.pid
nohup go run ./services/ai-gateway           > logs/ai.log                    2>&1 & echo $! > run/ai.pid
nohup go run ./services/billing-service      > logs/billing-service.log       2>&1 & echo $! > run/billing-service.pid
nohup go run ./services/notification-service > logs/notification-service.log  2>&1 & echo $! > run/notification-service.pid
nohup go run ./services/wa-gateway           > logs/wa-gateway.log            2>&1 & echo $! > run/wa-gateway.pid

# ============================================================
# UMKM Apps
# ============================================================
echo "▶ Starting UMKM apps..."
nohup go run ./apps/umkm/accounting  > logs/accounting.log  2>&1 & echo $! > run/accounting.pid
nohup go run ./apps/umkm/business    > logs/business.log    2>&1 & echo $! > run/business.pid
nohup go run ./apps/umkm/automation  > logs/automation.log  2>&1 & echo $! > run/automation.pid

# ============================================================
# Campaign App
# ============================================================
echo "▶ Starting campaign app..."
nohup go run ./apps/campaign/api > logs/campaign-api.log 2>&1 & echo $! > run/campaign-api.pid

# ============================================================
# Frontend Apps
# ============================================================
echo "▶ Starting frontend apps..."
nohup sh -c 'cd frontend/umkm-web     && npm run dev -- --port 3201' > logs/frontend-umkm.log     2>&1 & echo $! > run/frontend-umkm.pid
nohup sh -c 'cd frontend/campaign-web && npm run dev -- --port 3301' > logs/frontend-campaign.log 2>&1 & echo $! > run/frontend-campaign.pid

echo ""
echo "✓ Semua layanan berjalan di background!"
echo "  Pantau log: tail -f logs/<service>.log"
echo "  Semua log:  tail -f logs/*.log"
echo ""
echo "  Tekan Ctrl+C untuk mematikan semua layanan..."
echo ""

# Trap Ctrl+C untuk mematikan semua service
cleanup() {
    echo ""
    echo "▶ Mematikan semua layanan..."
    make stop-all 2>/dev/null || true
    exit 0
}
trap cleanup INT TERM EXIT

# Tetap aktif sampai Ctrl+C
wait
