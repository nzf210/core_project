# =============================================================================
# WCH Platform — Makefile
# =============================================================================
# Log files  → logs/<service>.log
# PID files  → run/<service>.pid
# Binaries   → bin/<service>  (untuk local build, bukan production)
#
# Gunakan `make help` untuk melihat semua perintah yang tersedia.
# Port service: lihat CLAUDE.md atau docs/FEATURE_MAP.md

.PHONY: help start-all stop-all status \
        run-gateway run-auth run-ai run-billing run-notification run-wa-gateway \
        run-accounting run-chatbot run-business run-automation \
        run-crypto-api run-crypto run-campaign \
        run-frontend \
        build build-all \
        test test-verbose test-race \
        vet tidy check \
        logs-auth logs-ai logs-accounting logs-chatbot logs-crypto \
        logs-campaign logs-gateway logs-all \
        clean clean-logs clean-build

# Direktori output (jangan letakkan file di root!)
LOG_DIR  := logs
RUN_DIR  := run
BIN_DIR  := bin

# Pastikan direktori output ada sebelum digunakan
$(LOG_DIR)/.gitkeep $(RUN_DIR)/.gitkeep $(BIN_DIR)/.gitkeep:
	@mkdir -p $(LOG_DIR) $(RUN_DIR) $(BIN_DIR)
	@touch $(LOG_DIR)/.gitkeep $(RUN_DIR)/.gitkeep $(BIN_DIR)/.gitkeep

# =============================================================================
# Help
# =============================================================================
help:
	@echo ""
	@echo "  WCH Platform — Available Make Targets"
	@echo "  ======================================"
	@echo ""
	@echo "  SERVICES:"
	@echo "    make run-gateway      — API Gateway (port 8000)"
	@echo "    make run-auth         — Auth Service (port 8001)"
	@echo "    make run-ai           — AI Gateway (port 8002)"
	@echo "    make run-billing      — Billing Service (port 8003)"
	@echo "    make run-notification — Notification Service (port 8005)"
	@echo "    make run-wa-gateway   — WhatsApp Gateway (port 8202)"
	@echo ""
	@echo "  APPS:"
	@echo "    make run-accounting   — UMKM Accounting (port 8201)"
	@echo "    make run-chatbot      — UMKM Chatbot (port 8202)"
	@echo "    make run-business     — UMKM Business (port 9001)"
	@echo "    make run-automation   — UMKM Automation (worker)"
	@echo "    make run-crypto-api   — Crypto API (port 8101)"
	@echo "    make run-crypto       — Crypto Worker (background)"
	@echo "    make run-campaign     — Campaign API (port 9002)"
	@echo ""
	@echo "  FRONTEND:"
	@echo "    make run-frontend     — Semua frontend (3101, 3201, 3301)"
	@echo ""
	@echo "  LIFECYCLE:"
	@echo "    make start-all        — Jalankan semua service di background"
	@echo "    make stop-all         — Matikan semua service"
	@echo "    make status           — Tampilkan status port yang aktif"
	@echo ""
	@echo "  BUILD (local binary):"
	@echo "    make build-all        — Build semua binary ke bin/"
	@echo "    make build            — Compile check (go build ./...)"
	@echo ""
	@echo "  LOGS:"
	@echo "    make logs-auth        — Tail log auth service"
	@echo "    make logs-ai          — Tail log AI gateway"
	@echo "    make logs-accounting  — Tail log UMKM accounting"
	@echo "    make logs-chatbot     — Tail log UMKM chatbot"
	@echo "    make logs-campaign    — Tail log campaign API"
	@echo "    make logs-crypto      — Tail log crypto worker"
	@echo "    make logs-gateway     — Tail log API gateway"
	@echo "    make logs-all         — Tail semua log sekaligus"
	@echo ""
	@echo "  QUALITY:"
	@echo "    make test             — Jalankan semua unit test"
	@echo "    make test-verbose     — Test verbose"
	@echo "    make test-race        — Test dengan race detector"
	@echo "    make vet              — Static analysis (go vet)"
	@echo "    make tidy             — Bersihkan dependencies"
	@echo "    make check            — tidy + vet + build + test"
	@echo ""
	@echo "  CLEANUP:"
	@echo "    make clean-logs       — Hapus semua log files di logs/"
	@echo "    make clean-build      — Hapus binary di bin/"
	@echo "    make clean            — clean-logs + clean-build"
	@echo ""

