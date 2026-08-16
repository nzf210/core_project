# WhatsApp Provider Routing Logic — Technical Documentation

**Date:** 2026-08-17  
**Scope:** P2-7 — Documentation for hybrid WhatsApp architecture routing  
**Services:** `wa-gateway`, `wa-cloud-api`, `auth-service`

---

## Executive Summary

WCH Platform uses a **hybrid WhatsApp architecture** combining two providers:

1. **whatsmeow (Unofficial)** — For conversational AI chatbot, internal OTP, low-volume notifications
2. **Meta Cloud API (Official)** — For mass broadcasting, transactional messages, compliance-critical flows

**Routing is dynamic** based on message type, tenant preference, and automatic fallback logic. This document explains how messages are routed between providers.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         wa-gateway (Port 8202)                  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  1. Resolve Provider Preference                           │  │
│  │     Priority: X-WA-Provider-Override > DB lookup > "auto" │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              ↓                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  2. Check Message Type                                     │  │
│  │     • isTransactional()?                                   │  │
│  │     • X-Message-Type header                                │  │
│  │     • X-Source service                                     │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              ↓                                   │
│         ┌────────────────────┴────────────────────┐             │
│         ↓ Cloud API                    ↓ whatsmeow              │
│  ┌──────────────────┐          ┌──────────────────┐             │
│  │ routeToCloudAPI()│          │ client.SendMessage│             │
│  │  (Meta Official) │          │   (QR session)    │             │
│  └──────────────────┘          └──────────────────┘             │
│         ↓                                 ↑                      │
│    ┌─────────┐                       Fallback                   │
│    │ wa-cloud│                      (auto mode only)            │
│    │   -api  │                                                   │
│    │ :8210   │                                                   │
│    └─────────┘                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Provider Resolution Logic

### Step 1: Resolve Provider Preference

**File:** `services/wa-gateway/cloud_routing.go:18-23`

```go
func resolveProviderPreference(r *http.Request, tenantID string) string {
    if override := r.Header.Get("X-WA-Provider-Override"); override != "" {
        return override
    }
    return getTenantWAProviderPreference(tenantID)
}
```

**Priority:**
1. **HTTP Header** `X-WA-Provider-Override` (highest) — Used by auth-service for OTP routing
2. **Database** `tenant_chatbot_configs.wa_provider_preference`
3. **Default** `"auto"` (if DB query fails or no config exists)

**Valid preference values:**
- `"auto"` (default) — Hybrid routing: transactional → Cloud API, conversational → whatsmeow
- `"whatsmeow"` — Force whatsmeow for ALL messages (skip Cloud API)
- `"cloud_api"` — Force Cloud API for ALL messages (no fallback to whatsmeow)

---

### Step 2: Check Message Type (Transactional vs Conversational)

**File:** `services/wa-gateway/cloud_routing.go:38-51`

```go
func isTransactional(r *http.Request) bool {
    msgType := r.Header.Get("X-Message-Type")
    switch msgType {
    case "otp", "invoice", "payment", "subscription", "system", "broadcast":
        return true
    }
    source := r.Header.Get("X-Source")
    switch source {
    case "auth-service", "billing-service", "notification-service":
        return true
    }
    return false
}
```

**Transactional message indicators:**

| Indicator | Values | Use Case |
|:----------|:-------|:---------|
| `X-Message-Type` | `otp`, `invoice`, `payment`, `subscription`, `system`, `broadcast` | Explicit message type classification |
| `X-Source` | `auth-service`, `billing-service`, `notification-service` | Service origin indicates transactional nature |

**Default:** Messages without these headers are treated as **conversational** (chatbot).

---

### Step 3: Route Decision

**File:** `services/wa-gateway/send_handlers.go:138-180`

```go
func handleCloudAPIRouting(w http.ResponseWriter, r *http.Request, tenantID, target, message string, skipCloud bool) bool {
    if skipCloud {
        return false  // Force whatsmeow
    }

    pref := resolveProviderPreference(r, tenantID)
    shouldTryCloud := pref == "cloud_api" || (pref == "auto" && isTransactional(r))

    if !shouldTryCloud {
        return false  // Use whatsmeow
    }

    waMsgID, err := routeToCloudAPI(tenantID, target, message, msgType)
    if err == nil {
        return true  // Cloud API success
    }

    // Cloud API failed
    if pref == "cloud_api" {
        // Forced Cloud API → no fallback, return error
        http.Error(w, `{"error":"Cloud API forced but failed"}`, http.StatusBadGateway)
        return true
    }

    // auto mode → fallback to whatsmeow
    return false
}
```

