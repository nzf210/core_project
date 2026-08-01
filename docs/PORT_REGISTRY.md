# WCH Platform — Port Registry

Dokumen ini adalah **sumber kebenaran tunggal** untuk semua port di WCH Platform.
Setiap perubahan port HARUS diupdate di sini terlebih dahulu.

---

## Skema Penomoran Port

```
X  Y  Z Z
│  │  └─── Nomor service (01–99)
│  └────── Kategori
│            0 = Infrastructure (DB, Cache)
│            1 = Backend Go service
│            2 = Frontend (Vue/Nginx)
│            3 = Observability & Tools (Grafana, N8N, Chatwoot)
└────────── Environment
             1 = DEV   (lokal developer)
             2 = STG   (staging server)
             3 = PROD  (production — sebagian besar internal only, expose via Nginx)
```

**Contoh baca port:**
- `11001` → Env **1** (dev), Kategori **1** (BE), Service **01** (Auth Service)
- `22003` → Env **2** (staging), Kategori **2** (FE), Service **03** (Campaign Web)
- `13001` → Env **1** (dev), Kategori **3** (tools), Service **01** (Grafana)

---

## Tabel Port Lengkap

### Infrastructure (Kategori 0)

| Service | Internal Port | DEV | STG | PROD | Catatan |
|:--------|:-------------|:----|:----|:-----|:--------|
| PostgreSQL (langsung) | 5432 | — | — | — | Internal only, akses via pgbouncer |
| pgbouncer | 6432 | `10433` | `20433` | internal | Connection pooler untuk PostgreSQL |
| Redis | 6379 | `10631` | `20631` | internal | Cache + queue |
| Chatwoot Redis | 6379 | — | — | — | Dedicated Redis untuk Chatwoot, tidak expose |

> **PROD note:** PostgreSQL dan Redis TIDAK expose port ke host. Akses hanya via Docker internal network.

---

### Backend Go Services (Kategori 1)

Semua BE service berkomunikasi via **Docker internal network** — tidak diexpose ke host di environment mana pun.
Akses dari luar selalu melalui `api-gateway` (port `8000` internal).

| Service | Internal Port | DEV | STG | PROD | Direktori |
|:--------|:-------------|:----|:----|:-----|:----------|
| API Gateway | 8000 | internal | `21000` | `127.0.0.1:8000` | `services/api-gateway` |
| Auth Service | 8001 | internal | internal | internal | `services/auth-service` |
| AI Gateway | 8002 | internal | internal | internal | `services/ai-gateway` |
| Billing Service | 8003 | internal | internal | internal | `services/billing-service` |
| Notification Service | 8005 | internal | internal | internal | `services/notification-service` |
| Subscription Worker | 8006 | internal | internal | internal | `services/subscription-worker` |
| UMKM Accounting | 8201 | internal | internal | internal | `apps/umkm/accounting` |
| WA Gateway | 8202 | internal | internal | internal | `services/wa-gateway` |
| WA Cloud API | 8210 | internal | internal | internal | `services/wa-cloud-api` |
| UMKM Business | 9001 | internal | internal | internal | `apps/umkm/business` |
| Campaign API | 9002 | internal | internal | internal | `apps/campaign/api` |

> **PROD exception:** `api-gateway` di-bind ke `127.0.0.1:8000` agar Nginx bisa proxy ke sana dari host.
> Semua service lain tidak perlu port karena komunikasi lewat Docker network (`wch-prod-network`).

> **No-port services** (tidak punya HTTP listener):
> - `umkm-chatbot` — consumer WA webhook
> - `umkm-automation` — background cron worker
> - `subscription-worker` — background cron worker

---

### Frontend (Kategori 2)

| Service | Internal Port | DEV | STG | PROD | Direktori |
|:--------|:-------------|:----|:----|:-----|:----------|
| UMKM Web | 80 | `12001` | `22001` | `127.0.0.1:8101` → Nginx 443 | `frontend/umkm-web` |
| Superadmin Web | 80 | `12002` | `22002` | `127.0.0.1:8102` → Nginx 443 | `frontend/superadmin-web` |
| Campaign Web | 80 | `12003` | `22003` | `127.0.0.1:8103` → Nginx 443 | `frontend/campaign-web` |

> **DEV native** (non-Docker, via Vite dev server):
> - UMKM Web: `3201`
> - Superadmin Web: `3401`
> - Campaign Web: `3301`
> Vite port hanya untuk `make dev` native, bukan Docker.

---

### Observability & Tools (Kategori 3)

