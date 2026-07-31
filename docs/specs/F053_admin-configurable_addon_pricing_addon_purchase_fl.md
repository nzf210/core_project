# F053: Admin-Configurable Addon Pricing + Addon Purchase Flow

**Date:** 2026-06-16  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F034](../FEATURE_MAP.md) (Wallet), [F052](../FEATURE_MAP.md) (Tier-First Feature System), [F054](F054_referral_system_discount_downline_commission_uplin.md) (Referral Discount)

---

## 🎯 Objectives

Superadmin dapat configure addon pricing via UI, tenant dapat beli addon via wallet, dan feature gating system yang extensible tanpa code change.

**Tujuan eksplisit:**
1. Superadmin atur harga addon di `available_features` table (dynamic pricing, no hardcoded values)
2. Tenant beli addon dari marketplace page → wallet deducted → `tenant_addons` row active → feature langsung available
3. Feature gating system (`CanUseFeature`) support bundled (plan) + addon (per-tenant) features — zero-hardcode extensibility

**Problem yang diselesaikan:**
- Addon pricing hardcoded di backend → setiap ubah harga perlu redeploy
- Feature gating logic tersebar (HasFeatureAccess, CheckQuota, RequireFeature) → sulit maintain consistency
- Tambah feature baru butuh DB migration + code change (Go struct + switch/case) → slow iteration

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: available_features Table**
  - *Verification:* Table `available_features` store feature metadata (feature_key, name, category, is_addon, price_cents, addon_unit)
  - *Example:* Row: `{ feature_key: 'ai_vision', name: 'AI Vision', is_addon: true, price_cents: 50000000, addon_unit: 'per_month' }`

- [x] **AC-2: tenant_addons Table**
  - *Verification:* Table `tenant_addons` track per-tenant addon ownership (tenant_id, addon_key, status, expires_at, auto_renew)
  - *Example:* Tenant beli `ai_vision` → INSERT row dengan `status='active'`, `expires_at=NOW()+30 days`

- [x] **AC-3: CanUseFeature Guard (Unified Logic)**
  - *Verification:* `CanUseFeature(tenantID, featureKey)` cek plan_features (bundled) OR tenant_addons (purchased) — return true jika salah satu allow
  - *Example:* Feature `ai_vision` OFF di plan lite → tenant beli addon → `CanUseFeature('ai_vision')` return true

- [x] **AC-4: GET Addon Marketplace**
  - *Verification:* `GET /api/addon-marketplace` return list addons dengan status (available, owned, expired)
  - *Example:* Response: `[{ addon_key: 'ai_vision', name: 'AI Vision', price_cents: 50000000, has_addon: false, addon_status: null }]`

- [x] **AC-5: POST Addon Purchase**
  - *Verification:* `POST /addons/purchase { addon_key }` → validate wallet balance → deduct → INSERT `tenant_addons` → return success
  - *Example:* Tenant wallet Rp 600.000, addon price Rp 500.000 → deduct → balance Rp 100.000 → addon active

- [x] **AC-6: Wallet Balance Validation**
  - *Verification:* Purchase dengan insufficient balance → 402 Payment Required
  - *Example:* Balance Rp 100.000, addon price Rp 500.000 → 402 "Saldo tidak cukup. Silakan top-up wallet."

- [x] **AC-7: Duplicate Purchase Prevention**
  - *Verification:* Purchase addon yang sudah active → 409 Conflict
  - *Example:* Tenant sudah punya `ai_vision` active → POST purchase → 409 "Addon sudah aktif"

- [x] **AC-8: Referral Discount Applied**
  - *Verification:* Tenant dengan `referred_by_affiliate_id` → apply discount before wallet deduct (F054 integration)
  - *Example:* Addon Rp 500.000 → discount 10% → final Rp 450.000 → deduct dari wallet

- [x] **AC-9: Frontend Addons Page**
  - *Verification:* `/addons` page tampilkan marketplace dengan list addons, status badge (Available/Active/Expired), tombol "Beli"
  - *Example:* Card addon `AI Vision` → badge "Available" → button "Beli Rp 500.000" → click → confirm modal → purchase