**Decision matrix:**

| Preference | Message Type | Cloud API | whatsmeow | Fallback? |
|:-----------|:-------------|:----------|:----------|:----------|
| `auto` | Transactional (`isTransactional()` = true) | ✅ Try first | ✅ Fallback if Cloud fails | YES |
| `auto` | Conversational (`isTransactional()` = false) | ❌ Skip | ✅ Direct | N/A |
| `whatsmeow` | Any | ❌ Skip | ✅ Direct | N/A |
| `cloud_api` | Any | ✅ Only | ❌ Never | NO — 502 error if Cloud fails |

---

## Routing Examples

### Example 1: OTP Login (Transactional)

**Request:**
```http
POST /api/wa/send
X-Message-Type: otp
X-Source: auth-service
X-WA-Provider-Override: auto

tenant_id=abc123&target=628123456789&message=Your OTP: 123456
```

**Flow:**
1. `resolveProviderPreference()` → `"auto"` (from header override)
2. `isTransactional()` → `true` (X-Message-Type: otp)
3. `shouldTryCloud` → `true` (auto + transactional)
4. `routeToCloudAPI()` → Sends via wa-cloud-api:8210
5. **If Cloud API fails** → Fallback to whatsmeow
6. **Response:** `{"success": true, "routed": "cloud_api", "wa_message_id": "wamid.xxx"}`

---

### Example 2: AI Chatbot Reply (Conversational)

**Request:**
```http
POST /api/wa/send
(no X-Message-Type header)
(no X-Source header)

tenant_id=abc123&target=628123456789&message=Halo, ada yang bisa saya bantu?
```

**Flow:**
1. `resolveProviderPreference()` → `"auto"` (DB lookup, default)
2. `isTransactional()` → `false` (no transactional indicators)
3. `shouldTryCloud` → `false` (auto + NOT transactional)
4. **Direct to whatsmeow** → `client.SendMessage()`
5. Rate limiter check (5 msg/min per tenant)
6. **Response:** `{"success": true}`

---

### Example 3: Mass Broadcast (Transactional + Cloud API Only)

**Request:**
```http
POST /api/wa/send
X-Message-Type: broadcast
(tenant has wa_provider_preference = "cloud_api" in DB)

tenant_id=abc123&target=628123456789&message=Promo Hari Ini: 50% OFF!
```

**Flow:**
1. `resolveProviderPreference()` → `"cloud_api"` (DB lookup)
2. `isTransactional()` → `true` (X-Message-Type: broadcast)
3. `shouldTryCloud` → `true` (cloud_api preference)
4. `routeToCloudAPI()` → Sends via wa-cloud-api:8210
5. **If Cloud API fails** → NO fallback (preference is forced)
6. **Error:** `502 Bad Gateway: "Cloud API forced but failed"`

---

### Example 4: Force whatsmeow (Skip Cloud)

**Request:**
```http
POST /api/wa/send
X-Message-Type: invoice
(tenant has wa_provider_preference = "whatsmeow" in DB)

tenant_id=abc123&target=628123456789&message=Invoice #12345: Rp 100,000
```

**Flow:**
1. `resolveProviderPreference()` → `"whatsmeow"` (DB lookup)
2. `skipCloud` → `true` (preference is whatsmeow)
3. **Skip Cloud API entirely** → `handleCloudAPIRouting()` returns false immediately
4. **Direct to whatsmeow** → `client.SendMessage()`
5. Rate limiter check (5 msg/min per tenant)
6. **Response:** `{"success": true}`

---

## Header Override for OTP Routing

**File:** `services/auth-service/main.go:356, 1371`

When auth-service sends OTP, it reads `tenants.auth_wa_provider_preference` and forwards to wa-gateway via `X-WA-Provider-Override` header:

