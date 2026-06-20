-- F052: Tier-First Feature System + Per-Tenant Addon Guard
-- 1. available_features: registry fitur + addon metadata
-- 2. tenant_addons: addon aktif per tenant
-- 3. plan_features.min_tier: tier minimum untuk addons

-- 1. available_features
CREATE TABLE IF NOT EXISTS available_features (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feature_key         VARCHAR(100) NOT NULL UNIQUE,
    feature_name        VARCHAR(255) NOT NULL,
    description         TEXT DEFAULT '',
    category            VARCHAR(50) NOT NULL,  -- 'core', 'ai', 'wa', 'storage'
    is_addon            BOOLEAN NOT NULL DEFAULT false,
    default_enabled     VARCHAR(20)[] DEFAULT '{}',  -- tier list where enabled by default
    addon_price_cents   BIGINT DEFAULT 0,
    addon_unit          VARCHAR(20) DEFAULT 'per_month',  -- per_month/per_request/per_minute/per_session
    created_at          TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_available_features_category ON available_features(category);
CREATE INDEX IF NOT EXISTS idx_available_features_is_addon ON available_features(is_addon) WHERE is_addon = true;

-- Seed bundled features (is_addon=FALSE — bundled per tier)
INSERT INTO available_features (feature_key, feature_name, category, is_addon, default_enabled, addon_price_cents, addon_unit)
VALUES
    ('accounting',       'Double-Entry Accounting',      'core',    false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('pos',              'Point of Sale',                'core',    false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('chatbot',          'AI Chatbot WhatsApp',           'ai',      false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('ai_text',          'AI Text (Chat)',               'ai',      false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('inventory',        'Inventory Management',         'core',    false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('reports',          'Laporan Keuangan',              'core',    false, ARRAY['lite','pro','ultimate'],    0, NULL),
    ('multi_user',       'Multi-User Access',            'core',    false, ARRAY['pro','ultimate'],          0, NULL),
    ('advanced_reports',  'Laporan Keuangan Lanjutan',    'core',    false, ARRAY['pro','ultimate'],          0, NULL),
    ('api_access',       'API Access',                   'core',    false, ARRAY['pro','ultimate'],          0, NULL),
    ('custom_branding',   'Custom Branding',              'core',    false, ARRAY['ultimate'],               0, NULL),
    ('priority_support',  'Priority Support',             'core',    false, ARRAY['pro','ultimate'],          0, NULL),
    ('wa_cloud_api',     'WA Cloud API Access (Meta)',   'wa',      false, ARRAY['pro','ultimate'],          0, NULL)
ON CONFLICT (feature_key) DO NOTHING;

-- Seed addon features (is_addon=TRUE — berbayar per wallet)
INSERT INTO available_features (feature_key, feature_name, category, is_addon, default_enabled, addon_price_cents, addon_unit)
VALUES
    ('ai_vision',        'AI Vision (Foto KTP/Produk)',  'ai',      true,  NULL,   500000,  'per_request'),
    ('ai_audio',         'AI Audio (Voice Note)',        'ai',      true,  NULL,   1000000, 'per_month'),
    ('wa_blast',         'WA Blast Massal',              'wa',      true,  NULL,   200000,  'per_request'),
    ('extra_store',      'Extra POS Store',             'storage', true,  NULL,   5000000, 'per_month'),
    ('extra_user',       'Extra User Seat',             'storage', true,  NULL,   1000000, 'per_month')
ON CONFLICT (feature_key) DO NOTHING;

-- 2. tenant_addons
CREATE TABLE IF NOT EXISTS tenant_addons (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    addon_key               VARCHAR(100) NOT NULL REFERENCES available_features(feature_key),
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',  -- active/expired/cancelled
    purchased_at            TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at              TIMESTAMPTZ,  -- NULL = unlimited/permanent
    auto_renew              BOOLEAN NOT NULL DEFAULT true,
    purchase_price_cents    BIGINT DEFAULT 0,
    wallet_transaction_id    UUID,
    created_at              TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, addon_key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_addons_lookup
    ON tenant_addons(tenant_id, addon_key)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_tenant_addons_expires
    ON tenant_addons(expires_at)
    WHERE status = 'active' AND expires_at IS NOT NULL;

-- 3. Sync addon features into plan_features so existing guards work during transition
-- (bundled addons like wa_cloud_api that are now is_addon=FALSE in available_features)
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
SELECT id, 'wa_cloud_api', 'WA Cloud API Access (Meta)', 'boolean', true
FROM saas_plans WHERE id IN ('pro', 'ultimate')
ON CONFLICT (plan_id, feature_key) DO NOTHING;

-- 4. Add min_tier to plan_features (for future addon gating by tier minimum)
ALTER TABLE plan_features ADD COLUMN IF NOT EXISTS min_tier VARCHAR(20) DEFAULT NULL;

-- Sync addon features that are wallet-gated into plan_features with is_enabled=false
-- (so GetPlanFeatures returns HasWACloudAPI etc for backward compat)
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled, min_tier)
SELECT saas_plans.id, af.feature_key, af.feature_name, 'addon', false, NULL
FROM saas_plans, available_features af
WHERE af.is_addon = true
  AND af.feature_key NOT IN (SELECT feature_key FROM plan_features WHERE plan_id = saas_plans.id)
ON CONFLICT (plan_id, feature_key) DO NOTHING;
