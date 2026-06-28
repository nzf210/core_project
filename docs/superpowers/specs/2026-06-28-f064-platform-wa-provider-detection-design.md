# F064: Platform WA Provider Detection & OTP Routing

**Spec Status:** ✅ Approved  
**Implementation:** 🔨 In Progress  
**Last Updated:** 2026-06-28

---

## 🎯 Objectives

Sistem harus bisa mendeteksi **platform-level WA provider aktif** yang digunakan superadmin — apakah **whatsmeow** (WA Center via QR) atau **Meta Cloud API** — dan menyesuaikan flow OTP/registrasi:

- Jika **whatsmeow** aktif: sistem **TIDAK BOLEH** kirim OTP/notifikasi proaktif ke user. User harus chat duluan ke WA Center dengan keyword `REG`, `OTP`, atau `VERIF`.
- Jika **Cloud API** aktif: sistem kirim OTP seperti biasa (proaktif).
- Superadmin bisa **manual override** pilihan provider kapan saja via dashboard.
- Default: **auto-detect** — sistem mendeteksi provider mana yang connect.

---

## 📝 Spec

### AC-1: Platform WA Provider — Storage

Tambahkan **1 row di Redis** sebagai source of truth untuk platform WA provider:

| Key | Value | TTL |
|:----|:------|:----|
| `platform:wa:provider` | `"auto"` / `"whatsmeow"` / `"cloud_api"` | No expiry (permanent) |

Tidak perlu tabel baru di PostgreSQL. Redis sudah dipakai untuk cache plan, OTP, dll. Auth-service dan wa-gateway sudah punya akses Redis.

**Fallback:** Jika Redis kosong (belum pernah di-set), auto-detect logic jalan.

**Superadmin override** → SET Redis key. **Auto-detect** → SET Redis key + reason.

### AC-2: Auto-Detect Logic

Function `getPlatformWAProvider(ctx) → (provider, reason)` di auth-service:

```
1. Cek Redis "platform:wa:provider"
   - Jika "whatsmeow" → return ("whatsmeow", "manual-override")
   - Jika "cloud_api" → return ("cloud_api", "manual-override")
   
2. Jika "auto" / kosong → auto-detect:
   a. Cek wa_sessions → apakah ada row dengan tenant_id = 'verifier'/'system' dan status = 'connected'?
      - Jika ya → whatsmeow available
   b. Cek wa_cloud_api_credentials → apakah ada row dengan is_active = true?
      - Jika ya + verification_status = 'verified' → cloud_api available
   c. Decision matrix:
      - Hanya whatsmeow available → return ("whatsmeow", "auto-detect:whatsmeow-only")
      - Hanya cloud_api available → return ("cloud_api", "auto-detect:cloud-api-only")
      - Keduanya available → return ("cloud_api", "auto-detect:cloud-api-priority")
      - Tidak ada → return ("whatsmeow", "auto-detect:no-connection-fallback")
```

**Cache:** Hasil auto-detect di-cache di Redis 5 menit dengan key `platform:wa:provider` (jika sebelumnya kosong/auto). Setiap ada perubahan koneksi WA (connect/disconnect), cache di-invalidate.

**Prioritas Cloud API:** Jika keduanya terhubung, Cloud API yang dipilih — karena lebih reliable untuk kirim OTP proaktif.

### AC-3: OTP Flow — Register (handleRegister)

File: `services/auth-service/auth_handlers.go`

**Perubahan di handleRegister:**

```go
waProvider := getPlatformWAProvider(ctx)

if waProvider == "whatsmeow" {
    // Simpan OTP di Redis (untuk VERIF flow via WA)
    // Tapi JANGAN kirim via sendWAGatewayOTP
    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "Untuk daftar, silakan kirim REG ke nomor WA Center.",
        Data:    map[string]any{"wa_center_required": true},
    })
    return
}

// cloud_api → kirim OTP seperti biasa
go sendWAGatewayOTP(...)
```

**Detail:**
- OTP tetap disimpan di Redis (`otp:{phone}`), tapi skip `sendWAGatewayOTP`.
- Response message berbeda: "Kirim REG ke WA Center" vs "OTP telah dikirim ke WA".
- wa_verify flow (`?wa_verify=true`) tetap bisa dipakai untuk web registration — user dapat `otp_code` di response web dan kirim `VERIF {code}` ke WA Center.

### AC-4: OTP Flow — Login (handlePhoneLogin)

File: `services/auth-service/phone_handlers.go`

**Perubahan di handlePhoneLogin:**

```go
waProvider := getPlatformWAProvider(ctx)

if waProvider == "whatsmeow" {
    // Simpan OTP di Redis (untuk OTP login via WA reply)
    // Tapi JANGAN kirim
    writeJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "Untuk login, silakan kirim OTP ke nomor WA Center.",
        Data:    map[string]any{"wa_center_required": true},
    })
    return
}

// cloud_api → kirim OTP seperti biasa
go sendLoginOTP(...)
```

