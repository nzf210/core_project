# Grafana Production-Ready Monitoring — Design Spec

**Date:** 2026-07-01
**Status:** Draft — Pending User Approval
**Related Feature:** F058 (upgrade Level 1 → Level 3) + new F067
**Author:** brainstorming session

---

## 🎯 Objectives

Membawa stack observability WCH Platform dari state "berjalan tapi belum production-grade" menjadi **production-ready** — sehingga untuk deploy ke production nantinya cukup ganti ENV dan compose file, tanpa perubahan kode.

**Tujuan eksplisit:**
1. Backend services expose `/metrics` dalam format Prometheus yang valid (bukan health-check agregasi).
2. Prometheus aktif scrape semua service + exporter DB/Redis.
3. Grafana auto-provision 8 dashboard + 2 datasource (zero manual setup).
4. Pemisahan dev vs prod via compose override (port exposure, retention, reverse-proxy).
5. ENV terdokumentasi lengkap di `.env.example`.

---

## 📌 Background — State Saat Ini

Sudah ada (hasil F058 Level 1):
- Grafana 10.2.2 + Loki 2.9.2 + Promtail running di `docker-compose.yml` (port `3001`).
- Promtail sudah scrape log Docker containers → kirim ke Loki ✅.
- Loki datasource auto-provisioned ✅.
- Frontend superadmin-web punya link external "📊 Grafana" (default `http://localhost:3001`).

Gap (yang harus diselesaikan):
1. **Tidak ada Prometheus** — `api-gateway` punya endpoint `/metrics` tapi isinya hanya agregasi health-check (string concatenation manual), bukan format Prometheus sebenarnya. Tidak ada yang scrape → tidak ada time-series.
2. **Tidak ada dashboard provisioned** — folder `provisioning/dashboards/` kosong. Setiap deploy Grafana mulai kosong.
3. **ENV inconsistency** — `GRAFANA_ADMIN_PASS` dipakai di compose tapi tidak didokumentasikan di `.env.example`.
4. **Backend metrics tidak standar** — handler `/metrics` di api-gateway hanya agregasi health-check. Service lain (auth, billing, accounting, chatbot, business, campaign-api) belum expose `/metrics` sama sekali.
5. **Frontend hardcode** fallback `localhost:3001` (production perlu domain).
6. **Belum ada auth/SSO hardening** — Grafana pakai login admin default.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Backend Services (Go) — 13 services                         │
│  api-gateway / auth-service / ai-gateway / billing-service / │
│  umkm-accounting / umkm-business / umkm-chatbot /            │
│  umkm-automation / campaign-api / wa-gateway / wa-cloud-api /│
│  notification-service / subscription-worker                  │
│  ↓ expose /metrics (Prometheus format via client_golang)     │
└─────────────────────────────────────────────────────────────┘
                          ↓ scrape (15s interval)
┌─────────────────────────────────────────────────────────────┐
│  Prometheus  (NEW)                                           │
│  - TSDB dengan 30 hari retention (dev) / 90d (prod)         │
│  - Service discovery via static_configs (container network) │
│  - Scrape: 13 Go services + postgres-exporter + redis-exporter│
└─────────────────────────────────────────────────────────────┘
                          ↓ query (PromQL)
┌─────────────────────────────────────────────────────────────┐
│  Grafana 10.2.2                                              │
│  - 2 datasources: Prometheus (default) + Loki              │
│  - 8 dashboard provisioned (auto-load on boot)             │
│  - Admin/password via ENV + opsi OAuth2 proxy (prod)       │
└─────────────────────────────────────────────────────────────┘
                          ↓ external link (new tab)
┌─────────────────────────────────────────────────────────────┐
│  Superadmin Web — link "📊 Grafana" → VITE_GRAFANA_URL     │
└─────────────────────────────────────────────────────────────┘

