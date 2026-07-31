# F065: Landing Page Content Management — Superadmin JSON Editor

**Date:** 2026-06-29  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F056](../FEATURE_MAP.md) (Dark/Light Mode Theme)

---

## 🎯 Objectives

Landing page content dapat di-manage oleh superadmin melalui JSON editor tanpa perlu edit kode atau re-deploy.

**Tujuan eksplisit:**
1. Superadmin dapat edit konten landing page (hero, features, steps, testimonials, CTA, footer) via dashboard UI
2. Perubahan konten langsung tampil di landing page publik tanpa perlu restart service
3. Content bersifat dynamic dari database dengan caching 6 jam untuk performance

**Problem yang diselesaikan:**
- Landing page content hardcoded di Vue component → setiap edit konten perlu commit + deploy
- Marketing team tidak bisa A/B test copy tanpa melibatkan developer
- SEO optimization dan seasonal campaign update terlalu slow karena harus melalui dev cycle

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Public Endpoint untuk Landing Page**
  - *Verification:* `GET /landing-configs` return semua section content (hero, features, steps, testimonials, cta, footer) tanpa auth
  - *Example:* `curl http://localhost:8000/landing-configs` → `{"hero": {...}, "features": [...]}`

- [x] **AC-2: Superadmin JSON Editor UI**
  - *Verification:* Superadmin dashboard memiliki section "Landing Page Editor" dengan 6 tabs (hero, features, steps, testimonials, cta, footer) + JSON editor per-tab
  - *Example:* Edit hero section → update title dari "Aplikasi Kasir UMKM" ke "Aplikasi POS Modern" → Save → refresh landing page → title berubah

- [x] **AC-3: Cache Invalidation**
  - *Verification:* Setiap kali superadmin update content via PUT → cache auto-invalidate → GET berikutnya serve fresh data
  - *Example:* `PUT /api/superadmin/landing-configs?id=hero` → response header `X-Cache-Invalidated: true` → `GET /landing-configs` return new content

- [x] **AC-4: Fallback Static Content**
  - *Verification:* Jika API gagal → landing page render static fallback content (hardcoded di LandingPage.vue)
  - *Example:* Kill billing-service → refresh landing page → tetap tampil dengan default content (tidak blank)

- [x] **AC-5: Authorization Guard**
  - *Verification:* Hanya superadmin yang bisa `PUT /api/superadmin/landing-configs` (JWT role check)
  - *Example:* Tenant owner coba akses editor → 403 Forbidden

- [x] **AC-6: Response Time < 100ms (Cached)**
  - *Verification:* `GET /landing-configs` dengan cache HIT → response time < 100ms
  - *Example:* `curl -w "%{time_total}" http://localhost:8000/landing-configs` → `0.015s`

- [x] **AC-7: Dark/Light Mode Compatible**
  - *Verification:* Landing page content render correctly di dark mode dan light mode (reuse F056 theme variables)
  - *Example:* Toggle dark mode → background color, text color, card shadow adjust otomatis

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│         Superadmin Dashboard (JSON Editor)          │
│  PUT /api/superadmin/landing-configs?id=hero        │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│            API Gateway :8000 (Proxy)                │
│  /api/superadmin/landing-configs → billing:8003     │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│         Billing Service :8003 (Handlers)            │
│  ┌──────────────────────────────────────────────┐  │
│  │ In-Memory Cache (sync.RWMutex, 6h TTL)      │  │
│  │  Key: "landing_configs"                     │  │
│  │  Value: map[string]interface{}              │  │
│  └──────────────────────────────────────────────┘  │
│              ↓ (cache MISS)                         │
│  ┌──────────────────────────────────────────────┐  │
│  │ PostgreSQL: landing_configs table           │  │
│  │  - id (hero, features, steps, ...)          │  │
│  │  - content (JSONB)                           │  │
│  │  - is_active (BOOLEAN)                       │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
         ↑ (public read)
