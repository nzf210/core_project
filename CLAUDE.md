# WCH Platform — Claude AI Project Memory

> **Dokumen ini adalah memori permanen AI coding assistant untuk monorepo WCH Platform.**
> Dibaca otomatis di setiap sesi. Selalu periksa file ini sebelum membuat perubahan.

---

## 🎯 Identitas Proyek

**WCH Platform** — SaaS multi-produk berbasis **Go (Golang)**. Satu monorepo, satu `go.mod`.

| Produk | Deskripsi | Direktori Utama |
|:-------|:----------|:----------------|
| **UMKM** | AI Agent UMKM: Double-Entry Accounting, AI Chatbot WA, POS | `apps/umkm/` |
| **Crypto** | ~~Trading Bot: DCA, Grid, Signal Bot berbasis Binance API~~ (ARCHIVED) | `apps/crypto/` |
| **Campaign** | Manajemen Pemilu: Relawan, Real Count, AI Sentiment | `apps/campaign/` |

Semua produk berbagi `services/` (auth, billing, ai-gateway, notification, wa-gateway, api-gateway).

---

## 🔒 Spec-First Workflow — WAJIB DIIKUTI

**Sebelum coding APAPUN, selalu cek `docs/FEATURE_MAP.md`.**

```
USER menulis SPEC      →       AI review & clarify      →       USER approve
     ↓                         ↓                                  ↓
 FEATURE_MAP.md         AI tanya clarifications           USER comment/approve
                              ↓                                  ↓
                      AI wait for approval          AI implement dari SPEC
                                                            ↓
                                                    USER review diff
                                                            ↓
                                                    JALANKAN TESTING 
```

### Checklist Sebelum Coding:

1. **Cek FEATURE_MAP.md** — Apakah fitur sudah ada di registry?
2. **Cek SPEC status** — Sudah ✅ Approved?
   - Jika ⏳ Draft: Tanya USER dulu, jangan implement
   - Jika ❌ Rejected: Jangan implement
3. **Ambiguitas?** — Tanya USER clarification dulu
4. **Implementasi selesai?** — Update `Implementation` status di FEATURE_MAP.md
5. **Testing Wajib** — Setiap kali ada *perubahan*, *tambah fungsi*, atau *hapus fungsi*, JALANKAN TEST sebelum menyelesaikan task:
   - `make check` (untuk menjalankan linter, build, dan semua test)
   - Atau `go test ./apps/umkm/... -v` (untuk test spesifik)

### Cara Menambah Fitur Baru:

1. User tambah entry di `docs/FEATURE_MAP.md` dengan format SPEC
2. User ubah status ke "✅ Approved" saat sudah siap
3. AI implement berdasarkan SPEC yang approved
4. AI update `Implementation` → "✅ Done" setelah selesai

---

## 🏗️ Peta Direktori (Aktual)

```
core_project/                   ← Root monorepo (satu go.mod)
├── apps/
│   ├── umkm/
│   │   ├── accounting/         ← Accounting engine + POS (Port 8201)
│   │   │   ├── main.go         ← Semua handler + router (flat pattern)
│   │   │   └── db.go           ← DB connection pool
│   │   ├── chatbot/            ← AI Chatbot via WhatsApp (Port 8202)
│   │   │   ├── main.go
│   │   │   └── db.go
│   │   ├── business/           ← Business management API (Port 9005)
│   │   │   └── main.go
│   │   └── automation/         ← Background worker (tanpa HTTP server)
│   │       └── main.go
│   └── campaign/
│       └── api/                ← Campaign REST API (Port 9002)
│           ├── main.go         ← Router saja
│           ├── handlers/       ← Handler per-resource (satu file = satu resource)
│           │   ├── campaign.go
│           │   ├── volunteer.go
│           │   ├── voter.go
│           │   └── responses.go  ← WriteJSON + ExtractTenantID helper
│           └── repository/
│               └── db.go       ← DB connection pool
│
├── services/
│   ├── api-gateway/            ← Reverse proxy + routing (Port 8000)
│   │   └── main.go
│   ├── auth-service/           ← JWT, Login, Register, RBAC (Port 8001)
│   │   ├── main.go
│   │   ├── jwt.go
│   │   └── db.go
│   ├── ai-gateway/             ← LLM Proxy + Semantic Cache (Port 8002)
│   │   ├── main.go
│   │   └── db.go
│   ├── billing-service/        ← Xendit subscription (Port 8003)
│   │   ├── main.go
│   │   └── db.go
│   ├── wa-gateway/             ← WhatsApp via whatsmeow (Port 8202)
│   │   └── main.go
│   ├── wa-cloud-api/           ← WhatsApp Cloud API — Meta Official (Port 8210)
│   │   ├── main.go
│   │   └── migrations.go
│   ├── subscription-worker/    ← Hold expired tenants (Port 8006)
│   │   └── main.go
│   └── notification-service/   ← Telegram/Email notif (Port 8005)
│       └── main.go
│
├── shared/
│   ├── sdk/
│   │   ├── config/config.go    ← SATU-SATUNYA cara baca konfigurasi
│   │   ├── auth/               ← JWT middleware untuk protect routes
│   │   ├── db/                 ← PostgreSQL connection helper
│   │   ├── cache/              ← Redis connection helper
│   │   ├── response/           ← Standard JSON response helper
│   │   ├── migrate/            ← Auto-migration runner (shared/sdk/migrate)
│   │   └── webhook/            ← Webhook utilities
│   └── migrations/             ← Database SQL migrations (000001 — 000063)
│
├── frontend/
│   ├── umkm-web/               ← Vue 3 + Vite (Port 3201, Docker scaled: 3201-3203)
│   ├── campaign-web/           ← Vue 3 + Vite (Port 3301)
│   └── superadmin-web/         ← Vue 3 + Vite (Port 3401)
│
├── tools/
│   ├── scripts/               ← Archived fix/patch scripts
│   └── testdata/              ← Sample data untuk testing manual
│
├── docs/                       ← Semua dokumentasi proyek
├── infra/                      ← Docker, Nginx, n8n, deploy scripts
├── scripts/                    ← CI/CD, loadtest, e2e, utility scripts
│
├── logs/                       ← LOG FILES (jangan edit manual!)
│   └── *.log                   ← Diisi otomatis oleh make start-all
├── run/                        ← PID FILES (jangan edit manual!)
│   └── *.pid                   ← Diisi otomatis oleh make start-all
├── bin/                        ← BINARY FILES lokal (di-gitignore)
│   └── <service>               ← Output dari make build-all
│
├── CLAUDE.md                   ← File ini (AI memory)
├── CONTRIBUTING.md             ← Panduan kontribusi & workflow
├── Makefile                    ← Shortcut commands
├── Dockerfile                  ← Multi-stage build (semua service)
├── docker-compose.yml          ← Docker orchestration
├── go.mod                      ← Go module (satu untuk semua)
└── .env.example                ← Template environment variables
```

