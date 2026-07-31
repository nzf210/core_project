# F025: Tier Restrictions Overhaul (Multimodal AI)

**Date:** 2026-06-15  
**Status:** ✅ Approved  
**Implementation:** ✅ Done (Phase 3 endpoints mocked, pending API key provisioning)  
**Related:** [F034](../FEATURE_MAP.md) (Addon Wallet), [F052](../FEATURE_MAP.md) (Addon Guard Foundation)

---

## 🎯 Objectives

Unify tier restrictions menjadi single source of truth dan siapkan fondasi untuk AI multimodal capabilities.

**Tujuan eksplisit:**
1. Sejajarkan Go `Plans` map dengan DB `plan_features` — satu source of truth untuk tier restrictions
2. Implement per-modality enforcement (text/vision/audio/image-gen) dengan quota tracking mechanism
3. Siapkan fondasi untuk AI multimodal (vision, STT/TTS, image gen) di `ultimate` tier

**Problem yang diselesaikan:**
- Tier restrictions hardcoded di banyak tempat (Go struct, switch/case, error messages) — tambah fitur baru butuh migration + code change
- Tidak ada Addon table untuk per-tenant feature unlock (F034 bikin wallet, F052 bikin guard, tapi belum ada unified system)
- Guard logic tersebar (HasFeatureAccess, CheckQuota, RequireFeature, RequireClinicType) — sulit maintain consistency
- Tidak ada quota tracking per-feature → tenant bisa abuse AI endpoint tanpa limit

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: DB-Driven Tier Restrictions**
  - *Verification:* `plan_features` table sebagai single source of truth, Go `Plans` map dihapus
  - *Example:* `GetPlanFeaturesRow(tier)` return struct dari DB query, bukan hardcoded map

- [x] **AC-2: Numeric Quota Columns**
  - *Verification:* `plan_features` table memiliki kolom quota per-modality (text, vision, audio, image_gen)
  - *Example:* `SELECT max_ai_requests_text, max_ai_requests_vision FROM plan_features WHERE plan_id = 'ultimate'`

- [x] **AC-3: Quota Counter Table**
  - *Verification:* `quota_counters` table track usage per-tenant per-feature per-period
  - *Example:* `INSERT INTO quota_counters (tenant_id, feature_key, period, count) VALUES (...)`

- [x] **AC-4: Quota Middleware**
  - *Verification:* AI Gateway endpoints protected dengan `QuotaMiddlewareFeature` — return 402 jika quota exceeded
  - *Example:* Tenant lite dengan 1000 AI requests → request ke-1001 return `402 Payment Required`

- [x] **AC-5: Redis Atomic Counter**
  - *Verification:* Quota increment via Redis INCR (atomic) → persist ke DB async untuk reporting
  - *Example:* `INCR quota:tenant_id:ai_text:2026-06` → real-time quota check tanpa DB hit

- [x] **AC-6: Chatbot Message Counter**
  - *Verification:* Setiap chatbot message processed → increment `chatbot_messages` counter
  - *Example:* Tenant process 50 WA messages → `quota_counters.count = 50`

- [x] **AC-7: Quota Usage Warning**
  - *Verification:* Notification service kirim warning saat tenant mencapai 80% quota (idempotent daily)
  - *Example:* Tenant lite 800/1000 AI requests → WA notification "Anda telah menggunakan 80% kuota AI..."

- [x] **AC-8: Superadmin Quota Dashboard**
  - *Verification:* Superadmin endpoint `/api/superadmin/quota/usage?tenant_id=...` return quota usage per-feature
  - *Example:* `{ ai_text: { used: 450, limit: 1000 }, chatbot_messages: { used: 120, limit: 500 } }`

- [x] **AC-9: Frontend Quota Display**
  - *Verification:* Settings page tampilkan quota usage dengan progress bar
  - *Example:* "AI Requests: 450/1000 (45%)" dengan progress bar hijau/kuning/merah