┌─────────────────────────────────────────────────────┐
│       Landing Page (Public, No Auth)                │
│  GET /landing-configs → render with fallback        │
└─────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: 000083_landing_configs.up.sql
CREATE TABLE landing_configs (
    id         VARCHAR(50) PRIMARY KEY,  -- 'hero', 'features', 'steps', 'testimonials', 'cta', 'footer'
    content    JSONB NOT NULL DEFAULT '{}',
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default content
INSERT INTO landing_configs (id, content) VALUES
('hero', '{"title":"Aplikasi Kasir, Pembukuan & AI untuk UMKM","subtitle":"Kelola transaksi, stok, dan pelanggan dalam satu platform","cta_text":"Coba Gratis","cta_url":"/register"}'),
('features', '[{"icon":"💳","title":"POS Modern","description":"Kasir cepat dengan barcode scanner"},{"icon":"📊","title":"Akuntansi Double-Entry","description":"Laporan keuangan akurat otomatis"},{"icon":"🤖","title":"AI Chatbot WhatsApp","description":"Customer service 24/7 tanpa operator"}]'),
('steps', '[{"number":1,"title":"Daftar Gratis","description":"Buat akun dalam 30 detik"},{"number":2,"title":"Setup Produk","description":"Import katalog atau input manual"},{"number":3,"title":"Mulai Jual","description":"Terima pembayaran, kirim invoice, otomatis"}]'),
('testimonials', '[{"name":"Ibu Siti","business":"Warung Makan Padang","quote":"Pembukuan jadi gampang, untung/rugi langsung keliatan"}]'),
('cta', '{"title":"Siap meningkatkan bisnis Anda?","subtitle":"Gabung 500+ UMKM yang sudah berkembang bersama WCH","button_text":"Mulai Sekarang","button_url":"/register"}'),
('footer', '{"company":"PT WCH Indonesia","tagline":"Solusi Digital untuk UMKM","links":[{"label":"Tentang Kami","url":"/about"},{"label":"Kontak","url":"/contact"},{"label":"Syarat & Ketentuan","url":"/terms"}]}');

CREATE INDEX idx_landing_configs_is_active ON landing_configs(is_active);
```

**Migration down:**
```sql
-- 000083_landing_configs.down.sql
DROP TABLE IF EXISTS landing_configs;
```

### API Endpoints

#### `GET /landing-configs` (Public)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "hero": {
      "title": "Aplikasi Kasir, Pembukuan & AI untuk UMKM",
      "subtitle": "Kelola transaksi, stok, dan pelanggan dalam satu platform",
      "cta_text": "Coba Gratis",
      "cta_url": "/register"
    },
    "features": [
      {
        "icon": "💳",
        "title": "POS Modern",
        "description": "Kasir cepat dengan barcode scanner"
      }
    ],
    "steps": [...],
    "testimonials": [...],
    "cta": {...},
    "footer": {...}
  }
}
```

**Response Headers:**
- `X-Cache: HIT` (jika dari cache) atau `X-Cache: MISS` (jika dari DB)
- `Cache-Control: public, max-age=21600` (6 jam)

**Error Cases:**
- `500 Internal Server Error` — DB error (frontend render fallback content)

#### `GET /api/superadmin/landing-configs` (Superadmin only)

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "hero",
      "content": {...},
      "is_active": true,
      "updated_at": "2026-06-29T10:30:00Z"
    },
    {
      "id": "features",
      "content": [...],
      "is_active": true,
      "updated_at": "2026-06-29T10:30:00Z"
    }
  ]
}
```

**Error Cases:**
- `401 Unauthorized` — Missing/invalid JWT token
- `403 Forbidden` — User role bukan `superadmin`

#### `PUT /api/superadmin/landing-configs?id=hero` (Superadmin only)

**Request:**
```json
{
  "content": {
    "title": "Solusi POS & Akuntansi Terlengkap",
    "subtitle": "Trusted by 500+ UMKM",
    "cta_text": "Daftar Sekarang",
    "cta_url": "/register"
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Landing config 'hero' updated successfully",
  "data": {
    "id": "hero",
    "content": {...},
    "updated_at": "2026-06-29T11:00:00Z"
  }
}
```

**Response Headers:**
- `X-Cache-Invalidated: true`

**Error Cases:**
- `400 Bad Request` — Invalid JSON structure
- `401 Unauthorized` — Missing/invalid JWT token
- `403 Forbidden` — User role bukan `superadmin`
- `404 Not Found` — `id` tidak ditemukan di table

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (billing-service):**
```go
// landing_config_handlers_test.go
func TestHandleLandingConfigsPublic(t *testing.T) {
    // Cache HIT scenario
    // Cache MISS scenario
    // DB error → return 500
}

func TestHandleLandingConfigsAdmin(t *testing.T) {
    // Superadmin list all configs
    // Non-superadmin → 403
}

func TestHandleUpdateLandingConfig(t *testing.T) {
    // Valid update → cache invalidate
    // Invalid JSON → 400
    // Unknown id → 404
    // Non-superadmin → 403
}
```

**Frontend (umkm-web):**
```typescript
// LandingPage.spec.ts
describe('LandingPage', () => {
  it('renders dynamic content from API', async () => {
    // Mock getLandingConfigs() → verify hero title rendered
  })

  it('falls back to static content when API fails', async () => {
    // Mock API error → verify fallback content rendered
  })

  it('respects dark/light mode theme', () => {
    // Toggle theme → verify CSS variables applied
  })
})
```

### Integration Tests

```bash
# 1. Public endpoint accessible tanpa auth
curl http://localhost:8000/landing-configs
# → 200 OK, return all sections

# 2. Superadmin update content
TOKEN=$(curl -X POST http://localhost:8001/superadmin-login \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

curl -X PUT "http://localhost:8000/api/superadmin/landing-configs?id=hero" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":{"title":"New Title"}}' 
# → 200 OK, X-Cache-Invalidated: true

# 3. Verify cache invalidated
curl http://localhost:8000/landing-configs | jq '.data.hero.title'
# → "New Title"

# 4. Non-superadmin forbidden
TENANT_TOKEN=$(curl -X POST http://localhost:8001/login ...)
curl -X PUT "http://localhost:8000/api/superadmin/landing-configs?id=hero" \
  -H "Authorization: Bearer $TENANT_TOKEN" 
# → 403 Forbidden
```

### Manual Testing Checklist

- [ ] Landing page render correctly dengan dynamic content
- [ ] Superadmin JSON editor load existing content
- [ ] Edit hero title → save → landing page update tanpa refresh (if using WebSocket) atau setelah refresh
- [ ] Kill billing-service → landing page tetap tampil dengan fallback content
- [ ] Dark/light mode toggle → landing page color adjust correctly
- [ ] Cache header `X-Cache: HIT` muncul setelah GET kedua kalinya (dalam 6 jam)
- [ ] Non-superadmin coba akses editor → redirect atau error message

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Landing config fetched", 
  "cache_status", cacheStatus,  // "HIT" or "MISS"
  "latency_ms", latency)

slog.Info("Landing config updated", 
  "id", configID,
  "updated_by", userID,
  "cache_invalidated", true)
```

**Metrics to track:**
- Cache hit rate (target: >95% karena landing page content jarang update)
- GET `/landing-configs` latency (target: p95 < 50ms untuk cache HIT)
- PUT update frequency (berapa kali per hari superadmin edit content)

**Alerts:**
- Cache hit rate < 80% → investigate cache TTL atau load pattern
- Landing configs GET error rate > 1% → DB connectivity issue

**Grafana Dashboard:**
- Panel 1: Landing page request rate + cache hit/miss split
- Panel 2: Content update timeline (PUT events)
- Panel 3: Response time histogram (HIT vs MISS)

---

## 🚀 Rollout Plan

### Phase 1: Backend + Migration (Done ✅)
- Deploy migration 000083 → create `landing_configs` table + seed default content
- Deploy billing-service dengan handlers baru
- Deploy api-gateway dengan proxy routes

### Phase 2: Superadmin UI (Done ✅)
- Deploy superadmin-web dengan Landing Page Editor section
- Test: Superadmin edit hero → verify API call + cache invalidate

### Phase 3: Public Landing Page Integration (Done ✅)
- Deploy umkm-web dengan dynamic content fetch dari `GET /landing-configs`
- Fallback static content tetap ada di component (fail-safe)
- Verify: Landing page load < 2s dengan cache HIT

### Phase 4: Monitoring (Current)
- Add Grafana dashboard untuk landing page metrics
- Add alert rule untuk cache hit rate anomaly

### Rollback
- **Config rollback:** Superadmin re-edit content via UI → restore previous version (no DB rollback needed)
- **Code rollback:** Revert api-gateway + billing-service → landing page fallback ke static content di LandingPage.vue
- **Emergency:** `UPDATE landing_configs SET is_active = false WHERE id = 'problematic_section'` → frontend skip section yang error

---

## 🔮 Future Enhancements (Out of Scope)

- **Versioning:** Track edit history → rollback ke version sebelumnya (audit log + restore button)
- **A/B Testing:** Multiple variant per section → serve different content ke different user segment
- **Preview Mode:** Superadmin preview perubahan sebelum publish (draft vs published state)
- **Rich Text Editor:** Upgrade dari JSON editor ke WYSIWYG editor (TipTap atau Quill.js)
- **Image Upload:** Support upload image untuk hero background, testimonial avatar via Cloudinary/S3
- **Multi-Language:** Support i18n → content per language (id, en) → detect dari browser locale

---

## 📚 References

- [F056: Dark/Light Mode Theme](../FEATURE_MAP.md) — CSS variables untuk theming
- [LandingPage.vue Implementation](../../frontend/umkm-web/src/components/LandingPage.vue) — Frontend component dengan fallback logic
- [Billing Service Handlers](../../services/billing-service/landing_config_handlers.go) — Backend handlers + caching logic
- [CommonMark Spec](https://spec.commonmark.org/) — Markdown standard untuk content formatting

---

## 📝 Notes & Decisions

**2026-06-29:** Decision: In-memory cache (bukan Redis) karena landing page content load frequency rendah + billing-service single instance di staging. Jika scale horizontal → migrate ke Redis Cluster.  
**2026-06-29:** Cache TTL 6 jam → balance antara freshness dan performance. Marketing team jarang update content >1x per hari.  
**2026-06-29:** Fallback static content wajib ada di LandingPage.vue → prevent blank page jika billing-service down atau DB migration belum jalan.  
**2026-06-29:** JSON editor (bukan WYSIWYG) untuk MVP → superadmin comfortable dengan JSON structure. Rich text editor defer ke v2 berdasarkan user feedback.