---

## 📡 Port Registry

| Port | Service | Direktori |
|:-----|:--------|:----------|
| `8010` | API Gateway | `services/api-gateway` (Docker mapped: `8010:8000`) |
| `8001` | Auth Service | `services/auth-service` (via API Gateway, no direct listen) |
| `8002` | AI Gateway | `services/ai-gateway` (via API Gateway, no direct listen) |
| `8013` | Billing Service | `services/billing-service` (Docker mapped: `8013:8003`) |
| `8015` | Notification Service | `services/notification-service` (Docker mapped: `8015:8005`) |
| `8016` | Subscription Worker | `services/subscription-worker` (Docker mapped: `8016:8006`) |
| `8201` | UMKM Accounting | `apps/umkm/accounting` (native, no Docker mapping) |
| `8202` | WA Gateway | `services/wa-gateway` (native, shared port with Chatbot) |
| `8210` | WA Cloud API | `services/wa-cloud-api` (Docker mapped: `8210:8210`) |
| `8202` | UMKM Chatbot | `apps/umkm/chatbot` (native, shared port with WA Gateway) |
| `8213` | UMKM Automation | `apps/umkm/automation` (Docker mapped: `8213:8203`) |
| `9005` | UMKM Business | `apps/umkm/business` |
| `9002` | Campaign API | `apps/campaign/api` |
| `3000` | Chatwoot (Self-hosted) | docker-compose (`3000:3000`) |
| `3201` | Frontend UMKM | `frontend/umkm-web` (Docker `3201:80`, scaled 3x → 3201-3203) |
| `3202` | Frontend Superadmin | `frontend/superadmin-web` |
| `3203` | Frontend Campaign | `frontend/campaign-web` |
| `5433` | PostgreSQL + pgvector (Docker) | docker-compose (`5433:5432`) |
| `5678` | N8N Main (Queue Mode) | docker-compose (`5678:5678`) |
| `6381` | Redis (Docker) | docker-compose (`6381:6379`) |

> ⚠️ WA Gateway dan UMKM Chatbot **tidak bisa jalan bersamaan di satu host** karena berbagi port 8202. Saat berjalan native, jalankan salah satu saja — jika dua-duanya perlu, gunakan Docker di mana Chatbot internal via API Gateway.

---

## 📱 Hybrid WhatsApp Architecture

WCH Platform menggunakan **arsitektur hybrid** untuk WhatsApp:

| Jalur | Library | Use Case | Risiko Ban |
|:------|:--------|:---------|:-----------|
| **Cloud API (Meta Official)** | `services/wa-cloud-api` (port 8210) | Broadcast Massal, Transaksional Eksternal | ✅ Nyaris 0%. Wajib pakai Wallet |
| **whatsmeow (Unofficial)** | `services/wa-gateway` (port 8202) | AI chatbot, OTP Login Internal, Notif 1-on-1 (Klinik) | ⚠️ Rate-limited. DILARANG untuk Blast |

**Message routing** terjadi di `services/wa-gateway` via header `X-Message-Type`:
- `X-Message-Type: broadcast` → WAJIB Cloud API (jika tenant QR, request gagal 402/400)
- `X-Message-Type: otp` → Auto-routing (WCH System untuk new tenant, QR tenant untuk login internal)
- `X-Message-Type: invoice` → Auto-routing
- `X-Message-Type: subscription` → Auto-routing
- `X-Message-Type: system` → Auto-routing
- _(tanpa header)_ → whatsmeow (chatbot conversational)

**Fallback:** Jika Cloud API gagal, wa-gateway otomatis fallback ke whatsmeow.

