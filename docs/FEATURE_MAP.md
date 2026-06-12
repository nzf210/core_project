# 🗺️ WCH Platform — Feature Map & Specification

> **Dokumen utama untuk AI governance.** Setiap fitur baru/wubah WAJIB ada SPEC di sini.
> User approve SPEC duluan, baru AI implement.

---

## 🔄 Spec-First Workflow

```
USER menulis SPEC      →       AI review & clarify      →       USER approve
     ↓                         ↓                                  ↓
 FEATURE_MAP.md         AI tanya clarifications           USER comment/approve
                              ↓                                  ↓
                      AI wait for approval          AI implement dari SPEC
                                                            ↓
                                                    USER review diff
```

### Aturan untuk AI:
1. Baca FEATURE_MAP.md sebelum coding
2. Kalau ada feature baru/wubah, tanya USER dulu:
   - "Ada SPEC untuk fitur ini?" → kalau belum, buat draft SPEC
   - "SPEC ini sudah diapprove?" → kalau belum, jangan implement
3. Kalau ada ambiguitas di SPEC, tanya clarification
4. Setelah implement, update kolom `Implementation` di tabel

---

## 📋 Feature Specifications

Format per feature:
```markdown
### FXXX: [Nama Feature]

**Spec Status:** ⏳ Draft | 🔍 In Review | ✅ Approved | ❌ Rejected
**Implementation:** ⏳ Pending | 🔨 In Progress | ✅ Done | ❌ Cancelled

**Deskripsi:** Apa yang fitur ini lakukan

**Spec:**
- Bullet point spesifikasi detail
- Include business rules
- Include validasi yang perlu

**Acceptance Criteria (AC):**
- [ ] AC-1: Kriteria yang bisa diverifikasi
- [ ] AC-2: User bisa test apakah fitur jalan

**Files yang perlu diubah:**
- `path/to/file.go` — deskripsi perubahan

**Notes:**
- Catatan implementasi jika ada
```

---

## 📊 Feature Registry

| ID | Feature | Spec Status | Implementation | Last Updated |
|:---|:--------|:------------|:---------------|:-------------|
| F001 | Multi-Store Quota | ✅ Approved | ✅ Done | 2026-06-12 |
| F002 | Voucher Link Subscription | ✅ Approved | ✅ Done | 2026-06-12 |
| F003 | Subscription Freeze Worker | ✅ Approved | ✅ Done | 2026-06-12 |
| F004 | Read-only Enforcement (Frozen) | ✅ Approved | ✅ Done | 2026-06-12 |
| F005 | Superadmin Dashboard | ✅ Approved | ✅ Done | 2026-06-12 |
| F006 | Multi-Tenant WA Session Pool | ✅ Approved | ✅ Done | 2026-06-01 |
| F007 | Chatbot with RAG | ✅ Approved | ✅ Done | 2026-06-01 |
| F008 | Escalation to Chatwoot | ✅ Approved | ✅ Done | 2026-06-01 |
| F009 | N8N Queue Mode Automation | ✅ Approved | ✅ Done | 2026-06-01 |
| F010 | Campaign Volunteer Management | ✅ Approved | ✅ Done | 2026-06-12 |
| F011 | Campaign Voter Onboarding | ✅ Approved | ✅ Done | 2026-06-12 |
| F012 | Sidebar Navigation UI | ✅ Approved | ✅ Done | 2026-06-12 |
| F013 | N8N Integration via Super Admin | ❌ Removed | — | — |
| F014 | Flexible LLM Model System | ✅ Approved | ✅ Done | 2026-06-12 |
| F015 | Onboarding Activation Flow | ✅ Approved | ✅ Done | 2026-06-13 |
| F016 | Hybrid WhatsApp (Cloud API + whatsmeow) | ✅ Approved | ✅ Done | 2026-06-13 |
| F017 | OTP 1-Hour Reuse Window | ✅ Approved | ✅ Done | 2026-06-13 |
| F018 | Telegram Auth (Register & Login) | ✅ Approved | ✅ Done | 2026-06-13 |

---

## F018: Telegram Auth (Register & Login via Telegram Bot)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User bisa daftar dan login via Telegram Bot. OTP dikirim melalui Telegram (bukan hanya WhatsApp), memanfaatkan bot Telegram yang sama dengan notification-service. Reuse Redis OTP key yang sudah ada — verify OTP endpoint yang sama tetap berfungsi untuk WA maupun Telegram.