```go
// auth-service/main.go:356 (handleRegister)
var authWAPreference string
err := DB.QueryRow(ctx, `SELECT auth_wa_provider_preference FROM tenants WHERE id = $1`, tenantID).Scan(&authWAPreference)
if err != nil {
    authWAPreference = "auto"
}

req.Header.Set("X-WA-Provider-Override", authWAPreference)
```

**Why separate from chatbot preference?**
- Chatbot messages: User preference (`wa_provider_preference`)
- Auth OTP messages: Tenant admin decision (`auth_wa_provider_preference`)

This allows a tenant to use whatsmeow for chatbot (conversational) but force Cloud API for auth OTP (compliance).

---

## Rate Limiting

### whatsmeow Rate Limiter

**File:** `services/wa-gateway/send_handlers.go:49-54`

```go
if !rateLimiter.Allow(tenantID) {
    slog.Warn("Rate limit exceeded for whatsmeow", "tenant_id", tenantID)
    waRateLimitedTotal.WithLabelValues().Inc()
    http.Error(w, `{"error":"Rate limit exceeded (max 5 msg/min). Use Cloud API for broadcasting."}`, http.StatusTooManyRequests)
    return
}
```

**Limits:**
- **whatsmeow:** 5 messages/minute per tenant (token bucket algorithm)
- **Cloud API:** No rate limit enforced by wa-gateway (Meta enforces their own limits)

**Anti-Ban Strategy:**
- Transactional messages → Cloud API (no ban risk)
- Conversational messages → whatsmeow with rate limiting (5 msg/min prevents spam detection)

---

## Cloud API Integration

### Routing to wa-cloud-api Service

**File:** `services/wa-gateway/cloud_routing.go:54-103`

```go
func routeToCloudAPI(tenantID, target, message, msgType string) (string, error) {
    cloudAPIHost := "http://localhost:8210"
    if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
        cloudAPIHost = "http://wa-cloud-api:8210"
    }

    payload := map[string]interface{}{
        "to":   target,
        "type": "text",
        "text": message,
    }

    req, err := http.NewRequest(http.MethodPost, cloudAPIHost+"/send", bytes.NewReader(body))
    req.Header.Set("X-Tenant-ID", tenantID)

    resp, err := http.DefaultClient.Do(req)
    // ... error handling ...

    return waMsgID, nil
}
```

**Environment-based routing:**
- **Development:** `http://localhost:8210` (wa-cloud-api running natively)
- **Production/Staging:** `http://wa-cloud-api:8210` (Docker container name)

**Tenant credentials:**
- wa-cloud-api reads `wa_cloud_api_credentials` table using `X-Tenant-ID` header
- Each tenant has separate Meta Business Account credentials

---

## Fallback Logic

### Auto Mode Fallback

**File:** `services/wa-gateway/send_handlers.go:176-179`

```go
// auto mode → fallback to whatsmeow
slog.Warn("Cloud API fallback to whatsmeow", "tenant_id", tenantID, "error", err)
waFallbackTotal.WithLabelValues().Inc()
return false
```

**Conditions for fallback:**
1. Preference is `"auto"`
2. Cloud API was attempted (transactional message)
3. Cloud API returned error (network, 4xx, 5xx)

**Metrics:** `wa_fallback_total` counter tracks fallback frequency per tenant.

### No Fallback for Forced Cloud API

```go
if pref == "cloud_api" {
    slog.Error("Cloud API forced but failed", "tenant_id", tenantID, "error", err)
    http.Error(w, `{"error":"Cloud API forced but failed"}`, http.StatusBadGateway)
    return true
}
```

When tenant explicitly sets `wa_provider_preference = "cloud_api"`, NO fallback occurs. This ensures compliance-critical flows don't accidentally use unofficial provider.

---

## Session Management

### whatsmeow Session Locking

**File:** `services/wa-gateway/send_handlers.go:56-59`

```go
if !acquireSessionLock(w, tenantID) {
    return
}
defer ReleaseSessionLock(context.Background(), tenantID)
```

**Why lock?**
- whatsmeow sessions are single-connection per tenant
- Concurrent sends from same tenant must be serialized
- Prevents "already connected" errors and message ordering issues

**Lock mechanism:**
- Redis SET NX with 5-minute TTL
- Heartbeat extends lock every 2 minutes
- Released on disconnect/logout

**Cloud API does NOT need locking** — Meta handles concurrency server-side.

---

## Reconnection Logic