**Rate Limiter (whatsmeow):** Token bucket 5 msg/menit/tenant — mencegah ban dari spam.

**Reconnect Backoff:** Exponential backoff (30s → 10m) + max 1 reconnect/5 menit.

**Credential per-tenant:** Tabel `wa_cloud_api_credentials` — setiap tenant bisa punya nomor WA bisnis sendiri di Meta.

### 🔀 WA Provider Preference (F048)

Tenant dapat meng-override hybrid routing dengan preferensi eksplisit di `tenant_chatbot_configs.wa_provider_preference`:

| Value | Behavior |
|:------|:---------|
| `auto` (default) | Hybrid: transactional→Cloud, conversational→whatsmeow |
| `whatsmeow` | Force ke whatsmeow, skip Cloud API total |
| `cloud_api` | Force Cloud API, NO fallback (jika gagal → error 502) |

**Addon Gate untuk Cloud API:** Cloud API option di-lock di UI kecuali tenant punya `plan_features.feature_key = 'wa_cloud_api' AND is_enabled = true` untuk plan-nya. Cek via endpoint `GET /api/umkm/chatbot/permissions`.

**Hybrid WA Setup Wizard (v2):**
- Frontend `WASetup.vue` menyediakan halaman setup WA terpadu dengan 2 tab: Koneksi & Provider, Pengaturan AI CS
- Flow validasi 2-step: Validate credential via Meta Graph API → Baru simpan ke DB
- Backend `POST /api/wa/validate` (via api-gateway → wa-cloud-api) untuk validasi real-time access token + phone number ID
- Migration `000070` menambah kolom `verification_status`, `verified_at`, `last_checked_at`, `check_error` ke `wa_cloud_api_credentials`

**Files involved:**
- `services/wa-gateway/main.go` → `getTenantWAProviderPreference()` (line ~172), override routing (line ~835)
- `apps/umkm/accounting/main.go` → `ChatbotConfig.WAProviderPreference`, handler `GET/PUT /chatbot/config`, enhanced `handleWACloudAPICredential`
- `services/wa-cloud-api/main.go` → `handleValidateCredential` (real-time Meta API validation)
- `services/api-gateway/main.go` → route `/api/wa/validate`
- `shared/migrations/000063_wa_provider_preferences.up.sql` → enum + kolom
- `shared/migrations/000064_wa_cloud_api_plan_feature.up.sql` → seed plan_features
- `shared/migrations/000070_wa_credential_verification.up.sql` → verification status columns
- `frontend/umkm-web/src/components/WASetup.vue` → 2-step validate+save wizard
- `frontend/umkm-web/src/api.ts` → `validateCloudAPICredential()`
- `frontend/umkm-web/src/components/ChatbotConfig.vue` → dropdown "📡 WA Provider"

### 📨 OTP Routing via Header Override (F048 AC-6)

**CATATAN PENTING**: OTP **hanya** digunakan untuk keperluan login internal (Owner/Staff UMKM) ke dashboard WCH, bukan untuk pelanggan UMKM. Pelanggan tidak perlu OTP untuk interaksi dengan toko/klinik.

Untuk OTP (auth-service → wa-gateway), auth-service membaca `tenants.auth_wa_provider_preference` dan forward ke wa-gateway via HTTP header `X-WA-Provider-Override`. wa-gateway override preference dari header (lebih tinggi prioritas dari DB lookup).

**Flow:**
1. `auth-service` saat generate OTP → query `SELECT auth_wa_provider_preference FROM tenants WHERE id = $1`
2. Set header `X-WA-Provider-Override: auto|whatsmeow|cloud_api`
3. `wa-gateway` baca header → override preference sebelum routing logic
4. Untuk **register baru** (tenant belum ada) → `senderTenant = "system"` → fallback `auto` → no override

**Files involved:**
- `services/auth-service/main.go` → line ~356 (handleRegister), line ~1371 (handlePhoneLogin) — baca `auth_wa_provider_preference`, set `X-WA-Provider-Override` header
- `services/wa-gateway/main.go` → line ~842 — `if override := r.Header.Get("X-WA-Provider-Override"); override != "" { preference = override }`

**Backward compatible:** Header optional. Jika tidak ada → wa-gateway pakai DB lookup (`getTenantWAProviderPreference`) seperti biasa.

---
- Sebelum kirim OTP baru → cek Redis apakah OTP aktif masih ada (`otp:{phone}` / `phone-login-otp:{phone}`)
- Jika masih aktif → return "OTP sudah dikirim" (TIDAK kirim ulang), mengurangi volume pesan WA
- OTP tidak dihapus setelah verifikasi sukses — tetap berlaku selama 1 jam
- Redis TTL (1 jam) menangani auto-expiry
- Detail: `docs/FEATURE_MAP.md` → F017

---

## 🤖 Telegram Auth Bot

**User bisa daftar dan login via Telegram Bot** sebagai alternatif WhatsApp:

| Endpoint | Method | Deskripsi |
|:---------|:-------|:----------|
| `/auth/telegram/register` | POST | Daftar via Telegram — kirim `telegramChatId` + data registrasi |
| `/auth/telegram/login` | POST | Login via Telegram — kirim `telegramChatId` + `phoneNumber` |
| `/auth/telegram/webhook` | POST | Webhook bot Telegram — handle `/start` command |

