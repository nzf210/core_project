# WCH Platform — Claude Project Memory

> Dokumen ini adalah **memori permanen** Claude Code Agent untuk monorepo WCH Platform.
> Dibaca otomatis di setiap sesi. Jaga agar tetap singkat dan relevan.

---

## 🎯 Identitas Proyek

Platform SaaS multi-produk berbasis **Go (Golang)** dengan 3 produk utama:

1. **`apps/crypto`** — SaaS Crypto Trading Bot (DCA, Grid, Signal Bot)
2. **`apps/umkm`** — AI Agent UMKM & Pembukuan (Chatbot WA + Double-Entry Accounting)
3. **`apps/campaign`** — Manajemen Pemenangan Pemilu (Relawan, Real Count, AI Sentiment)

Semua produk berbagi layanan di `services/` (auth, billing, ai-gateway, notification, tenant, workflow).

---

## 🏗️ Peta Arsitektur & Direktori

```
core_project/          ← Root monorepo (satu go.mod)
├── apps/
│   ├── crypto/        ← Trading bot app (api/, domain/, worker/)
│   ├── umkm/          ← UMKM app (api/, chatbot/, accounting/, automation/)
│   └── campaign/      ← Campaign app (api/, volunteer/, analytics/, premium/)
├── services/
│   ├── auth-service/  ← JWT, login, register, RBAC (Port 8000)
│   ├── ai-gateway/    ← MiniMax M2.7 proxy + cache (Port 8003)
│   ├── billing-service/   ← Xendit subscription (Port 8002)
│   ├── notification-service/ ← WA Fonnte + Email SMTP (Port 8004)
│   ├── tenant-service/    ← Multi-tenant management (Port 8001)
│   └── workflow-service/  ← n8n integration (Port 8005)
├── shared/
│   └── sdk/config/config.go  ← SATU-SATUNYA cara baca config
├── frontend/          ← Next.js apps (crypto-web, umkm-web, campaign-web, admin-web)
├── infra/             ← Docker, Nginx, deploy scripts
├── docs/              ← Semua dokumentasi proyek
├── CLAUDE.md          ← File ini
└── .env               ← Environment variables (JANGAN commit ke git)
```

---

## ⚙️ Konvensi Kode — WAJIB DIIKUTI

### Go Backend
- **Framework HTTP**: `net/http` standar atau `github.com/gin-gonic/gin`
- **Database driver**: `github.com/jackc/pgx/v5` — **JANGAN** pakai GORM
- **JWT**: `github.com/golang-jwt/jwt/v5`
- **Password hashing**: `golang.org/x/crypto/bcrypt`
- **Error handling**: Kembalikan `error` eksplisit — **JANGAN** `panic()` di luar `main()`
- **Logging**: `log/slog` (structured) — **JANGAN** `fmt.Println` atau `log.Println` biasa
- **Config**: SELALU gunakan `config.LoadConfig()` dari `shared/sdk/config/config.go`
- **Uang/Harga**: Gunakan `int64` dalam satuan **sen (1 rupiah = 100 sen)** — **JANGAN** `float64`
- **UUID**: Gunakan `github.com/google/uuid` untuk generate ID

### Testing
- Setiap handler/service WAJIB ada unit test di file `*_test.go`
- Perintah test: `go test ./...` (jalankan setelah setiap perubahan besar)
- Gunakan `httptest` package untuk test HTTP handler

### Database
- Migration files di: `shared/migrations/[timestamp]_[nama].[up|down].sql`
- Semua tabel: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Kolom timestamp wajib: `created_at TIMESTAMPTZ DEFAULT NOW()`, `updated_at TIMESTAMPTZ`
- Semua query: gunakan parameterized statements via pgx — **JANGAN** string concatenation

---

## 🤖 MiniMax M2.7 — LLM Utama Platform

**SELALU** panggil LLM melalui `services/ai-gateway` — **JANGAN** panggil API eksternal langsung dari `apps/`.

```go
// Cara BENAR memanggil AI dari apps/
resp, err := http.Post("http://localhost:8003/v1/chat", "application/json", payload)

// Cara SALAH — JANGAN lakukan ini dari apps/
client := openai.NewClient(cfg.AI.MiniMaxAPIKey) // ❌
```

Konfigurasi di `services/ai-gateway`:
- **Base URL**: `https://api.minimax.io/v1`
- **Model**: `MiniMax-M2.7`
- **Go client**: `github.com/sashabaranov/go-openai` dengan `config.BaseURL` override
- **Semantic cache**: Selalu cek Redis sebelum call API — key: `ai:cache:{sha256(prompt)}`

---

## 🔒 Aturan Keamanan — KRITIS

| Data Sensitif | Metode Enkripsi | Lokasi |
|:---|:---|:---|
| API Key Exchange Crypto | AES-256-GCM | DB kolom `encrypted_api_key` |
| NIK Pemilih (Campaign) | AES-256-GCM | DB kolom `encrypted_nik` |
| Refresh Token | SHA-256 hash | Redis + DB |
| Password User | bcrypt (cost=12) | DB kolom `password_hash` |

**Kunci enkripsi**: Ambil dari `cfg.EncryptionKey` — panjang wajib 32 byte.

---

## 🚫 Larangan Keras (JANGAN LANGGAR)

1. ❌ **JANGAN** commit file `.env` ke git
2. ❌ **JANGAN** hardcode API key, password, atau secret di kode
3. ❌ **JANGAN** panggil exchange API langsung dari HTTP handler — gunakan channel/worker
4. ❌ **JANGAN** gunakan `float64` untuk kalkulasi uang
5. ❌ **JANGAN** hapus atau modifikasi `shared/sdk/config/config.go` tanpa konfirmasi
6. ❌ **JANGAN** panggil MiniMax API langsung dari `apps/` — selalu lewat `ai-gateway`
7. ❌ **JANGAN** simpan data PII pemilih tanpa enkripsi

---

## 📋 Perintah Penting

```bash
# Infrastruktur lokal
docker compose -f infra/docker/docker-compose.yml up -d

# Jalankan service individual
go run ./services/auth-service/main.go
go run ./services/ai-gateway/main.go

# Test & build
go test ./...
go build ./...
go vet ./...
go mod tidy

# Jalankan migrasi (setelah migration tooling tersedia)
# go run ./shared/tools/migrate/main.go up
```

---

## 📖 Referensi Dokumentasi

- [Master Plan Produk](docs/master_plan.md)
- [Master Plan AI Development](docs/master_plan_ai_dev.md)
- [Roadmap](docs/roadmap.md)
- [Monetisasi](docs/monetization.md)
- [Deployment](docs/deployment.md)