# =============================================================================
# Helper: Pastikan direktori output ada
# =============================================================================
_ensure_dirs:
	@mkdir -p $(LOG_DIR) $(RUN_DIR) $(BIN_DIR)

# =============================================================================
# Shared Services
# =============================================================================
run-gateway: _ensure_dirs
	@echo "▶ Starting API Gateway on port 8000..."
	@go run ./services/api-gateway

run-auth: _ensure_dirs
	@echo "▶ Starting Auth Service on port 8001..."
	@go run ./services/auth-service

run-ai: _ensure_dirs
	@echo "▶ Starting AI Gateway on port 8002..."
	@go run ./services/ai-gateway

run-billing: _ensure_dirs
	@echo "▶ Starting Billing Service on port 8003..."
	@go run ./services/billing-service

run-notification: _ensure_dirs
	@echo "▶ Starting Notification Service on port 8005..."
	@go run ./services/notification-service

run-wa-gateway: _ensure_dirs
	@echo "▶ Starting WA Gateway on port 8202..."
	@go run ./services/wa-gateway

# =============================================================================
# UMKM Apps
# =============================================================================
run-accounting: _ensure_dirs
	@echo "▶ Starting UMKM Accounting on port 8201..."
	@go run ./apps/umkm/accounting

run-chatbot: _ensure_dirs
	@echo "▶ Starting UMKM Chatbot on port 8202..."
	@echo "⚠  Port 8202 juga dipakai wa-gateway. Pastikan tidak konflik!"
	@go run ./apps/umkm/chatbot

run-business: _ensure_dirs
	@echo "▶ Starting UMKM Business on port 9001..."
	@go run ./apps/umkm/business

run-automation: _ensure_dirs
	@echo "▶ Starting UMKM Automation Worker..."
	@go run ./apps/umkm/automation

# =============================================================================
# Crypto Apps
# =============================================================================
run-crypto-api: _ensure_dirs
	@echo "▶ Starting Crypto API on port 8101..."
	@go run ./apps/crypto/api

run-crypto: _ensure_dirs
	@echo "▶ Starting Crypto Trading Bot Worker..."
	@go run ./apps/crypto/worker

# =============================================================================
# Campaign App
# =============================================================================
run-campaign: _ensure_dirs
	@echo "▶ Starting Campaign API on port 9002..."
	@go run ./apps/campaign/api

# =============================================================================
# Frontend
# =============================================================================
run-frontend: _ensure_dirs
	@echo "▶ Starting Crypto Frontend on port 3101..."
	@cd frontend/crypto-web && npm run dev -- --port 3101 &
	@echo "▶ Starting UMKM Frontend on port 3201..."
	@cd frontend/umkm-web && npm run dev -- --port 3201 &
	@echo "▶ Starting Campaign Frontend on port 3301..."
	@cd frontend/campaign-web && npm run dev -- --port 3301 &
	@echo "✓ Frontend services started!"