**Flow:**
1. User chat bot Telegram WCH → bot reply dengan Chat ID
2. User kirim `telegramChatId` + data ke `/auth/telegram/register` atau `/auth/telegram/login`
3. Auth-service generate OTP dan kirim via Telegram Bot API (`sendMessage`)
4. OTP disimpan di Redis dengan key yang sama dengan WA (`otp:{phone}` / `phone-login-otp:{phone}`)
5. Verifikasi OTP via `/auth/verify-otp` atau `/auth/verify-phone-login` — endpoint sama untuk WA & Telegram
6. Setelah login, `telegram_chat_id` di-update di tabel `users`

**Setup Webhook:**
```bash
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<domain>/auth/telegram/webhook"
```

**Bot token:** `TELEGRAM_BOT_TOKEN` di `.env` — shared dengan notification-service (bisa pakai bot yang sama)

**Detail:** `docs/FEATURE_MAP.md` → F018

---

## 🔄 N8N Queue Mode Architecture

**N8N berjalan dalam Queue Mode** dengan horizontal scaling via Redis:

| Komponen | Container | Fungsi |
|:---------|:----------|:-------|
| `n8n-main` | `wch-n8n-main` | UI + Webhook Receiver + Workflow Editor |
| `n8n-worker` | `wch-n8n-worker` | Execution Worker (scalable) |
| Redis DB 2 | `wch-redis` | Bull Queue untuk job distribution |

**Workflows (8 total):**

| Workflow | Trigger | Deskripsi |
|:---------|:--------|:----------|
| `universal_chatbot.json` | Webhook POST | Multi-tenant chatbot: Config → Session → RAG → LLM → Save → Escalation |
| `rag_indexer.json` | Webhook POST | Index FAQ & Products ke pgvector |
| `escalation_handler.json` | Webhook POST | Escalation ke Chatwoot |
| `master_automations.json` | Cron (setiap menit) | Execute due automations |
| `daily_revenue_digest.json` | Cron (harian) | Revenue digest ke Telegram |
| `expiration_reminder.json` | Cron | Reminder untuk expired subscriptions |
| `campaign_voter_onboard.json` | Webhook | Campaign voter onboarding |
| `voucher_wa_distribute.json` | Webhook | Distribusi voucher via WA |

**Persistence — Dedicated Database:**

N8N menggunakan database PostgreSQL terpisah (`wch_n8n`) dari database utama platform. Auto-created via `infra/postgres/init.sql` saat postgres pertama kali start.

| Var | Nilai Default | Deskripsi |
|:----|:--------------|:----------|
| `N8N_DB_NAME` | `wch_n8n` | Database name |
| `N8N_DB_HOST` | `127.0.0.1` | PostgreSQL host |
| `N8N_DB_PORT` | `5433` | PostgreSQL port (Docker) |
| `N8N_DB_USER` | `wch_admin` | Database user |
| `N8N_ENCRYPTION_KEY` | *(wajib di-set)* | 32-byte key untuk enkripsi credential di DB |

Scaling worker tinggal: `docker-compose up -d --scale n8n-worker=3` — worker auto-sync workflow & credential dari database yang sama.

**Migrasi ke Production Server Baru:**
```bash
# 1. Backup
pg_dump -h oldserver -U wch_admin -d wch_n8n > n8n_backup.sql
# 2. Restore
psql -h newserver -U wch_admin -d wch_n8n < n8n_backup.sql
# 3. Set N8N_ENCRYPTION_KEY yang sama di server baru
```
⚠️ `N8N_ENCRYPTION_KEY` harus SAMA antara backup dan restore — jika berbeda, semua credential (API keys, tokens) tidak bisa didekripsi.

**Multi-Tenant WA Session Pool:**

Setiap tenant memiliki WA session sendiri di tabel `wa_sessions`:
```
tenant_a → wa-001 (connected)
tenant_b → wa-002 (connected)
tenant_c → wa-003 (qr_pending)
```

---

## ⚙️ Konvensi Kode — WAJIB DIIKUTI

### Go Backend

| Aspek | Aturan | Larangan |
|:------|:-------|:---------|
| HTTP Framework | `net/http` standar | ❌ Gin, Echo, Fiber |
| Database | `github.com/jackc/pgx/v5` | ❌ GORM, `database/sql` + lib/pq |
| JWT | `github.com/golang-jwt/jwt/v5` | ❌ Library lain |
| Password | `golang.org/x/crypto/bcrypt` (cost=12) | ❌ MD5, SHA, plain text |
| Logging | `log/slog` (structured JSON) | ❌ `fmt.Println`, `log.Println` |
| Config | `config.LoadConfig()` dari `shared/sdk/config` | ❌ `os.Getenv()` langsung |
| Uang/Harga | `int64` satuan **sen** (1 rupiah = 100 sen) | ❌ `float64` |
| UUID | `github.com/google/uuid` | ❌ Auto-increment integer |
| Error Handling | `return error` eksplisit | ❌ `panic()` di luar main |
| AI/LLM | Via `services/ai-gateway` | ❌ Panggil MiniMax/OpenAI langsung dari `apps/` |

### Pola Kode Handler