**Spec:**
- User memulai chat dengan bot Telegram WCH → bot reply dengan Chat ID
- Frontend UMKM mengirimkan `telegramChatId` bersama data pendaftaran ke `POST /auth/telegram/register`
- Auth-service generate OTP dan kirim via `sendMessage` Telegram Bot API
- OTP disimpan di Redis dengan key yang sama (`otp:{phone}`) — verifikasi via `POST /auth/verify-otp` tetap berfungsi
- Untuk login: `POST /auth/telegram/login` — verifikasi nomor WA terdaftar, kirim OTP via Telegram, update `telegram_chat_id` di DB
- Webhook bot: `POST /auth/telegram/webhook` — handle command `/start` untuk mengembalikan Chat ID
- Reuse 1-hour OTP reuse window dari F017 — baik WA maupun Telegram

**Acceptance Criteria (AC):**
- [x] AC-1: POST `/auth/telegram/register` dengan `telegramChatId` + data → OTP terkirim ke Telegram
- [x] AC-2: POST `/auth/telegram/login` dengan `telegramChatId` + `phoneNumber` → OTP login terkirim ke Telegram
- [x] AC-3: POST `/auth/verify-otp` tetap berfungsi untuk verifikasi OTP (dari WA maupun Telegram)
- [x] AC-4: POST `/auth/verify-phone-login` tetap berfungsi untuk verifikasi login (dari WA maupun Telegram)
- [x] AC-5: Webhook `/auth/telegram/webhook` handle /start command dan return Chat ID
- [x] AC-6: `telegram_chat_id` tersimpan di tabel users saat registrasi & login via Telegram
- [x] AC-7: OTP 1-hour reuse window (F017) berfungsi untuk Telegram juga

**Files Changed:**
- `services/auth-service/main.go` — Telegram request types, `handleTelegramRegister()`, `handleTelegramLogin()`, `handleTelegramWebhook()`, `sendTelegramOTP()`, updated `handleVerifyOTP()` to parse map-based reg data
- `shared/sdk/config/config.go` — Telegram Bot config struct + loading
- `shared/migrations/000031_telegram_auth.up.sql` — `telegram_chat_id` column + index
- `.env.example` — `TELEGRAM_BOT_TOKEN` documentation

**Notes:**
- Bot token dibaca dari `TELEGRAM_BOT_TOKEN` env — shared dengan notification-service (bisa pakai bot yang sama)
- Webhook URL: `POST https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<domain>/auth/telegram/webhook`
- Chat ID Telegram user berbeda dengan WA number — mapping disimpan di `users.telegram_chat_id`
- Tidak perlu perubahan di API Gateway — `/auth/telegram/*` otomatis di-proxy oleh existing `/auth/` prefix

---

## F015: Onboarding Activation Flow

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User baru yang baru daftar bisa lanjut ke step 1 & 2 onboarding tanpa gate. Aktivasi (beli paket / masukkan voucher) baru diminta via modal dialog setelah step 2 selesai. Sistem auto-generate kode voucher sebagai bukti langganan. Superadmin bisa generate voucher dalam jumlah dengan masa aktif day-duration.

**Spec:**

### Onboarding Page (/onboarding)

- Step 1 (Pilih Jenis Usaha) — **tanpa gate**, user boleh pilih atau skip
- Step 2 (Detail Usaha: nama, alamat, nomor WA) — **tanpa gate**, boleh lanjut tanpa harus aktifkan
- Setelah step 2 selesai (klik "Lanjut"), muncul **Modal Activation**:
  - **Opsi A: Beli Paket** — pilih paket (Lite/Pro/Business) → generate Xendit invoice → status subscription = `pending`
  - **Opsi B: Masukkan Kode Voucher** — input kode → validasi → langsung aktivasi jika valid

### Subscription Status Lifecycle

```
Tenant dibuat (register OTP)     → plan=inactive, is_frozen=true
User sampai modal activation
    ├─ Beli Paket                → status=pending, expires_at=now+pending_timeout
    │                              Xendit callback CONFIRMED → activateSubscription()
    │                              Pending > 24 jam tiddk dibayar → hapus tenant + user
    └─ Masukkan Voucher          → validate → activateSubscription() + generate system voucher
```

### Auto-Generate System Voucher (setelah aktivasi via Xendit)

- Saat payment confirmed via Xendit webhook → sistem generate `voucher_codes` entry untuk tenant tersebut
- Format kode: `WCH-{short_tenant_id}-{timestamp}` (contoh: `WCH-a1b2-1750000000`)
- Jenis: `system_generated`, `is_used=true`, `plan_id` sesuai paket yang dibeli
- Kode ini dikirim via WhatsApp notification ke user sebagai "bukti langganan"

### Day-Duration Voucher System (bukan tanggal fixed)

