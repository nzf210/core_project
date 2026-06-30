# F064: Platform WA Provider Detection & OTP Routing


## F064: Platform WA Provider Detection & OTP Routing

**Spec Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Last Updated:** 2026-06-28

### 🎯 Objectives

Sistem mendeteksi **platform-level WA provider aktif** (whatsmeow vs Meta Cloud API) dan menyesuaikan flow OTP:
- **whatsmeow** → user HARUS chat REG/OTP/VERIF ke WA Center, sistem TIDAK kirim OTP proaktif
- **Cloud API** → sistem kirim OTP secara proaktif, user tidak perlu chat duluan
- Superadmin bisa **manual override** provider via dashboard
- Default: **auto-detect** (Cloud API prioritas jika keduanya aktif)

### 📝 Spec

#### AC-1: Redis Storage
- [x] Key `platform:wa:provider` → `"auto"` / `"whatsmeow"` / `"cloud_api"`
- [x] No TTL (permanent), manual override atau auto-detect SET key

#### AC-2: Auto-Detect Logic
- [x] `getPlatformWAProvider()` di `services/auth-service/wa_platform.go`
- [x] Cek `wa_sessions` (tenant_id IN ('verifier','system'), status='connected') → whatsmeow
- [x] Cek `wa_cloud_api_credentials` (is_active=true, verification_status='verified') → cloud_api
- [x] Decision matrix: cloud_api only → cloud_api | whatsmeow only → whatsmeow | both → cloud_api priority | none → whatsmeow fallback
- [x] Return `(provider, reason)` string

#### AC-3: Register — Skip OTP if Whatsmeow
- [x] `handleRegister` di `auth-service/auth_handlers.go` call `getPlatformWAProvider()`
- [x] Jika `"whatsmeow"` → skip `sendWAGatewayOTP()`, response `wa_center_required: true`
- [x] Message: "Untuk daftar, silakan kirim REG ke nomor WA Center. Atau kirim VERIF {code} jika daftar dari web."

#### AC-4: Login — Skip OTP if Whatsmeow
- [x] `handlePhoneLogin` di `auth-service/phone_handlers.go` call `getPlatformWAProvider()`
- [x] Jika `"whatsmeow"` → skip `sendLoginOTP()`, response message: "Untuk login, silakan kirim OTP ke nomor WA Center..."

#### AC-5: Cloud API OTP Tidak Terganggu
- [x] Jika provider `"cloud_api"` atau `"auto"` (resolved ke cloud_api) → OTP kirim via `sendWAGatewayOTP` / `sendLoginOTP` seperti biasa

#### AC-6: Superadmin API Endpoints
- [x] `GET /api/superadmin/wa/platform-provider` → return `wa_provider`, `effective_provider`, `reason`, `connections`
- [x] `PUT /api/superadmin/wa/platform-provider` → set Redis key (auto → DELETE key, else → SET)
- [x] Routing: `api-gateway/main.go` line 74 → auth-service `/superadmin/wa/platform-provider`

#### AC-7: Frontend Provider Selector UI
- [x] `WACenter.vue` — dropdown: Auto Detect / Paksa Whatsmeow / Paksa Cloud API
- [x] Tampilkan effective provider, reason, connection status (whatsmeow ✅/❌, cloud_api ✅/❌)
- [x] Hint message: "User harus chat REG/OTP/VERIF..." (whatsmeow) atau "Sistem kirim OTP otomatis..." (cloud_api)
- [x] `client.ts` — `getPlatformProvider()`, `setPlatformProvider()`

#### AC-8: Cache Invalidation on Connect/Disconnect
- [x] `invalidatePlatformWAProviderCache()` di `wa-gateway/event_handler.go`
- [x] Dipanggil di: `handleConnectedEvent` (connect), `handleLogoutRequest` (disconnect), `scheduleQRExpiry` (QR timeout → no connect)

#### AC-9: Web Registration via VERIF Code
- [x] `otp_code` tetap di-return dalam response untuk register/login whatsmeow mode
- [x] User bisa kirim `VERIF {code}` ke WA Center untuk complete registration

### 🛠️ Implementation Details

**Backend Files:**
- `services/auth-service/wa_platform.go` (NEW) — `getPlatformWAProvider()`, `handleGetPlatformProvider()`, `handleSetPlatformProvider()`
- `services/auth-service/auth_handlers.go` (MODIFIED) — `handleRegister` cek provider, skip send
- `services/auth-service/phone_handlers.go` (MODIFIED) — `handlePhoneLogin` cek provider, skip send
- `services/auth-service/main.go` (MODIFIED) — route `/superadmin/wa/platform-provider`
- `services/api-gateway/main.go` (MODIFIED) — proxy `/api/superadmin/wa/platform-provider` → auth-service
- `services/wa-gateway/event_handler.go` (MODIFIED) — `invalidatePlatformWAProviderCache()` + call on connect
- `services/wa-gateway/logout_handlers.go` (MODIFIED) — call `invalidatePlatformWAProviderCache()` on disconnect
- `services/wa-gateway/qr_handlers.go` (MODIFIED) — call `invalidatePlatformWAProviderCache()` on QR expiry

**Frontend Files:**
- `frontend/superadmin-web/src/components/WACenter.vue` (MODIFIED) — provider selector UI
- `frontend/superadmin-web/src/api/client.ts` (MODIFIED) — `getPlatformProvider()`, `setPlatformProvider()`

**Redis Keys:**
| Key | Value | TTL |
|:----|:------|:----|
| `platform:wa:provider` | `auto\|whatsmeow\|cloud_api` | None |

**No migration needed** — storage via Redis. Existing `wa_sessions` dan `wa_cloud_api_credentials` tables used for detection.

### ✅ Testing

```bash
# 1. go vet + build
go vet ./services/wa-gateway/... ./services/auth-service/... && go build ./services/wa-gateway/... ./services/auth-service/...

# 2. Auto-detect (no Redis override)
# Pastikan wa_sessions (verifier/system) connected → effective_provider = cloud_api (priority)
# Pastikan wa_sessions disconnected + cloud_api active → effective_provider = cloud_api
# Pastikan wa_sessions connected + cloud_api inactive → effective_provider = whatsmeow

# 3. Manual override
# curl -X PUT http://localhost:8001/superadmin/wa/platform-provider \
#   -H "Content-Type: application/json" -d '{"wa_provider":"whatsmeow"}'
# → Register/login harus return "wa_center_required: true"

# 4. API endpoint
# curl http://localhost:8001/superadmin/wa/platform-provider
# → return wa_provider, effective_provider, reason, connections
```
