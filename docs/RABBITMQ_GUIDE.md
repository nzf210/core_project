# RabbitMQ Queue Architecture — WCH Platform

**Dokumentasi operasional RabbitMQ untuk async job processing di WCH Platform.**

---

## Overview

WCH Platform menggunakan **RabbitMQ 3.13** untuk memisahkan operasi berat dari HTTP request cycle. Arsitektur hybrid: operasi read langsung ke database, operasi write berat masuk queue.

**Target kapasitas:**
- HTTP throughput: 10K-15K concurrent requests
- Queue buffer: 50K-100K+ jobs
- Worker scaling: Horizontal via Docker replicas

---

## Infrastructure Setup

### Docker Compose

RabbitMQ sudah terkonfigurasi di `docker-compose.yml`:

```yaml
rabbitmq:
  image: rabbitmq:3.13-management-alpine
  container_name: wch-rabbitmq
  environment:
    RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER:-wch_admin}
    RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD:-rabbitmq_pass}
    RABBITMQ_DEFAULT_VHOST: /
  ports:
    - "10672:5672"   # DEV: AMQP protocol
    - "10673:15672"  # DEV: Management UI
  volumes:
    - rabbitmq_data:/var/lib/rabbitmq
  healthcheck:
    test: rabbitmq-diagnostics -q ping
    interval: 10s
    timeout: 5s
    retries: 5
  restart: unless-stopped
```

### Environment Variables

```bash
# .env / .env.staging
RABBITMQ_URL=amqp://wch_admin:rabbitmq_pass@rabbitmq:5672/
RABBITMQ_USER=wch_admin
RABBITMQ_PASSWORD=ganti_dengan_password_kuat
```

### Ports

| Environment | AMQP Port | Management UI | Access |
|:------------|:----------|:--------------|:-------|
| Development | `10672` | `10673` | http://localhost:10673 |
| Staging | `20672` | `20673` | http://157.15.40.27:20673 (via Cloudflare Tunnel) |
| Production | internal | internal | Internal only, monitoring via Prometheus |

**Login Management UI:**
- Username: `wch_admin` (dari `RABBITMQ_USER`)
- Password: `rabbitmq_pass` (dari `RABBITMQ_PASSWORD`)

---

## Queue Design

### Naming Convention

Format: `{domain}.{operation}`

| Queue Name | Purpose | Producer | Consumer | Priority |
|:-----------|:--------|:---------|:---------|:---------|
| `accounting.transactions` | Journal entry batch | `umkm-accounting` | `umkm-automation` | High |
| `voucher.distribution` | Voucher WA blast | `billing-service` | `umkm-automation` | Medium |
| `chatbot.replies` | AI chatbot response | `umkm-chatbot` | `umkm-automation` | High |
| `notifications.email` | Email notifications | `notification-service` | `notification-service` | Low |
| `notifications.telegram` | Telegram notifications | `notification-service` | `notification-service` | Low |
| `notifications.wa` | WhatsApp notifications | `wa-gateway` | `wa-gateway` | Medium |

### Job Structure

```json
{
  "job_id": "uuid-v4",
  "tenant_id": "tenant-uuid",
  "type": "accounting.journal_batch",
  "data": {
    "entries": [...],
    "metadata": {...}
  },
  "created_at": "2026-08-22T10:30:00Z"
}
```

---

## Code Integration

### Shared Client Library

**Location:** `shared/sdk/queue/rabbitmq.go`

**Initialize in service main.go:**

```go
import "core_project/shared/sdk/queue"

func main() {
    rabbitMQ, err := queue.NewClient(os.Getenv("RABBITMQ_URL"))
    if err != nil {
        log.Fatal("Failed to connect to RabbitMQ:", err)
    }
    defer rabbitMQ.Close()
    
    // Start consumers (blocking call)
    go rabbitMQ.Consume("accounting.transactions", handleAccountingJob)
    
    // Start HTTP server
    http.ListenAndServe(":8201", router)
}
```

### Producer Pattern (HTTP Handler)

```go
func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Header.Get("X-Tenant-ID")
    
    var req TransactionRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Validate sync
    if err := validateTransaction(&req); err != nil {
        writeJSON(w, 400, Response{Message: err.Error()})
        return
    }
    
    // Enqueue async processing
    jobID := uuid.New().String()
    job := queue.Job{
        JobID:    jobID,
        TenantID: tenantID,
        Type:     "accounting.journal_batch",
        Data:     map[string]interface{}{"request": req},
        CreatedAt: time.Now(),
    }
    
    if err := rabbitMQ.Publish(r.Context(), "accounting.transactions", job); err != nil {
        writeJSON(w, 500, Response{Message: "Failed to enqueue job"})
        return
    }
    
    // Save job status to DB
    _, err = DB.Exec(r.Context(), `
        INSERT INTO async_jobs (job_id, tenant_id, type, status, data)
        VALUES ($1, $2, $3, 'pending', $4)
    `, jobID, tenantID, job.Type, job.Data)
    
    // Immediate response
    writeJSON(w, 202, Response{
        Success: true,
        Message: "Transaction queued for processing",
        Data:    map[string]string{"job_id": jobID},
    })
}
```

