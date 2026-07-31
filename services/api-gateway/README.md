# API Gateway

**Port:** 8000  
**Database:** PostgreSQL (tenant lookup)  
**Cache:** Redis (rate limiting)

## Deskripsi

Reverse proxy dan routing gateway untuk seluruh WCH Platform. Menangani rate limiting, CORS, authentication middleware, dan routing ke backend services.

## Fitur Utama

- 🚦 **Rate Limiting** — Per-IP (public) & per-tenant (authenticated)
- 🔐 **JWT Middleware** — Auto-inject `X-Tenant-ID` header
- 🌐 **CORS** — Cross-origin request handling
- 📊 **Metrics** — Request count, latency, error rate
- 🔄 **Service Routing** — Proxy ke auth, billing, UMKM, campaign
- 🎫 **Quota Enforcement** — Feature gating per plan
- 🔌 **Webhook Proxy** — Xendit, WA, N8N callbacks

## Environment Variables

```bash
# Database (untuk tenant lookup & quota check)
DATABASE_URL=postgres://user:pass@localhost:5433/wch_platform

# Redis (rate limiting)
REDIS_ADDR=localhost:6381
REDIS_PASSWORD=
REDIS_DB=0

# Server
PORT=8000
ENV=development  # or production

# JWT (untuk validasi token)
JWT_SECRET=your-secret-key-min-32-chars
```

## Routing Table

### Public Routes (No Auth)

| Path | Target Service | Rate Limit |
|:-----|:--------------|:-----------|
| `/auth/*` | auth-service:8001 | 100 req/min per IP |
| `/api/public/campaign/*` | campaign-api:9002 | 100 req/min per IP |
| `/uploads/*` | auth-service:8001/static | No limit |

### Webhook Routes (Signature Verified)

| Path | Target Service | Rate Limit |
|:-----|:--------------|:-----------|
| `/webhooks/xendit/*` | billing-service:8003 | 500 req/min per IP |
| `/webhooks/xendit/campaign/*` | campaign-api:9002 | 500 req/min per IP |
| `/webhooks/wa-cloud` | wa-cloud-api:8210 | 500 req/min per IP |
| `/webhooks/n8n/*` | n8n:5678 | 500 req/min per IP |

### Protected Routes (JWT Required)

| Path | Target Service | Rate Limit | Features |
|:-----|:--------------|:-----------|:---------|
| `/api/umkm/*` | umkm-accounting:8201 | 1000 req/min per tenant | - |
| `/api/umkm/business/*` | umkm-business:9005 | 1000 req/min per tenant | - |
| `/api/umkm/chat` | umkm-chatbot:8203 | 100 req/min per tenant | AI quota |
| `/api/ai/*` | ai-gateway:8002 | 50 req/min per tenant | AI feature gate |
| `/api/campaign/*` | campaign-api:9002 | 1000 req/min per tenant | - |
| `/api/billing/*` | billing-service:8003 | 100 req/min per tenant | - |
| `/api/wa/*` | wa-gateway:8202 | 60 req/min per tenant | - |
| `/api/wa-cloud/*` | wa-cloud-api:8210 | 60 req/min per tenant | - |

### Superadmin Routes (Superadmin Role Only)

| Path | Target Service |
|:-----|:--------------|
| `/api/superadmin/billing/*` | billing-service:8003/admin |
| `/api/superadmin/tenants/*` | auth-service:8001/superadmin |
| `/api/superadmin/n8n/*` | n8n:5678 |

## Rate Limiting

### Public Rate Limit (Per-IP)
- **Limit:** 100 requests/menit
- **Scope:** Per source IP address
- **Key:** `rate:public:{ip}`
- **Window:** Sliding window 60 detik

### Tenant Rate Limit (Per-Tenant)
- **Limit:** 1000 requests/menit (default)
- **Scope:** Per tenant ID dari JWT
- **Key:** `rate:tenant:{tenantId}`
- **Window:** Sliding window 60 detik

### Custom Limits
- **AI Gateway:** 50 req/min (mahal, butuh quota)
- **WA Gateway:** 60 req/min (anti-ban WhatsApp)
- **Webhooks:** 500 req/min (high throughput)

**Response saat rate limited:**
```json
HTTP/1.1 429 Too Many Requests
{
  "success": false,
  "message": "Rate limit exceeded. Try again later."
}
```

## Quota Middleware

Quota middleware meng-enforce feature gating berdasarkan plan tenant:

```go
// Cek apakah tenant punya akses ke fitur tertentu
func quotaMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get("X-Tenant-ID")
        
        // Cek plan tenant dari Redis/DB
        plan := getTenantPlan(tenantID)
        
        // Cek feature requirement
        feature := r.Header.Get("X-Require-Feature")
        if feature != "" && !plan.HasFeature(feature) {
            http.Error(w, "Feature not available in your plan", 403)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

**Plan Features:**
- **Free:** Basic accounting, 10 transaksi/bulan
- **Lite:** + Chatbot, 100 transaksi/bulan
- **Pro:** + AI features, unlimited transaksi
- **Ultimate:** + Multi-location, advanced analytics

## CORS Configuration

```go
// Allowed origins
allowedOrigins := []string{
    "http://localhost:3201",  // UMKM Web (dev)
    "http://localhost:3301",  // Campaign Web (dev)
    "http://localhost:3401",  // Superadmin Web (dev)
    "https://app.wch-platform.com",
    "https://campaign.wch-platform.com",
    "https://admin.wch-platform.com",
}

// CORS headers
Access-Control-Allow-Origin: <origin>
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Tenant-ID
Access-Control-Max-Age: 3600
```

## Security Headers

Gateway menambahkan security headers ke semua response:

```
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## Request Flow