### Automatic Reconnection Before Send

**File:** `services/wa-gateway/send_handlers.go:116-130`

```go
func ensureConnection(w http.ResponseWriter, client *whatsmeow.Client, tenantID string) bool {
    if client.IsConnected() {
        return true
    }
    if !shouldReconnect(tenantID) {
        writeServiceUnavailable(w, "WhatsApp disconnected. Reconnect backoff active.")
        return false
    }
    if err := client.Connect(); err != nil {
        writeServiceUnavailable(w, "Failed to reconnect to WA server")
        return false
    }
    return true
}
```

**Backoff schedule:**
- Attempt 1: 30s
- Attempt 2: 60s
- Attempt 3: 2m
- Attempt 4: 4m
- Attempt 5: 8m
- Attempt 6+: 10m (capped)

**Rate limit:** Max 1 reconnect attempt per 5 minutes per tenant.

---

## Metrics & Observability

### Prometheus Metrics

**File:** `services/wa-gateway/metrics.go`

```go
// Message routing
waRoutedTotal.WithLabelValues("cloud_api").Inc()    // Cloud API success
waRoutedTotal.WithLabelValues("whatsmeow").Inc()    // whatsmeow success

// Fallback tracking
waFallbackTotal.WithLabelValues().Inc()              // Cloud → whatsmeow fallback

// Rate limiting
waRateLimitedTotal.WithLabelValues().Inc()           // whatsmeow rate limit hit

// Message status
waMessagesTotal.WithLabelValues("whatsmeow", "out", "sent").Inc()
waMessagesTotal.WithLabelValues("whatsmeow", "out", "failed").Inc()
```

**Dashboard queries:**
- Cloud API usage: `rate(wa_routed_total{provider="cloud_api"}[5m])`
- Fallback rate: `rate(wa_fallback_total[5m])`
- Rate limit hits: `rate(wa_rate_limited_total[5m])`

---

## Configuration Summary

### Database Tables

**`tenant_chatbot_configs`:**
```sql
wa_provider_preference VARCHAR(20) DEFAULT 'auto' CHECK (wa_provider_preference IN ('auto', 'whatsmeow', 'cloud_api'))
```

**`tenants`:**
```sql
auth_wa_provider_preference VARCHAR(20) DEFAULT 'auto' CHECK (auth_wa_provider_preference IN ('auto', 'whatsmeow', 'cloud_api'))
```

**`wa_cloud_api_credentials`:**
```sql
tenant_id UUID PRIMARY KEY REFERENCES tenants(id)
access_token TEXT NOT NULL
phone_number_id VARCHAR(50) NOT NULL
is_active BOOLEAN DEFAULT true
```

### HTTP Headers

| Header | Values | Purpose |
|:-------|:-------|:--------|
| `X-WA-Provider-Override` | `auto`, `whatsmeow`, `cloud_api` | Override DB preference (priority 1) |
| `X-Message-Type` | `otp`, `invoice`, `payment`, `subscription`, `system`, `broadcast` | Mark as transactional |
| `X-Source` | `auth-service`, `billing-service`, `notification-service` | Service origin indicates transactional |
| `X-Tenant-ID` | UUID | Multi-tenant isolation |

---

## Error Handling

### Common Error Scenarios

**1. Cloud API Forced But Failed (502)**
```json
{
  "error": "Cloud API forced but failed"
}
```
**Cause:** Tenant set `wa_provider_preference = "cloud_api"` but:
- wa-cloud-api service is down
- Tenant credentials expired/invalid
- Meta API rate limit exceeded

**Fix:** Check `wa_cloud_api_credentials` table, verify Meta Business Manager account.

---

**2. Rate Limit Exceeded (429)**
```json
{
  "error": "Rate limit exceeded (max 5 msg/min). Use Cloud API for broadcasting."
}
```
**Cause:** Tenant sent >5 messages in 1 minute via whatsmeow.

**Fix:** Switch to Cloud API for high-volume messaging or wait 1 minute.

---

**3. Not Connected to WhatsApp (401)**
```json
{
  "error": "Not connected to WhatsApp. Please scan QR first."
}
```
**Cause:** whatsmeow session expired, tenant hasn't scanned QR code.

**Fix:** Tenant must visit `/settings` in umkm-web and scan QR code to establish session.

