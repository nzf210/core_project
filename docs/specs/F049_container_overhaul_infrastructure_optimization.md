# F049: Container Overhaul & Infrastructure Optimization

**Date:** 2026-06-17  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [Deployment Docs](../../DEPLOYMENT.md), [Docker Compose Staging](../../docker-compose.staging.yml)

---

## 🎯 Objectives

Optimize Docker infrastructure untuk production deployment dengan multi-stage builds, resource limits, dan observability stack.

**Tujuan eksplisit:**
1. Multi-stage Docker builds untuk reduce image size (from ~2GB to ~500MB per service)
2. Resource limits (CPU/memory) per container untuk prevent resource starvation
3. Production-ready observability stack (Prometheus + Grafana + Loki + Promtail) dengan auto-provisioned dashboards

**Problem yang diselesaikan:**
- Docker images terlalu besar (2GB+) → slow deployment, high bandwidth cost
- No resource limits → satu service bisa monopolize host resources, impact service lainnya
- Manual observability setup → setiap deploy baru perlu setup Grafana dashboard manually

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Multi-Stage Dockerfile**
  - *Verification:* Dockerfile menggunakan multi-stage build (builder stage + final runtime stage), final image <600MB
  - *Example:* Image `wch-auth-service` before: 2.1GB, after: 480MB

- [x] **AC-2: Resource Limits in docker-compose**
  - *Verification:* Setiap service di `docker-compose.yml` memiliki `deploy.resources.limits` (cpu, memory)
  - *Example:* `auth-service: { cpu: '0.5', memory: '512M' }`

- [x] **AC-3: Healthcheck Endpoints**
  - *Verification:* Setiap Go service expose `/health` endpoint, Docker HEALTHCHECK directive configured
  - *Example:* `curl http://localhost:8001/health` → `200 OK { "status": "healthy" }`

- [x] **AC-4: Prometheus Metrics Scraping**
  - *Verification:* Prometheus scrape all services, targets show "UP" status
  - *Example:* Prometheus UI → Targets page → all services green

- [x] **AC-5: Grafana Auto-Provisioned Dashboards**
  - *Verification:* Grafana load dashboards from `infra/observability/grafana/dashboards/*.json` on startup
  - *Example:* Fresh Grafana instance → 8 dashboards already available (no manual import)

- [x] **AC-6: Loki Log Aggregation**
  - *Verification:* Promtail forward Docker container logs ke Loki, Grafana Explore query Loki datasource
  - *Example:* Grafana Explore → Loki → query `{container_name="wch-auth-service"}` → show logs

- [x] **AC-7: Staging Environment Setup**
  - *Verification:* `docker-compose.staging.yml` with separate config, `scripts/staging-setup.sh` automate initialization
  - *Example:* `./scripts/staging-setup.sh` → generate secrets, init DB, start all services

- [x] **AC-8: CI/CD Health Checks**
  - *Verification:* GitHub Actions workflow deploy ke staging → health check all services → rollback jika failed
  - *Example:* Deploy workflow step "Health Check" → cURL all `/health` endpoints → fail job if any 5xx

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│       Multi-Stage Docker Build (Per Service)        │
│  Stage 1 (builder): golang:1.23-alpine             │
│    - go build ./cmd/service                         │
│  Stage 2 (runtime): alpine:3.19                    │
│    - COPY --from=builder /app/binary               │
│    - HEALTHCHECK --interval=30s /health            │
│  Result: Final image ~500MB (vs 2GB before)        │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Docker Compose Resource Limits                 │
│  services:                                           │
│    auth-service:                                     │
│      deploy:                                         │
│        resources:                                    │
│          limits: { cpu: '0.5', memory: '512M' }     │
│          reservations: { cpu: '0.25', memory: '256M'}│
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Observability Stack (Prometheus + Grafana)     │
│  Prometheus :9090 scrape /metrics from all services │
│  Grafana :3001 auto-load dashboards from volume     │
│  Loki :3100 log aggregation from Promtail          │
│  Promtail read Docker logs → push to Loki          │
└─────────────────────────────────────────────────────┘
```

### Dockerfile Multi-Stage Pattern

```dockerfile
# Stage 1: Builder
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/service ./services/auth-service

