# N8N Setup & Integration Guide

> **Panduan setup dan integrasi n8n workflows untuk WCH Platform.**

---

## 🎯 N8N Architecture di WCH Platform

**Mode:** Queue Mode (Redis-based job distribution)

```
n8n-main (UI + Webhook Receiver)
    ↓
  Redis (Bull Queue)
    ↓
n8n-worker (Execution Worker) — Scalable
```

**Database:** PostgreSQL terpisah (`wch_n8n`)

---

## 🚀 Quick Start

### 1. Start n8n

```bash
# Start n8n services
docker compose up -d n8n-main n8n-worker

# Verify
docker ps | grep n8n
curl http://localhost:5678/healthz
```

### 2. Access Web UI

**URL:** http://localhost:5678

**Credentials:** (dari `.env`)
```bash
Username: admin
Password: ganti_password_n8n_yang_kuat
```

⚠️ **IMPORTANT:** Ubah password di `.env` sebelum production!

### 3. Import Workflows

**Via UI:**
1. Login ke n8n → http://localhost:5678
2. Click "Add workflow" → "Import from file"
3. Select file dari `infra/n8n/workflows/*.json`
4. Save & Activate

**Via CLI (Bulk Import):**
```bash
# Copy workflows ke n8n container
for file in infra/n8n/workflows/*.json; do
  docker cp "$file" wch-n8n-main:/tmp/
done

# Import via n8n CLI (inside container)
docker exec wch-n8n-main n8n import:workflow --input=/tmp/*.json
```

---

## 📋 Available Workflows (19 total)

### 1️⃣ Core Workflows

| Workflow | Trigger | Deskripsi |
|:---------|:--------|:----------|
| `universal_chatbot.json` | Webhook POST | Multi-tenant chatbot: Config → Session → RAG → LLM → Save |
| `rag_indexer.json` | Webhook POST | Index FAQ & Products ke pgvector |
| `escalation_handler.json` | Webhook POST | Escalation ke Chatwoot |
| `master_automations.json` | Cron (setiap menit) | Execute due automations |

### 2️⃣ UMKM Workflows

| Workflow | Trigger | Deskripsi |
|:---------|:--------|:----------|
| `daily_revenue_digest.json` | Cron (harian) | Revenue digest ke Telegram |
| `freeze_reminder.json` | Cron | Reminder expired subscriptions |
| `inventory_alert.json` | Webhook | Low stock alert |
| `umkm_low_stock_alert.json` | Webhook | Stock threshold notifications |
| `umkm_clinic_reminder.json` | Cron | Appointment reminders (klinik) |
| `voucher_wa_distribute.json` | Webhook | Distribusi voucher via WA |
| `ocr_accounting.json` | Webhook | OCR receipt → journal entry |
| `courier_tracking.json` | Webhook | Track pengiriman (JNE/SiCepat) |

### 3️⃣ Campaign Workflows

| Workflow | Trigger | Deskripsi |
|:---------|:--------|:----------|
| `campaign_voter_onboard.json` | Webhook | Voter onboarding automation |
| `campaign_blast.json` | Webhook | Mass WhatsApp blast |
| `campaign_fraud_alert.json` | Webhook | Deteksi fraud voting |
| `realcount_aggregator.json` | Webhook | Aggregate real count data |

### 4️⃣ Utility Workflows

| Workflow | Trigger | Deskripsi |
|:---------|:--------|:----------|
| `global_error_handler.json` | Error event | Centralized error handling |
| `webhook_retry.json` | Webhook | Retry failed webhook deliveries |

---

## 🔗 Integration dengan Backend

### Trigger Workflow dari Go Service

**Pattern: Webhook POST ke n8n**

```go
// Trigger n8n workflow via webhook
func triggerN8NWorkflow(workflowWebhook string, payload map[string]interface{}) error {
    jsonData, _ := json.Marshal(payload)
    
    resp, err := http.Post(
        workflowWebhook,
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("n8n webhook failed: %d", resp.StatusCode)
    }
    return nil
}

// Example: Trigger chatbot workflow
func handleChatbotMessage(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...
    
    payload := map[string]interface{}{
        "tenant_id": tenantID,
        "message": userMessage,
        "phone": phoneNumber,
    }
    
    webhookURL := "http://n8n-main:5678/webhook/chatbot"
    if err := triggerN8NWorkflow(webhookURL, payload); err != nil {
        slog.Error("Failed to trigger n8n", "error", err)
    }
}
```

### Get Webhook URL

**Di n8n UI:**
1. Open workflow
2. Click "Webhook" node
3. Copy "Production URL" → contoh: `http://localhost:5678/webhook/abc123`

**Untuk internal (Go → n8n):**
```
http://n8n-main:5678/webhook/<webhook-id>
```

**Untuk external (WhatsApp → n8n):**
```
https://yourdomain.com/n8n/webhook/<webhook-id>
```

---

## 🧪 Testing Workflows

### Manual Test via UI

1. Open workflow di n8n
2. Click "Execute Workflow" (play button)
3. Lihat execution log
4. Debug jika error

### Test via cURL

```bash
# Test chatbot webhook
curl -X POST http://localhost:5678/webhook/chatbot \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "test-tenant",
    "message": "Halo, saya mau tanya stok",
    "phone": "6281234567890"
  }'

# Expected: 200 OK + workflow execution started
```