Promtail (sudah ada) → Loki (sudah ada) → Grafana Log Explorer panel
```

### Komponen Baru
1. **Prometheus container** (`prom/prometheus:v2.51.0`) — belum ada.
2. **Prometheus scrape config** — `infra/observability/prometheus/prometheus.yml`.
3. **postgres-exporter** (`prometheuscommunity/postgres-exporter:v0.15.0`) — untuk dashboard DB.
4. **redis-exporter** (`oliver006/redis_exporter:v1.59.0`) — untuk dashboard Redis.
5. **Reusable Go package** — `shared/observability/metrics.go` (middleware + collector helpers).
6. **8 dashboard JSON** — `infra/observability/grafana/provisioning/dashboards/`.
7. **Dashboard provider config** — `provisioning/dashboards/dashboards.yaml`.
8. **Datasource Prometheus config** — `provisioning/datasources/prometheus.yaml`.
9. **Compose override prod** — `docker-compose.prod.yml`.

### Komponen yang Dimodifikasi
- 13 service Go: tambah `/metrics` endpoint + HTTP middleware instrumentasi.
- `api-gateway/handlers.go`: hapus `handleAggregatedMetrics` (diganti scrape individual oleh Prometheus).
- `frontend/superadmin-web/.env` + `.env.example`: dokumentasi `VITE_GRAFANA_URL`.
- `docker-compose.yml`: tambah `prometheus`, `postgres-exporter`, `redis-exporter`, `prometheus_data` volume.
- `infra/observability/grafana/provisioning/datasources/loki.yaml`: set `isDefault: false` (Prometheus jadi default).
- `docs/FEATURE_MAP.md`: update F058 + entry baru F067.

---

## 📊 Metrics Specification

### Standard metrics (otomatis dari `prometheus/client_golang`)
| Metric | Deskripsi |
|:-------|:----------|
| `go_*` (goroutines, memstats, gc, threads) | Runtime Go — otomatis |
| `process_*` (cpu, fd, resident_memory) | Process — otomatis |

### Custom WCH HTTP middleware metrics (`shared/observability/metrics.go`)
```go
http_requests_total{service,method,route,status}      // Counter
http_request_duration_seconds{service,method,route}   // Histogram (buckets: 0.005..10s)
http_requests_in_flight{service}                      // Gauge (active concurrent)
```

### Per-service business metrics (selective)
| Service | Metric | Tipe |
|:--------|:-------|:-----|
| auth-service | `auth_logins_total{method,success}` | Counter |
| auth-service | `auth_active_sessions` | Gauge |
| ai-gateway | `ai_requests_total{provider,model,modality,status}` | Counter |
| ai-gateway | `ai_request_duration_seconds{provider,model}` | Histogram |
| wa-gateway | `wa_messages_total{channel,direction,status}` | Counter |
| wa-cloud-api | `wa_cloud_messages_total{template,status}` | Counter |
| billing-service | `billing_subscriptions_active` | Gauge |
| billing-service | `billing_payments_total{method,status}` | Counter |
| billing-service | `billing_tenants_active` | Gauge (Dashboard 6) |
| billing-service | `billing_revenue_cents` | Counter (Dashboard 6) |
| billing-service | `billing_subscriptions_new` | Counter (Dashboard 6) |
| notification-service | `notifications_sent_total{channel,status}` | Counter |
| subscription-worker | `subscription_worker_runs_total{action}` | Counter |
| campaign-api | `campaign_volunteers_active` | Gauge (Dashboard 8) |
| campaign-api | `campaign_voters_onboarded` | Counter (Dashboard 8) |
| campaign-api | `campaign_realcount_progress` | Gauge (Dashboard 8) |
| campaign-api | `campaign_logistics_status{status}` | Gauge (Dashboard 8) |
| umkm-accounting / umkm-business / umkm-chatbot / umkm-automation | (HTTP middleware only — no custom business metrics needed) | — |

### Implementasi per service
```go
// main.go di setiap service
import "core_project/shared/observability"

