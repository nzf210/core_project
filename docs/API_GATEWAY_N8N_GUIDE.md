# 🌐 WCH Platform — API Gateway & n8n Integration Guide

> Single source of truth untuk routing, middleware chain, n8n workflow orchestration, dan webhook contracts.

---

## 1. Posisi API Gateway

API Gateway (`services/api-gateway`, port 8000) adalah **single entry point** untuk seluruh request eksternal. Tanggung jawabnya:

- **Routing** — path → service backend
- **Auth inject** — JWT di-parse, klaim di-propagate via `X-Tenant-ID` / `X-User-ID` / `X-User-Role` header
- **Rate limiting** — IP, public, dan per-tenant
- **CORS** — header di-strip dari upstream (gateway yang pegang kendali)
- **Quota** — blokir `lite` tier jika transaksi quota habis

```
[Client] → [Nginx :80/443] → [API Gateway :8000] → [Backend service :XXXX]
                                          ↓
                                    [Redis :6381]   (rate limit + cache)
```

---

## 2. Routing Table (Final)

### Public Routes (no auth)

| Path | Target | Notes |
|:-----|:-------|:------|
| `/auth/*` | auth-service:8001 | Login, register, refresh, public profile |
| `/api/public/campaign/*` | campaign-api:9002 | Public election info |
| `/uploads/*` | auth-service:8001/static | Avatar, attachments |
| `/webhooks/xendit/*` | billing-service:8003 | Xendit callbacks (signature verified by billing) |
| `/webhooks/fonnte/*` | notification-service:8005 | Fonnte WA delivery status |
| `/webhooks/n8n/*` | n8n:5678 | n8n workflow callbacks |
| `/healthz` | (self) | Aggregated health — pings all downstreams |

### Superadmin Routes (`X-User-Role: superadmin`)

| Path | Target | Purpose |
|:-----|:-------|:--------|
| `/api/superadmin/*` | auth-service:8001/superadmin | User management, tenant CRUD |
| `/api/superadmin/billing/*` | billing-service:8003/admin | Plans, vouchers, dashboard |
| `/api/superadmin/n8n/*` | n8n:5678 | n8n UI proxy (Basic Auth header auto-injected) |

### Tenant Routes (`auth` + `tenantRateLimit` + optional `quota`)

| Path | Target | Middleware |
|:-----|:-------|:-----------|
| `/api/profile*` | auth-service:8001 | tenantRateLimit |
| `/api/ai/*` | ai-gateway:8002 | tenantRateLimit + quota |
| `/api/umkm/business/*` | umkm-business:9005 | tenantRateLimit |
| `/api/umkm/automation/*` | umkm-automation:8203 | tenantRateLimit |
| `/api/umkm/*` | umkm-accounting:8201 | tenantRateLimit + quota |
| `/api/campaign/*` | campaign-api:9002 | tenantRateLimit + quota |
| `/api/billing/*` | billing-service:8003 | tenantRateLimit |
| `/api/crypto/*` | crypto-api:8101/api/v1 | tenantRateLimit + quota |
| `/api/wa/*` | wa-gateway:8202 | tenantRateLimit |
| `/api/notifications/*` | notification-service:8005 | tenantRateLimit |

---

## 3. Middleware Chain (Execution Order)

Setiap request melalui pipeline ini **top-down**:

```
1. corsMiddleware           ← set CORS headers, handle OPTIONS preflight
2. ipRateLimitMiddleware    ← 200 req/min/IP, key: rate_limit:ip:<addr>
3. auth.Middleware          ← parse JWT, populate context (TenantID, UserID, Role)
4. tenantRateLimit          ← per plan (free=60, lite=300, pro=1000)
5. quotaMiddleware          ← lite: transactions quota check
6. (rate limit public)      ← /auth, /api/public/* (10-30 req/min/IP)
7. handler / reverse proxy  ← forward to backend with X-* headers
```

**Urutan penting:** auth HARUS sebelum tenantRateLimit (butuh tenantID dari JWT). Rate limit IP HARUS paling luar (DDoS protection).

---

## 4. Header Convention

### Request Headers (klien → gateway)

| Header | Wajib | Tujuan |
|:-------|:------|:-------|
| `Authorization: Bearer <jwt>` | ✅ (route privat) | JWT hasil login |
| `X-Request-ID` | optional | Trace ID untuk logging |
| `X-Tenant-ID` | ❌ (otomatis) | Di-inject gateway dari JWT — client tidak perlu set |

### Response Headers (backend → klien)

| Header | Set Oleh | Tujuan |
|:-------|:---------|:-------|
| `X-Subscription-Status` | subscription_guard | `active` atau `frozen` |
| `X-Request-ID` | gateway | Trace ID untuk debugging |
| `X-RateLimit-Remaining` | (planned) | Sisa quota rate limit |

---

## 5. n8n Integration

### 5.1 Arsitektur

```
[API Gateway :8000]
    ├── /api/superadmin/n8n/*  → n8n UI (Basic Auth injected)
    └── /webhooks/n8n/*        → n8n webhook receiver
    
[n8n :5678]
    ├── workflows (JSON di /workflows/*.json, di-import saat start)
    ├── DB: postgres (shared DB dengan backend)
    └── Scheduler: cron jobs untuk reminder, digest, dsb
```

### 5.2 Workflow yang Tersedia