# Stage 2: Runtime
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/service ./
COPY --from=builder /app/shared/migrations ./shared/migrations

EXPOSE 8001
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

CMD ["./service"]
```

### Resource Limits (docker-compose.yml)

```yaml
services:
  auth-service:
    image: wch-auth-service:latest
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8001/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
```

### Prometheus Scrape Config

```yaml
# infra/observability/prometheus/prometheus.yml
scrape_configs:
  - job_name: 'wch-services'
    static_configs:
      - targets:
          - 'auth-service:8001'
          - 'billing-service:8003'
          - 'api-gateway:8000'
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Auto-Provisioned Dashboards

```yaml
# infra/observability/grafana/provisioning/dashboards/default.yml
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    options:
      path: /etc/grafana/dashboards
```

Dashboards (8 total):
1. `service-health.json` — Service uptime + health check status
2. `http-metrics.json` — Request rate, latency (p50/p95/p99), error rate
3. `database-metrics.json` — Connection pool, query duration, slow queries
4. `redis-metrics.json` — Hit rate, memory usage, command latency
5. `container-metrics.json` — CPU, memory, disk I/O per container
6. `business-metrics.json` — Revenue, transactions, active tenants
7. `logs-overview.json` — Log volume, error rate, top error messages
8. `alerts-dashboard.json` — Active alerts, alert history

---

## 🧪 Testing Strategy

### Build Tests

```bash
# Test multi-stage build (image size < 600MB)
docker build -t wch-auth-service:test -f Dockerfile --target=runtime .
docker images wch-auth-service:test --format "{{.Size}}"
# Expect: <600MB

# Test healthcheck
docker run -d --name test-auth wch-auth-service:test
sleep 10
docker inspect test-auth | jq '.[0].State.Health.Status'
# Expect: "healthy"
```

### Resource Limit Tests

```bash
# Deploy with resource limits
docker-compose -f docker-compose.staging.yml up -d

# Verify limits applied
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
# Expect: auth-service CPU <50%, memory <512MB

# Stress test (attempt to exceed limit)
docker exec wch-auth-service stress --cpu 2 --timeout 60s
# Expect: CPU throttled at 50%, not 200%
```

### Observability Tests

```bash
# Prometheus targets
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job, health}'
# Expect: all targets "up"

# Grafana dashboards
curl -u admin:admin http://localhost:3001/api/search | jq 'length'
# Expect: 8 dashboards

# Loki logs
curl -G http://localhost:3100/loki/api/v1/query \
  --data-urlencode 'query={container_name="wch-auth-service"}' | jq '.data.result | length'
# Expect: >0 log entries
```

### Manual Testing Checklist

- [ ] Build all service images → verify size <600MB each
- [ ] `docker-compose up` → all services start healthy
- [ ] Prometheus UI → all targets green
- [ ] Grafana UI → 8 dashboards auto-loaded
- [ ] Grafana Explore → Loki datasource → query logs
- [ ] Kill one service → Prometheus alert fires
- [ ] Restart service → health check pass → alert resolves
- [ ] Staging deploy script → `./scripts/staging-setup.sh` → all services running

---

## 📊 Monitoring & Observability

**Prometheus Alerts (infra/observability/prometheus/alerts.yml):**
```yaml
groups:
  - name: service_health
    rules:
      - alert: ServiceDown
        expr: up{job="wch-services"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.instance }} is down"
      
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate on {{ $labels.instance }}"
```

**Metrics to track:**
- Container resource usage (CPU, memory, disk) vs limits
- Image pull time (track image size impact on deployment speed)
- Service startup time (from container start to health check pass)
- Log volume per service (detect log spam)