# =============================================================================
# Start All — Log ke logs/, PID ke run/
# =============================================================================
start-all: _ensure_dirs
	@echo "▶ Memulai seluruh ekosistem WCH Platform..."
	@echo "  Log files  → $(LOG_DIR)/"
	@echo "  PID files  → $(RUN_DIR)/"
	@nohup go run ./services/api-gateway     > $(LOG_DIR)/api-gateway.log     2>&1 & echo $$! > $(RUN_DIR)/api-gateway.pid
	@nohup go run ./services/auth-service    > $(LOG_DIR)/auth.log             2>&1 & echo $$! > $(RUN_DIR)/auth.pid
	@nohup go run ./services/ai-gateway      > $(LOG_DIR)/ai.log               2>&1 & echo $$! > $(RUN_DIR)/ai.pid
	@nohup go run ./services/billing-service > $(LOG_DIR)/billing-service.log  2>&1 & echo $$! > $(RUN_DIR)/billing-service.pid
	@nohup go run ./services/notification-service > $(LOG_DIR)/notification-service.log 2>&1 & echo $$! > $(RUN_DIR)/notification-service.pid
	@nohup go run ./services/wa-gateway      > $(LOG_DIR)/wa-gateway.log       2>&1 & echo $$! > $(RUN_DIR)/wa-gateway.pid
	@nohup go run ./apps/umkm/accounting     > $(LOG_DIR)/accounting.log       2>&1 & echo $$! > $(RUN_DIR)/accounting.pid
	@nohup go run ./apps/umkm/business       > $(LOG_DIR)/business.log         2>&1 & echo $$! > $(RUN_DIR)/business.pid
	@nohup go run ./apps/umkm/automation     > $(LOG_DIR)/automation.log       2>&1 & echo $$! > $(RUN_DIR)/automation.pid
	@nohup go run ./apps/campaign/api        > $(LOG_DIR)/campaign-api.log     2>&1 & echo $$! > $(RUN_DIR)/campaign-api.pid
	@nohup go run ./apps/crypto/api          > $(LOG_DIR)/crypto-api.log       2>&1 & echo $$! > $(RUN_DIR)/crypto-api.pid
	@nohup go run ./apps/crypto/worker       > $(LOG_DIR)/crypto.log           2>&1 & echo $$! > $(RUN_DIR)/crypto.pid
	@nohup sh -c 'cd frontend/crypto-web   && npm run dev -- --port 3101' > $(LOG_DIR)/frontend-crypto.log   2>&1 & echo $$! > $(RUN_DIR)/frontend-crypto.pid
	@nohup sh -c 'cd frontend/umkm-web     && npm run dev -- --port 3201' > $(LOG_DIR)/frontend-umkm.log     2>&1 & echo $$! > $(RUN_DIR)/frontend-umkm.pid
	@nohup sh -c 'cd frontend/campaign-web && npm run dev -- --port 3301' > $(LOG_DIR)/frontend-campaign.log 2>&1 & echo $$! > $(RUN_DIR)/frontend-campaign.pid
	@echo ""
	@echo "✓ Semua layanan berjalan di background!"
	@echo "  Pantau log : tail -f $(LOG_DIR)/<service>.log"
	@echo "  Semua log  : make logs-all"
	@echo "  Matikan    : make stop-all"
	@echo ""