| Service | Internal Port | DEV | STG | PROD | Catatan |
|:--------|:-------------|:----|:----|:-----|:--------|
| Grafana | 3000 | `13001` | `23001` | internal | Dashboard monitoring |
| Loki | 3100 | `13100` | `23100` | internal | Log aggregator |
| Prometheus | 9090 | `13090` | `23090` | internal | Metrics scraper |
| N8N Main | 5678 | `13678` | `23678` | internal | Workflow editor + webhook receiver |
| Chatwoot | 3000 | `13000` | `23000` | internal | Escalation CRM |
| Postgres Exporter | 9187 | `13187` | `23187` | internal | Metrics untuk Prometheus |
| Redis Exporter | 9121 | `13121` | `23121` | internal | Metrics untuk Prometheus |

> **N8N Worker** tidak perlu port — hanya consume jobs dari Redis Bull queue.
> **Promtail** tidak perlu port — hanya push logs ke Loki.

---

## Arsitektur Akses per Environment

### DEV (lokal developer)

```
Developer Browser
      │
      ├── http://localhost:8000   → API Gateway (native dev, Go binary langsung)
      │
      ├── http://localhost:12001  → UMKM Frontend     (Docker Nginx)
      ├── http://localhost:12002  → Superadmin Frontend (Docker Nginx)
      ├── http://localhost:12003  → Campaign Frontend  (Docker Nginx)
      │
      ├── http://localhost:13001  → Grafana
      ├── http://localhost:13678  → N8N
      └── http://localhost:13000  → Chatwoot

[make dev-all — native hot-reload, tanpa Docker FE]
      ├── localhost:3201  → UMKM Web (Vite)
      ├── localhost:3401  → Superadmin Web (Vite)
      └── localhost:3301  → Campaign Web (Vite)
```

### STAGING

```
Developer / QA Browser
      │
      ├── http://staging-server:21000  → API Gateway (untuk testing API langsung)
      ├── http://staging-server:22001  → UMKM Frontend
      ├── http://staging-server:22002  → Superadmin Frontend
      ├── http://staging-server:22003  → Campaign Frontend
      ├── http://staging-server:23001  → Grafana (monitoring)
      ├── http://staging-server:23678  → N8N
      └── http://staging-server:23000  → Chatwoot

[BE tidak diexpose kecuali api-gateway — akses service lain via gateway]
[Di VPS: semua port di atas di-proxy via Nginx dengan auth/SSL]
```

Dev dan Staging bisa berjalan di **server yang sama** tanpa konflik karena prefix berbeda (1xxxx vs 2xxxx).

### PRODUCTION

```
Internet
   │
   └── Nginx (80/443)
         ├── umkm.domain.com       → proxy → umkm-frontend:80
         ├── admin.domain.com      → proxy → superadmin-frontend:80
         ├── campaign.domain.com   → proxy → campaign-frontend:80
         ├── api.domain.com        → proxy → api-gateway:8000
         └── n8n.domain.com        → proxy → n8n-main:5678

[Semua port BE/infra = internal Docker network only, tidak expose ke host]
```

---

## Aturan Penggunaan

1. **Port kategori 0 (infra)** — hanya expose di DEV dan STG untuk keperluan debugging. PROD internal only.
2. **Port kategori 1 (BE)** — tidak diexpose kecuali `api-gateway`. DEV: internal (akses via Go binary langsung port 8000). STG: `api-gateway` expose di `21000` untuk testing. PROD: `api-gateway` bind ke `127.0.0.1:8000` untuk Nginx.
3. **Port kategori 2 (FE)** — selalu expose karena ini akses utama user.
4. **Port kategori 3 (tools)** — expose di DEV/STG untuk admin/ops. PROD akses via Nginx dengan auth.
5. **Jangan pernah expose PostgreSQL (5432) langsung** — selalu via pgbouncer.
6. **N8N Main wajib expose** — butuh browser untuk setup workflow dan webhook dari luar.

---

## File Terkait

| File | Relevansi |
|:-----|:----------|
| `docker-compose.yml` | Port mapping untuk Docker deployment |
| `docker-compose.staging.yml` | Override port untuk staging (prefix 2xxxx) |
| `scripts/dev-native.sh` | Port untuk native dev (Vite ports: 3201, 3301, 3401) |
| `frontend/umkm-web/vite.config.ts` | Vite dev server port 3201 |
| `frontend/campaign-web/vite.config.ts` | Vite dev server port 3301 |
| `frontend/superadmin-web/vite.config.ts` | Vite dev server port 3401 |
| `CLAUDE.md` | Port registry ringkas (sinkronkan saat ada perubahan) |

---

*Last updated: 2026-08-01*
