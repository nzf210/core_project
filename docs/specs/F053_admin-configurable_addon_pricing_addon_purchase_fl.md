# F053: Admin-Configurable Addon Pricing + Addon Purchase Flow


### 📌 Background — State Saat Ini

```
现状 (Current):
  plan_features (DB) → feature_key/enabled per plan
    → PlanFeaturesRow (Go struct, hardcoded fields: HasPOS, HasChatbot, ...)
    → HasFeatureAccess() switch/case per feature name
    → hardcoded "Fitur X memerlukan paket Lite..."

问题 (Problems):
  1. Tambah fitur baru → migration (DB) + code change (Go struct + switch)
  2. Tidak ada Addon table → F034 (addon wallet) done; F052 bikin Addon guard foundation
  3. Guard tersebar (HasFeatureAccess, CheckQuota, RequireFeature, RequireClinicType, ...)
  4. Enum "lite/pro/ultimate" hardcoded di banyak tempat
```


### 🏗️ Arsitektur Baru

```
┌──────────────────────────────────────────────────────────────┐
│  Guards (shared/sdk/auth/)                                  │
│                                                              │
│  CanUse(tenantID, "ai_vision")                              │
│    ├─ 1. TierFeatureEnabled?(tier, "ai_vision")             │
│    │     → SELECT is_enabled FROM plan_features              │
│    │       WHERE plan_id = $tier AND feature_key = "ai_vision"│
│    │                                                         │
│    └─ 2. TenantHasAddon?(tenantID, "ai_vision")             │
│          → SELECT 1 FROM tenant_addons                       │
│            WHERE tenant_id = $tid AND addon_key = "ai_vision"│
│            AND status = 'active' AND expires_at > NOW()      │
│                                                              │
│  Result: Tier ON? → allowed                                  │
│          Tier OFF but Addon active? → allowed                │
│          Tier OFF and no Addon? → denied                     │
└──────────────────────────────────────────────────────────────┘
```


### 🔐 Guard Logic

#### 1. `CanUseFeature(tenantID, featureKey)` — core guard
```go
func CanUseFeature(ctx context.Context, tenantID, featureKey string) (bool, string) {
    tier := GetTenantPlan(ctx, tenantID)
    feat := GetFeatureDef(featureKey)  // from available_features cache

    // Addon-only check (feature tidak ada di plan_features)
    if feat != nil && feat.IsAddon {
        return CanUseAddon(ctx, tenantID, featureKey)
    }

    // Bundled feature check
    row, _ := GetPlanFeaturesRow(ctx, tier)
    // cek plan_features.is_enabled untuk featureKey
    enabled := row.IsFeatureEnabled(featureKey)
    if enabled {
        return true, ""
    }

    // Fallback: apakah ini addon yang di-upgrade dari tier?
    addonOK, _ := CanUseAddon(ctx, tenantID, featureKey)
    if addonOK {
        return true, ""
    }

    return false, fmt.Sprintf("Fitur %s tidak tersedia di paket %s.", feat.FeatureName, tier)
}
```

#### 2. `CanUseAddon(ctx, tenantID, addonKey)` — addon guard
```go
func CanUseAddon(ctx context.Context, tenantID, addonKey string) (bool, error) {
    feat := GetFeatureDef(addonKey)
    if feat == nil || !feat.IsAddon {
        return false, nil // bukan addon
    }

    // Cek tier minimum
    row, _ := GetPlanFeaturesRow(ctx, GetTenantPlan(ctx, tenantID))
    if feat.MinTier != "" && !row.TierAllowsFeature(addonKey) {
        return false, nil // tier tidak memenuhi min tier
    }

    // Cek tenant_addons
    var exists bool
    err := db.Pool.QueryRow(ctx,
        `SELECT EXISTS(
            SELECT 1 FROM tenant_addons
            WHERE tenant_id=$1 AND addon_key=$2
            AND status='active'
            AND (expires_at IS NULL OR expires_at > NOW())
        )`, tenantID, addonKey).Scan(&exists)
    return exists, err
}
```

