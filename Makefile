.PHONY: run-auth run-ai run-accounting run-chatbot run-automation run-crypto run-crypto-api run-frontend start-all stop-all

run-auth:
	@echo "Starting Auth Service on port 8001..."
	@go run ./services/auth-service

run-ai:
	@echo "Starting AI Gateway on port 8002..."
	@go run ./services/ai-gateway

run-accounting:
	@echo "Starting UMKM Accounting on port 8201..."
	@go run ./apps/umkm/accounting

run-chatbot:
	@echo "Starting UMKM Chatbot on port 8202..."
	@go run ./apps/umkm/chatbot

run-automation:
	@echo "Starting Automation Worker..."
	@go run ./apps/umkm/automation

run-crypto-api:
	@echo "Starting Crypto API Service on port 8101..."
	@go run ./apps/crypto/api

run-crypto:
	@echo "Starting Crypto Trading Bot Worker..."
	@go run ./apps/crypto/worker

run-frontend:
	@echo "Starting Crypto Frontend on port 3101..."
	@cd frontend/crypto-web && npm run dev -- --port 3101 &
	@echo "Starting UMKM Frontend on port 3201..."
	@cd frontend/umkm-web && npm run dev -- --port 3201 &
	@echo "Starting Campaign Frontend on port 3301..."
	@cd frontend/campaign-web && npm run dev -- --port 3301 &

# Perintah bantu untuk menjalankan semua layanan di background (Linux/Mac)
start-all:
	@echo "Memulai seluruh ekosistem WCH..."
	@nohup go run ./services/api-gateway > api-gateway.log 2>&1 & echo $$! > api-gateway.pid
	@nohup go run ./services/auth-service > auth.log 2>&1 & echo $$! > auth.pid
	@nohup go run ./services/ai-gateway > ai.log 2>&1 & echo $$! > ai.pid
	@nohup go run ./apps/umkm/accounting > accounting.log 2>&1 & echo $$! > accounting.pid
	@nohup go run ./apps/umkm/chatbot > chatbot.log 2>&1 & echo $$! > chatbot.pid
	@nohup go run ./apps/umkm/automation > automation.log 2>&1 & echo $$! > automation.pid
	@nohup go run ./apps/campaign/api > campaign-api.log 2>&1 & echo $$! > campaign-api.pid
	@nohup go run ./apps/crypto/api > crypto-api.log 2>&1 & echo $$! > crypto-api.pid
	@nohup go run ./apps/crypto/worker > crypto.log 2>&1 & echo $$! > crypto.pid
	@nohup sh -c 'cd frontend/crypto-web && npm run dev -- --port 3101' > frontend-crypto.log 2>&1 & echo $$! > frontend-crypto.pid
	@nohup sh -c 'cd frontend/umkm-web && npm run dev -- --port 3201' > frontend-umkm.log 2>&1 & echo $$! > frontend-umkm.pid
	@nohup sh -c 'cd frontend/campaign-web && npm run dev -- --port 3301' > frontend-campaign.log 2>&1 & echo $$! > frontend-campaign.pid
	@echo "Semua layanan berjalan di background! Cek file *.log untuk output."

stop-all:
	@echo "Mematikan seluruh ekosistem WCH..."
	@pkill -f "go run ./services" || true
	@pkill -f "go run ./apps" || true
		@lsof -ti :8000 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8001 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8002 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8003 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :9002 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8202 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8301 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3301 | xargs kill -9 2>/dev/null || true
	@pkill -f "npm run dev" || true
	@rm -f *.pid
	@echo "Layanan berhasil dihentikan."