mux.Handle("/metrics", observability.PrometheusHandler())
r.Use(observability.Middleware("auth-service"))  // instrument HTTP otomatis
```

api-gateway change: hapus `handleAggregatedMetrics` (string concatenation jelek) → ganti dengan `/metrics` real dari api-gateway sendiri. Prometheus scrape individual service, bukan gateway agregasi.

---

## 📈 Dashboard Specifications (8 total)

Semua dashboard diletakkan di folder "WCH Platform" dan auto-provision.

| # | Nama | Datasource | Tujuan |
|:--|:-----|:-----------|:-------|
| 1 | 🏠 Platform Overview | Prometheus | Health semua service, at-a-glance |
| 2 | 🔧 Per-Service Deep Dive | Prometheus | Drill-down per service |
| 3 | 🤖 AI & WA Gateway | Prometheus | Business-critical path |
| 4 | 🗄️ Database & System | Prometheus (+ exporters) | PostgreSQL, Redis, container |
| 5 | 📜 Log Explorer | Loki | Real-time log stream, filter & search |
| 6 | 💼 Business KPIs | Prometheus | Tenant growth, revenue, top usage |
| 7 | 🚨 SLI/SLO Tracker | Prometheus | Uptime %, MTTR, error budget burn |
| 8 | 🗳️ Campaign Ops | Prometheus | Volunteer/voter/real-count/logistics |

### Dashboard 1: 🏠 Platform Overview
- Row "Services Health": `up{job="wch-services"}` per service (table/statusmap)
- Row "HTTP Throughput": total request/sec across all services (timeseries)
- Row "Error Rate": `% request dengan status >= 500` (stat + alert color)
- Row "P95 Latency": per-service p95 latency (timeseries)
- Variables: `$service` (dropdown), `$env` (dev/staging/prod)

### Dashboard 2: 🔧 Per-Service Deep Dive
- Row "Request Rate": `rate(http_requests_total{service="$service"}[5m])` per route
- Row "Latency Histogram": p50/p90/p99 (timeseries, template variable `$route`)
- Row "In-Flight Requests": active concurrent (gauge)
- Row "Go Runtime": goroutines, heap alloc, GC pause (timeseries)
- Variables: `$service`, `$route`

### Dashboard 3: 🤖 AI & WA Gateway
- Row "AI Requests": per provider/model (OpenAI/Anthropic/Gemini) — rate, latency, cost-estimate
- Row "AI Error Rate": per provider (yang outage langsung kelihatan)
- Row "WA Messages": per channel (whatsmeow/cloud_api), direction (in/out), status (sent/failed)
- Row "WA Session Pool": active sessions, connection failures
- Variables: `$provider`, `$model`, `$channel`

### Dashboard 4: 🗄️ Database & System
- Row "PostgreSQL" (via postgres-exporter): active connections vs pool limit, slow queries, transaction rate
- Row "Redis" (via redis-exporter): hit ratio, memory usage, evictions, connected clients
- Row "Container Resources": CPU/mem per service (dari process metrics Go runtime)
- Row "Disk & Network": tsdb growth, network I/O
- Variables: `$service`

### Dashboard 5: 📜 Log Explorer (Loki)
- Row "Live Log Tail": `{container=~".+"} |= "$keyword"` (log panel, real-time)
- Row "Log Level Filter": `|= "ERROR" or |= "WARN"` (toggle via `$level` variable)
- Row "Log Volume per Service": `count_over_time($query[5m])` (bar chart)
- Row "Top Error Patterns": regex extract dari log (table)
- Variables: `$service`, `$level` (INFO/WARN/ERROR/DEBUG), `$keyword`

### Dashboard 6: 💼 Business KPIs
- Row "Tenant Growth": `billing_tenants_active` (timeseries, growth trend)
- Row "Revenue": `billing_revenue_cents` cumulative per day (timeseries)
- Row "New Subscriptions": `rate(billing_subscriptions_new[1d])` per plan (bar chart)
- Row "Top Tenants by Usage": dari http_requests_total atau business metric (table)
- Variables: `$plan`, `$timerange`

### Dashboard 7: 🚨 SLI/SLO Tracker
- Row "Uptime %": per service, 30-day rolling (stat, 99.9% target)
- Row "Error Budget Burn": rate vs budget, alert threshold (timeseries)
- Row "MTTR": mean time to recovery (stat)
- Row "Incident History": annotate dari alertmanager (opsional, future)
- Variables: `$service`, `$slo_target`

### Dashboard 8: 🗳️ Campaign Ops
- Row "Volunteers": `campaign_volunteers_active` per campaign (timeseries)
- Row "Voter Onboarding": `rate(campaign_voters_onboarded[1h])` (timeseries)
- Row "Real Count C1 Progress": `campaign_realcount_progress` completion % (gauge per campaign)
- Row "Logistics Status": `campaign_logistics_status{status}` distribution (pie chart)
- Variables: `$campaign_id`

### Provisioning config (`provisioning/dashboards/dashboards.yaml`)
```yaml
apiVersion: 1
providers:
  - name: 'WCH Dashboards'
    orgId: 1
    folder: 'WCH Platform'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /etc/grafana/provisioning/dashboards
