# Local Development Setup — WCH Platform

Panduan ini untuk menjalankan WCH Platform di lingkungan development lokal.

## Prerequisites

1. **Docker & Docker Compose** — untuk PostgreSQL, Redis, dan infrastructure services
2. **Go 1.21+** — untuk compile Go services
3. **Node.js 18+** — untuk frontend development
4. **Air** (optional, untuk hot-reload) — `go install github.com/air-verse/air@latest`

## Quick Start

### 1. Clone & Setup Environment

```bash
cd /home/syahril/dev/core_project
cp .env.example .env
```

### 2. Edit `.env` — Minimal Configuration

Buka `.env` dan isi minimal berikut:

```bash
# Core Security (WAJIB diisi)
JWT_SECRET=dev_secret_32_characters_long_key
ENCRYPTION_KEY=dev_encryption_32_characters_key

# Database (sudah sesuai docker-compose, via pgbouncer port 10433)
DB_HOST=127.0.0.1
DB_PORT=10433
DB_USER=wch_admin
DB_PASSWORD=your_strong_password_here
DB_NAME=wch_platform

# Redis (sudah sesuai docker-compose, expose ke port 10631)
REDIS_HOST=127.0.0.1
REDIS_PORT=10631
REDIS_PASSWORD=your_redis_password_here

# AI/LLM (minimal satu provider)
MINIMAX_API_KEY=your_anthropic_api_key_here
MINIMAX_BASE_URL=https://api.anthropic.com/v1
MINIMAX_MODELS=claude-sonnet-4-5
```

### 3. Start Infrastructure (Docker)

```bash
# Start PostgreSQL + Redis saja
docker compose up -d postgres redis

# Verify
docker compose ps postgres redis
```

Output seharusnya:
```
NAME           STATUS
wch-postgres   Up (healthy)
wch-redis      Up (healthy)
```

### 4. Start Go Services (Native Hot-Reload)

**Option A: Start semua service sekaligus**
```bash
make dev-all
# atau
./scripts/dev-native.sh
```

**Option B: Start service individual**
```bash
make dev-auth          # Auth Service (port 8001)
make dev-gateway       # API Gateway (port 8000)
make dev-accounting    # UMKM Accounting (port 8201)
make dev-chatbot       # UMKM Chatbot (port 8202)
```

### 5. Start Frontend (Vue + Vite)

```bash
# UMKM Web
cd frontend/umkm-web
npm install
npm run dev          # port 3201

# Campaign Web (optional)
cd frontend/campaign-web
npm install
npm run dev          # port 3301

# Superadmin Web (optional)
cd frontend/superadmin-web
npm install
npm run dev          # port 3401
```

## Verify Setup

### Health Check Endpoints

```bash
# API Gateway
curl http://localhost:8000/livez
# Expected: {"status":"healthy","timestamp":"..."}

# Auth Service
curl http://localhost:8001/health
# Expected: {"status":"ok"}

# UMKM Accounting
curl http://localhost:8201/health
# Expected: {"status":"ok"}
```

### Frontend URLs

- UMKM: http://localhost:3201
- Campaign: http://localhost:3301
- Superadmin: http://localhost:3401

## Common Issues & Fixes

### Issue 1: Redis "Bad file format" / Service Restarting

**Symptom:** Services terus restart dengan error `unable to connect to redis`

**Root Cause:** Redis AOF (Append Only File) corrupted

**Fix:**
```bash
# Stop all
docker compose down

# Remove corrupted Redis volume
docker volume rm core_project_redisdata

# Restart
docker compose up -d postgres redis
```

### Issue 2: Port Already in Use

**Symptom:** `bind: address already in use`

**Fix:**
```bash
# Cek siapa yang pakai port (contoh: 8001)
lsof -ti:8001

# Kill process
kill $(lsof -ti:8001)

# Atau gunakan cleanup script
./scripts/dev-native.sh --stop
```

### Issue 3: Database Connection Failed

**Symptom:** `failed to connect to postgres`

**Fix:**
```bash
# Cek container status
docker compose ps postgres

# Cek logs
docker logs wch-postgres

# Pastikan DB_HOST, DB_PORT, DB_PASSWORD di .env match dengan docker-compose.yml
```

### Issue 4: Migration Failed

**Symptom:** `migration xxx failed`

**Fix:**
```bash
# Cek tabel schema_migrations
docker exec -it wch-postgres psql -U wch_admin -d wch_platform -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;"

# Jika ada migration stuck, reset manual (DANGER!)
# docker exec -it wch-postgres psql -U wch_admin -d wch_platform -c "DELETE FROM schema_migrations WHERE version = 'XXX';"
```

## Development Workflow

### 1. Code Changes dengan Hot-Reload

Services yang jalan via `make dev-*` atau `./scripts/dev-native.sh` otomatis **hot-reload** saat ada perubahan kode Go.

**Log files:**
```bash
tail -f logs/auth.log
tail -f logs/accounting.log
tail -f logs/api-gateway.log
```

### 2. Testing

```bash
# Test satu service
go test ./services/auth-service/ -v

# Test semua
go test ./... -v

# Dengan coverage
go test ./... -cover
```

### 3. Build Binary

```bash
# Build semua service ke bin/
make build-all

# Jalankan binary langsung
./bin/auth-service
```

### 4. Stop Services

```bash
# Stop native services
./scripts/dev-native.sh --stop

# Stop Docker
docker compose down

# Stop + remove volumes (DANGER: data hilang!)
docker compose down -v
```

## Architecture — Native vs Docker

### Recommended Setup (Hybrid)

| Component | Runtime | Reason |
|:----------|:--------|:-------|
| PostgreSQL | Docker | Persistent data, pgvector extension |
| Redis | Docker | Shared cache |
| Go Services | Native (air hot-reload) | Fast iteration, debugging |
| Frontend | Native (Vite dev server) | HMR, instant feedback |
| N8N, Grafana, Chatwoot | Docker | Complex setup, stable |

### Full Docker (Production-like)

```bash
# Build images
docker compose build

# Start all
docker compose up -d

# Access
# API Gateway: http://localhost:8010 (note: port 8010, not 8000)
# Frontend UMKM: http://localhost:3201
```

## Environment Variables Reference

Lihat file `.env.example` untuk daftar lengkap. Variabel **WAJIB** untuk dev lokal:

- `JWT_SECRET` — 32 karakter minimum
- `ENCRYPTION_KEY` — 32 karakter minimum
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `MINIMAX_API_KEY` — untuk AI Chatbot (Anthropic Claude)

## Next Steps

- Baca [CONTRIBUTING.md](../CONTRIBUTING.md) untuk workflow development
- Baca [FEATURE_MAP.md](FEATURE_MAP.md) untuk fitur yang tersedia
- Baca [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) untuk contoh skenario development