#### 3. `RequireFeature(feature string)` middleware
```go
// Supercedes existing RequireFeature — delegates to CanUseFeature
func RequireFeature(feature string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID, ok := r.Context().Value(TenantIDKey).(string)
            if !ok || tenantID == "" {
                response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
                return
            }
            allowed, reason := CanUseFeature(r.Context(), tenantID, feature)
            if !allowed {
                w.Header().Set("X-Feature-Gate", "denied")
                w.Header().Set("X-Feature-Required", feature)
                response.Error(w, http.StatusForbidden, reason, nil)
                return
            }
            w.Header().Set("X-Feature-Gate", "allowed")
            next.ServeHTTP(w, r)
        })
    }
}
```


### 📋 Seed Data (Migration 000065)

```sql
-- available_features registry
INSERT INTO available_features (feature_key, feature_name, category, is_addon, default_enabled, addon_price_cents, addon_unit) VALUES
    -- Bundled features (is_addon=FALSE, default_enabled sesuai tier)
    ('accounting',       'Double-Entry Accounting',      'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('pos',              'Point of Sale',                  'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('chatbot',          'AI Chatbot WhatsApp',           'ai',        false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('ai_text',          'AI Text (Chat)',                 'ai',        false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('inventory',        'Inventory Management',            'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('reports',         'Laporan Keuangan',               'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('multi_user',       'Multi-User Access',             'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('advanced_reports', 'Laporan Keuangan Lanjutan',     'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('api_access',       'API Access',                    'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('custom_branding',  'Custom Branding',               'core',      false, ARRAY['ultimate'],            0,        NULL),
    ('priority_support', 'Priority Support',              'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    -- Addon features (is_addon=TRUE)
    ('ai_vision',        'AI Vision (Foto KTP/Produk)',   'ai',        true,  NULL,                           5000,      'per_request'),
    ('ai_audio',         'AI Audio (Voice Note)',          'ai',        true,  NULL,                           1000,      'per_minute'),
    ('wa_cloud_api',     'WA Cloud API Broadcast',        'wa',        true,  NULL,                           5000,      'per_session'),
    ('wa_blast',         'WA Blast Massal',               'wa',        true,  NULL,                           10000,     'per_request'),
    ('extra_store',      'Extra POS Store',               'storage',   true,  NULL,                           5000000,  'per_month'),
    ('extra_user',       'Extra User Seat',               'storage',   true,  NULL,                           1000000,  'per_month');
```


### 🔄 Phase Plan

| Phase | Scope | Effort |
|:------|:------|:-------|
| Phase 1 | DB schema + `CanUseFeature` SDK + `available_features` seed | Backend only |
| Phase 2 | Migrasi `HasFeatureAccess` → `CanUseFeature`, remove hardcoded switch | Backend refactor |
| Phase 3 | Addon purchase flow (wallet deduction + tenant_addons INSERT) | Backend + FE |
| Phase 4 | UI "Add-ons" page (list + buy + my addons) | Frontend only |
| Phase 5 | Superadmin: UI plan matrix editor (add/remove features per tier) | Frontend |


### 📁 Files Changed (Phase 1)

**Backend:**
- `shared/migrations/000068_tier_addon_system.up.sql` (NEW) — available_features + tenant_addons + min_tier
- `shared/migrations/000068_tier_addon_system.down.sql` (NEW)
- `shared/sdk/auth/can_use.go` (NEW) — `CanUseFeature()` + `CanUseAddon()` + `GetFeatureDef()` + cache
- `shared/sdk/auth/quota.go` — `RequireFeature()` delegates to `CanUseFeature()`, `HasFeatureAccess()` deprecated

**Frontend:** F053 scope (Addons.vue purchase UI).
- `frontend/umkm-web/src/router/index.ts` — route `/addons`
- `frontend/umkm-web/src/config/menu.ts` — menu "Add-ons" (if addon_count > 0)


## F053: Admin-Configurable Addon Pricing + Addon Purchase Flow

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Harga addon dikonfigurasi oleh superadmin via UI di `available_features` (BUKAN `addon_prices` — lihat F057 untuk konsolidasi). Tenant membeli addon → wallet deducted → `tenant_addons` row dibuat → fitur langsung aktif. Ini adalah kelanjutan dari F052 (Tier-First Feature System) dan F034 (Wallet).