```

---

## ⚙️ Environment & Compose Configuration

### `.env.example` — tambahan baru
```bash
# =============================================================================
# Observability — Grafana / Prometheus / Loki
# =============================================================================
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASS=ganti_password_grafana_yang_kuat   # WAJIB ganti di prod!
PROMETHEUS_RETENTION=30d
LOKI_RETENTION=336h                                     # 14 hari (sudah ada)

# Frontend (superadmin-web)
VITE_GRAFANA_URL=http://localhost:3001                  # dev
# PRODUCTION: VITE_GRAFANA_URL=https://grafana.yourdomain.com
```

### `docker-compose.yml` (dev — existing + tambahan)
```yaml
  prometheus:
    image: prom/prometheus:v2.51.0
    container_name: wch-prometheus
    ports:
      - "9091:9090"
    volumes:
      - ./infra/observability/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=${PROMETHEUS_RETENTION:-30d}
    restart: unless-stopped

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.15.0
    container_name: wch-postgres-exporter
    environment:
      - DATA_SOURCE_NAME=postgresql://wch_admin:${DB_PASSWORD}@postgres:5432/wch_platform?sslmode=disable
    ports:
      - "9187:9187"
    restart: unless-stopped

  redis-exporter:
    image: oliver006/redis_exporter:v1.59.0
    container_name: wch-redis-exporter
    environment:
      - REDIS_ADDR=redis://redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    ports:
      - "9121:9121"
    restart: unless-stopped

  grafana:
    # ...existing...
    environment:
      - GF_SECURITY_ADMIN_USER=${GRAFANA_ADMIN_USER:-admin}
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASS:-admin123}
      - GF_USERS_ALLOW_SIGN_UP=false

volumes:
  prometheus_data: null
```

### `docker-compose.prod.yml` (NEW — override)
```yaml
# Usage: docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
services:
  prometheus:
    ports: !reset []
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=${PROMETHEUS_RETENTION:-90d}

  grafana:
    ports: !reset []
    environment:
      - GF_SERVER_ROOT_URL=https://grafana.${DOMAIN}
      - GF_SERVER_DOMAIN=grafana.${DOMAIN}

  postgres-exporter:
    ports: !reset []

  redis-exporter:
    ports: !reset []
```

### `infra/observability/prometheus/prometheus.yml` (NEW)
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    env: dev

scrape_configs:
  - job_name: 'wch-services'
    static_configs:
      - targets:
          - 'api-gateway:8000'
          - 'auth-service:8001'
          - 'ai-gateway:8002'
          - 'billing-service:8003'
          - 'umkm-accounting:8201'
          - 'umkm-business:9005'
          - 'umkm-chatbot:8203'
          - 'umkm-automation:8204'
          - 'campaign-api:9002'
          - 'wa-gateway:8202'
          - 'wa-cloud-api:8210'
          - 'notification-service:8005'
          - 'subscription-worker:8006'
    relabel_configs:
      - source_labels: ['__address__']
        regex: '([a-z0-9\-]+):\d+'
        target_label: 'service'

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### `provisioning/datasources/prometheus.yaml` (NEW)
```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

### `provisioning/datasources/loki.yaml` (MODIFIED)
```yaml
apiVersion: 1
datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: false       # Prometheus jadi default
    version: 1
    editable: false
```

---

## 🛡️ Auth Strategy

**Dev (default):** Admin/password via ENV (`GRAFANA_ADMIN_USER` + `GRAFANA_ADMIN_PASS`). Default fallback `admin/admin123` hanya untuk dev.

**Prod (opsional OAuth2 proxy):**
- Grafana di reverse-proxy nginx, tidak expose port langsung.
- Untuk SSO: deploy `oauth2-proxy` container, connect ke auth-service JWT (future enhancement — terdokumentasi di F058 Level 3).
- Minimum hardening prod: ganti `GRAFANA_ADMIN_PASS`, enable TLS via nginx, restrict access via IP allowlist / VPN.