### Authenticated Request
```
Frontend (port 3201)
    ↓ POST /api/umkm/transactions
    ↓ Headers: Authorization: Bearer <jwt>, X-Tenant-ID: <uuid>
    ↓
API Gateway (port 8000)
    ↓ Validate JWT (shared/sdk/auth/middleware.go)
    ↓ Extract tenant_id dari JWT claims
    ↓ Check rate limit (Redis: rate:tenant:<tenantId>)
    ↓ Check quota (plan features)
    ↓ Proxy request ke umkm-accounting:8201/transactions
    ↓
UMKM Accounting Service
    ↓ Process transaction
    ↓ Return response
    ↓
API Gateway
    ↓ Add security headers
    ↓ Return to frontend
```

### Webhook Request
```
Xendit Server
    ↓ POST /webhooks/xendit/invoice.paid
    ↓ Headers: X-Callback-Token: <signature>
    ↓
API Gateway (port 8000)
    ↓ Rate limit check (500 req/min per IP)
    ↓ Proxy ke billing-service:8003/webhook
    ↓
Billing Service
    ↓ Verify signature (Xendit webhook token)
    ↓ Process payment
    ↓ Return 200 OK
```

## Monitoring

### Metrics (Prometheus)
```
# Request metrics
http_requests_total{service, method, status}
http_request_duration_seconds{service, method}

# Rate limit metrics
rate_limit_hits_total{scope}  # scope: public/tenant
rate_limit_blocked_total{scope}

# Proxy metrics
proxy_backend_errors_total{service}
proxy_backend_latency_seconds{service}
```

### Health Check
```bash
# Gateway health
curl http://localhost:8000/health

# Response
{
  "status": "ok",
  "redis": "connected",
  "database": "connected",
  "uptime": "2h35m"
}
```

## Testing

```bash
# Run tests
go test ./services/api-gateway/... -v

# Security tests
go test -run TestRateLimit -v
go test -run TestCORS -v
go test -run TestSecurityHeaders -v

# Load testing
hey -n 10000 -c 100 http://localhost:8000/api/umkm/dashboard \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <uuid>"
```

## Troubleshooting

### Error: 429 Too Many Requests
**Penyebab:** Rate limit exceeded.  
**Solusi:**
```bash
# Cek rate limit key di Redis
redis-cli -p 6381
GET rate:tenant:<tenantId>

# Reset manual (development only)
DEL rate:tenant:<tenantId>
```

### Error: 502 Bad Gateway
**Penyebab:** Target service down atau unreachable.  
**Solusi:**
```bash
# Cek service status
make status

# Restart service
make dev-accounting  # contoh untuk accounting service
```

### Error: 403 Feature not available
**Penyebab:** Tenant plan tidak support fitur yang di-request.  
**Solusi:**
```bash
# Cek plan tenant
psql -h localhost -p 5433 -U wch_admin -d wch_platform
SELECT id, plan FROM tenants WHERE id = '<tenantId>';

# Upgrade plan via billing dashboard
```

### Rate Limiting tidak bekerja
**Penyebab:** Redis connection failed.  
**Solusi:**
```bash
# Cek Redis
docker ps | grep redis
redis-cli -p 6381 PING

# Restart Redis
docker compose restart redis
```

## Development

```bash
# Run dengan hot reload
make dev-gateway

# Run manual
go run services/api-gateway/*.go

# Test single endpoint
curl -X POST http://localhost:8000/api/umkm/transactions \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <uuid>" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100000, "description": "Test"}'
```

## Production Checklist

- [ ] Set `ENV=production`
- [ ] Enable HTTPS (Nginx/Caddy upstream)
- [ ] Set Redis password
- [ ] Configure rate limits sesuai traffic
- [ ] Enable request logging (JSON format)
- [ ] Set up Prometheus scraping
- [ ] Configure CORS allowed origins (production domains only)
- [ ] Test all protected routes dengan JWT
- [ ] Verify webhook signature validation
- [ ] Set up alerting untuk 5xx errors
- [ ] Load test (target: 1000 RPS)

## Load Testing Results

Target: **1000 RPS**, 95th percentile latency < 100ms

```bash
# Command
hey -n 100000 -c 100 -q 1000 http://localhost:8000/api/umkm/dashboard \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: <uuid>"

# Expected Results
Summary:
  Total:        100.0 secs
  Requests/sec: 999.8
  
Latency distribution:
  50% in 45ms
  95% in 82ms
  99% in 120ms
```

## Architecture Notes

### Why API Gateway?

1. **Centralized Auth** — JWT validation satu tempat, tidak duplikat di tiap service
2. **Rate Limiting** — Anti-abuse, protect backend dari overload
3. **Tenant Isolation** — Automatic tenant_id injection dari JWT
4. **CORS** — Satu konfigurasi untuk semua frontend
5. **Observability** — Central logging & metrics
6. **Feature Gating** — Quota enforcement per plan

### Pattern: Service Discovery

API Gateway menggunakan environment-based service discovery:

```go
func getTarget(service string, port string) string {
    if cfg.Env == "production" {
        return "http://" + service + ":" + port  // Docker network
    }
    return "http://localhost:" + port  // Native dev
}
```

**Production (Docker):** Service name resolves via Docker network DNS  
**Development (Native):** Services run di localhost dengan port masing-masing

## Related Services

- **Auth Service** (8001) — JWT generation & validation
- **All backend services** — Proxied via gateway
- **Redis** (6381) — Rate limit storage
- **PostgreSQL** (5433) — Tenant & plan data
