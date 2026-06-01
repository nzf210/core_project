# 🗺️ WCH Platform — Feature Location Map (Cheat Sheet)

> **Quick reference:** Di mana saya harus menulis kode untuk fitur ini?
> Untuk panduan lengkap dengan contoh kode, lihat [CONTRIBUTING.md](../CONTRIBUTING.md).

---

## Mau Tambah Endpoint/API Baru?

```
┌─────────────────────────────────────────────────────────────────┐
│  Ini untuk produk mana?                                         │
│                                                                 │
│  UMKM Accounting ──→ apps/umkm/accounting/main.go (flat)       │
│  UMKM Business   ──→ apps/umkm/business/main.go (flat)         │
│  UMKM Chatbot    ──→ apps/umkm/chatbot/main.go (flat)          │
│  UMKM Automation ──→ apps/umkm/automation/main.go (worker)     │
│                                                                 │
│  Campaign API    ──→ apps/campaign/api/handlers/<nama>.go       │
│                      + daftarkan di apps/campaign/api/main.go  │
│                                                                 │
│  Crypto API      ──→ apps/crypto/api/handlers.go               │
│  Crypto Worker   ──→ apps/crypto/worker/<engine>.go            │
│                                                                 │
│  Auth / JWT      ──→ services/auth-service/main.go             │
│  AI Proxy        ──→ services/ai-gateway/main.go               │
│  Billing/Xendit  ──→ services/billing-service/main.go          │
│  WA (whatsmeow)  ──→ services/wa-gateway/main.go              │
│  Notifikasi      ──→ services/notification-service/main.go     │
│  API Gateway     ──→ services/api-gateway/main.go              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Mau Tambah Tabel Database?

```
# Cari nomor migration terakhir:
ls shared/migrations/*.up.sql | tail -1

# Buat file baru:
shared/migrations/NNNNNN_nama_fitur.up.sql
shared/migrations/NNNNNN_nama_fitur.down.sql
```

> Lihat [CONTRIBUTING.md → Cara Menambah Tabel](../CONTRIBUTING.md#️-cara-menambah-tabel-database) untuk aturan lengkap.

---

## Mau Tambah/Ubah Config?

```
1. shared/sdk/config/config.go  ← Tambah field struct + LoadConfig()
2. .env.example                 ← Tambah dengan nilai contoh
3. docker-compose.yml           ← Tambah environment variable
```

---

## Mau Tambah UI Frontend?

```
┌─────────────────────────────────────────────────────────────────┐
│  UMKM     ──→ frontend/umkm-web/src/components/<Nama>.vue       │
│                + import & daftarkan di src/App.vue              │
│  Crypto   ──→ frontend/crypto-web/src/                         │
│  Campaign ──→ frontend/campaign-web/src/                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Mau Tambah Service/Worker Baru?

```
WAJIB update semua file ini:
☐ Makefile                      ← Tambah run target
☐ Dockerfile                    ← Tambah go build + COPY
☐ docker-compose.yml            ← Tambah service definition
☐ services/api-gateway/main.go  ← Tambah proxy route (jika REST API)
☐ CLAUDE.md                     ← Update Port Registry
☐ .env.example                  ← Tambah env vars baru
```

---

## 📡 Port Allocation

| Port | Service | Type | Direktori |
|:-----|:--------|:-----|:----------|
| `8000` | API Gateway | REST Proxy | `services/api-gateway` |
| `8001` | Auth Service | REST API | `services/auth-service` |
| `8002` | AI Gateway | REST API | `services/ai-gateway` |
| `8003` | Billing Service | REST API | `services/billing-service` |
| `8005` | Notification Service | REST API | `services/notification-service` |
| `8101` | Crypto API | REST API | `apps/crypto/api` |
| `8201` | UMKM Accounting | REST API | `apps/umkm/accounting` |
| `8202` | WA Gateway | WebSocket/REST | `services/wa-gateway` |
| `8202` | UMKM Chatbot | REST API | `apps/umkm/chatbot` ⚠️ |
| `9001` | UMKM Business | REST API | `apps/umkm/business` |
| `9002` | Campaign API | REST API | `apps/campaign/api` |
| `3101` | Frontend Crypto | Vite Dev | `frontend/crypto-web` |
| `3201` | Frontend UMKM | Vite Dev | `frontend/umkm-web` |
| `3301` | Frontend Campaign | Vite Dev | `frontend/campaign-web` |
| `5433` | PostgreSQL (Docker) | DB | docker-compose |
| `6381` | Redis (Docker) | Cache | docker-compose |

> ⚠️ WA Gateway dan UMKM Chatbot keduanya menggunakan port 8202. Jalankan hanya satu pada satu waktu saat dev lokal.

### API Gateway Routing (Port 8000)

Requests ke API Gateway diarahkan berdasarkan prefix URL:

| Prefix URL | Target Service | Port |
|:-----------|:--------------|:-----|
| `/api/v1/auth/*` | Auth Service | `8001` |
| `/api/v1/ai/*` | AI Gateway | `8002` |
| `/api/v1/billing/*` | Billing Service | `8003` |
| `/api/v1/accounting/*` | UMKM Accounting | `8201` |
| `/api/v1/campaign/*` | Campaign API | `9002` |
| `/api/v1/crypto/*` | Crypto API | `8101` |

---

## Multi-Store (UMKM)

1 owner bisa membuat banyak toko dengan business_type berbeda (restoran + cafe, dll) di bawah 1 subscription. Quota di-enforce via `plan_features.feature_key='max_stores'`.

| Endpoint | Method | Tujuan |
|:---------|:-------|:-------|
| `/api/umkm/stores` | GET | List semua toko milik owner (include `max_stores` & `can_add` quota info) |
| `/api/umkm/stores` | POST | Buat toko baru (cek quota) |
| `/api/umkm/stores/{id}` | GET | Detail 1 toko |
| `/api/umkm/stores/{id}` | PATCH | Update nama/alamat/business_type/is_active |
| `/api/umkm/stores/{id}` | DELETE | Hapus toko |

**Quota Default:**

| Tier | max_stores |
|:-----|:-----------|
| Lite | 1 |
| Pro | 1 |
| Business | 5 |

Quota di-baca langsung dari `plan_features` table — superadmin bisa ubah via endpoint `/admin/plan-features` di `services/billing-service/` (port 8003) tanpa tulis migration baru.

**Superadmin Dynamic Plan Features:**

| Endpoint | Method | Tujuan |
|:---------|:-------|:-------|
| `/admin/plan-features?plan_id=business` | GET | List features per tier (semua kalau tanpa filter) |
| `/admin/plan-features` | POST | Upsert feature (insert atau update kalau `(plan_id, feature_key)` sudah ada) |
| `/admin/plan-features/{id}` | PATCH | Edit nama/value/is_enabled |
| `/admin/plan-features/{id}` | DELETE | Hapus feature |

Header wajib: `X-User-Role: superadmin` (di-inject otomatis oleh api-gateway dari JWT claim).

---

## Voucher Link (Subscription Activation)

**Model:** Voucher = primary activation. Akun freeze otomatis saat `current_period_end` lewat (grace 0 hari). Freeze = read-only + banner, user masih bisa login dan lihat data historis.

**Voucher Lifecycle:**

```
[Superadmin] /admin/voucher-links/generate
    { program_id, count, valid_days, base_url }
    → Returns: { count, expires_at, links: [{token, url}, ...] }
    → Distribute via WA/email/reseller

[User] Klik link → POST /voucher/redeem-link { token, tenant_id }
    1. Verify JWT signature
    2. Lookup voucher_links by SHA-256(token)
    3. Validate: is_active, not redeemed, not expired
    4. Check max_uses_per_tenant (default 1)
    5. Mark link redeemed (tx)
    6. Extend or create subscription (tx):
       - If existing active: current_period_end += duration_months
       - Else: new subscription with period_end = NOW() + duration
    7. Un-freeze tenant if was frozen
    8. Generate subscription_tickets + notification
```

**Key Tables:**

| Table | Purpose |
|:------|:--------|
| `voucher_programs` | Program definition (type, plan, duration, max_uses) |
| `voucher_links` | Individual links (token_hash, expires_at, redeemed_by) |
| `voucher_generation_logs` | Audit trail (siapa generate berapa) |
| `tenant_subscriptions.status` | `active` | `frozen` | `cancelled` |
| `tenants.is_frozen` | Denormalized flag (fast read by middleware) |

**Endpoints:**

| Endpoint | Method | Auth | Purpose |
|:---------|:-------|:-----|:--------|
| `/voucher/redeem-link` | POST | public (signed token) | Customer redeem |
| `/admin/voucher-links/generate` | POST | superadmin | Bulk generate |
| `/admin/voucher-links?program_id=X` | GET | superadmin | List links per program |
| `/admin/dashboard` | GET | superadmin | Aggregated overview |

**Freeze Worker (`services/subscription-worker/`, port 8006):**

- Cek `tenant_subscriptions` setiap `FREEZE_CHECK_INTERVAL` (default 1 jam)
- Subscription dengan `current_period_end < NOW() - GRACE_PERIOD_HOURS` → freeze
- Batch update: status='frozen', tenants.is_frozen=true
- Liveness: GET `/healthz`
- Env: `FREEZE_CHECK_INTERVAL`, `GRACE_PERIOD_HOURS`

**Read-only Enforcement (`shared/sdk/auth/subscription_guard.go`):**

Middleware `auth.RequireActiveSubscription` — block POST/PATCH/PUT/DELETE saat frozen, GET tetap pass. Set header `X-Subscription-Status: active|frozen`. Pakai pattern:

```go
handler := auth.RequireActiveSubscription(auth.Middleware(mux))
```

**Superadmin Dashboard (`frontend/superadmin-web/`, port 3401):**

1 unified dashboard (bukan per-product). Sections:
- **Overview** — tenant counts, voucher stats 30d, revenue (Xendit), subs by plan, recent frozen
- **Voucher Programs** — list, create new
- **Generate Links** — bulk generate + download CSV
- **Frozen Accounts** — list + kirim reminder WA

---

## Pola Kode (Pattern Reference)

| Situasi | Lihat Contoh |
|:--------|:-------------|
| App flat (handler + router di main.go) | `apps/umkm/accounting/main.go` |
| App terstruktur (handler terpisah) | `apps/campaign/api/handlers/voter.go` |
| Service sederhana | `services/notification-service/main.go` |
| Background worker (ticker) | `apps/umkm/automation/main.go` |
| Background worker (Redis pubsub) | `apps/crypto/worker/main.go` |
| Enkripsi AES-256-GCM | `apps/crypto/domain/encryption.go` |
| JWT middleware | `shared/sdk/auth/` |
| Standard response helper | `apps/campaign/api/handlers/responses.go` |

---

*Lihat [CONTRIBUTING.md](../CONTRIBUTING.md) untuk panduan lengkap dengan contoh kode.*