- [x] **AC-10: Multimodal Endpoint Stubs**
  - *Verification:* AI Gateway memiliki endpoints `/v1/vision`, `/v1/audio/stt`, `/v1/audio/tts`, `/v1/image/generate` (mocked response)
  - *Example:* `POST /v1/vision { image_url }` → return mock JSON (real implementation pending API key)

- [x] **AC-11: Chatbot Multimodal Config**
  - *Verification:* `tenant_chatbot_configs` memiliki toggle `enable_vision`, `enable_voice`
  - *Example:* ChatbotConfig.vue → checkbox "Enable Vision AI" → update DB column

- [x] **AC-12: WA Media Download Helper**
  - *Verification:* WhatsApp image/audio messages di-download via local tmp proxy → forward ke AI Gateway
  - *Example:* User kirim foto produk via WA → chatbot process via `/v1/vision` → reply deskripsi produk

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│         Plan Features (DB Single Source)            │
│  plan_features: plan_id, feature_key, is_enabled,   │
│    max_ai_requests_text, max_ai_requests_vision,    │
│    max_ai_requests_audio, max_image_gen             │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│            Quota Middleware (AI Gateway)            │
│  1. Get tenant tier → query plan_features           │
│  2. Check Redis: quota:tenant:feature:period        │
│  3. If exceeded → 402 Payment Required              │
│  4. If OK → INCR Redis → forward request            │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│         Quota Counters (Async DB Persist)           │
│  Cron job: Redis → DB sync every hour               │
│  quota_counters: tenant_id, feature_key, period,    │
│    count, created_at                                │
└─────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────┐
│    Notification Service (80% Warning)               │
│  Cron job: check quota usage daily                  │
│  If >= 80% → send WA notification (idempotent)      │
└─────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: 000035_quota_counters.up.sql
CREATE TABLE quota_counters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_key VARCHAR(50) NOT NULL,  -- 'ai_text', 'ai_vision', 'chatbot_messages', etc.
    period VARCHAR(20) NOT NULL,       -- '2026-06' (YYYY-MM format)
    count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_quota_counters_tenant_feature_period 
ON quota_counters(tenant_id, feature_key, period);

CREATE INDEX idx_quota_counters_period ON quota_counters(period);

-- Migration: 000036_multimodal_features.up.sql
ALTER TABLE plan_features 
ADD COLUMN max_ai_requests_text BIGINT DEFAULT 1000,
ADD COLUMN max_ai_requests_vision BIGINT DEFAULT 0,
ADD COLUMN max_ai_requests_audio BIGINT DEFAULT 0,
ADD COLUMN max_image_gen BIGINT DEFAULT 0;

-- Seed multimodal quotas
UPDATE plan_features SET 
  max_ai_requests_text = 1000, 
  max_ai_requests_vision = 0, 
  max_ai_requests_audio = 0, 
  max_image_gen = 0
WHERE plan_id = 'lite';

UPDATE plan_features SET 
  max_ai_requests_text = 5000, 
  max_ai_requests_vision = 100, 
  max_ai_requests_audio = 100, 
  max_image_gen = 0
WHERE plan_id = 'pro';

UPDATE plan_features SET 
  max_ai_requests_text = -1,  -- -1 = unlimited
  max_ai_requests_vision = 1000, 
  max_ai_requests_audio = 500, 
  max_image_gen = 100
WHERE plan_id = 'ultimate';