```go
// Pattern WAJIB untuk setiap HTTP handler:
func handleResource(w http.ResponseWriter, r *http.Request) {
    // 1. Baca tenant ID dari header (multi-tenant!)
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        writeJSON(w, http.StatusBadRequest, Response{Message: "Missing X-Tenant-ID"})
        return
    }

    // 2. Dispatch berdasarkan method
    switch r.Method {
    case http.MethodGet:
        // handle GET
    case http.MethodPost:
        // handle POST
    default:
        writeJSON(w, http.StatusMethodNotAllowed, Response{Message: "Method not allowed"})
    }
}

// Pattern standard writeJSON:
func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}
```

### Pola Database

```go
// ✅ BENAR — Parameterized query
rows, err := DB.Query(ctx, "SELECT * FROM users WHERE tenant_id = $1", tenantID)

// ❌ SALAH — String concatenation (SQL Injection!)
rows, err := DB.Query(ctx, "SELECT * FROM users WHERE tenant_id = '" + tenantID + "'")
```

### Pola Struct Response

```go
// Gunakan struct ini secara konsisten:
type Response struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

---

## 🗄️ Konvensi Database

### Aturan Tabel Baru

```sql
-- SETIAP tabel harus punya kolom ini:
CREATE TABLE nama_tabel (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- ✅ UUID, bukan SERIAL
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- ✅ Multi-tenant
    -- ... kolom bisnis ...
    created_at TIMESTAMPTZ DEFAULT NOW(),   -- ✅ Wajib
    updated_at TIMESTAMPTZ DEFAULT NOW()    -- ✅ Wajib
);
-- ✅ Index pada tenant_id wajib:
CREATE INDEX idx_nama_tabel_tenant_id ON nama_tabel(tenant_id);
```

### Tipe Data

| Data | SQL | Go |
|:-----|:----|:---|
| ID | `UUID` | `string` |
| Uang/Harga | `BIGINT` (satuan sen) | `int64` |
| Timestamp | `TIMESTAMPTZ` | `time.Time` |
| JSON fleksibel | `JSONB` | `map[string]interface{}` |
| Data terenkripsi | `TEXT` | `string` (ciphertext AES-GCM) |

### Penamaan File Migration

```
# Format: nomor_urut_nama_fitur.up.sql / .down.sql
shared/migrations/000025_nama_feature.up.sql
shared/migrations/000025_nama_feature.down.sql
```

### Auto-Migration (AKTIF)

**Semua services jalankan migration otomatis saat startup.** Tidak perlu `psql` manual.

- Package: `shared/sdk/migrate` — migration runner + tracker
- Table: `schema_migrations` — daftar versi yang sudah dijalankan
- Setiap migration dieksekusi dalam **transaction** (gagal → rollback)
- Idempotent: migration yang sudah jalan tidak dijalankan lagi

```bash
# Tambah migration baru
make migrate-new NAME=add_invoice_table
# → shared/migrations/000029_add_invoice_table.up.sql
# → shared/migrations/000029_add_invoice_table.down.sql

# Cek status migrations
make migrate-status
```

**Services yang sudah terintegrasi auto-migration:**
- `services/auth-service` ✅
- `services/billing-service` ✅
- `apps/umkm/accounting` ✅
- `apps/umkm/chatbot` ✅
- `apps/campaign/api` ✅

**Tabel terkait N8N Queue Mode / Chatbot Upgrade (migration 000029):**
- `wa_sessions` — Multi-tenant WA session pool
- `tenant_chatbot_configs` — Per-tenant chatbot config (LLM, prompt, escalation, RAG)
- `conversation_sessions` — Multi-channel conversation sessions
- `conversation_logs` — Structured conversation logs + analytics
- `vector_embeddings` — pgvector embeddings untuk RAG (FAQ & Products)
- `escalation_history` — History escalation ke Chatwoot

---

## 🤖 MiniMax M2.7 — LLM Utama Platform

**SELALU** panggil LLM melalui `services/ai-gateway` — **JANGAN** panggil API AI langsung dari `apps/`.

```go
// ✅ BENAR — Panggil via AI Gateway
resp, err := http.Post("http://localhost:8002/v1/chat", "application/json", payload)