- Kolom `validity_days` (INT) — jumlah hari aktif (bukan `valid_until` date)
- Kolom `remaining_days` — hari tersisa, dihitung saat dibaca
- Saat aktivasi voucher baru:
  - Jika tenant sudah punya voucher aktif dengan **plan yang sama** → akumulasi: `remaining_days += new_validity_days`
  - Jika plan **berbeda** → buat voucher baru secara terpisah (bukan overwrite)
- Priority plan: **Pro > Business > Lite** — sistem baca voucher dengan plan tertinggi sebagai plan aktif

### Auto-Delete Pending Tenant

- Worker `subscription-worker` atau cron di `billing-service` cek tenant dengan `status=pending` dan `created_at < now - 24 jam`
- Hapus row `tenants` + `users` terkait dari DB (CASCADE)
- Log penghapusan ke `subscription_tickets` dengan status `expired`

### Superadmin Voucher Management

- `POST /admin/vouchers/generate` — generate N voucher codes sekaligus
  - Body: `{ plan_id, validity_days, quantity, program_name }`
  - Generate N kode acak, simpan ke `voucher_codes`
- `GET /admin/vouchers` — list semua voucher (filter: used/unused, plan_id, program)
- `GET /admin/tenants/{id}/vouchers` — list voucher aktif per tenant (untuk melihat masa aktif)

### WhatsApp Notification (Activation)

- Pesan template saat aktivasi:
  ```
  🎉 Langganan WCH Platform berhasil diaktifkan!

  📋 Paket: {plan_name}
  ⏱️  Masa Aktif: {validity_days} hari
  🔑 Kode Voucher: {system_generated_voucher_code}

  Simpan kode ini sebagai bukti langganan Anda.
  ```

**Acceptance Criteria:**
- [ ] AC-1: User baru daftar → sampai step 2 onboarding → tidak diblokir, modal activation muncul
- [ ] AC-2: Pilih "Beli Paket" → invoice Xendit dibuat, status subscription = `pending`
- [ ] AC-3: Bayar Xendit → webhook confirmed → tenant aktif, kode voucher sistem di-generate, dikirim via WA
- [ ] AC-4: Pending > 24 jam tidak dibayar → tenant + user dihapus otomatis
- [ ] AC-5: Pilih "Masukkan Voucher" → valid → langsung aktivasi + kode voucher sistem dikirim via WA
- [ ] AC-6: Redeem voucher plan sama → hari aktif diakumulasi
- [ ] AC-7: Redeem voucher plan berbeda → buat voucher baru, priority tetap plan tertinggi
- [ ] AC-8: Superadmin bisa generate N voucher codes sekaligus via API
- [ ] AC-9: Superadmin bisa lihat voucher aktif per tenant

**Files yang perlu diubah:**
- `frontend/umkm-web/src/components/Onboarding.vue` — hapus gate di step 1 & 2, tambah modal activation
- `services/billing-service/main.go` — `pending` subscription status, auto-delete expired, generate system voucher, day-duration logic
- `services/auth-service/main.go` — sync `is_frozen` dan plan cache saat activate
- `shared/migrations/` — add `validity_days` / `remaining_days` columns, `pending_timeout` di `tenant_subscriptions`
- `services/subscription-worker/main.go` — cron job auto-delete expired pending tenants
- `services/wa-gateway/` — WhatsApp notification template untuk activation
- `apps/umkm/accounting/main.go` — quota middleware baca priority plan (Pro > Lite)

**Notes:**
- Billing-service adalah source of truth untuk subscription state
- Auth-service baca dari Redis cache, di-sync saat `activateSubscription()` dipanggil
- Pending timeout default: 24 jam (bisa di-config via env `SUBSCRIPTION_PENDING_TIMEOUT_HOURS`)
- Superadmin generate voucher: kode di-generate client-side (frontend) atau server-side? Disarankan server-side via billing-service admin endpoint

---

## F016: Hybrid WhatsApp (Cloud API + whatsmeow)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Pisahkan jalur pengiriman WhatsApp: Meta Cloud API (official) untuk pesan transaksional kritis, whatsmeow (unofficial) untuk chatbot dengan rate limiter ketat. Mengurangi risiko banned nomor WA pengguna.

**Spec:**
- WhatsApp Cloud API service baru (`services/wa-cloud-api`, port 8210) wrap Meta Graph API v22.0
- Per-tenant credentials di tabel `wa_cloud_api_credentials` (phone_number_id, access_token, verify_token)
- Message routing via header `X-Message-Type` di wa-gateway:
  - `otp` → Cloud API (auth-service OTP login/register)
  - `invoice` → Cloud API (billing-service payment notifications)
  - `subscription` → Cloud API (accounting revenue digest)
  - `system` → Cloud API (notification-service system alerts)
  - _(tanpa header)_ → whatsmeow (chatbot conversational)