### Consumer Pattern (Worker)

```go
func handleAccountingJob(job queue.Job) error {
    ctx := context.Background()
    
    // Update status to processing
    DB.Exec(ctx, `
        UPDATE async_jobs SET status = 'processing', started_at = NOW()
        WHERE job_id = $1
    `, job.JobID)
    
    // Process job
    var req TransactionRequest
    json.Unmarshal([]byte(job.Data["request"].(string)), &req)
    
    result, err := processTransaction(ctx, job.TenantID, &req)
    if err != nil {
        // Mark as failed
        DB.Exec(ctx, `
            UPDATE async_jobs SET status = 'failed', error = $1, completed_at = NOW()
            WHERE job_id = $2
        `, err.Error(), job.JobID)
        return err
    }
    
    // Mark as completed
    DB.Exec(ctx, `
        UPDATE async_jobs SET status = 'completed', result = $1, completed_at = NOW()
        WHERE job_id = $2
    `, result, job.JobID)
    
    return nil
}
```

### Job Status API

```go
func handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
    jobID := r.URL.Query().Get("job_id")
    tenantID := r.Header.Get("X-Tenant-ID")
    
    var status, errorMsg string
    var result interface{}
    
    err := DB.QueryRow(r.Context(), `
        SELECT status, result, error
        FROM async_jobs
        WHERE job_id = $1 AND tenant_id = $2
    `, jobID, tenantID).Scan(&status, &result, &errorMsg)
    
    if err != nil {
        writeJSON(w, 404, Response{Message: "Job not found"})
        return
    }
    
    writeJSON(w, 200, Response{
        Success: true,
        Data: map[string]interface{}{
            "job_id": jobID,
            "status": status,
            "result": result,
            "error":  errorMsg,
        },
    })
}
```

---

## Database Schema

### async_jobs Table

**Migration:** `000076_async_jobs.up.sql`

```sql
CREATE TABLE async_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    data JSONB,
    result JSONB,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_async_jobs_tenant_id ON async_jobs(tenant_id);
CREATE INDEX idx_async_jobs_status ON async_jobs(status);
CREATE INDEX idx_async_jobs_type ON async_jobs(type);
CREATE INDEX idx_async_jobs_created_at ON async_jobs(created_at DESC);
```

**Status flow:** `pending` → `processing` → `completed` / `failed`

---

## Scaling Workers

### Horizontal Scaling

```bash
# Development
docker compose up -d --scale umkm-automation=3

# Staging
COMPOSE_PROJECT_NAME=wch-stg docker compose \
  -f docker-compose.yml -f docker-compose.staging.yml \
  up -d --scale umkm-automation=3

# Check running workers
docker ps | grep umkm-automation
```

### Auto-scaling Guidelines

**CPU-based:**
- Scale up: avg CPU > 70% for 5 minutes
- Scale down: avg CPU < 30% for 10 minutes
- Min replicas: 1
- Max replicas: 5

**Queue depth-based:**
- Scale up: queue depth > 1000 messages for 2 minutes
- Scale down: queue depth < 100 for 5 minutes

---

## Monitoring

### RabbitMQ Management UI

**Access:** http://localhost:10673 (dev) | http://157.15.40.27:20673 (staging)

**Key metrics:**
- **Queue depth:** Jumlah message yang belum diproses
- **Consumer count:** Jumlah worker aktif per queue
- **Message rate:** Publish/deliver/ack per second
- **Unacked messages:** Message yang sedang diproses

### Grafana Dashboard

**Metrics via Prometheus:**

```promql
# Queue depth
rabbitmq_queue_messages{queue="accounting.transactions"}

# Message rate
rate(rabbitmq_queue_messages_published_total[5m])

# Consumer lag
rabbitmq_queue_messages - rabbitmq_queue_messages_ready
```

**Alerts:**
- Queue depth > 5000 untuk 10 menit → WARNING
- Queue depth > 10000 untuk 5 menit → CRITICAL
- No consumers untuk 3 menit → CRITICAL
- Message age > 30 menit → WARNING

### Database Monitoring

```sql
-- Pending jobs count
SELECT COUNT(*) FROM async_jobs WHERE status = 'pending';

-- Failed jobs count (last 1 hour)
SELECT COUNT(*) FROM async_jobs 
WHERE status = 'failed' AND created_at > NOW() - INTERVAL '1 hour';

-- Average processing time
SELECT AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) AS avg_seconds
FROM async_jobs 
WHERE status = 'completed' AND completed_at > NOW() - INTERVAL '1 hour';

-- Top slow jobs
SELECT job_id, type, EXTRACT(EPOCH FROM (completed_at - started_at)) AS duration_seconds
FROM async_jobs 
WHERE status = 'completed'
ORDER BY duration_seconds DESC
LIMIT 10;
```

---

## Troubleshooting

### Queue Depth Terus Naik