// ❌ SALAH — Langsung dari apps/
client := openai.NewClient(cfg.AI.MiniMaxAPIKey)
```

AI Gateway sudah menangani:
- Semantic caching via Redis (key: `ai:cache:{sha256(prompt)}`)
- Billing tracking per-tenant di tabel `ai_usage_logs`
- Fallback ke Gemini jika MiniMax gagal

---

## 🔒 Keamanan — KRITIS

| Data Sensitif | Metode | Lokasi DB |
|:--------------|:-------|:----------|
| API Key Exchange Crypto | ~~AES-256-GCM~~ (ARCHIVED) | `encrypted_api_key` |
| NIK Pemilih (Campaign) | AES-256-GCM | `encrypted_nik` |
| Refresh Token | SHA-256 hash | Redis + `refresh_tokens` |
| Password User | bcrypt (cost=12) | `password_hash` |

- Kunci enkripsi: `cfg.EncryptionKey` — **wajib** 32 byte
- Contoh enkripsi: `shared/sdk/encryption/encryption.go`

---

## 👥 Hirarki User & Akses (WCH Platform)
Sistem memiliki 3 level/jenis user, yaitu:
1. **Superadmin** (Akses penuh ke dashboard superadmin untuk manajemen tenant & voucher).
2. **Tenant Owner / Admin** (Pemilik UMKM/Campaign, memiliki kontrol penuh atas resource aplikasinya, menagih/membayar langganan, dan manajemen onboarding).
3. **Staff / Kasir / Relawan** (Anggota dari tenant yang hanya memiliki akses operasional sesuai role RBAC, tidak bisa mengakses menu billing/langganan, tidak bisa Add/Edit/Delete staff).

*Saran perbaikan:* 
- Pastikan saat user login, status `onboarding_completed` dan `plan` terbaca dari profil backend lalu disimpan ke `localStorage`. Redirect loop ke halaman payment/onboarding terjadi karena saat login di device baru, flag `onboarding_completed` di `localStorage` kosong, sehingga Router memaksa user aktif masuk ke halaman Onboarding (yang juga merupakan form pemilihan paket).
- Sembunyikan menu/akses yang tidak relevan bagi Staff (seperti billing atau upgrade paket).
- Sediakan endpoint khusus untuk `GET /me` guna mensinkronisasi `status` dan `role` saat reload.

---

## 🎫 Superadmin Voucher Management UI

Card "Voucher Billing" di `frontend/umkm-web/src/components/SuperAdminDashboard.vue` menyediakan UI untuk generate dan lihat voucher:

**Generate Voucher Modal:**
- Input: program name (opsional), paket (lite/pro/ultimate), jumlah (1-1000), masa aktif (hari), max uses (opsional)
- Tombol "Generate Sekarang" → `POST /api/superadmin/billing/vouchers/generate`
- Setelah generate: tampilkan semua kode voucher + tombol **Download CSV** + tombol **Copy** per kode
- Tombol "Generate Lagi" untuk reset dan generate batch baru

**Voucher List Modal:**
- Tombol "Lihat Daftar" di header card → buka modal daftar voucher
- Filter: used/unused, paket
- Tabel: #, Kode, Program, Paket, Status, Digunakan Oleh, Tanggal
- Tombol Refresh untuk reload data

**API Methods (`superadminApi.ts`):**
- `generateVouchers({ plan_id, validity_days, quantity, program_name?, max_uses? })` → returns `{plan_id, validity_days, count, codes: [{code, days}]}`
- `listVouchers({ plan_id?, used?, limit? })` → returns `{total, used, unused, codes: [...]}`
- `deleteVoucher(id)` → `DELETE /api/superadmin/billing/vouchers?id=...` — hanya unused vouchers

**Delete Voucher (Button Hapus):**
- Muncul di tabel Daftar Voucher untuk setiap voucher yang belum terpakai (`is_redeemed = false`)
- Konfirmasi dialog sebelum hapus, lalu `DELETE /admin/vouchers?id=<id>` di billing-service
- Backend validasi: hanya superadmin, hanya unused vouchers (redeemed → 400 error), 404 jika tidak ditemukan

**Routing:** `/api/superadmin/billing/` → strips prefix → proxies ke billing-service:8003 `/admin`

**⚠️ Bug Fix — Voucher Redemption (v2):**
- `handleAdminGenerateVouchers` menyimpan `validity_days` ke kolom `voucher_codes.validity_days` saat INSERT (sebelumnya hanya di-echo di response, tidak disimpan)
- `handleRedeemVoucher` membaca `validity_days` langsung dari row `voucher_codes` via JOIN (sebelumnya membaca `duration_months` dari `voucher_programs` yang selalu 0 untuk `bonus_months` → menghasilkan 0 hari aktif)
- `activateSubscription` menggunakan `validity_days` as-is (bukan `duration_months*30`)

---

## 🚫 Larangan Keras

1. ❌ **JANGAN** commit file `.env` ke git
2. ❌ **JANGAN** hardcode API key, password, atau secret di kode
3. ❌ **JANGAN** gunakan `float64` untuk kalkulasi uang
4. ❌ **JANGAN** panggil MiniMax/OpenAI/Gemini langsung dari `apps/`
5. ❌ **JANGAN** hapus/modifikasi `shared/sdk/config/config.go` tanpa diskusi
6. ❌ **JANGAN** simpan data PII (NIK, password) tanpa enkripsi/hashing
7. ❌ **JANGAN** gunakan string concatenation di SQL query
8. ❌ **JANGAN** `panic()` di luar fungsi `main()`
9. ❌ **JANGAN** implement fitur yang belum di-approve di FEATURE_MAP.md

---

## 🧪 Testing Convention

**Setiap perubahan kode WAJIB disertai unit test minimal:**

1. **Validation tests** (pure functions, no DB) — cek error message & status code
2. **JSON binding tests** — verifikasi field mapping benar
3. **Enum tests** — verifikasi valid + invalid values dari schema real

**Pattern test yang dipakai project:**
- `httptest.NewRecorder()` + `http.HandlerFunc(handler)` untuk handler test
- Pure function test (validation, parsing) tanpa DB stub
- Mock data sesuai real schema (UUIDs, VARCHAR limits, enum values dari migration)

**Test files per-fitur:**
- F046 Coordinator: `apps/campaign/api/handlers/coordinator_test.go`
- F047 Clinic modules: `apps/umkm/accounting/clinic_test.go`
- F048 Chatbot config: `apps/umkm/accounting/chatbot_config_test.go`
- F048 WA Provider routing: `services/wa-gateway/wa_gateway_test.go`

**F048 AC-8 — Chatbot Activation Guard (integration-style):**
- `apps/umkm/accounting/chatbot_config_test.go` (TestValidateWAConnectionForChatbot_*)
- Skip otomatis jika `DB == nil` (tidak ada DB pool di test env)
- Untuk CI lengkap: jalankan dengan `DATABASE_URL` env pointing ke test DB

**Contoh mock data real-schema:**
```go
const mockTenantID = "11111111-1111-1111-1111-111111111111" // UUID
const mockCampaignID = "22222222-2222-2222-2222-222222222222"
// business_types.id VARCHAR(50): "umum", "warung", "clinic", ...
// coordinator_level: "korprov", "korKab", "korKec", "korKades", "saksi_tps"
// wa_provider_preference: "auto", "whatsmeow", "cloud_api"
```

---

## 🛡️ Chatbot Activation Guard (F048 AC-8)

Saat user klik tombol **"Simpan & Aktifkan"** di `/chatbot-config`, BE WAJIB cek apakah toko sudah punya koneksi WA aktif **sebelum** `is_active = true` disimpan.

**Lokasi kode:** `apps/umkm/accounting/main.go` → `validateWAConnectionForChatbot()`

**Logika validasi (OR, dua-duanya harus return true):**
1. Whatsmeow device connected:
   ```sql
   SELECT 1 FROM wa_sessions 
   WHERE tenant_id = $1 AND status = 'connected'
   ```
2. Meta Cloud API active:
   ```sql
   SELECT 1 FROM wa_cloud_api_credentials
   WHERE tenant_id = $1 AND is_active = true
   ```

**Response jika tidak ada koneksi:**
```http
HTTP/1.1 400 Bad Request
{
  "success": false,
  "message": "Nomor WhatsApp (CS) belum terhubung. Silakan hubungkan WhatsApp terlebih dahulu sebelum mengaktifkan Chatbot."
}
```

**Flow UI:**
```
User tekan "Simpan & Aktifkan"
   ↓