### Monitor Executions

**Via UI:**
- n8n → "Executions" tab
- Filter by workflow / status / date
- Click execution → lihat detail tiap node

**Via Logs:**
```bash
docker logs wch-n8n-main -f --tail 100
docker logs wch-n8n-worker -f --tail 100
```

---

## ⚙️ Configuration

### Environment Variables

**Dari `.env`:**
```bash
# n8n Admin
N8N_ADMIN_USER=admin
N8N_ADMIN_PASSWORD=ganti_password_n8n_yang_kuat

# Database (dedicated DB)
N8N_DB_NAME=wch_n8n
N8N_DB_HOST=postgres
N8N_DB_PORT=5432
N8N_DB_USER=wch_admin
N8N_DB_PASSWORD=secure_postgres_password_123

# Encryption (WAJIB 32-byte hex)
N8N_ENCRYPTION_KEY=replace_with_32_byte_hex_key

# Queue Mode
N8N_WORKER_REPLICAS=3
N8N_CONCURRENCY_PRODUCTION_LIMIT=20
N8N_WORKER_CONCURRENCY=10

# URLs
N8N_PUBLIC_URL=http://localhost:5678
N8N_INTERNAL_URL=http://n8n-main:5678

# Telegram (untuk notifications)
OWNER_TELEGRAM_CHAT_ID=
```

### Scaling Workers

```bash
# Scale to 5 workers
docker compose up -d --scale n8n-worker=5

# Verify
docker ps | grep n8n-worker
```

---

## 🔐 Security

### Production Checklist

- [ ] Ubah `N8N_ADMIN_PASSWORD` di `.env`
- [ ] Set `N8N_ENCRYPTION_KEY` unik (32 bytes)
- [ ] Simpan encryption key di vault (HashiCorp/AWS Secrets)
- [ ] Restrict n8n UI via nginx (auth_basic atau IP whitelist)
- [ ] HTTPS untuk webhook eksternal
- [ ] Rate limit webhook endpoints

### Backup & Restore

**Backup:**
```bash
# Export workflows
docker exec wch-n8n-main n8n export:workflow --all --output=/tmp/workflows.json
docker cp wch-n8n-main:/tmp/workflows.json ./backup/

# Backup database (pgbouncer host-mapped port 10433)
pg_dump -h localhost -p 10433 -U wch_admin wch_n8n > n8n_backup.sql
```

**Restore:**
```bash
# Restore workflows
docker cp ./backup/workflows.json wch-n8n-main:/tmp/
docker exec wch-n8n-main n8n import:workflow --input=/tmp/workflows.json

# Restore database
psql -h localhost -p 10433 -U wch_admin -d wch_n8n < n8n_backup.sql
```

⚠️ **CRITICAL:** `N8N_ENCRYPTION_KEY` harus SAMA saat restore. Jika berbeda, credentials di DB tidak bisa didekripsi.

---

## 📊 Monitoring

### Health Check

```bash
# n8n health
curl http://localhost:5678/healthz

# Redis (queue)
redis-cli -h localhost -p 10631 PING

# PostgreSQL (persistence)
psql -h localhost -p 10433 -U wch_admin -d wch_n8n -c "SELECT COUNT(*) FROM workflows;"
```

### Metrics (Future: Prometheus)

**Planned metrics:**
- `n8n_workflow_executions_total{workflow,status}`
- `n8n_workflow_duration_seconds{workflow}`
- `n8n_queue_size{queue}`
- `n8n_worker_busy{worker_id}`

---

## 🐛 Troubleshooting

### Issue: Workflow tidak execute

**Check:**
1. Workflow aktif? (toggle di UI)
2. Webhook URL benar? (cek di node settings)
3. n8n-worker jalan? (`docker ps | grep n8n-worker`)
4. Redis connection? (`docker logs wch-n8n-main | grep redis`)

### Issue: Database connection error

**Fix:**
```bash
# Verify DB exists
psql -h localhost -p 10433 -U wch_admin -l | grep wch_n8n

# Create if missing
psql -h localhost -p 10433 -U wch_admin -c "CREATE DATABASE wch_n8n;"

# Restart n8n
docker compose restart n8n-main n8n-worker
```

### Issue: Execution stuck

**Debug:**
```bash
# Check worker logs
docker logs wch-n8n-worker --tail 100

# Check queue
redis-cli -h localhost -p 10631 LLEN bull:n8n:jobs:waiting

# Restart worker
docker compose restart n8n-worker
```

---

## 📚 Resources

- [n8n Official Docs](https://docs.n8n.io/)
- [n8n Queue Mode](https://docs.n8n.io/hosting/scaling/queue-mode/)
- [n8n API](https://docs.n8n.io/api/)
- [Community Workflows](https://n8n.io/workflows/)

---

## 🔄 Migration Checklist (Dev → Production)

- [ ] Export workflows dari dev n8n
- [ ] Backup `wch_n8n` database (`pg_dump`)
- [ ] Copy `.env` dengan updated credentials
- [ ] Restore database di production server
- [ ] Import workflows di production n8n
- [ ] Update webhook URLs di backend code
- [ ] Test 1 workflow end-to-end
- [ ] Monitor executions selama 24 jam
- [ ] Setup alerts untuk failed executions
