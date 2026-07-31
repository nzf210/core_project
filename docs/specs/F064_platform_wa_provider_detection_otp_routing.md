# F064: Platform WA Provider Detection & OTP Routing

**Date:** 2026-06-28  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F016](../CLAUDE.md#-hybrid-whatsapp-architecture), [F048](../FEATURE_MAP.md)

---

## 🎯 Objectives

Sistem dapat mendeteksi platform-level WA provider aktif dan menyesuaikan flow OTP registration/login secara otomatis.

**Tujuan eksplisit:**
1. Auto-detect provider aktif (whatsmeow vs Meta Cloud API) — prioritas Cloud API jika keduanya tersedia
2. Skip proactive OTP send untuk whatsmeow (user harus chat REG/OTP/VERIF ke WA Center)
3. Superadmin dapat manual override provider via dashboard untuk testing/troubleshooting

**Problem yang diselesaikan:**
- Platform tidak tahu provider mana yang aktif → kirim OTP via provider yang salah/tidak tersedia
- User experience berbeda antara whatsmeow (chat-based) vs Cloud API (proactive send) tapi sistem tidak menyesuaikan
- Perlu manual override untuk testing atau saat Cloud API down

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Redis Storage**
  - *Verification:* Key `platform:wa:provider` → `"auto"` / `"whatsmeow"` / `"cloud_api"`, no TTL
  - *Example:* `redis-cli GET platform:wa:provider` → `"auto"` (default)

- [x] **AC-2: Auto-Detect Logic**
  - *Verification:* `getPlatformWAProvider()` di `services/auth-service/wa_platform.go` query `wa_sessions` + `wa_cloud_api_credentials`, return `(provider, reason)`
  - *Example:* Cloud API active → return `("cloud_api", "Meta Cloud API active (1 credentials verified)")`

- [x] **AC-3: Register — Skip OTP if Whatsmeow**
  - *Verification:* `POST /api/auth/register` dengan provider whatsmeow → response `wa_center_required: true`, `otp_code` tetap di-return untuk VERIF flow
  - *Example:* Response message: *"Untuk daftar, silakan kirim REG ke nomor WA Center. Atau kirim VERIF {code} jika daftar dari web."*

- [x] **AC-4: Login — Skip OTP if Whatsmeow**
  - *Verification:* `POST /api/auth/phone-login` dengan provider whatsmeow → skip `sendLoginOTP()`, return message
  - *Example:* *"Untuk login, silakan kirim OTP ke nomor WA Center..."*

- [x] **AC-5: Cloud API OTP Tidak Terganggu**
  - *Verification:* Provider `"cloud_api"` atau `"auto"` (resolved ke cloud_api) → OTP kirim via `sendWAGatewayOTP` seperti biasa
  - *Example:* `POST /api/auth/register` → OTP terkirim via Cloud API, user terima WA dalam 3-5 detik

- [x] **AC-6: Superadmin API Endpoints**
  - *Verification:* `GET /api/superadmin/wa/platform-provider` → return `wa_provider`, `effective_provider`, `reason`, `connections`
  - *Example:* `PUT /api/superadmin/wa/platform-provider { "wa_provider": "whatsmeow" }` → Redis key set

- [x] **AC-7: Frontend Provider Selector UI**
  - *Verification:* `WACenter.vue` — dropdown: Auto Detect / Paksa Whatsmeow / Paksa Cloud API, tampilkan effective provider + reason + connection status
  - *Example:* Dropdown pilih "Paksa Cloud API" → effective_provider berubah → hint message update

- [x] **AC-8: Cache Invalidation on Connect/Disconnect**
  - *Verification:* `invalidatePlatformWAProviderCache()` dipanggil saat whatsmeow connect/disconnect/QR timeout
  - *Example:* Whatsmeow connect → cache invalidate → auto-detect re-run → effective_provider update

- [x] **AC-9: Web Registration via VERIF Code**
  - *Verification:* Whatsmeow mode → `otp_code` tetap di-return, user bisa kirim `VERIF {code}` ke WA Center
  - *Example:* User daftar via web → dapat OTP code → kirim `VERIF 123456` ke WA Center → registration complete

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│           Auth Service (Registration/Login)         │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │ getPlatformWAProvider()                      │  │
│  │  1. Check Redis: platform:wa:provider       │  │
│  │  2. If "auto" or empty:                     │  │
│  │     - Query wa_sessions (verifier/system)   │  │
│  │     - Query wa_cloud_api_credentials        │  │
│  │     - Decision: cloud_api > whatsmeow       │  │
│  │  3. Return (provider, reason)               │  │
│  └──────────────────────────────────────────────┘  │
│              ↓                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │ handleRegister / handlePhoneLogin            │  │
│  │  - If whatsmeow: skip sendOTP, return hint  │  │
│  │  - If cloud_api: sendOTP via wa-gateway     │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
         ↓ (for whatsmeow mode)
┌─────────────────────────────────────────────────────┐
│         WA Center (user-initiated commands)         │
│  REG → create tenant + user                         │
│  OTP → send OTP code                                │
│  VERIF {code} → verify OTP + activate account       │
└─────────────────────────────────────────────────────┘
```

### Database Schema

**No migration needed** — storage via Redis. Query existing tables:

```sql
-- Auto-detect whatsmeow
SELECT 1 FROM wa_sessions 
WHERE tenant_id IN ('verifier', 'system') 
  AND status = 'connected';

-- Auto-detect Cloud API
SELECT 1 FROM wa_cloud_api_credentials 
WHERE is_active = true 
  AND verification_status = 'verified';
```

### API Endpoints

#### `GET /api/superadmin/wa/platform-provider`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "wa_provider": "auto",
    "effective_provider": "cloud_api",
    "reason": "Meta Cloud API active (2 credentials verified)",
    "connections": {
      "whatsmeow": false,
      "cloud_api": true
    }
  }
}
```

#### `PUT /api/superadmin/wa/platform-provider`

**Request:**
```json
{
  "wa_provider": "whatsmeow"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Platform WA provider updated to whatsmeow"
}
```

**Error Cases:**
- `400 Bad Request` — Invalid `wa_provider` value (not in `auto|whatsmeow|cloud_api`)
- `401 Unauthorized` — Not superadmin role

#### `POST /api/auth/register` (modified behavior)

**Whatsmeow mode response:**
```json
{
  "success": true,
  "data": {
    "wa_center_required": true,
    "otp_code": "123456",
    "message": "Untuk daftar, silakan kirim REG ke nomor WA Center. Atau kirim VERIF 123456 jika daftar dari web."
  }
}
```

**Cloud API mode response** (unchanged):
```json
{
  "success": true,
  "data": {
    "message": "OTP sent to WhatsApp"
  }
}
```

### Redis Keys

| Key | Value | TTL | Usage |
|:----|:------|:----|:------|
| `platform:wa:provider` | `auto` / `whatsmeow` / `cloud_api` | None | Manual override setting. If empty/auto → run auto-detect |

---

## 🧪 Testing Strategy

### Unit Tests
```bash
# Auto-detect logic (mock DB queries)
go test ./services/auth-service/ -run TestGetPlatformWAProvider -v

# Cache invalidation
go test ./services/wa-gateway/ -run TestInvalidatePlatformWAProviderCache -v
```

### Integration Tests
```bash
# 1. Auto-detect scenarios
# - Cloud API only active
# - Whatsmeow only active
# - Both active (Cloud API priority)
# - None active (whatsmeow fallback)

# 2. Register/Login flow
# - Whatsmeow mode → wa_center_required = true
# - Cloud API mode → OTP sent

# 3. Manual override
curl -X PUT http://localhost:8001/superadmin/wa/platform-provider \
  -H "Content-Type: application/json" \
  -d '{"wa_provider":"whatsmeow"}'
```

### Manual Testing
1. Set provider via Superadmin → WACenter → dropdown
2. Verify effective_provider updates
3. Register new user → check OTP send behavior
4. Connect/disconnect whatsmeow → verify cache invalidation

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Platform WA provider resolved", 
  "provider", provider, 
  "reason", reason, 
  "source", "auto-detect")