**Symptom:** Queue depth naik terus, worker tidak bisa keep up

**Diagnosis:**
1. Cek jumlah consumer: Management UI → Queues → {queue_name} → Consumers
2. Cek CPU worker: `docker stats | grep umkm-automation`
3. Cek error rate di logs: `docker logs wch-umkm-automation 2>&1 | grep ERROR`

**Solution:**
- Scale up workers: `docker compose up -d --scale umkm-automation=5`
- Optimize job handler (reduce external API calls, optimize DB queries)
- Increase worker QoS (prefetch count) di `queue.NewClient()`

### Message Stuck di "Unacked"

**Symptom:** Message masuk "unacked" tapi tidak completed/failed

**Diagnosis:**
1. Cek worker logs: `docker logs wch-umkm-automation --tail 100`
2. Cek apakah handler throw panic (message tidak di-ack/nack)

**Solution:**
- Restart worker: `docker restart wch-umkm-automation`
- Fix panic di handler (add recover)
- Set consumer timeout di RabbitMQ config

### Job Status API Lambat

**Symptom:** `GET /jobs/{id}` response time > 500ms

**Diagnosis:**
1. Cek index: `EXPLAIN SELECT * FROM async_jobs WHERE job_id = '...'`
2. Cek table size: `SELECT pg_size_pretty(pg_total_relation_size('async_jobs'))`

**Solution:**
- Add index: `CREATE INDEX idx_async_jobs_job_id ON async_jobs(job_id)`
- Cleanup old jobs (retention 30 hari):
  ```sql
  DELETE FROM async_jobs 
  WHERE completed_at < NOW() - INTERVAL '30 days' 
  AND status IN ('completed', 'failed');
  ```

### RabbitMQ Disk Space Penuh

**Symptom:** RabbitMQ stop accepting message, logs: `disk_free_limit`

**Diagnosis:**
```bash
docker exec wch-rabbitmq rabbitmq-diagnostics status | grep disk
```

**Solution:**
- Cleanup old messages via Management UI → Queues → Purge
- Increase disk space or move data directory
- Set retention policy untuk queues

---

## Backup & Recovery

### Backup RabbitMQ Definitions

```bash
# Export definitions (exchanges, queues, bindings)
docker exec wch-rabbitmq rabbitmqadmin export rabbitmq_definitions.json

# Backup to Git
cp rabbitmq_definitions.json infra/rabbitmq/definitions.json
git add infra/rabbitmq/definitions.json
git commit -m "chore: backup RabbitMQ definitions"
```

### Restore Definitions

```bash
# Import definitions
docker exec -i wch-rabbitmq rabbitmqadmin import < infra/rabbitmq/definitions.json
```

### Backup Job Status Database

```bash
# Dump async_jobs table
pg_dump -h localhost -p 10433 -U wch_admin -d wch_platform \
  -t async_jobs > backup_async_jobs.sql

# Restore
psql -h localhost -p 10433 -U wch_admin -d wch_platform < backup_async_jobs.sql
```

---

## Migration dari N8N Queue Mode

**Context:** WCH Platform sebelumnya menggunakan N8N Queue Mode (Redis Bull Queue) untuk workflow automation. RabbitMQ menggantikan ini untuk job processing yang lebih general-purpose.

**N8N tetap digunakan untuk:**
- Workflow automation kompleks (multi-step, conditional branching)
- External API orchestration (Meta, Gemini, Xendit webhooks)

**RabbitMQ digunakan untuk:**
- Heavy operations dari HTTP handlers (batch processing)
- Background jobs yang butuh retry & DLQ
- High-throughput message processing

**Migration path:**
1. Keep N8N workflows untuk automation use cases
2. Move heavy HTTP operations ke RabbitMQ queues
3. Integrate: N8N workflow bisa publish ke RabbitMQ queue via HTTP node

---

## Production Checklist

**Before deploy:**
- [ ] `RABBITMQ_PASSWORD` diganti dari default
- [ ] Management UI tidak expose ke public (hanya internal atau VPN)
- [ ] Disk space monitoring aktif (min 10GB free)
- [ ] Prometheus alerts configured
- [ ] Backup script scheduled (daily definitions export)
- [ ] Worker auto-scaling policy tested
- [ ] DLQ (Dead Letter Queue) configured untuk failed jobs
- [ ] Job retention policy implemented (cleanup after 30 days)

**Monitoring dashboard:**
- [ ] Queue depth per queue
- [ ] Message rate (publish/consume)
- [ ] Worker count per queue
- [ ] Average job processing time
- [ ] Failed job count (last 1 hour)

---

## References

- **RabbitMQ Documentation:** https://www.rabbitmq.com/documentation.html
- **Go AMQP Client:** https://github.com/rabbitmq/amqp091-go
- **WCH Platform Architecture:** `docs/DEVELOPER_GUIDE.md`
- **Port Registry:** `docs/PORT_REGISTRY.md`
- **Migration Registry:** `docs/MIGRATION_REGISTRY.md`
