-- 000026_multi_store.up.sql
-- Multi-store support: 1 owner bisa punya banyak toko di tier Business
--
-- Design:
--   - stores = child entity di bawah tenants (1 subscription)
--   - owner_user_id = user yang punya toko (1 owner = N stores)
--   - tenant_id tetap = parent subscription (pakai billing existing)
--   - quota di-enforce via plan_features.feature_key = 'max_stores'
--
-- Backward compatible: existing tenants otomatis punya 1 store (default)

-- ─────────────────────────────────────────────
-- 1. Tabel stores
-- ─────────────────────────────────────────────
CREATE TABLE stores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    business_type   VARCHAR(50) NOT NULL REFERENCES business_types(id),
    business_name   VARCHAR(255),
    address         TEXT,
    phone           VARCHAR(50),
    logo_url        TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stores_owner ON stores(owner_user_id);
CREATE INDEX idx_stores_tenant ON stores(tenant_id);
CREATE INDEX idx_stores_business_type ON stores(business_type);
CREATE INDEX idx_stores_active ON stores(is_active) WHERE is_active = true;

-- ─────────────────────────────────────────────
-- 2. Seed: auto-create 1 store per existing tenant
-- Backfill supaya existing data tidak orphan
-- ─────────────────────────────────────────────
INSERT INTO stores (owner_user_id, tenant_id, name, business_type, is_active)
SELECT
    (SELECT id FROM users WHERE tenant_id = t.id ORDER BY created_at ASC LIMIT 1),
    t.id,
    COALESCE(t.business_name, t.name, 'Toko Utama'),
    COALESCE(t.business_type, 'umum'),
    true
FROM tenants t
WHERE EXISTS (SELECT 1 FROM users WHERE tenant_id = t.id)
  AND NOT EXISTS (SELECT 1 FROM stores s WHERE s.tenant_id = t.id);

-- ─────────────────────────────────────────────
-- 3. Add max_stores feature ke plan_features
-- Superadmin bisa ubah via API nanti
-- ─────────────────────────────────────────────
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled) VALUES
    ('lite',     'max_stores', 'Jumlah toko maksimum',         '1',     true),
    ('pro',      'max_stores', 'Jumlah toko maksimum',         '1',     true),
    ('ultimate', 'max_stores', 'Jumlah toko maksimum',         '5',     true)
ON CONFLICT (plan_id, feature_key) DO UPDATE SET
    feature_value = EXCLUDED.feature_value;
