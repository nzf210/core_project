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
```

### Checklist Sebelum Coding:

1. **Cek FEATURE_MAP.md** — Apakah fitur sudah ada di registry?
2. **Cek SPEC status** — Sudah ✅ Approved?
   - Jika ⏳ Draft: Tanya USER dulu, jangan implement
   - Jika ❌ Rejected: Jangan implement
3. **Ambiguitas?** — Tanya USER clarification dulu
4. **Implementasi selesai?** — Update `Implementation` status di FEATURE_MAP.md

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
│   │   ├── business/           ← Business management API (Port 9001)
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
│   ├── subscription-worker/    ← Freeze expired tenants (Port 8006)
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
│   └── migrations/             ← Database SQL migrations (000001 — 000027)
│
├── frontend/
│   ├── umkm-web/               ← Vue 3 + Vite (Port 3201)
│   └── campaign-web/           ← Vue 3 + Vite (Port 3301)
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
| `8010` | API Gateway | `services/api-gateway` (Docker mapped port) |
| `8001` | Auth Service | `services/auth-service` |
| `8002` | AI Gateway | `services/ai-gateway` |
| `8013` | Billing Service | `services/billing-service` (Docker mapped port) |
| `8015` | Notification Service | `services/notification-service` (Docker mapped port) |
| `8016` | Subscription Worker | `services/subscription-worker` (Docker mapped port) |
| `8201` | UMKM Accounting | `apps/umkm/accounting` |
| `8212` | WA Gateway | `services/wa-gateway` (Docker mapped port) |
| `8202` | UMKM Chatbot | `apps/umkm/chatbot` |
| `8213` | UMKM Automation | `apps/umkm/automation` (Docker mapped port) |
| `9001` | UMKM Business | `apps/umkm/business` |
| `9002` | Campaign API | `apps/campaign/api` |
| `3000` | Chatwoot (Self-hosted) | docker-compose (chatwoot) |
| `3201` | Frontend UMKM | `frontend/umkm-web` |
| `3301` | Frontend Campaign | `frontend/campaign-web` |
| `5433` | PostgreSQL + pgvector (Docker) | docker-compose |
| `5678` | N8N Main (Queue Mode) | docker-compose (n8n-main) |
| `6381` | Redis (Docker) | docker-compose |

> ⚠️ Jika berjalan tanpa Docker (native), WA Gateway dan UMKM Chatbot secara default sama-sama menggunakan port 8202. Saat berjalan dengan Docker, WA Gateway di-expose ke port host 8212.

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
| `freeze_reminder.json` | Cron | Reminder untuk expired subscriptions |
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

## 📋 Perintah Cepat

```bash
# Jalankan semua service di background
make start-all
# Log otomatis ke logs/*.log, PID ke run/*.pid

# Matikan semua service
make stop-all

# Cek status semua port
make status

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
| [CONTRIBUTING.md](CONTRIBUTING.md) | Panduan lengkap menambah & mengubah fitur |
| [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) | Skenario-skenario pengembangan + contoh kode |
| [docs/MIGRATION_REGISTRY.md](docs/MIGRATION_REGISTRY.md) | Daftar semua migrasi database |
| [docs/master_plan.md](docs/master_plan.md) | Rencana bisnis & roadmap produk |
| [docs/deployment.md](docs/deployment.md) | Panduan deploy ke production |
