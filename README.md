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
├── infra/         ← Docker, n8n, observability (Grafana/Prometheus/Loki)
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
- **Observability:** Prometheus + Grafana + Loki
- **CI/CD:** GitHub Actions

## 🚀 Deployment

### Development (Hot-Reload)

```bash
# 1. Infrastruktur + Observability
docker compose up -d postgres redis prometheus grafana loki

# 2. Jalankan semua service dengan hot-reload
./scripts/dev-native.sh

# Atau per service:
make dev-auth          # Auth Service (8001)
make dev-gateway       # API Gateway (8000)
make dev-accounting    # UMKM Accounting (8201)

# Stop semua:
./scripts/dev-native.sh --stop
```

**Benefits:**
- Go services auto-rebuild saat file `.go` disave (via `air`)
- Frontend Vue hot-reload native via Vite
- Grafana dashboard: http://localhost:3001 (admin/admin123)
- Prometheus metrics: http://localhost:9091

### Production

```bash
# Deploy dengan compose override
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Grafana accessible via reverse proxy (nginx)
# Internal services + Prometheus tidak expose port ke host
```

**Production differences:**
- Prometheus retention: 90d (vs 30d dev)
- No port exposure (internal network only)
- Grafana `GF_SERVER_ROOT_URL` configured for reverse proxy