- Fallback: Cloud API gagal → otomatis whatsmeow (logged as WARN)
- Rate limiter whatsmeow: token bucket 5 msg/menit/tenant (mencegah spam ban)
- Reconnect backoff whatsmeow: exponential (30s → 60s → 240s → 10m), max 1 reconnect/5 menit
- Webhook Meta di `/webhooks/wa-cloud/` untuk status callback & incoming messages
- Superadmin CRUD credentials via `/admin/credentials`

**Acceptance Criteria (AC):**
- [x] AC-1: OTP terkirim via Cloud API saat auth-service kirim dengan `X-Message-Type: otp`
- [x] AC-2: Invoice/payment notification terkirim via Cloud API
- [x] AC-3: Chatbot tetap bisa kirim/terima via whatsmeow (tanpa header khusus)
- [x] AC-4: Rate limiter memblokir pesan whatsmeow ke-6+ dalam 1 menit (HTTP 429)
- [x] AC-5: Cloud API gagal → otomatis fallback ke whatsmeow (logged warning)
- [x] AC-6: Webhook Meta diterima di `/webhooks/wa-cloud/`
- [x] AC-7: Superadmin bisa CRUD credential via `POST/GET /admin/credentials`

**Files yang diubah:**
- `services/wa-cloud-api/main.go` — Service baru wrap Meta Graph API
- `services/wa-cloud-api/migrations.go` — Auto-migration runner
- `services/wa-gateway/main.go` — Message router + rate limiter + reconnect backoff
- `shared/sdk/config/config.go` — WhatsApp Cloud API config fields
- `shared/migrations/000030_wa_cloud_api_credentials.up.sql` — New credential table
- `services/api-gateway/main.go` — Webhook route `/webhooks/wa-cloud/` + health check
- `services/auth-service/main.go` — `X-Message-Type: otp` + `X-Source: auth-service`
- `services/billing-service/main.go` — `X-Message-Type: invoice` + `X-Source: billing-service`
- `services/notification-service/main.go` — `X-Message-Type: system` + `X-Source: notification-service`
- `apps/umkm/accounting/main.go` — `X-Message-Type: subscription` + `X-Source: umkm-accounting`
- `Dockerfile` + `docker-compose.yml` + `Makefile` + `.env.example` — Infrastructure

**Notes:**
- WhatsApp Cloud API pricing ~$0.005-0.08/message tergantung tipe. Lebih mahal dari whatsmeow (gratis) tapi zero ban risk.
- Perlu Meta Business App + phone_number_id + permanent access token per tenant
- whatsmeow tetap dipakai untuk chatbot karena conversational messages via Cloud API akan mahal
- Nomor whatsmeow sebaiknya nomor "tumbal" khusus, bukan nomor bisnis utama
- Lihat `CLAUDE.md` section "📱 Hybrid WhatsApp Architecture" untuk detail arsitektur

---

## F017: OTP 1-Hour Reuse Window

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** OTP (registrasi & login via WhatsApp) berlaku selama 1 jam penuh. Selama masa aktif, sistem tidak mengirim ulang OTP — mengurangi volume pesan WA keluar dan menurunkan risiko banned oleh WhatsApp.

**Spec:**
- OTP disimpan di Redis dengan TTL 1 jam (sebelumnya 5 menit)
- Saat user minta OTP baru: cek dulu apakah OTP yang masih aktif ada di Redis
  - Jika ada → return success dengan pesan "OTP sudah dikirim. Masih berlaku 1 jam." (TIDAK kirim ulang)
  - Jika tidak ada → generate OTP baru, simpan ke Redis, kirim via WA Gateway
- OTP TIDAK dihapus setelah verifikasi berhasil — tetap berlaku selama 1 jam penuh
- Redis TTL menangani auto-expiry otomatis setelah 1 jam
- Mencakup 3 endpoint OTP:
  - `POST /register` → OTP registrasi (`otp:{phone}`)
  - `POST /phone-login` → OTP login (`phone-login-otp:{phone}`)
  - `POST /forgot-password` → tidak terpengaruh (email-based, bukan WA)

**Acceptance Criteria (AC):**
- [x] AC-1: Request OTP pertama → OTP baru dikirim via WA, TTL Redis 1 jam
- [x] AC-2: Request OTP kedua dalam 1 jam → tidak kirim ulang, return "OTP sudah dikirim"
- [x] AC-3: Verifikasi OTP sukses → OTP tetap bisa dipakai ulang selama 1 jam
- [x] AC-4: Setelah 1 jam → OTP expired otomatis, request baru generate OTP baru
- [x] AC-5: Login OTP dan Register OTP punya key Redis terpisah (tidak konflik)