**Catatan keamanan exporter:**
- `DATA_SOURCE_NAME` postgres-exporter mengandung DB password. Di prod HARUS via Docker secret / vault, bukan plain ENV. Didokumentasikan di `.env.example`.
- redis-exporter password juga sensitif — sama perlakuan-nya.

---

## 🧪 Testing Strategy

```bash
# 1. Compile semua service (instrumentasi tidak boleh break build)
go build ./... && go vet ./...

# 2. Unit test observability package
go test ./shared/observability/... -v

# 3. Manual — verify /metrics per service (format Prometheus valid)
for port in 8000 8001 8002 8003 8005 8006 8201 8202 8203 8204 8210 9002 9005; do
  echo "=== :$port/metrics ==="
  curl -s http://localhost:$port/metrics | head -20
done

# 4. Prometheus target health
curl -s http://localhost:9091/api/v1/targets | jq '.data.activeTargets[] | {job:.labels.job, health}'

# 5. Grafana — semua dashboard ter-provision
curl -s -u admin:$GRAFANA_ADMIN_PASS http://localhost:3001/api/search | jq '.[].title'
# → harus return 8 dashboard

# 6. Datasource
curl -s -u admin:$GRAFANA_ADMIN_PASS http://localhost:3001/api/datasources | jq '.[].name'
# → ["Prometheus","Loki"]

# 7. Full make check
make check
```

---

## ✅ Acceptance Criteria

- [ ] AC-1: Semua 13 service Go expose `/metrics` format Prometheus valid (curl return `# HELP`/`# TYPE`)
- [ ] AC-2: Prometheus scrape semua target — status `up` di `/api/v1/targets` (13 services + 2 exporters = 15 targets)
- [ ] AC-3: Grafana auto-load 8 dashboard + 2 datasource tanpa manual setup
- [ ] AC-4: `postgres-exporter` & `redis-exporter` aktif, datanya muncul di Dashboard 4
- [ ] AC-5: Dashboard 5 (Log Explorer) query Loki real-time
- [ ] AC-6: `docker-compose.prod.yml` tidak expose port internal, hanya via nginx
- [ ] AC-7: `.env.example` dokumentasi semua ENV observability (GRAFANA_ADMIN_USER/PASS, PROMETHEUS_RETENTION, VITE_GRAFANA_URL)
- [ ] AC-8: `go build ./...`, `go vet`, `make check` clean
- [ ] AC-9: Link Grafana di superadmin-web pakai `VITE_GRAFANA_URL` (tidak hardcode localhost)
- [ ] AC-10: Business metrics (billing/campaign) muncul di Dashboard 6 & 8
- [ ] AC-11: Package `shared/observability` lulus unit test

---

## 📁 File Changes Summary

### Files Baru
| File | Deskripsi |
|:-----|:----------|
| `shared/observability/metrics.go` | Package reusable: `PrometheusHandler()`, `Middleware(svc)`, collector helpers |
| `shared/observability/metrics_test.go` | Unit test untuk middleware & counter |
| `infra/observability/prometheus/prometheus.yml` | Scrape config (9 service Go + 2 exporter) |
| `infra/observability/grafana/provisioning/dashboards/dashboards.yaml` | Dashboard provider config |
| `infra/observability/grafana/provisioning/dashboards/01-platform-overview.json` | Dashboard 1 |
| `infra/observability/grafana/provisioning/dashboards/02-service-deep-dive.json` | Dashboard 2 |
| `infra/observability/grafana/provisioning/dashboards/03-ai-wa-gateway.json` | Dashboard 3 |
| `infra/observability/grafana/provisioning/dashboards/04-database-system.json` | Dashboard 4 |
| `infra/observability/grafana/provisioning/dashboards/05-log-explorer.json` | Dashboard 5 (Loki) |
| `infra/observability/grafana/provisioning/dashboards/06-business-kpis.json` | Dashboard 6 |
| `infra/observability/grafana/provisioning/dashboards/07-sli-slo-tracker.json` | Dashboard 7 |
| `infra/observability/grafana/provisioning/dashboards/08-campaign-ops.json` | Dashboard 8 |
| `infra/observability/grafana/provisioning/datasources/prometheus.yaml` | Datasource baru (default) |
| `docker-compose.prod.yml` | Override untuk production |

