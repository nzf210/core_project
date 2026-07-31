# WA Gateway

**Port:** 8202  
**Database:** PostgreSQL (wa_sessions, tenant_chatbot_configs)  
**Cache:** Redis DB 9 (distributed locks), Redis DB 0 (OTP shared)

## Deskripsi

WhatsApp Gateway dengan **hybrid architecture**: whatsmeow (unofficial) untuk chatbot conversational + Meta Cloud API (official) untuk broadcast transactional. Mendukung multi-tenant dengan session pool dan rate limiting anti-ban.

## Fitur Utama

- 🔄 **Hybrid Provider** — Auto-routing berdasarkan message type
- 📱 **Multi-Tenant Sessions** — Pool WA session per-tenant
- 🚦 **Rate Limiter** — Token bucket 5 msg/min per tenant (whatsmeow)
- 🔐 **QR Code Auth** — Scan QR untuk koneksi whatsmeow
- ☁️ **Cloud API Integration** — Meta Business API untuk broadcast
- 🔄 **Auto-Reconnect** — Exponential backoff (30s → 10min)
- 🔒 **Distributed Lock** — Redis-based multi-instance coordination
- 💓 **Heartbeat** — Instance health monitoring

## Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5433/wch_platform

# Redis
REDIS_ADDR=localhost:6381
REDIS_PASSWORD=
# DB 9 untuk WA coordination (locks, heartbeat)
# DB 0 untuk shared OTP keys

# Server
PORT=8202
ENV=development

# WhatsApp (whatsmeow) — auto store di DB
# No extra config needed

# Meta Cloud API
# Per-tenant credentials di tabel wa_cloud_api_credentials
```

## Hybrid Architecture

### Provider Routing Logic

```
Request ke /send
    ↓
Baca header X-Message-Type
    ↓
    ├─ "broadcast" → WAJIB Cloud API (jika QR tenant → reject 402)
    ├─ "otp" → Auto-routing (WCH System gunakan Cloud, QR tenant gunakan whatsmeow)
    ├─ "invoice" → Auto-routing
    ├─ "subscription" → Auto-routing
    ├─ "system" → Auto-routing
    └─ (no header) → whatsmeow (chatbot conversational)
    ↓
Cek tenant preference (wa_provider_preference)
    ↓
    ├─ "auto" → Hybrid logic di atas
    ├─ "whatsmeow" → Force whatsmeow (skip Cloud API)
    └─ "cloud_api" → Force Cloud API (no fallback, error jika gagal)
    ↓
Send via provider → Fallback jika Cloud API gagal
```

### Message Types

| Type | Default Provider | Use Case |
|:-----|:----------------|:---------|
| `broadcast` | Cloud API (WAJIB) | Marketing blast, announcement massal |
| `otp` | Auto-routing | OTP login/register |
| `invoice` | Auto-routing | Invoice payment reminder |
| `subscription` | Auto-routing | Subscription expiry notification |
| `system` | Auto-routing | System alerts |
| *(empty)* | whatsmeow | Chatbot conversational, customer service |

## API Endpoints

### POST `/send`
Kirim pesan WhatsApp (auto-routed).

**Headers:**
```
X-Tenant-ID: <tenant-uuid>
X-Message-Type: otp|invoice|subscription|system|broadcast  # optional
X-WA-Provider-Override: auto|whatsmeow|cloud_api  # optional (auth-service OTP)
```

**Request:**
```json
{
  "to": "6281234567890",
  "message": "Halo dari Toko Berkah!"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "messageId": "wamid.xxx",
    "provider": "cloud_api"  // or "whatsmeow"
  }
}
```

**Error Responses:**
```json
// 402 Payment Required (broadcast via QR tenant)
{
  "success": false,
  "message": "Broadcast requires Cloud API. Tenant using QR connection."
}

// 429 Rate Limited (whatsmeow 5 msg/min exceeded)
{
  "success": false,
  "message": "Rate limit exceeded. Max 5 messages/minute."
}