**Files Changed:**
- `services/auth-service/main.go` — `handleRegister()`, `handleVerifyOTP()`, `handlePhoneLogin()`, `handleVerifyPhoneLogin()`

**Notes:**
- Mengurangi jumlah pesan WA keluar drastis saat user berkali-kali minta OTP
- Kombinasi dengan F016 (Hybrid WA) + rate limiter memperkuat mitigasi ban
- Risiko keamanan rendah: OTP tetap 6-digit, brute-force dalam 1 jam tidak feasible
- Test OTP `000000` tetap berfungsi di development

---

## F014: Flexible LLM Model System

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem LLM yang flexible dan dynamic dengan capability-based routing, mendukung multiple providers dan per-use-case model selection.

**Spec:**
- **Model Registry**: Konfigurasi model dari environment variables dengan capability tags
- **Capability Routing**: Otomatis pilih model berdasarkan `use_case`:
  - `product` — untuk mengambil data product (murah, fast model)
  - `faq` — untuk menjawab FAQ (murah, fast model)
  - `general` — untuk tugas umum (default, full model)
- **Multi-Provider Support**: MiniMax (primary), Gemini (fallback), OpenAI (optional)
- **Fallback Chain**: Automatic fallback ke provider lain jika primary gagal
- **Per-Model Metrics**: Track usage per model (requests, tokens, cost)
- **Prometheus Endpoint**: `/metrics` untuk monitoring
- **API Endpoint**: `/v1/models` untuk list available models

**Environment Variables:**
```bash
# Single model (default)
MINIMAX_MODELS=MiniMax-M2.7
MINIMAX_CAPABILITIES=general,product,faq

# Multiple models (semicolon-separated)
MINIMAX_MODELS=MiniMax-M2.7;MiniMax-M2.7-Fast
MINIMAX_CAPABILITIES=general,product,faq;general
MINIMAX_COST_PER_1M_IN=0.30;0.10
MINIMAX_FALLBACKS=gemini:gemini-1.5-flash
```

**API Usage:**
```json
// Chat request dengan use_case routing
POST /v1/chat
{
  "message": "Apa harga produk X?",
  "use_case": "product"  // → auto-route ke model dengan capability "product"
}

// Override specific model
POST /v1/chat
{
  "message": "Explain code...",
  "provider": "openai",
  "model": "gpt-4o"
}

// List available models
GET /v1/models
```

**Acceptance Criteria:**
- [x] AC-1: Model registry loaded dari environment variables
- [x] AC-2: `use_case` field mengarahkan ke model yang sesuai capability
- [x] AC-3: Fallback chain berfungsi (MiniMax → Gemini → mock)
- [x] AC-4: Per-model metrics trackable via `/metrics`
- [x] AC-5: `/v1/models` endpoint return semua available models

**Files:**
- `shared/sdk/config/config.go` — LLMModel / LLMConfig structs + loadLLMModels()
- `services/ai-gateway/main.go` — capability-based routing + metrics
- `.env.example` — updated dengan flexible model config

**Notes:**
- Fokus saat ini: MiniMax sebagai primary model
- OpenAI/Gemini sebagai fallback/optional
- Per-tenant model override bisa ditambahkan di future (via DB config)

---

## F012: Sidebar Navigation UI

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Ganti horizontal header menu dengan sidebar kiri yang grouped dan collapsible.

**Spec:**
- Sidebar kiri dengan grouped menu items
- Groups: Operasi (Dashboard, Kasir, Katalog), Keuangan (Jurnal), Sistem (Automasi, Pengaturan, Super Admin)
- Collapsible groups — klik header untuk expand/collapse
- Active route highlighting
- User profile di bottom sidebar
- Responsive: sidebar di desktop, drawer di mobile
- Data-driven menu config (bukan hardcoded HTML)

**Acceptance Criteria:**
- [x] AC-1: Sidebar menampilkan grouped menu items
- [x] AC-2: Groups bisa collapse/expand
- [x] AC-3: Active route di-highlight
- [x] AC-4: User profile terlihat di sidebar
- [x] AC-5: Mobile: hamburger → drawer sidebar
- [x] AC-6: Smooth transition animations

**Files:**
- `frontend/umkm-web/src/components/AppSidebar.vue` — sidebar component baru
- `frontend/umkm-web/src/config/menu.ts` — menu configuration
- `frontend/umkm-web/src/App.vue` — use sidebar
- `frontend/umkm-web/src/style.css` — global sidebar styles

**Notes:** Icon menggunakan emoji untuk simplicity (bisa upgrade ke lucide-icons nanti).

---

## F013: N8N Integration via Super Admin

**Spec Status:** ❌ Removed
**Implementation:** —

**Deskripsi:** Integrate N8N ke Super Admin dashboard sebagai monitoring hub, bukan custom UI.