### Files Modified
| File | Perubahan |
|:-----|:----------|
| `services/auth-service/main.go` | `/metrics` endpoint + middleware + business metrics |
| `services/ai-gateway/main.go` | ganti custom handler → `observability.PrometheusHandler()` + AI metrics |
| `services/wa-gateway/routes.go` | ganti → `observability.PrometheusHandler()` + WA metrics |
| `services/wa-cloud-api/main.go` | `/metrics` + middleware + cloud WA metrics |
| `services/billing-service/main.go` | `/metrics` + wallet/subscription/tenant metrics |
| `services/notification-service/main.go` | `/metrics` + middleware + notification metrics |
| `services/subscription-worker/main.go` | `/metrics` + middleware + worker metrics |
| `apps/umkm/accounting/main.go` | `/metrics` + middleware |
| `apps/umkm/chatbot/main.go` | `/metrics` + middleware |
| `apps/umkm/business/main.go` | `/metrics` + middleware |
| `apps/umkm/automation/main.go` | `/metrics` + middleware |
| `apps/campaign/api/main.go` | `/metrics` + campaign business metrics |
| `services/api-gateway/main.go` | hapus `handleAggregatedMetrics` → pakai real `/metrics` |
| `services/api-gateway/handlers.go` | hapus `handleAggregatedMetrics` function |
| `docker-compose.yml` | + `prometheus`, `postgres-exporter`, `redis-exporter`, `prometheus_data` volume |
| `infra/observability/grafana/provisioning/datasources/loki.yaml` | `isDefault: false` |
| `.env.example` | + section observability |
| `frontend/superadmin-web/.env` | dokumentasi `VITE_GRAFANA_URL` untuk prod |
| `docs/FEATURE_MAP.md` | update F058 + entry baru F067 |

---

## ⚠️ Risks & Mitigation

1. **`prometheus/client_golang` dependency** — tambah 1 dep baru ke `go.mod`. Aman, library official, widely used. Mitigasi: pin versi stabil.
2. **Performance overhead** — middleware metrics ~microseconds per request. Negligible untuk volume WCH.
3. **Dashboard JSON validity** — paling labor-intensive. Akan generate via Grafana json schema yang valid (bukan hand-draw) supaya pasti load.
4. **Prometheus retention disk** — 30 hari dev ~500MB-2GB. Pastikan disk cukup. Prod override 90d ~3-6GB.
5. **Exporter credentials** — `DATA_SOURCE_NAME` mengandung DB password. Di prod HARUS via Docker secret / vault, bukan plain ENV. Sudah didokumentasikan.
6. **Prometheus relabel `service` label** — perlu test regex extract agar label `service` benar untuk dashboard drill-down. Fallback: hardcode label per target group jika regex bermasalah.

---

## 🔮 Future Enhancements (Out of Scope)

- **Alertmanager** + notification ke WA/Telegram/Slack saat service down (F068 candidate)
- **OAuth2 proxy** untuk SSO Grafana ↔ auth-service JWT (F058 Level 3)
- **node-exporter / cAdvisor** untuk host-level metrics (host CPU/mem/disk)
- **Tempo / Jaeger** untuk distributed tracing
- **Grafana OnCall** untuk incident management
- **Embed iframe** di superadmin-web (Level 2 — butuh `GF_SECURITY_ALLOW_EMBEDDING`)
- **Long-term storage** Prometheus → Thanos / Mimir untuk retention > 90d

---

## 📝 Implementation Order (high-level — detail di plan)

1. Buat `shared/observability` package + unit test
2. Instrument 13 service Go (batch — bisa parallel)
3. Setup Prometheus container + scrape config
4. Setup exporters (postgres + redis)
5. Generate 8 dashboard JSON
6. Update Grafana datasource + provisioning
7. Update compose (dev + prod override)
8. Update `.env.example` + frontend env
9. Update FEATURE_MAP.md
10. Testing end-to-end