Frontend PUT /chatbot/config { is_active: true, ... }
   ↓
BE validateChatbotConfig(&merged)   ← cek schema fields
   ↓
BE validateWAConnectionForChatbot() ← cek WA connected?
   ↓
   ├─ Ya (salah satu) → UPDATE tenant_chatbot_configs SET is_active = true
   └─ Tidak → 400 "Nomor WhatsApp (CS) belum terhubung..."
```

**Run test pattern:**
```bash
# Quick single test
go test ./apps/umkm/accounting/ -run "TestValidateChatbotConfig_WAProviderPreference" -v

# All tests in package
go test ./apps/umkm/accounting/ -v

# Full check (lint + build + test) - same as `make check`
export PATH=/usr/local/go/bin:$PATH
go vet ./... && go test ./... -count=1
```

## 📋 Perintah Cepat

```bash
# Jalankan semua service di background
make start-all
# Log otomatis ke logs/*.log, PID ke run/*.pid

# Matikan semua service
make stop-all

# Cek status semua port
make status

# Jalankan test per-service
export PATH=/usr/local/go/bin:$PATH
go test ./services/wa-gateway/ -v -run "TestResolveProviderPreference|TestIsTransactional"
go test ./apps/umkm/accounting/ -v -run "TestValidateChatbotConfig|TestRequireClinicType"
go test ./apps/campaign/api/handlers/ -v -run "TestHandleAssignCoordinator|TestCoordinatorLevel"

# Full check (lint + build + test)
make check

# Jalankan service individual
make run-auth          # Auth Service (port 8001)
make run-ai            # AI Gateway (port 8002)
make run-accounting    # UMKM Accounting (port 8201)
make run-chatbot       # UMKM Chatbot (port 8202)
make run-campaign      # Campaign API (port 9002)
make run-frontend      # Semua frontend

# Build binary ke bin/ (BUKAN ke root!)
make build-all         # Semua service → bin/<service>
make build             # Compile check saja (go build ./...)

# Pantau log
make logs-auth         # tail -f logs/auth.log
make logs-accounting   # tail -f logs/accounting.log
make logs-all          # tail -f logs/*.log

# Testing & Quality
go test ./...          # Test semua package
make test-umkm         # Test seluruh layanan UMKM saja
go build ./...         # Compile check
go vet ./...           # Linting
go mod tidy            # Bersihkan dependencies
make check             # tidy + vet + build + test sekaligus

# Cleanup
make clean-logs        # Hapus semua file di logs/
make clean-build       # Hapus semua binary di bin/
make clean             # clean-logs + clean-build