**Spec:**
- Super Admin dashboard → link ke N8N UI (new tab)
- N8N status indicator (connected/running/error)
- Recent executions widget (fetch from N8N API)
- Quick action: "Buka Workflow Editor"

**Acceptance Criteria:**
- [x] AC-1: N8N status visible di Super Admin
- [x] AC-2: Direct link to N8N editor
- [x] AC-3: Recent executions shown

**Files:**
- `services/billing-service/main.go` — N8N status & executions endpoints
- `frontend/superadmin-web/src/views/Dashboard.vue` — Direct link ke N8N editor
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Direct link ke N8N editor

**Notes:** N8N UI tetap digunakan untuk workflow editing. Super Admin hanya sebagai hub + monitoring.

---

**REMOVED (2026-06-12):** F013 dihapus karena:
- Tidak perlu dedicated `/n8n` page — N8N editor langsung diakses via `http://localhost:5678`
- Fitur sudah terpenuhi cukup dengan link di Dashboard.vue (direct ke N8N editor)

---

## F001: Multi-Store Quota Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** 1 owner bisa buat banyak toko dengan quota per plan.

**Spec:**
- 1 owner = banyak `stores` dengan `business_type` berbeda (restoran + cafe, dll)
- Quota di-enforce via `plan_features.feature_key='max_stores'`
- Default per tier: Lite=1, Pro=1, Business=5
- Superadmin bisa ubah quota via billing-service tanpa migration

**Acceptance Criteria:**
- [x] AC-1: GET `/api/umkm/stores` return quota info (`max_stores`, `can_add`)
- [x] AC-2: POST `/api/umkm/stores` check quota sebelum create
- [x] AC-3: Superadmin bisa CRUD plan-features via `/admin/plan-features`
- [x] AC-4: Header `X-User-Role: superadmin` injected by api-gateway

**Files:**
- `apps/umkm/accounting/main.go` — stores CRUD + quota check
- `services/billing-service/main.go` — superadmin plan-features CRUD

**Notes:** Quota dibaca langsung dari `plan_features` table, tidak di-cache.

---

## F002: Voucher Link Subscription Model

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Subscription = link-based voucher (primary) + Xendit (hybrid B2B).

**Spec:**
- Superadmin generate bulk voucher links via `/admin/voucher-links/generate`
- User klik link → redeem → subscription extend/created
- Grace period 0 hari (langsung freeze saat expired)
- Freeze = read-only + banner, user masih bisa login

**Voucher Lifecycle:**
```
[Superadmin] POST /admin/voucher-links/generate
    { program_id, count, valid_days, base_url }
    → Returns: { links: [{token, url}, ...] }

[User] Klik link → POST /voucher/redeem-link { token, tenant_id }
    1. Verify JWT signature
    2. Lookup voucher_links by SHA-256(token)
    3. Validate: is_active, not redeemed, not expired
    4. Check max_uses_per_tenant
    5. Mark link redeemed
    6. Extend or create subscription
    7. Un-freeze if was frozen
```

**Acceptance Criteria:**
- [x] AC-1: Superadmin generate voucher links (bulk)
- [x] AC-2: User redeem via signed token link
- [x] AC-3: Subscription extend/created on redeem
- [x] AC-4: Tenant un-frozen on successful redeem

**Files:**
- `services/billing-service/main.go` — voucher generation + redemption
- `shared/migrations/000028_voucher_subscription.up.sql` — schema

**Notes:** Voucher token di-hash SHA-256 sebelum save ke DB.

---

## F003: Subscription Freeze Worker

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Background worker yang freeze tenant expired.

**Spec:**
- Cek `tenant_subscriptions` setiap `FREEZE_CHECK_INTERVAL` (default 1 jam)
- Subscription dengan `current_period_end < NOW()` → freeze
- Batch update: `status='frozen'`, `tenants.is_frozen=true`
- Liveness check: GET `/healthz`

**Acceptance Criteria:**
- [x] AC-1: Worker running dengan interval configurable
- [x] AC-2: Expired subscriptions frozen automatically
- [x] AC-3: `is_frozen` denormalized flag updated

**Files:**
- `services/subscription-worker/main.go` — freeze worker
- `docker-compose.yml` — worker service definition

**Notes:** GRACE_PERIOD_HOURS=0 (0-day freeze).

---

## F004: Read-only Enforcement (Frozen Tenant)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Middleware block write operations saat tenant frozen.

**Spec:**
- Middleware `auth.RequireActiveSubscription`
- Block POST/PATCH/PUT/DELETE saat frozen
- GET tetap pass (user bisa lihat data)
- Set header `X-Subscription-Status: active|frozen`

