# F067: Grafana Production-Ready Monitoring — Prometheus + 8 Dashboards

**Date:** 2026-07-01
**Status:** ✅ Approved
**Implementation:** ✅ Done
**Related:** F058 (Grafana Level 1 → Level 3 upgrade)

---

## 🎯 Objectives

Membawa stack observability WCH Platform dari "berjalan tapi belum production-grade" menjadi **production-ready** — sehingga deploy ke production tinggal ganti ENV dan compose file, tanpa perubahan kode.

**Tujuan eksplisit:**
1. Backend services expose `/metrics` Prometheus format yang valid
2. Prometheus aktif scrape 13 services + postgres-exporter + redis-exporter
3. Grafana auto-provision 8 dashboard + 2 datasource (zero manual setup)
4. Pemisahan dev vs prod via compose override (port exposure, retention, reverse-proxy)
5. ENV terdokumentasi lengkap di `.env.example`

---

## 📊 Metrics Specification

### Standard metrics (otomatis dari `prometheus/client_golang`)
- `go_*` (goroutines, memstats, gc, threads)
- `process_*` (cpu, fd, resident_memory)

### Custom WCH HTTP middleware (`shared/observability/metrics.go`)
```
http_requests_total{service,method,route,status}      // Counter
http_request_duration_seconds{service,method,route}   // Histogram
http_requests_in_flight{service}                      // Gauge
```

### Per-service business metrics
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
| billing-service | `billing_tenants_active` | Gauge |
| billing-service | `billing_revenue_cents` | Counter |
| billing-service | `billing_subscriptions_new` | Counter |
| notification-service | `notifications_sent_total{channel,status}` | Counter |
| subscription-worker | `subscription_worker_runs_total{action}` | Counter |
| campaign-api | `campaign_volunteers_active` | Gauge |
| campaign-api | `campaign_voters_onboarded` | Counter |
| campaign-api | `campaign_realcount_progress` | Gauge |
| campaign-api | `campaign_logistics_status{status}` | Gauge |

---

## 📈 Dashboard Specifications (8 total)

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

Semua dashboard auto-provision di folder "WCH Platform". Minimal valid JSON dengan 1 placeholder panel — edit di Grafana UI untuk customize.

---

## ⚙️ Environment Configuration

### `.env.example` — tambahan baru
```bash
# Observability — Grafana / Prometheus / Loki
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASS=ganti_password_grafana_yang_kuat
PROMETHEUS_RETENTION=30d
LOKI_RETENTION=336h  # 14 hari

# Frontend (superadmin-web)
VITE_GRAFANA_URL=http://localhost:3001  # dev
# PRODUCTION: VITE_GRAFANA_URL=https://grafana.yourdomain.com
```

### `docker-compose.yml` — tambahan
- `prometheus` (prom/prometheus:v2.51.0) → port 9091
- `postgres-exporter` (v0.15.0) → port 9187
- `redis-exporter` (v1.59.0) → port 9121
- Volume: `prometheus_data`

### `docker-compose.prod.yml` — NEW override
- Prometheus retention 90d (vs 30d dev)
- Port exposure stripped (internal network only)
- Grafana `GF_SERVER_ROOT_URL` + `GF_SERVER_DOMAIN` for reverse proxy

---

## ✅ Implementation Summary

### Files Baru (19 total)
- `shared/observability/metrics.go` + `metrics_test.go`
- `infra/observability/prometheus/prometheus.yml`
- `infra/observability/grafana/provisioning/dashboards/dashboards.yaml`
- `infra/observability/grafana/provisioning/dashboards/01-platform-overview.json` (+ 7 lainnya)
- `infra/observability/grafana/provisioning/datasources/prometheus.yaml`
- `docker-compose.prod.yml`

### Files Modified (18 total)
- 13 Go services: tambah `/metrics` endpoint + middleware + business metrics
- `docker-compose.yml`: tambah prometheus + exporters + volume
- `infra/observability/grafana/provisioning/datasources/loki.yaml`: `isDefault: false`
- `.env.example`: section observability
- `frontend/superadmin-web/.env`: dokumentasi `VITE_GRAFANA_URL`
- `docs/FEATURE_MAP.md`: entry F067

### Instrumented Services (13)
1. auth-service ✅
2. api-gateway ✅
3. ai-gateway ✅
4. billing-service ✅
5. wa-gateway ✅
6. wa-cloud-api ✅
7. notification-service ✅
8. subscription-worker ✅
9. umkm-accounting ✅
10. umkm-business ✅
11. umkm-chatbot ✅
12. umkm-automation ✅
13. campaign-api ✅

---

## 🧪 Acceptance Criteria

- [x] AC-1: Semua 13 service expose `/metrics` format Prometheus valid
- [x] AC-2: Prometheus scrape 15 targets (13 services + 2 exporters) dengan status `up`
- [x] AC-3: Grafana auto-load 8 dashboard + 2 datasource
- [x] AC-4: postgres-exporter & redis-exporter aktif
- [x] AC-5: Dashboard 5 (Log Explorer) query Loki real-time
- [x] AC-6: `docker-compose.prod.yml` tidak expose port internal
- [x] AC-7: `.env.example` dokumentasi ENV observability
- [x] AC-8: `go build ./...` clean
- [x] AC-9: Link Grafana di superadmin-web pakai `VITE_GRAFANA_URL`
- [x] AC-10: Business metrics declared di service yang relevan
- [x] AC-11: `shared/observability` lulus unit test

---

## 🚀 Deployment

**Dev:**
```bash
docker compose up -d prometheus postgres-exporter redis-exporter grafana
make start-all  # atau make dev-all untuk hot-reload
curl http://localhost:9091/targets  # cek Prometheus targets
open http://localhost:3001  # Grafana (admin/admin123 default dev)
```

**Production:**
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
# Grafana accessible via reverse proxy (nginx) di https://grafana.yourdomain.com
# Internal services + Prometheus tidak expose port ke host
```

---

## 🔮 Future Enhancements (Out of Scope)

- **Alertmanager** + notification ke WA/Telegram (F068 candidate)
- **OAuth2 proxy** untuk SSO Grafana ↔ auth-service JWT
- **node-exporter / cAdvisor** untuk host-level metrics
- **Tempo / Jaeger** untuk distributed tracing
- **Grafana OnCall** untuk incident management
- **Embed iframe** di superadmin-web (Level 2 — butuh `GF_SECURITY_ALLOW_EMBEDDING`)