// 502 Bad Gateway (Cloud API failed, no fallback on force mode)
{
  "success": false,
  "message": "Cloud API request failed"
}
```

### GET `/status`
Cek status koneksi WA tenant.

**Headers:**
```
X-Tenant-ID: <tenant-uuid>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "whatsmeow": {
      "status": "connected",  // or "disconnected", "qr_pending"
      "phoneNumber": "6281234567890",
      "lastSeen": "2026-07-31T10:30:00Z"
    },
    "cloudApi": {
      "status": "active",  // or "inactive", "error"
      "phoneNumberId": "123456789",
      "verifiedAt": "2026-07-25T08:00:00Z"
    },
    "preference": "auto"  // wa_provider_preference
  }
}
```

### GET `/qr`
Generate QR code untuk koneksi whatsmeow.

**Headers:**
```
X-Tenant-ID: <tenant-uuid>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "qr": "data:image/png;base64,iVBORw0KGgo...",
    "expiresIn": 60  // seconds
  }
}
```

**Flow:**
1. Frontend request `/qr`
2. Gateway generate QR via whatsmeow
3. Frontend display QR
4. User scan dengan WA mobile
5. Gateway detect connection → status jadi "connected"

### POST `/disconnect`
Disconnect whatsmeow session.

**Headers:**
```
X-Tenant-ID: <tenant-uuid>
```

**Response:**
```json
{
  "success": true,
  "message": "Session disconnected"
}
```

## Rate Limiting (Token Bucket)

### Whatsmeow Rate Limiter
- **Algorithm:** Token bucket
- **Capacity:** 5 tokens
- **Refill rate:** 5 tokens/minute (1 token per 12 detik)
- **Scope:** Per-tenant

**Implementation:**
```go
type tokenBucket struct {
    tokens   float64
    lastTime time.Time
}

func (rl *TenantRateLimiter) Allow(tenantID string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    bucket := rl.buckets[tenantID]
    if bucket == nil {
        bucket = &tokenBucket{tokens: 5, lastTime: time.Now()}
        rl.buckets[tenantID] = bucket
    }
    
    // Refill tokens based on time elapsed
    elapsed := time.Since(bucket.lastTime)
    refill := elapsed.Seconds() / 60.0 * float64(rl.rate)
    bucket.tokens = math.Min(float64(rl.rate), bucket.tokens + refill)
    bucket.lastTime = time.Now()
    
    if bucket.tokens >= 1 {
        bucket.tokens -= 1
        return true  // Allow
    }
    return false  // Rate limited
}
```

### Cloud API Rate Limit
- **No rate limit di gateway** (Meta handles it server-side)
- **Meta limit:** ~1000 msg/day (Tier 1), scale up dengan verification

## Multi-Tenant Session Management

### Session Pool (whatsmeow)
Tabel `wa_sessions`:
```sql
CREATE TABLE wa_sessions (
    id UUID PRIMARY KEY,
    tenant_id UUID UNIQUE REFERENCES tenants(id),
    phone_number VARCHAR(20),
    status VARCHAR(20),  -- connected, disconnected, qr_pending
    device_id VARCHAR(50),
    created_at TIMESTAMPTZ,
    last_connected_at TIMESTAMPTZ
);
```

### Cloud API Credentials
Tabel `wa_cloud_api_credentials`:
```sql
CREATE TABLE wa_cloud_api_credentials (
    id UUID PRIMARY KEY,
    tenant_id UUID UNIQUE REFERENCES tenants(id),
    access_token TEXT,  -- Encrypted AES-256-GCM
    phone_number_id VARCHAR(50),
    business_account_id VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    verification_status VARCHAR(20),  -- pending, verified, failed
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ
);
```

## Distributed Lock (Multi-Instance)

Untuk mencegah race condition saat multi-instance (horizontal scaling), WA Gateway menggunakan Redis distributed lock:

```go
// Acquire lock sebelum connect
lockKey := "wa:lock:" + tenantID
acquired := tryAcquireLock(redisClient, lockKey, instanceID, 5*time.Minute)

if !acquired {
    // Instance lain sudah handle tenant ini
    return
}

// Connect & maintain session
// ...

