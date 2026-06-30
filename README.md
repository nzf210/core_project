# WCH Platform

**Multi-produk SaaS berbasis Go (Golang)** — satu monorepo untuk semua.

## 🏗️ Produk

| Produk | Deskripsi | Port |
|:-------|:----------|:-----|
| **UMKM** | AI Agent: Double-Entry Accounting, Chatbot WA, POS | 8201, 8202, 9001 |
| **Crypto** | ~~Trading Bot: DCA, Grid, Signal (Binance API)~~ (ARCHIVED) | 8101 |
| **Campaign** | Manajemen Pemilu: Relawan, Real Count, AI Sentiment | 9002 |

## ⚡ Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Node.js 18+ (untuk frontend)

### 1. Setup Environment

```bash
cp .env.example .env
# Edit .env dengan kredensial Anda
```

### 2. Jalankan Infrastruktur (DB + Redis)

```bash
docker compose up -d postgres redis
```

### 3. Jalankan Service

```bash
# Semua service sekaligus (background):
make start-all

# Atau service individual:
make run-auth         # Auth Service (port 8001)
make run-accounting   # UMKM Accounting (port 8201)
make run-campaign     # Campaign API (port 9002)

# Cek status:
make status
```

### 4. Frontend

```bash
cd frontend/umkm-web && npm install && npm run dev   # port 3201
```

## 📚 Dokumentasi

| Dokumen | Isi |
|:--------|:----|
| [CONTRIBUTING.md](CONTRIBUTING.md) | **Panduan utama** — cara tambah/ubah fitur |
| [CLAUDE.md](CLAUDE.md) | AI assistant memory — konvensi & arsitektur |
| [docs/FEATURE_MAP.md](docs/FEATURE_MAP.md) | Cheat sheet: di mana menulis kode |
| [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) | Panduan lengkap per skenario |
| [docs/MIGRATION_REGISTRY.md](docs/MIGRATION_REGISTRY.md) | Registry semua migrasi DB |
| [docs/master_plan.md](docs/master_plan.md) | Rencana bisnis & roadmap |

## 🏗️ Arsitektur

```
core_project/
├── apps/          ← Produk bisnis (umkm, campaign)
├── services/      ← Shared services (auth, ai-gateway, billing, dll)
├── shared/        ← SDK bersama (config, db, cache, migrations)
├── frontend/      ← Vue 3 apps (umkm-web, campaign-web)
├── infra/         ← Docker, n8n, deploy scripts
└── docs/          ← Dokumentasi
```

## 🧪 Testing

```bash
make test          # Semua unit test
make build         # Compile check
make vet           # Static analysis
make check         # Semua quality checks sekaligus
```

## 🔧 Tech Stack

- **Backend:** Go 1.21+ / `net/http` standar
- **Database:** PostgreSQL 16 via `pgx/v5`
- **Cache:** Redis via `go-redis/v9`
- **Auth:** JWT via `golang-jwt/jwt/v5`
- **AI/LLM:** MiniMax M2.7 (via internal AI Gateway)
- **WhatsApp:** whatsmeow library
- **Billing:** Xendit
- **Frontend:** Vue 3 + TypeScript + Vite
- **CI/CD:** GitHub Actions

<!-- Run new Config -->
Cara pakai:

# 1. Pastikan Docker postgres+redis jalan
docker compose up -d postgres redis

# 2. Jalankan semua dengan hot-reload
./scripts/dev-native.sh

# Atau per service:
make dev-auth      # Auth Service (8001)
make dev-gateway   # API Gateway (8000)
make dev-accounting # UMKM Accounting (8201)

# Stop:
./scripts/dev-native.sh --stop

Perubahan yang dirasakan:
- Go service auto-rebuild saat file .go di-save (tanpa restart manual)
- Frontend Vue sudah natively hot-reload via Vite
- Production = Docker semua (env sama, tinggal docker compose up)