#!/bin/bash
set -e

TENANT_ID="d55169b1-9431-4104-af52-61c80811e40d"

echo "=== 1. Checking umkm-accounting internal chatbot config ==="
curl -s "http://localhost:8201/internal/tenant/$TENANT_ID/chatbot-config" | jq . || curl -s "http://localhost:8201/internal/tenant/$TENANT_ID/chatbot-config"
echo ""

echo "=== 2. Testing N8N incoming chatbot webhook ==="
curl -s -X POST "http://localhost:13678/webhook/chatbot/incoming" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT_ID"'",
    "message": "Halo, apakah toko buka hari ini?",
    "sender_jid": "6281355492003@s.whatsapp.net",
    "sender_phone": "6281355492003",
    "platform": "whatsapp"
  }' | jq . || true
echo ""