**Acceptance Criteria:**
- [x] AC-1: Write operations blocked saat frozen
- [x] AC-2: Read operations tetap jalan
- [x] AC-3: Response include subscription status header

**Files:**
- `shared/sdk/auth/subscription_guard.go` — middleware
- `apps/umkm/accounting/main.go` — applied ke router

**Notes:** Banner message untuk UI frontend dari header `X-Subscription-Status`.

---

## F005: Superadmin Dashboard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Unified dashboard untuk superadmin (bukan per-product).

**Spec:**
- 1 unified dashboard di `frontend/superadmin-web/` (port 3401)
- Sections: Overview, Voucher Programs, Generate Links, Frozen Accounts
- Overview: tenant counts, voucher stats 30d, revenue (Xendit), subs by plan
- Frozen Accounts: list + kirim reminder WA

**Acceptance Criteria:**
- [x] AC-1: Overview dengan aggregated stats
- [x] AC-2: Voucher program CRUD
- [x] AC-3: Bulk generate + download CSV
- [x] AC-4: Frozen accounts list dengan WA reminder action

**Files:**
- `frontend/superadmin-web/` — Vue 3 frontend
- `services/billing-service/main.go` — dashboard APIs

**Notes:** API Gateway inject `X-User-Role: superadmin` dari JWT claim.

---

## F006: Multi-Tenant WA Session Pool

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Setiap tenant punya WA session sendiri untuk chatbot.

**Spec:**
- Tabel `wa_sessions` store session per tenant
- Status: `connected`, `qr_pending`, `disconnected`
- WA Gateway handle multi-device
- Session di-manage via N8N workflow

**Acceptance Criteria:**
- [x] AC-1: Tenant punya dedicated WA session
- [x] AC-2: Session status trackable
- [x] AC-3: QR code generation per tenant

**Files:**
- `services/wa-gateway/main.go` — WA session management
- `shared/migrations/000029_n8n_queue_mode.up.sql` — schema

**Notes:** Saat dev lokal, hanya satu WA device aktif.

---

## F007: Chatbot with RAG

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** AI chatbot dengan Retrieval-Augmented Generation.

**Spec:**
- FAQ dan Products di-index ke pgvector
- Chatbot retrieve relevant context sebelum LLM call
- Configurable per tenant (LLM, prompt, escalation settings)
- N8N workflow: Config → Session → RAG → LLM → Save

**Acceptance Criteria:**
- [x] AC-1: FAQ/Products indexed ke vector store
- [x] AC-2: Chatbot retrieve relevant context
- [x] AC-3: Per-tenant chatbot config
- [x] AC-4: Multi-channel session (WA, web, etc)

**Files:**
- `apps/umkm/chatbot/main.go` — chatbot API
- `services/ai-gateway/main.go` — embeddings endpoint
- `n8n/workflows/rag_indexer.json` — index workflow
- `n8n/workflows/universal_chatbot.json` — chatbot workflow

**Notes:** Embeddings via OpenAI/Anthropic melalui ai-gateway.

---

## F008: Escalation to Chatwoot

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Chatbot escalation ke human agent via Chatwoot.

**Spec:**
- Trigger escalation berdasarkan keyword atau fallback
- Buat conversation di Chatwoot
- Transfer context (conversation history, customer info)
- Log escalation history

**Acceptance Criteria:**
- [x] AC-1: Auto-escalation based on config
- [x] AC-2: Conversation created in Chatwoot
- [x] AC-3: Context transferred to agent
- [x] AC-4: Escalation history logged

**Files:**
- `n8n/workflows/escalation_handler.json` — escalation workflow
- `shared/migrations/000029_n8n_queue_mode.up.sql` — escalation_history table

**Notes:** Chatwoot running di port 3000 (docker-compose).

---

## F009: N8N Queue Mode Automation

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** N8N dengan Redis queue untuk horizontal scaling.

**Spec:**
- N8N Main: UI + Webhook Receiver + Workflow Editor
- N8N Worker: Execution Worker (scalable)
- Redis DB 2: Bull Queue untuk job distribution
- 8 workflows configured

**Workflows:**
| Workflow | Trigger | Purpose |
|:---------|:--------|:--------|
| `universal_chatbot.json` | Webhook | Multi-tenant chatbot |
| `rag_indexer.json` | Webhook | Index FAQ/Products |
| `escalation_handler.json` | Webhook | Escalation to Chatwoot |
| `master_automations.json` | Cron (1m) | Execute due automations |
| `daily_revenue_digest.json` | Cron | Revenue digest to Telegram |
| `freeze_reminder.json` | Cron | Expired subscription reminder |
| `campaign_voter_onboard.json` | Webhook | Voter onboarding |
| `voucher_wa_distribute.json` | Webhook | Voucher WA distribution |