-- Add multimodal config to chatbot
ALTER TABLE tenant_chatbot_configs
ADD COLUMN enable_vision BOOLEAN DEFAULT false,
ADD COLUMN enable_voice BOOLEAN DEFAULT false;
```

### API Endpoints

#### `POST /v1/vision` (AI Gateway)

**Request:**
```json
{
  "image_url": "https://example.com/image.jpg",
  "prompt": "Describe this product"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "description": "A red t-shirt with logo on the front",
    "confidence": 0.92
  }
}
```

**Error Cases:**
- `402 Payment Required` — Quota exceeded: `{ message: "Kuota AI Vision habis. Upgrade ke Ultimate untuk lanjut." }`
- `403 Forbidden` — Feature not enabled for tier

#### `POST /v1/audio/stt` (Speech-to-Text)

**Request:**
```json
{
  "audio_url": "https://example.com/voice.ogg",
  "language": "id"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "transcript": "Halo, saya mau pesan nasi goreng"
  }
}
```

#### `POST /v1/audio/tts` (Text-to-Speech)

**Request:**
```json
{
  "text": "Terima kasih sudah order",
  "voice": "id-female"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "audio_url": "https://storage.example.com/tts/abc123.mp3"
  }
}
```

#### `POST /v1/image/generate`

**Request:**
```json
{
  "prompt": "A modern coffee shop logo with minimalist design",
  "size": "512x512"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "image_url": "https://storage.example.com/gen/xyz789.png"
  }
}
```

#### `GET /api/superadmin/quota/usage?tenant_id=...`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "tenant_id": "uuid",
    "period": "2026-06",
    "quotas": [
      {
        "feature_key": "ai_text",
        "used": 450,
        "limit": 1000,
        "percentage": 45
      },
      {
        "feature_key": "ai_vision",
        "used": 0,
        "limit": 0,
        "percentage": 0
      }
    ]
  }
}
```

### Redis Keys

| Key | Value | TTL | Usage |
|:----|:------|:----|:------|
| `quota:{tenant_id}:{feature}:{period}` | Integer (count) | End of month | Atomic INCR for quota check |
| `quota_warn:{tenant_id}:{period}` | Timestamp | 24h | Idempotent warning flag |

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (shared/sdk/auth):**
```go
// quota_test.go
func TestGetPlanFeaturesRow_DBDriven(t *testing.T) {
    // Mock DB query return plan_features row
    // Verify PlanFeaturesRow struct populated correctly
}

func TestQuotaMiddleware_ExceedsLimit(t *testing.T) {
    // Redis counter > plan limit
    // Expect 402 Payment Required
}

func TestQuotaMiddleware_UnlimitedTier(t *testing.T) {
    // plan.max_ai_requests_text = -1
    // Expect no 402 regardless of usage
}
```

**Integration Tests:**
```bash
# 1. Quota enforcement
curl -X POST http://localhost:8002/v1/chat/completions \
  -H "X-Tenant-ID: lite-tenant-uuid" \
  -d '{"prompt":"hello"}' 
# → 200 OK (request 1-1000)

# Simulate 1000 requests via Redis
redis-cli INCRBY quota:lite-tenant-uuid:ai_text:2026-06 1000

curl -X POST http://localhost:8002/v1/chat/completions \
  -H "X-Tenant-ID: lite-tenant-uuid" \
  -d '{"prompt":"hello"}'
# → 402 Payment Required

# 2. Multimodal endpoint stubs
curl -X POST http://localhost:8002/v1/vision \
  -d '{"image_url":"https://example.com/test.jpg","prompt":"describe"}'
# → 200 OK with mock response

# 3. Quota usage dashboard
curl http://localhost:8000/api/superadmin/quota/usage?tenant_id=lite-tenant-uuid \
  -H "Authorization: Bearer <superadmin_token>"
# → return usage breakdown
```

### Manual Testing Checklist

- [ ] Lite tenant hit 1000 AI requests → 402 error
- [ ] Ultimate tenant unlimited text → no 402 regardless of usage
- [ ] Chatbot process WA message → quota counter increment
- [ ] Settings page quota progress bar update real-time
- [ ] 80% quota warning sent once per day (idempotent)
- [ ] Superadmin quota dashboard show all tenants usage
- [ ] Multimodal toggle di ChatbotConfig save ke DB
- [ ] WA image message → chatbot call `/v1/vision` (mock response)

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Quota check", 
  "tenant_id", tenantID,
  "feature", featureKey,
  "used", usedCount,
  "limit", limitCount,
  "percentage", percentage)

slog.Warn("Quota exceeded", 
  "tenant_id", tenantID,
  "feature", featureKey,
  "used", usedCount,
  "limit", limitCount)