| File | Trigger | Aksi |
|:-----|:--------|:-----|
| `master_automations.json` | Cron `* * * * *` | Poll UMKM automation due → execute → send WA |
| `freeze_reminder.json` | Cron `0 8 * * *` | H-3/H-1/H-0 → WA reminder ke owner |
| `daily_revenue_digest.json` | Cron `0 20 * * *` | Fetch dashboard → kirim ke Telegram owner |
| `voucher_wa_distribute.json` | Webhook `POST /voucher-bulk-distribute` | Bulk distribute voucher via WA |
| `campaign_voter_onboard.json` | Webhook `POST /campaign-voter-onboard` | Sentiment classification via AI Gateway |

### 5.3 Menambah Workflow Baru

1. Buat JSON di `infra/n8n/workflows/<nama>.json`
2. Stop n8n: `docker compose stop n8n`
3. Hapus marker: `docker compose exec n8n rm /home/node/.n8n/.workflow_imported`
4. Start ulang: `docker compose up -d n8n` (akan re-import semua workflow)

Atau, lebih simple — **import manual via UI**:
- Buka `http://localhost:5678` (Basic Auth dari `.env`)
- New workflow → paste JSON → save & activate

### 5.4 Webhook Pattern

n8n workflow trigger via webhook dengan konvensi:

```
POST /api/superadmin/n8n/webhook/<workflow-path>
Content-Type: application/json
X-Webhook-Secret: <shared-secret>     # opsional, divalidasi n8n

{ "field": "value", ... }
```

Atau via direct port (jika expose n8n di network internal):
```
POST http://n8n:5678/webhook/<workflow-path>
```

---

## 6. Webhook Contracts (External Services)

### 6.1 Xendit (Payment)

| Event | Path | Payload |
|:------|:-----|:--------|
| Invoice paid | `POST /webhooks/xendit/invoice.paid` | Xendit invoice object |
| Subscription activated | `POST /webhooks/xendit/subscription.activated` | Xendit subscription object |
| Subscription expired | `POST /webhooks/xendit/subscription.expired` | Xendit subscription object |

**Verifikasi signature** — di-handle `billing-service`:
- Header `x-callback-token` harus sama dengan `cfg.Xendit.WebhookToken`
- Reject 401 jika beda.

### 6.2 wa-gateway Internal (WhatsApp)

| Event | Path | Payload |
|:------|:-----|:--------|
| Message status | `POST /webhooks/wa/message.status` | Delivery report dari whatsmeow |
| Device status | `POST /webhooks/wa/device.status` | Connection status |

**Verifikasi** — internal service, tidak perlu signature (hanya accessible dari docker network).

### 6.3 Custom Webhook (Internal)

Untuk trigger n8n dari backend service:
```
POST /webhooks/n8n/<workflow-name>
```

Contoh: ketika voucher baru di-redeem → trigger n8n workflow "send welcome email".

---

## 7. Subscription Worker

`services/subscription-worker` (port 8006) — background cron:

| Env Var | Default | Tujuan |
|:--------|:--------|:-------|
| `FREEZE_CHECK_INTERVAL` | `1h` | Seberapa sering cek subscription |
| `GRACE_PERIOD_HOURS` | `0` | Toleransi (0 = freeze tepat waktu) |

Endpoint: `GET /healthz` (untuk liveness probe).

**Logika:**
1. Query `tenant_subscriptions` dengan `status='active' AND current_period_end < NOW() - GRACE`
2. Update batch: `status='frozen'`, `tenants.is_frozen=true`
3. Log: `slog.Info("tenant frozen", "tenant_id", id, "expired_at", ...)`

Frontend (UMKM) membaca `tenants.is_frozen` via `/api/profile` dan tampilkan banner.

---

## 8. Troubleshooting

| Gejala | Penyebab | Fix |
|:-------|:---------|:----|
| 401 Unauthorized | JWT invalid/expired | Re-login, refresh token |
| 403 Forbidden (frozen) | Subscription expired | Redeem voucher |
| 429 Too Many Requests | Rate limit hit | Tunggu 60s atau upgrade plan |
| 502 Bad Gateway | Backend service down | Cek `docker compose ps`, restart service |
| `/healthz` return 503 | Salah satu service down | Cek field `services` di response untuk tahu yang mana |
| n8n workflow tidak jalan | Workflow belum di-import | Hapus marker, restart container |
| Webhook 401 | Signature invalid | Cek `XENDIT_WEBHOOK_TOKEN` / `FONNTE_TOKEN` di `.env` |

---

## 9. Security Checklist

- [ ] Ganti `N8N_ADMIN_PASSWORD` di `.env` (default = `change_me_in_env`)
- [ ] Ganti `XENDIT_WEBHOOK_TOKEN` (generate dari Xendit dashboard)
- [ ] Ganti `FONNTE_TOKEN` (dari fonnte.com dashboard)
- [ ] n8n UI di-bind ke `127.0.0.1:5678` (tidak exposed publik) — akses via SSH tunnel atau Nginx dengan IP allowlist
- [ ] Webhook signature selalu diverifikasi di downstream service, bukan di gateway
- [ ] Redis password di-set di production (`REDIS_PASSWORD`)

---

## 10. Referensi Cepat

- **Routes**: Lihat `services/api-gateway/main.go:47-72`
- **n8n workflows**: `infra/n8n/workflows/`
- **Subscription worker**: `services/subscription-worker/main.go`
- **Billing endpoints**: `services/billing-service/main.go` (cari `handleAdmin*`)
- **Port registry**: `CLAUDE.md` (tabel Port Registry)
- **Fitur per service**: `docs/FEATURE_MAP.md`