**Grafana Dashboard Highlights:**
- Service Health: Uptime %, last restart time, health check failure count
- HTTP Metrics: Request rate, p95 latency, error rate (4xx/5xx split)
- Database: Slow query count, connection pool exhaustion events

---

## 🚀 Rollout Plan

### Phase 1: Multi-Stage Dockerfile (Done ✅)
- Refactor all service Dockerfiles → multi-stage pattern
- Build + test locally → verify image size reduction
- Push to container registry

### Phase 2: Resource Limits (Done ✅)
- Add `deploy.resources` to docker-compose.yml
- Test on staging → verify limits enforced, no OOM kills
- Document recommended limits per service in `DEPLOYMENT.md`

### Phase 3: Observability Stack (Done ✅)
- Deploy Prometheus + Grafana + Loki + Promtail
- Auto-provision 8 Grafana dashboards
- Configure Prometheus alerts
- Test: kill service → alert fires → Slack notification

### Phase 4: CI/CD Integration (Done ✅)
- GitHub Actions workflow deploy ke staging
- Health check step post-deploy
- Rollback on failure
- Slack notification on success/failure

### Phase 5: Production Deployment (Future)
- Same infra setup di production environment
- Blue-green deployment strategy
- Automated backup before deploy

### Rollback
- **Dockerfile rollback:** Revert to single-stage Dockerfile → larger images but simpler build
- **Resource limits rollback:** Remove `deploy.resources` → unlimited resource usage (risky on shared host)
- **Observability rollback:** Stop Prometheus/Grafana/Loki containers → lose metrics/logs (no impact on service functionality)

---

## 🔮 Future Enhancements (Out of Scope)

- **Kubernetes Migration:** Migrate from docker-compose ke K8s untuk better orchestration, auto-scaling, self-healing
- **Service Mesh:** Istio/Linkerd untuk mTLS, circuit breaking, traffic splitting
- **Horizontal Autoscaling:** Scale service replicas based on CPU/memory usage or request rate
- **Distributed Tracing:** Jaeger/Tempo untuk trace request flow across services
- **Log Retention Policy:** Auto-delete old logs dari Loki (e.g., keep 30 days) untuk manage disk usage

---

## 📚 References

- [Docker Multi-Stage Builds](https://docs.docker.com/build/building/multi-stage/) — Official Docker docs
- [Prometheus Scrape Config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) — Scraping configuration reference
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/) — Auto-provision dashboards
- [Deployment Runbook](../../docs/DEPLOYMENT_RUNBOOK.md) — Step-by-step production deployment guide

---

## 📝 Notes & Decisions

**2026-06-17:** Decision: Multi-stage build with `alpine:3.19` runtime (bukan `scratch`) karena butuh ca-certificates + tzdata untuk HTTPS calls + timezone handling. Trade-off: +10MB image size, gain standard tooling (wget for healthcheck).  
**2026-06-17:** Resource limits: CPU 0.5 core (50%), memory 512MB per service untuk shared VPS host dengan 8 cores, 16GB RAM, 12 services → reserve 6 cores, 6GB RAM, sisanya untuk DB + Redis + observability stack.  
**2026-06-17:** Grafana dashboards auto-provisioned via file mount (bukan API import) karena idempotent + version-controlled. Dashboard JSON stored di git, auto-load on container restart.  
**2026-06-17:** Prometheus alert routing ke Slack via Alertmanager webhook — config in `infra/observability/alertmanager/config.yml` (not covered in this spec, separate F067).  
**2026-06-17:** Staging environment separate docker-compose file (`docker-compose.staging.yml`) untuk isolate config, avoid accidental production deploy. Production akan punya `docker-compose.prod.yml` dengan different secrets + resource allocation.

**Note:** File ini sebelumnya berisi spec tentang Dashboard Reports (sales chart, top products) yang tidak relevan dengan judul "Container Overhaul". Konten dashboard reports seharusnya di file spec terpisah (e.g., F0XX_dashboard_reports.md).