- [x] **AC-10: Superadmin Addon Config UI (Future)**
  - *Verification:* Superadmin dashboard page untuk edit `available_features` (add/edit/delete addon, set price, set min_tier)
  - *Example:* Superadmin edit `ai_vision` price dari Rp 500k → Rp 400k → save → tenant berikutnya bayar Rp 400k

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│      Feature Gating Guard (shared/sdk/auth)         │
│  CanUseFeature(tenantID, "ai_vision"):              │
│    1. Check plan_features (bundled)                 │
│    2. Check tenant_addons (purchased)               │
│    3. Return true if EITHER allow                   │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Frontend (/addons Marketplace)                 │
│  GET /api/addon-marketplace                         │
│  → List addons dengan status per-tenant             │
│  → Button "Beli" → POST /addons/purchase            │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Backend (billing-service Addon Purchase)       │
│  POST /addons/purchase { addon_key }:               │
│    1. Validate addon exists in available_features   │
│    2. Check tenant NOT already have active addon    │
│    3. Apply referral discount (F054)                │
│    4. Check wallet balance >= final_price           │
│    5. Deduct wallet → INSERT tenant_addons          │
│    6. Invalidate addon cache                        │
└─────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: 000068_tier_addon_system.up.sql

CREATE TABLE available_features (
    feature_key VARCHAR(50) PRIMARY KEY,
    feature_name VARCHAR(200) NOT NULL,
    category VARCHAR(50) NOT NULL,  -- 'bundled', 'addon', 'storage'
    is_addon BOOLEAN NOT NULL DEFAULT false,
    min_tier VARCHAR(20),  -- 'lite', 'pro', 'ultimate', NULL (available untuk all)
    price_cents BIGINT,  -- NULL for bundled features
    addon_unit VARCHAR(20),  -- 'per_month', 'per_year', 'one_time'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_available_features_category ON available_features(category);
CREATE INDEX idx_available_features_is_addon ON available_features(is_addon);

CREATE TABLE tenant_addons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    addon_key VARCHAR(50) NOT NULL REFERENCES available_features(feature_key),
    status VARCHAR(20) NOT NULL DEFAULT 'active',  -- 'active', 'expired', 'cancelled'
    purchased_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    auto_renew BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tenant_addons_tenant_addon ON tenant_addons(tenant_id, addon_key) WHERE status = 'active';
CREATE INDEX idx_tenant_addons_tenant ON tenant_addons(tenant_id);
CREATE INDEX idx_tenant_addons_status ON tenant_addons(status);
CREATE INDEX idx_tenant_addons_expires_at ON tenant_addons(expires_at);

-- Seed default addons
INSERT INTO available_features (feature_key, feature_name, category, is_addon, min_tier, price_cents, addon_unit) VALUES
    ('ai_vision',        'AI Vision Analysis',            'addon',     true,  NULL,                           50000000, 'per_month'),
    ('ai_voice_stt',     'AI Voice to Text (STT)',        'addon',     true,  NULL,                           30000000, 'per_month'),
    ('ai_voice_tts',     'AI Text to Voice (TTS)',        'addon',     true,  NULL,                           30000000, 'per_month'),
    ('ai_image_gen',     'AI Image Generation',           'addon',     true,  NULL,                           40000000, 'per_month'),
    ('extra_store',      'Extra POS Store',               'storage',   true,  NULL,                           5000000,  'per_month'),
    ('extra_user',       'Extra User Seat',               'storage',   true,  NULL,                           1000000,  'per_month');
```

### API Endpoints

#### `GET /api/addon-marketplace`

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "addon_key": "ai_vision",
      "feature_name": "AI Vision Analysis",
      "category": "addon",
      "price_cents": 50000000,
      "addon_unit": "per_month",
      "has_addon": false,
      "addon_status": null
    },
    {
      "addon_key": "ai_voice_stt",
      "feature_name": "AI Voice to Text (STT)",
      "category": "addon",
      "price_cents": 30000000,
      "addon_unit": "per_month",
      "has_addon": true,
      "addon_status": "active",
      "expires_at": "2026-07-16T10:30:00Z"
    }
  ]
}
```

**Error Cases:**
- `401 Unauthorized` — Missing/invalid JWT token

#### `POST /addons/purchase`

**Request:**
```json
{
  "addon_key": "ai_vision"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Addon purchased successfully",
  "data": {
    "addon_key": "ai_vision",
    "expires_at": "2026-07-16T10:30:00Z",
    "amount_paid_cents": 45000000
  }
}
```

**Error Cases:**
- `400 Bad Request` — Invalid addon_key (not exists in available_features)
- `402 Payment Required` — Insufficient wallet balance
- `409 Conflict` — Addon already active for tenant
- `401 Unauthorized` — Missing/invalid JWT token

### Feature Gating Logic

**Backend Guard (shared/sdk/auth/can_use.go):**
```go
func CanUseFeature(ctx context.Context, tenantID, featureKey string) (bool, string) {
    tier := GetTenantPlan(ctx, tenantID)
    
    // Check bundled features (plan_features)
    row, _ := GetPlanFeaturesRow(ctx, tier)
    if row.IsFeatureEnabled(featureKey) {
        return true, ""
    }
    
    // Check purchased addons (tenant_addons)
    var count int
    err := DB.QueryRow(ctx, `
        SELECT COUNT(1) FROM tenant_addons 
        WHERE tenant_id = $1 
          AND addon_key = $2 
          AND status = 'active' 
          AND expires_at > NOW()
    `, tenantID, featureKey).Scan(&count)
    
    if err == nil && count > 0 {
        return true, ""
    }
    
    return false, fmt.Sprintf("Feature %s not available for your plan", featureKey)
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (billing-service):**
```go
// addon_test.go
func TestPurchaseAddon_Success(t *testing.T) {
    // Mock: wallet balance Rp 600k, addon price Rp 500k
    // Expect: deduct Rp 500k, tenant_addons INSERT, status active
}

func TestPurchaseAddon_InsufficientBalance(t *testing.T) {
    // Mock: wallet balance Rp 100k, addon price Rp 500k
    // Expect: 402 Payment Required
}

func TestPurchaseAddon_AlreadyActive(t *testing.T) {
    // Mock: tenant already has active addon
    // Expect: 409 Conflict
}

func TestCanUseFeature_PlanBundled(t *testing.T) {
    // Feature enabled in plan_features
    // Expect: CanUseFeature return true
}

func TestCanUseFeature_AddonPurchased(t *testing.T) {
    // Feature OFF in plan, but tenant has active addon
    // Expect: CanUseFeature return true
}
```

### Integration Tests

```bash
# 1. GET marketplace
curl -X GET http://localhost:8003/addon-marketplace \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
# → 200 OK with addon list

# 2. Purchase addon (sufficient balance)
curl -X POST http://localhost:8003/addons/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"addon_key":"ai_vision"}'
# → 200 OK

# 3. Verify addon active
curl -X GET http://localhost:8003/addon-marketplace \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
# → ai_vision status = active

# 4. Purchase again (duplicate)
curl -X POST http://localhost:8003/addons/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"addon_key":"ai_vision"}'
# → 409 Conflict

# 5. Purchase with insufficient balance
# Set wallet balance < addon price via DB
curl -X POST http://localhost:8003/addons/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"addon_key":"ai_voice_stt"}'
# → 402 Payment Required
```

### Manual Testing Checklist

- [ ] Addon marketplace page load dengan list addons
- [ ] Available addon show "Beli Rp X" button
- [ ] Active addon show "Active" badge + expiry date
- [ ] Click "Beli" → confirm modal → purchase success
- [ ] Wallet balance deducted correctly
- [ ] Feature gating (CanUseFeature) allow access setelah purchase
- [ ] Purchase dengan insufficient balance → error toast
- [ ] Purchase addon yang sudah active → error toast
- [ ] Referral discount applied (final price < base price)
- [ ] Superadmin dapat edit addon price (future AC-10)

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Addon purchased", 
  "tenant_id", tenantID,
  "addon_key", addonKey,
  "amount_paid_cents", amountPaid,
  "expires_at", expiresAt)

slog.Warn("Addon purchase failed", 
  "tenant_id", tenantID,
  "addon_key", addonKey,
  "reason", "insufficient_balance",
  "required_cents", price,
  "available_cents", balance)
```

**Metrics to track:**
- Addon purchase count per addon per month
- Addon revenue per month
- Average wallet balance before/after addon purchase
- Addon expiry/renewal rate (for auto-renew future feature)

**Alerts:**
- Addon purchase failure rate > 20% → investigate wallet balance issue or pricing config
- Feature gating `CanUseFeature` latency > 50ms → cache issue or DB query optimization needed

---

## 🚀 Rollout Plan

### Phase 1: DB Schema + Feature Gating (Done ✅)
- Migration 000068: `available_features` + `tenant_addons` tables
- Deploy `shared/sdk/auth/can_use.go` (CanUseFeature guard)
- Seed default addons
- Test: feature gating logic via unit tests

### Phase 2: Backend Addon Purchase API (Done ✅)
- Deploy billing-service dengan `/addon-marketplace` + `/addons/purchase` endpoints
- Referral discount integration (F054)
- Test: addon purchase flow via cURL

### Phase 3: Frontend Addons Page (Done ✅)
- Deploy umkm-web dengan `/addons` route + `Addons.vue` component
- Marketplace UI dengan addon cards + status badges
- Test: end-to-end purchase flow

### Phase 4: Superadmin Addon Config UI (Future)
- Superadmin dashboard page untuk CRUD `available_features`
- Edit addon price, set min_tier, enable/disable addon
- Test: superadmin edit price → tenant berikutnya bayar new price

### Phase 5: Auto-Renew (Future)
- Cron job check `tenant_addons` dengan `expires_at < NOW() + 7 days` + `auto_renew = true`
- Auto-deduct wallet → extend `expires_at` += 30 days
- Email notification sebelum auto-renew

### Rollback
- **Phase 1 rollback:** Revert migration 000068 → `available_features` + `tenant_addons` tables dropped
- **Phase 2 rollback:** Remove addon purchase endpoints dari billing-service routing → 404
- **Emergency:** Disable addon purchase via feature flag → `/addons/purchase` return 503 Service Unavailable

---

## 🔮 Future Enhancements (Out of Scope)

- **Addon Bundles:** Package multiple addons dengan discount (e.g., "Multimodal AI Bundle": vision + STT + TTS = Rp 100k/month, save Rp 10k)
- **Annual Addon:** Support `addon_unit='per_year'` dengan discount (e.g., pay 10 months, get 12 months)
- **Addon Trial:** Free trial 7 hari untuk new tenants (auto-expire jika tidak purchase)
- **Addon Gifting:** Tenant A bisa gift addon ke Tenant B (referral incentive)
- **Usage-Based Addon:** Pay-per-use pricing (e.g., Rp 1000 per AI Vision request) bukan flat monthly fee

---

## 📚 References

- [F034: Wallet Top-up](../FEATURE_MAP.md) — Wallet system integration
- [F052: Tier-First Feature System](../FEATURE_MAP.md) — Plan features + feature gating foundation
- [F054: Referral System](F054_referral_system_discount_downline_commission_uplin.md) — Referral discount calculation
- [Addons.vue Component](../../frontend/umkm-web/src/components/Addons.vue) — Frontend marketplace UI

---

## 📝 Notes & Decisions

**2026-06-16:** Decision: `available_features` single table untuk bundled + addon features (bukan split ke 2 tables) — simplify query + avoid JOIN.  
**2026-06-16:** `tenant_addons` UNIQUE constraint pada `(tenant_id, addon_key) WHERE status='active'` — allow expired addon re-purchase.  
**2026-06-16:** Addon pricing di `available_features` (bukan dedicated `addon_prices` table) — F057 akan konsolidasi pricing logic, untuk MVP cukup 1 table.  
**2026-06-16:** `CanUseFeature` guard support OR logic (plan bundled OR addon purchased) — flexibility untuk tenant upgrade via addon tanpa plan upgrade.  
**2026-06-16:** Referral discount applied BEFORE wallet deduct (integrate dengan F054) — affiliate commission calculated dari final_price (setelah discount).