---

**4. Reconnect Backoff Active (503)**
```json
{
  "error": "WhatsApp disconnected. Reconnect backoff active."
}
```
**Cause:** whatsmeow session disconnected and currently in exponential backoff window.

**Fix:** Wait for backoff window to expire (30s-10m depending on attempt count).

---

## Testing & Validation

### Manual Testing Commands

```bash
# Test conversational message (whatsmeow)
curl -X POST http://localhost:8202/api/wa/send \
  -d "tenant_id=abc123" \
  -d "target=628123456789@s.whatsapp.net" \
  -d "message=Test conversational message"

# Test transactional message (Cloud API with fallback)
curl -X POST http://localhost:8202/api/wa/send \
  -H "X-Message-Type: otp" \
  -d "tenant_id=abc123" \
  -d "target=628123456789@s.whatsapp.net" \
  -d "message=Your OTP: 123456"

# Test forced Cloud API (no fallback)
curl -X POST http://localhost:8202/api/wa/send \
  -H "X-WA-Provider-Override: cloud_api" \
  -d "tenant_id=abc123" \
  -d "target=628123456789@s.whatsapp.net" \
  -d "message=Test forced Cloud API"

# Test forced whatsmeow (skip Cloud)
curl -X POST http://localhost:8202/api/wa/send \
  -H "X-WA-Provider-Override: whatsmeow" \
  -d "tenant_id=abc123" \
  -d "target=628123456789@s.whatsapp.net" \
  -d "message=Test forced whatsmeow"
```

### Unit Tests

**File:** `services/wa-gateway/wa_gateway_test.go`

```go
func TestResolveProviderPreference_HeaderOverride(t *testing.T)
func TestResolveProviderPreference_DBLookup(t *testing.T)
func TestIsTransactional_MessageType(t *testing.T)
func TestIsTransactional_SourceService(t *testing.T)
```

---

## Architecture Decision Records

### ADR-001: Why Hybrid Architecture?

**Decision:** Use both whatsmeow + Cloud API instead of single provider.

**Context:**
- Cloud API: Expensive (per-message cost), no ban risk, requires Meta Business Manager
- whatsmeow: Free, ban risk if spamming, no official support

**Rationale:**
- **Cost:** Conversational chatbot (high volume) → whatsmeow saves ~70% cost vs Cloud API
- **Compliance:** Transactional messages (OTP, invoices) → Cloud API reduces ban risk
- **Flexibility:** Tenants can choose preference per use case

**Consequences:**
- More complex routing logic
- Fallback handling required
- Two sets of credentials to manage

---

### ADR-002: Why `auto` is Default?

**Decision:** Default `wa_provider_preference = 'auto'` for new tenants.

**Rationale:**
- Most tenants don't understand the difference
- `auto` provides best cost/reliability balance
- Transactional messages (high risk) → Cloud API automatically
- Conversational messages (low risk) → whatsmeow saves cost

**Consequences:**
- Need clear UI explanation of routing behavior
- Metrics to track fallback rate (Cloud API failures)

---

## Future Enhancements

1. **Dynamic Fallback Threshold**
   - Track Cloud API failure rate per tenant
   - Auto-switch to whatsmeow-only if Cloud API fails >50% for 1 hour

2. **Provider Health Checks**
   - Periodic ping to Cloud API and whatsmeow
   - Pre-emptive routing based on provider health

3. **Cost Optimization Dashboard**
   - Show tenant: "You could save Rp X/month by switching to whatsmeow"
   - Recommend preference based on message volume and type distribution

4. **A/B Testing Framework**
   - Split traffic 50/50 between providers for same message type
   - Measure delivery rate, latency, ban rate

---

## References

- **Hybrid Architecture Overview:** `CLAUDE.md` → "📱 Hybrid WhatsApp Architecture"
- **WA Provider Operations Guide:** `docs/WA_PROVIDER_GUIDE.md`
- **Feature Spec:** `docs/specs/F048_WA_Provider_Preferences.md`
- **Cloud API Service:** `services/wa-cloud-api/main.go`
- **Routing Logic:** `services/wa-gateway/cloud_routing.go`, `services/wa-gateway/send_handlers.go`
- **Auth OTP Integration:** `services/auth-service/main.go:356, 1371`