**Acceptance Criteria:**
- [x] AC-1: N8N running dengan queue mode
- [x] AC-2: Redis queue configured
- [x] AC-3: All 8 workflows deployed

**Files:**
- `docker-compose.yml` — n8n-main, n8n-worker, redis config
- `n8n/workflows/*.json` — workflow definitions
- `infra/postgres/init.sql` — auto-create `wch_n8n` database
- `.env` / `.env.example` — `N8N_DB_*`, `N8N_ENCRYPTION_KEY` vars

**Notes:** Worker auto-configure dari shared database — scaling tinggal `docker-compose up -d --scale n8n-worker=N`. Persistence via dedicated `wch_n8n` database, backup: `pg_dump wch_n8n > n8n_backup.sql`.

---

## F010: Campaign Volunteer Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Manajemen volunteer untuk campaign.

**Spec:**
- CRUD volunteer dengan role (ketua, saksi, dll)
- Assign volunteer ke TPS/area
- Track volunteer activity
- Encrypted NIK storage

**Acceptance Criteria:**
- [x] AC-1: Volunteer CRUD
- [x] AC-2: Volunteer assignment to area
- [x] AC-3: NIK encrypted at rest
- [x] AC-4: Activity tracking

**Files:**
- `apps/campaign/api/handlers/volunteer.go`
- `apps/campaign/api/main.go`

**Notes:** NIK di-encrypt AES-256-GCM, key dari `cfg.EncryptionKey`.

---

## F011: Campaign Voter Onboarding

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Voter registration via webhook dari N8N.

**Spec:**
- N8N workflow trigger voter onboarding
- Voter data di-encrypt sebelum save
- Link voter ke TPS

**Acceptance Criteria:**
- [x] AC-1: Webhook endpoint untuk voter creation
- [x] AC-2: Voter data encrypted
- [x] AC-3: TPS assignment
- [x] AC-4: Bulk import support

**Files:**
- `apps/campaign/api/handlers/voter.go`
- `n8n/workflows/campaign_voter_onboard.json`

**Notes:** Bulk import via CSV dengan async processing.

---

## 📍 Lokasi Kode (Quick Reference)

### Mau Tambah Endpoint/API?

```
UMKM Accounting    ──→ apps/umkm/accounting/main.go (flat pattern)
UMKM Business      ──→ apps/umkm/business/main.go (flat pattern)
UMKM Chatbot       ──→ apps/umkm/chatbot/main.go (flat pattern)
UMKM Automation    ──→ apps/umkm/automation/main.go (worker)

Campaign API       ──→ apps/campaign/api/handlers/<nama>.go
                     + daftarkan di apps/campaign/api/main.go

Auth Service       ──→ services/auth-service/main.go
AI Gateway         ──→ services/ai-gateway/main.go
Billing Service    ──→ services/billing-service/main.go
WA Gateway         ──→ services/wa-gateway/main.go
Notification       ──→ services/notification-service/main.go
API Gateway        ──→ services/api-gateway/main.go
```

### Mau Tambah Tabel Database?

```bash
# Cek nomor terakhir:
ls shared/migrations/*.up.sql | tail -1

# Buat migration baru:
shared/migrations/NNNNNN_nama_fitur.up.sql
shared/migrations/NNNNNN_nama_fitur.down.sql
```

### Mau Tambah Config?

```
1. shared/sdk/config/config.go  ← Tambah field + LoadConfig()
2. .env.example                 ← Tambah dengan contoh nilai
3. docker-compose.yml           ← Tambah env var
```

### Mau Tambah UI Frontend?

```
UMKM      ──→ frontend/umkm-web/src/components/<Nama>.vue
Campaign  ──→ frontend/campaign-web/src/
Superadmin ──→ frontend/superadmin-web/src/
```

### Mau Tambah Service/Worker?

```
Wajib update:
☐ Makefile
☐ Dockerfile
☐ docker-compose.yml
☐ services/api-gateway/main.go (jika REST API)
☐ CLAUDE.md (Port Registry)
☐ .env.example
```

---

## 🔧 Cara Menambah Feature Baru

1. **Tambah SPEC entry** di section ini dengan format:
   ```
   ### F012: [Nama Feature]
   **Spec Status:** ⏳ Draft
   **Implementation:** ⏳ Pending
   ...
   ```

2. **User approve** — tambahkan comment atau ubah status ke "✅ Approved"

3. **AI implement** — setelah approved, AI coding berdasarkan SPEC

4. **Update implementation status** — ubah ke "✅ Done" setelah selesai

5. **Update Feature Registry table** di atas

---

*Lihat [CONTRIBUTING.md](../CONTRIBUTING.md) untuk panduan coding.*