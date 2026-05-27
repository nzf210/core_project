#!/bin/bash
make stop-all
sleep 2

# Jalankan semua di background, tapi kemudian tunggu
nohup go run ./services/api-gateway > api-gateway.log 2>&1 &
nohup go run ./services/auth-service > auth.log 2>&1 &
nohup go run ./services/ai-gateway > ai.log 2>&1 &
nohup go run ./apps/umkm/accounting > accounting.log 2>&1 &
nohup go run ./apps/umkm/chatbot > chatbot.log 2>&1 &
nohup go run ./apps/umkm/automation > automation.log 2>&1 &
nohup go run ./apps/crypto/api > crypto-api.log 2>&1 &
nohup go run ./apps/crypto/worker > crypto.log 2>&1 &
nohup sh -c 'cd frontend/crypto-web && npm run dev' > frontend.log 2>&1 &

echo "Semua layanan berjalan, keep-alive script dimulai..."
wait