# Frontend UMKM
cd frontend/umkm-web && npm run dev
```

---

## 📖 Dokumentasi Referensi

| Dokumen | Tujuan |
|:--------|:-------|
| **[docs/FEATURE_MAP.md](docs/FEATURE_MAP.md)** | **SPEC & feature registry — WAJIB baca sebelum coding** |
| **[docs/WA_PROVIDER_GUIDE.md](docs/WA_PROVIDER_GUIDE.md)** | **WA provider operations — QR scan, send, reconnect, anti-ban** |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Panduan lengkap menambah & mengubah fitur |
| [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) | Skenario-skenario pengembangan + contoh kode |
| [docs/MIGRATION_REGISTRY.md](docs/MIGRATION_REGISTRY.md) | Daftar semua migrasi database |
| [docs/master_plan.md](docs/master_plan.md) | Rencana bisnis & roadmap produk |
| [docs/deployment.md](docs/deployment.md) | Panduan deploy ke production |

---

## 💳 Xendit Per-Tenant Architecture (F034 Extension)

Setiap tenant memiliki kredensial Xendit sendiri di tabel `tenants`:

| Kolom | Tipe | Fungsi |
|:------|:-----|:-------|
| `xendit_api_key` | VARCHAR(255) | API key akun Xendit tenant |
| `xendit_merchant_id` | VARCHAR(255) | Merchant account ID (pembayaran langsung ke akun tenant) |
| `xendit_webhook_token` | VARCHAR(255) | Token verifikasi webhook per-tenant |

**Architecture:**
```
Subscription/Topup Request
  → getTenantXenditClient(tenantID)  ← cache 5-min TTL (sync.RWMutex)
  → baca xendit_api_key dari DB
  → CreateInvoice di merchantID tenant
  → Dana masuk ke bank account tenant ✅

Payment Webhook Callback
  → Extract tenantID dari external_id
     INV-{uuid}|{tenantID}  (invoice)
     {uuid}-wallet-topup-{tenantID}  (topup)
  → Verify webhook token: prioritas 1 = DB per-tenant, fallback = env XENDIT_WEBHOOK_TOKEN
```

**Client Caching:** 5-minute TTL — menghindari DB hit per request, mendukung key rotation.

**Backward Compat:** Webhook fallback ke env var `XENDIT_WEBHOOK_TOKEN` untuk tenant lama.

---

## 🔧 Known Issues & Fixes (2026-06-14 Session)

### Fix 1: journal_entries.metadata Column Missing
- **Symptom:** `GET /api/umkm/transactions` → 500 "DB error"
- **Root cause:** `handleGetTransactions` query SELECT `e.metadata` tapi kolom tidak ada
- **Fix:** Migration 000033 menambah `metadata JSONB` column + GIN index di `journal_entries`
- **Files:** `shared/migrations/000033_journal_entries_metadata.{up,down}.sql`

### Fix 2: Settings Page 500 — wa_provider Column
- **Symptom:** `GET /api/umkm/settings` → 500 "DB error"
- **Root cause:** Query SELECT `wa_provider` dari `tenants` table, tapi kolom tidak ada
- **Fix:** Hapus semua referensi `wa_provider` dari SELECT, Go variable, response JSON, dan PUT UPDATE
- **File:** `apps/umkm/accounting/main.go` — `handleSettings` function

### Fix 3: Nginx Drops Authorization Header
- **Symptom:** Frontend `/settings` → 401 "Missing Authorization header"
- **Root cause:** `nginx.conf` tidak explicit-pass `Authorization` dan `X-Tenant-ID` headers
- **Fix:** Tambah `proxy_set_header Authorization $http_authorization` dan `proxy_set_header X-Tenant-ID $http_x_tenant_id` di semua proxy location blocks
- **File:** `frontend/umkm-web/nginx.conf`

### Fix 4: WA Gateway 404 — StripPrefix Mismatch
- **Symptom:** `POST /api/wa/status` → 404
- **Root cause:** API Gateway pakai `http.StripPrefix("/api/wa", ...)` tapi wa-gateway handler registered di `/api/wa/status` (full path)
- **Fix:** Hapus `http.StripPrefix` dari proxy `/api/wa/` agar path diteruskan utuh
- **File:** `services/api-gateway/main.go` (line 92)

### Fix 5: Feature Gating — Plan Cache Not Populated
- **Symptom:** Tenant dengan plan "lite" dapat 403 "Fitur Chatbot memerlukan paket Lite"
- **Root cause:** `GetTenantPlan()` di `quota.go` baca Redis key `tenant:plan:{id}`, tapi login handler TIDAK populate cache. Redis miss → fallback ke tier tanpa akses (`inactive` setelah 2026-06-14) → `HasChatbot: false`
- **Fix:**
  - Tambah `"superadmin"` tier di `Plans` map (`shared/sdk/auth/quota.go`)
  - Auth-service login handlers (`handleLogin`, `handlePhoneLogin`, `handleSuperAdminLogin`) sekarang set Redis key setelah login sukses
- **Files:** `shared/sdk/auth/quota.go`, `services/auth-service/main.go`

### Fix 6: FAQ Edit Button
- **Symptom:** User mau edit FAQ setelah AI generate — tidak ada tombol edit
- **Fix:** Tambah inline edit mode di Settings.vue FAQ section + PUT handler di backend
- **Files:** `frontend/umkm-web/src/components/Settings.vue`, `apps/umkm/accounting/main.go` (`handleFaqs` PUT)

### Architecture Note: Plan Redis Cache Dependency
- Auth-service login populate cache. Untuk existing tenant sebelum fix ini, set manual: `docker exec wch-redis redis-cli SET "tenant:plan:{id}" "{plan}"`
- `GetTenantPlan()` akan refactored untuk fallback ke DB di versi berikutnya
### Frontend CSS
- **WAJIB**: Edit `frontend/umkm-web/src/assets/main.css` untuk merubah gaya CSS utama (termasuk .modal-overlay, .modal-content). File `style.css` hanya legacy/secondary.