```

**Metrics to track:**
- Redis cache hit/miss for `platform:wa:provider`
- Auto-detect execution time
- Provider distribution (whatsmeow vs cloud_api usage %)

**Alerts:**
- Both providers down → superadmin notification

---

## 🚀 Rollout Plan

### Phase 1: Backend + Superadmin UI (Done ✅)
- Deploy `auth-service` + `wa-gateway` dengan auto-detect logic
- Deploy `superadmin-web` dengan provider selector UI
- Default: `auto` (Cloud API priority)

### Phase 2: Monitoring (Current)
- Add Grafana dashboard untuk provider distribution
- Add alert rule untuk "both providers down"

### Rollback
- Redis key delete: `redis-cli DEL platform:wa:provider` → revert ke auto-detect default
- Code rollback: revert `wa_platform.go` → system kirim OTP via wa-gateway tanpa cek provider

---

## 🔮 Future Enhancements (Out of Scope)

- **Per-tenant provider preference:** Tenant bisa pilih sendiri whatsmeow vs Cloud API (saat ini platform-level)
- **A/B test OTP delivery:** Random assign provider untuk compare success rate
- **Auto-failover:** Jika Cloud API down → auto-switch ke whatsmeow dengan graceful UX

---

## 📚 References

- [F048: WA Provider Preference (Tenant-Level)](../FEATURE_MAP.md) — tenant-level routing logic
- [F016: Hybrid WhatsApp Architecture](../CLAUDE.md#-hybrid-whatsapp-architecture) — original whatsmeow + Cloud API design
- [WhatsApp Cloud API Docs](https://developers.facebook.com/docs/whatsapp/cloud-api/)

---

## 📝 Notes & Decisions

**2026-06-28:** Implemented platform-level detection. Decision: Cloud API priority over whatsmeow karena compliance + reliability (Meta official API).  
**2026-06-28:** `otp_code` tetap di-return untuk whatsmeow mode — support hybrid flow (web registration via VERIF command).  
**2026-06-28:** Cache invalidation di wa-gateway event handlers (connect/disconnect/QR timeout) — ensure auto-detect selalu fresh setelah provider state change.