**Detail:**
- OTP tetap disimpan di Redis (`phone-login-otp:{phone}`), tapi skip `sendLoginOTP`.
- Response: "Kirim OTP ke nomor WA Center" — user tahu harus chat ke WA Center.

### AC-5: OTP Flow — Existing WhatsApp Flow masih jalan

Yang **tidak berubah:**
- `event_handler.go` di wa-gateway — keyword REG, OTP, VERIF tetap bekerja seperti biasa (F063).
- `handleRegisterWA` (register via WA) — tetap bisa dipanggil dari wa-gateway.
- `handleVerifyOTPWA`, `handleVerifyPhoneLoginWA` — tetap bisa diverifikasi via WA reply.
- `wa_verify=true` flow — web registration dengan VERIF code tetap jalan.

Yang berubah HANYA **apakah auth-service mengirim OTP proaktif** atau TIDAK.

### AC-6: Superadmin API Endpoints

Di **auth-service** (sudah punya route `/superadmin/...`):

| Method | Path | Deskripsi |
|:-------|:-----|:----------|
| `GET` | `/superadmin/wa/platform-provider` | Return provider aktif + status koneksi (whatsmeow + cloud_api) |
| `PUT` | `/superadmin/wa/platform-provider` | Set override: `{ "wa_provider": "auto" / "whatsmeow" / "cloud_api" }` |

**GET Response:**
```json
{
  "success": true,
  "data": {
    "wa_provider": "auto",
    "effective_provider": "whatsmeow",
    "reason": "auto-detect:whatsmeow-only",
    "connections": {
      "whatsmeow": { "status": "connected", "jid": "62xxx@whatsapp.net" },
      "cloud_api": { "status": "disconnected", "is_active": false }
    },
    "updated_at": "2026-06-28T10:00:00Z"
  }
}
```

**PUT Request:**
```json
{ "wa_provider": "cloud_api" }
```

**PUT Response:**
```json
{
  "success": true,
  "message": "Platform WA provider updated to cloud_api"
}
```

**Routing via api-gateway:** `/api/superadmin/wa/platform-provider` → auth-service (same pattern as existing `/api/superadmin/verifier/...`).

### AC-7: Superadmin Dashboard UI

**Tambah di WACenter.vue** (atau card terpisah di SuperAdminDashboard):

```
📡 Provider WhatsApp Platform
─────────────────────────────────
Provider Aktif: [whatsmeow ▼]
Status: ✅ whatsmeow (62xxx-xxxx@whatsapp.net)
        ⚡ Cloud API (tidak aktif)

Mode saat ini: Deteksi Otomatis (whatsmeow)
               [Ganti Provider →]

Ketika provider = whatsmeow:
  • User harus chat REG/OTP/VERIF ke WA Center
  • Sistem TIDAK kirim notifikasi proaktif
```

Dropdown: Auto Detect / Paksa whatsmeow / Paksa Cloud API

Setiap perubahan -> PUT `/api/superadmin/wa/platform-provider` -> toast success.

### AC-8: Cache Invalidation

Setiap kali ada perubahan koneksi WA:
- whatsmeow connect/disconnect (via `handleStatusRequest`, `handleQR`, `handleLogout`)
- Cloud API credential toggle (via `PUT /chatbot/config` -> wa_cloud_api update)

→ Invalidate Redis `platform:wa:provider` key agar auto-detect ulang.

### AC-9: Web Registration tetap Jalan

**Register via Web** (POST `/register` tanpa `?wa_verify=true`):
- Jika provider = whatsmeow: response `"Kirim REG ke WA Center"`. User gak bisa daftar via web? 
  - **Keputusan:** web registration tetap bisa, tapi user harus kirim `VERIF {otp_code}` ke WA Center.
  - Auth-service generate OTP, return `otp_code` di response data → user kirim VERIF code via WA.
  - Tapi... kalau web registration dari PC, user harus buka WA dan chat. It's fine — itu yang diharapkan.

Atau **lebih baik**: jika provider = whatsmeow, web registration langsung `wa_verify=true` mode secara otomatis:
- Response otp_code → user kirim VERIF ke WA Center
- Tanpa perlu query param `?wa_verify=true`

## 🛠️ Implementation Plan

### Files Changed:

| File | Change |
|:-----|:-------|
| `services/auth-service/auth_handlers.go` | Add `getPlatformWAProvider()`, modify `handleRegister` — skip send if whatsmeow |
| `services/auth-service/phone_handlers.go` | Modify `handlePhoneLogin` — skip send if whatsmeow |
| `services/auth-service/main.go` | Register 2 new routes: GET/PUT `/superadmin/wa/platform-provider` |
| `services/auth-service/wa_platform.go` | NEW — `getPlatformWAProvider()`, `handleGetPlatformProvider()`, `handleSetPlatformProvider()` |
| `services/api-gateway/main.go` | Proxy route `/api/superadmin/wa/platform-provider` → auth-service |
| `services/wa-gateway/routes.go` | Invalidate Redis `platform:wa:provider` on connect/disconnect |
| `frontend/superadmin-web/src/components/WACenter.vue` | Add provider selector UI |
| `frontend/superadmin-web/src/api/client.ts` | Add `getPlatformProvider()`, `setPlatformProvider()` |

### Migration:
- **Tidak perlu migration database baru** — storage via Redis. Query `wa_sessions` dan `wa_cloud_api_credentials` sudah existing.

### Redis Keys:
| Key | Value | TTL |
|:----|:------|:-----|
| `platform:wa:provider` | `auto|whatsmeow|cloud_api` | None |
| `platform:wa:reason` | `"auto-detect:whatsmeow-only"` dll | None |

### Pseudo-code getPlatformWAProvider:

```go
func getPlatformWAProvider(ctx context.Context) (string, string) {
    // 1. Cek Redis override
    provider, err := Redis.Get(ctx, "platform:wa:provider").Result()
    if err == nil && provider != "auto" && provider != "" {
        return provider, "manual-override"
    }

    // 2. Auto-detect
    var whatsmeowConnected bool
    var cloudAPIActive bool

    DB.QueryRow(ctx, `SELECT EXISTS(
        SELECT 1 FROM wa_sessions WHERE tenant_id IN ('verifier','system') AND status = 'connected'
    )`).Scan(&whatsmeowConnected)

    DB.QueryRow(ctx, `SELECT EXISTS(
        SELECT 1 FROM wa_cloud_api_credentials WHERE is_active = true
        AND (verification_status = 'verified' OR verification_status IS NULL)
    )`).Scan(&cloudAPIActive)

    // Decision matrix
    if cloudAPIActive && whatsmeowConnected {
        // Cache auto-detect result
        Redis.Set(ctx, "platform:wa:provider", "cloud_api", 0)
        return "cloud_api", "auto-detect:both-connected-cloud-priority"
    }
    if cloudAPIActive {
        Redis.Set(ctx, "platform:wa:provider", "cloud_api", 0)
        return "cloud_api", "auto-detect:cloud-api-only"
    }
    if whatsmeowConnected {
        Redis.Set(ctx, "platform:wa:provider", "whatsmeow", 0)
        return "whatsmeow", "auto-detect:whatsmeow-only"
    }

    // Fallback
    return "whatsmeow", "auto-detect:no-connection-fallback"
}
```

### Flow Diagram:

```
Register/Login Request
       │
       v
getPlatformWAProvider()
       │
       ├─ Redis: "cloud_api" (manual)
       │     └─ Kirim OTP via Cloud API (existing flow)
       │
       ├─ Redis: "whatsmeow" (manual)
       │     └─ Skip send. Response: "Kirim REG/OTP ke WA Center"
       │
       └─ Redis: "auto" / empty → auto-detect
             │
             ├─ whatsmeow connected, cloud_api connected/verified
             │     └─ Cloud API → Kirim OTP (prioritas)
             │
             ├─ cloud_api connected/verified only
             │     └─ Kirim OTP via Cloud API
             │
             ├─ whatsmeow connected only
             │     └─ Skip send. Response: "Kirim REG/OTP ke WA Center"
             │
             └─ No connection
                   └─ Skip send (fallback whatsmeow behavior)
```

---

## ✅ Acceptance Criteria

- [ ] AC-1: Redis key `platform:wa:provider` menyimpan provider aktif (auto/whatsmeow/cloud_api)
- [ ] AC-2: Auto-detect logic membaca wa_sessions + wa_cloud_api_credentials, decision matrix benar
- [ ] AC-3: handleRegister → jika whatsmeow → skip OTP kirim, response "Kirim REG ke WA Center"
- [ ] AC-4: handlePhoneLogin → jika whatsmeow → skip OTP kirim, response "Kirim OTP ke WA Center"
- [ ] AC-5: Jika Cloud API → OTP tetap terkirim seperti biasa (existing flow tidak rusak)
- [ ] AC-6: GET/PUT `/superadmin/wa/platform-provider` berfungsi — return status koneksi + set override
- [ ] AC-7: Superadmin dashboard — dropdown provider selector di WACenter.vue
- [ ] AC-8: Cache invalidation saat WA connect/disconnect
- [ ] AC-9: Web registration tetap bisa via VERIF code walau provider = whatsmeow
- [ ] AC-10: `go build ./...`, `go vet ./...` clean