```

**Metrics to track:**
- Quota 402 error rate per tenant (detect abuse or need upgrade)
- Redis counter sync lag (Redis → DB persistence delay)
- Quota warning delivery success rate

**Alerts:**
- Quota 402 error rate > 10% for any tenant → investigate quota limit misconfiguration
- Redis counter sync failed → DB persistence issue

**Grafana Dashboard:**
- Panel 1: Quota usage heatmap (all tenants, all features)
- Panel 2: 402 error rate timeline
- Panel 3: Top 10 tenants by AI usage

---

## 🚀 Rollout Plan

### Phase 1: Align Source of Truth (Done ✅)
- Migration 000035: quota_counters table
- Migration 000036: multimodal quota columns
- Remove Go `Plans` map → use DB `plan_features` only
- Deploy: auth-service, billing-service dengan DB-driven quota logic

### Phase 2: Quota Counter Mechanism (Done ✅)
- Redis atomic counter + async DB persist
- Quota middleware di AI Gateway
- Notification service 80% warning
- Superadmin quota dashboard
- Frontend Settings quota display

### Phase 3: Multimodal Endpoints (Done ✅ — mocked)
- AI Gateway stubs: `/v1/vision`, `/v1/audio/*`, `/v1/image/generate`
- Chatbot multimodal config toggles (DB + UI)
- WA media download helper → forward ke AI Gateway
- **Pending:** Real API integration (MiniMax Vision, Whisper, ElevenLabs) — need API keys

### Phase 4: Production API Integration (Future)
- Provision API keys untuk MiniMax Vision, Whisper STT, ElevenLabs TTS, MiniMax Image-1
- Replace mock responses dengan real API calls
- Load testing untuk multimodal endpoints (latency, throughput)
- Cost monitoring (per-request API cost tracking)

### Rollback
- **Phase 1 rollback:** Revert migration 000036, restore Go `Plans` map dari git history
- **Phase 2 rollback:** Disable quota middleware via feature flag → allow unlimited usage (emergency)
- **Phase 3 rollback:** Set `enable_vision`, `enable_voice` columns ke `false` untuk all tenants

---

## 🔮 Future Enhancements (Out of Scope)

- **Dynamic Quota Adjustment:** Superadmin dapat adjust quota per-tenant via UI (override plan default)
- **Quota Pooling:** Shared quota untuk multi-store tenant (satu tenant, banyak store, quota dibagi)
- **Burst Quota:** Allow temporary quota overage (e.g., 110% limit) dengan charge ke wallet
- **Real-time Quota Dashboard:** WebSocket push untuk live quota updates di frontend (no refresh needed)
- **API Key Management UI:** Tenant bisa input own API key untuk MiniMax/OpenAI → bypass WCH quota

---

## 📚 References

- [F034: Addon Wallet](../FEATURE_MAP.md) — Wallet top-up system untuk addon purchase
- [F052: Addon Guard Foundation](../FEATURE_MAP.md) — Guard logic foundation untuk addon checking
- [MiniMax API Docs](https://www.minimaxi.com/document/guides/chat-model/V2) — Vision model integration reference
- [Whisper OpenAI](https://platform.openai.com/docs/guides/speech-to-text) — STT API reference

---

## 📝 Notes & Decisions

**2026-06-15:** Decision: Quota limit `-1` = unlimited (bukan `NULL`) untuk consistency dengan numeric type. Simplify SQL query (`WHERE count < limit OR limit = -1`).  
**2026-06-15:** Redis atomic INCR → DB async persist every hour (bukan real-time INSERT) untuk reduce DB load. Trade-off: reporting lag 1 jam max.  
**2026-06-15:** 80% warning idempotent daily (bukan per-increment) → avoid spam. Use Redis flag `quota_warn:{tenant}:{period}` TTL 24h.  
**2026-06-15:** Phase 3 multimodal endpoints mocked — real API integration defer sampai API key provisioned + budget approved. Mock response structure final, tinggal swap implementation.  
**2026-06-15:** Vendor choice (MiniMax Vision, Whisper, ElevenLabs) pending owner confirmation. Architecture designed vendor-agnostic via `/v1/vision` abstraction — mudah swap provider nanti.