# =============================================================================
# Stop All — Baca PID dari run/
# =============================================================================
stop-all:
	@echo "▶ Mematikan seluruh ekosistem WCH Platform..."
	@if ls $(RUN_DIR)/*.pid 2>/dev/null | grep -q .; then \
		for pid_file in $(RUN_DIR)/*.pid; do \
			pid=$$(cat "$$pid_file" 2>/dev/null); \
			if [ -n "$$pid" ] && kill -0 "$$pid" 2>/dev/null; then \
				kill "$$pid" 2>/dev/null && echo "  Stopped PID $$pid ($$pid_file)"; \
			fi; \
		done; \
		rm -f $(RUN_DIR)/*.pid; \
	fi
	@pkill -f "go run ./services" 2>/dev/null || true
	@pkill -f "go run ./apps" 2>/dev/null || true
	@pkill -f "npm run dev" 2>/dev/null || true
	@lsof -ti :8000 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8001 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8002 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8003 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8005 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8202 | xargs kill -9 2>/dev/null || true
	@lsof -ti :9001 | xargs kill -9 2>/dev/null || true
	@lsof -ti :9002 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3301 | xargs kill -9 2>/dev/null || true
	@echo "✓ Semua layanan berhasil dihentikan."

# =============================================================================
# Status — Cek port aktif
# =============================================================================
status:
	@echo "▶ Status Port WCH Platform:"
	@echo ""
	@echo "  Port 8000 (API Gateway):       $$(lsof -ti :8000 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8001 (Auth Service):      $$(lsof -ti :8001 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8002 (AI Gateway):        $$(lsof -ti :8002 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8003 (Billing Service):   $$(lsof -ti :8003 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8005 (Notification):      $$(lsof -ti :8005 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8101 (Crypto API):        $$(lsof -ti :8101 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8201 (UMKM Accounting):   $$(lsof -ti :8201 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 8202 (WA/Chatbot):        $$(lsof -ti :8202 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 9001 (UMKM Business):     $$(lsof -ti :9001 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 9002 (Campaign API):      $$(lsof -ti :9002 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 3101 (Frontend Crypto):   $$(lsof -ti :3101 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 3201 (Frontend UMKM):     $$(lsof -ti :3201 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo "  Port 3301 (Frontend Campaign): $$(lsof -ti :3301 > /dev/null 2>&1 && echo '✓ RUNNING' || echo '✗ stopped')"
	@echo ""
	@echo "▶ PID Files di $(RUN_DIR)/:"
	@ls $(RUN_DIR)/*.pid 2>/dev/null | while read f; do \
		svc=$$(basename "$$f" .pid); \
		pid=$$(cat "$$f" 2>/dev/null); \
		if kill -0 "$$pid" 2>/dev/null; then \
			echo "  ✓ $$svc (PID $$pid)"; \
		else \
			echo "  ✗ $$svc (PID $$pid — sudah berhenti)"; \
		fi; \
	done || echo "  (tidak ada PID file — service belum dijalankan via make start-all)"
	@echo ""

# =============================================================================
# Logs — Tail log files dari logs/
# =============================================================================
logs-auth:
	@tail -f $(LOG_DIR)/auth.log

logs-ai:
	@tail -f $(LOG_DIR)/ai.log

logs-accounting:
	@tail -f $(LOG_DIR)/accounting.log

logs-chatbot:
	@tail -f $(LOG_DIR)/chatbot.log

logs-crypto:
	@tail -f $(LOG_DIR)/crypto.log

logs-campaign:
	@tail -f $(LOG_DIR)/campaign-api.log

logs-gateway:
	@tail -f $(LOG_DIR)/api-gateway.log

logs-billing:
	@tail -f $(LOG_DIR)/billing-service.log

logs-wa:
	@tail -f $(LOG_DIR)/wa-gateway.log

logs-all:
	@echo "▶ Tailing semua log (Ctrl+C untuk berhenti)..."
	@tail -f $(LOG_DIR)/*.log 2>/dev/null || echo "Tidak ada log files di $(LOG_DIR)/"

# =============================================================================
# Build — Compile binary ke bin/ (bukan ke root!)
# =============================================================================
build-all: _ensure_dirs
	@echo "▶ Building all services to $(BIN_DIR)/..."
	@go build -o $(BIN_DIR)/api-gateway        ./services/api-gateway
	@go build -o $(BIN_DIR)/auth-service       ./services/auth-service
	@go build -o $(BIN_DIR)/ai-gateway         ./services/ai-gateway
	@go build -o $(BIN_DIR)/billing-service    ./services/billing-service
	@go build -o $(BIN_DIR)/notification-service ./services/notification-service
	@go build -o $(BIN_DIR)/wa-gateway         ./services/wa-gateway
	@go build -o $(BIN_DIR)/umkm-accounting    ./apps/umkm/accounting
	@go build -o $(BIN_DIR)/umkm-business      ./apps/umkm/business
	@go build -o $(BIN_DIR)/umkm-chatbot       ./apps/umkm/chatbot
	@go build -o $(BIN_DIR)/umkm-automation    ./apps/umkm/automation
	@go build -o $(BIN_DIR)/crypto-api         ./apps/crypto/api
	@go build -o $(BIN_DIR)/crypto-worker      ./apps/crypto/worker
	@go build -o $(BIN_DIR)/campaign-api       ./apps/campaign/api
	@echo "✓ Semua binary berhasil di-build ke $(BIN_DIR)/"
	@ls -lh $(BIN_DIR)/ | grep -v gitkeep

build:
	@echo "▶ Compile check (go build ./...)..."
	@go build ./...
	@echo "✓ Build successful!"

# =============================================================================
# Quality
# =============================================================================
test:
	@echo "▶ Running all tests..."
	@go test ./...

test-verbose:
	@echo "▶ Running all tests (verbose)..."
	@go test -v ./...

test-race:
	@echo "▶ Running tests with race detector..."
	@go test -race ./...

vet:
	@echo "▶ Running go vet..."
	@go vet ./...
	@echo "✓ Vet passed!"

tidy:
	@echo "▶ Running go mod tidy..."
	@go mod tidy
	@echo "✓ Dependencies cleaned!"

# CI-like check — jalankan semua quality checks sekaligus
check: tidy vet build test
	@echo ""
	@echo "✓ All checks passed!"

# =============================================================================
# Cleanup
# =============================================================================
clean-logs:
	@echo "▶ Menghapus log files..."
	@rm -f $(LOG_DIR)/*.log
	@echo "✓ Log files dihapus dari $(LOG_DIR)/"

clean-build:
	@echo "▶ Menghapus binary files..."
	@ls $(BIN_DIR)/ | grep -v ".gitkeep" | while read f; do rm -f "$(BIN_DIR)/$$f"; done
	@echo "✓ Binary files dihapus dari $(BIN_DIR)/"

clean: clean-logs clean-build
	@echo "✓ Cleanup selesai."