// Heartbeat setiap 30 detik
go func() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        extendLock(redisClient, lockKey, instanceID, 5*time.Minute)
    }
}()
```

**Keys:**
- `wa:lock:{tenantId}` — Distributed lock (TTL 5 min)
- `wa:owner:{tenantId}` — Instance ID yang owns session
- `wa:instance:{instanceId}` — Heartbeat timestamp

## Reconnect Strategy

### Exponential Backoff
```
Attempt 1: 30 seconds
Attempt 2: 60 seconds
Attempt 3: 120 seconds (2 min)
Attempt 4: 240 seconds (4 min)
Attempt 5+: 600 seconds (10 min) — max backoff
```

### Backoff Reset
- **Success reconnect** → Reset backoff ke 30s
- **Max 1 reconnect/5 menit** — Anti-spam protection

### Code:
```go
func calculateBackoff(attempt int) time.Duration {
    baseBackoff := 30 * time.Second
    maxBackoff := 10 * time.Minute
    
    backoff := baseBackoff * (1 << (attempt - 1))  // Exponential
    if backoff > maxBackoff {
        return maxBackoff
    }
    return backoff
}
```

## Security

### Phone Number Validation
```go
phoneRE := regexp.MustCompile(`^62[0-9]{9,13}$`)

if !phoneRE.MatchString(phoneNumber) {
    return errors.New("Invalid Indonesian phone number")
}
```

### Anti-Ban Best Practices
✅ **DO:**
- Rate limit 5 msg/min (whatsmeow)
- Gunakan Cloud API untuk broadcast
- Reconnect dengan exponential backoff
- Avoid spam patterns

❌ **DON'T:**
- Blast >100 msg/hari via whatsmeow
- Reconnect terlalu sering (<5 menit)
- Kirim identical messages ke banyak kontak
- Ignore Meta Business verification

## Testing

```bash
# Run tests
go test ./services/wa-gateway/... -v

# Security tests
go test -run TestTokenBucket -v
go test -run TestPhoneNumber -v
go test -run TestReconnect -v

# Integration test (butuh Redis + DB)
DATABASE_URL=postgres://... go test -run TestWARouting -v
```

## Monitoring

### Metrics (Prometheus)
```
wa_messages_total{channel, direction, status}
wa_messages_routed_total{provider}  # cloud_api vs whatsmeow
wa_fallback_total  # Cloud API → whatsmeow fallback count
wa_rate_limited_total  # Messages blocked by rate limiter
```

### Logs (slog JSON)
- Message sent (provider, tenant, recipient)
- Rate limit hit
- Reconnect attempts
- QR code generated
- Cloud API fallback triggered

## Troubleshooting

### Error: "Rate limit exceeded"
**Solusi:**
```bash
# Cek bucket tokens di memory (restart gateway untuk reset)
# atau tunggu 1 menit untuk refill
```

### QR Code expired
**Solusi:** Request `/qr` lagi. QR expire setelah 60 detik.

### whatsmeow disconnected randomly
**Penyebab:** WhatsApp server kicked session (spam detection).  
**Solusi:**
- Kurangi frequency message
- Gunakan Cloud API untuk transactional messages
- Pastikan rate limit aktif

### Cloud API 402 error
**Penyebab:** Tenant belum setup Cloud API credentials.  
**Solusi:**
1. Tenant register Meta Business Account
2. Create WhatsApp Business API
3. Save credentials via `/api/umkm/wa-cloud-api/credentials`

## Production Checklist

- [ ] Redis DB 9 aktif (distributed locks)
- [ ] Rate limiter enabled
- [ ] Exponential backoff configured
- [ ] Monitor reconnect frequency
- [ ] Alert on >10 rate limit hits/tenant/hour
- [ ] Cloud API credentials encrypted (AES-256-GCM)
- [ ] Test fallback Cloud API → whatsmeow
- [ ] Verify multi-instance coordination (jika scaled)

## Related Services

- **Auth Service** (8001) — OTP sender
- **WA Cloud API** (8210) — Cloud API proxy
- **N8N** — Chatbot workflow engine
- **API Gateway** (8000) — Rate limiting & routing
