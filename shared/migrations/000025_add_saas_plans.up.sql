-- 000025_add_saas_plans.up.sql
-- SaaS Plans: Lite, Pro, Business | Voucher Programs | Auto Subscription Tickets

CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────
-- 1. SaaS Plan Definitions
-- ─────────────────────────────────────────────
CREATE TABLE saas_plans (
    id            VARCHAR(20) PRIMARY KEY,  -- 'lite', 'pro', 'ultimate'
    name          VARCHAR(50) NOT NULL,
    description   TEXT,
    price_monthly BIGINT NOT NULL,           -- harga dalam SEN (1 rupiah = 100 sen)
    price_yearly  BIGINT,                   -- diskon yearly
    is_active     BOOLEAN NOT NULL DEFAULT true,
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────
-- 2. Plan Features Matrix
-- ─────────────────────────────────────────────
CREATE TABLE plan_features (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id     VARCHAR(20) NOT NULL REFERENCES saas_plans(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,    -- e.g., 'accounting', 'chatbot', 'pos', 'ai_requests'
    feature_name VARCHAR(255) NOT NULL,   -- label human-readable
    feature_value VARCHAR(50) NOT NULL,   -- 'unlimited', '50', 'true', 'false'
    is_enabled  BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plan_id, feature_key)
);

CREATE INDEX idx_plan_features_plan_id ON plan_features(plan_id);

-- ─────────────────────────────────────────────
-- 3. Voucher Programs
-- ─────────────────────────────────────────────
CREATE TABLE voucher_programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID REFERENCES tenants(id) ON DELETE CASCADE,  -- NULL = global, tenant-specific if set
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    voucher_type    VARCHAR(20) NOT NULL,  -- 'discount_percent', 'discount_fixed', 'free_months', 'plan_upgrade'
    discount_value  INT NOT NULL DEFAULT 0,
    target_plan_id  VARCHAR(20),           -- NULL = all plans, e.g. 'lite', 'pro', 'ultimate'
    duration_months INT NOT NULL DEFAULT 1,
    max_uses        INT NOT NULL DEFAULT 0, -- 0 = unlimited
    max_uses_per_tenant INT NOT NULL DEFAULT 1,
    uses_count      INT NOT NULL DEFAULT 0,
    starts_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voucher_programs_tenant ON voucher_programs(tenant_id);
CREATE INDEX idx_voucher_programs_target_plan ON voucher_programs(target_plan_id);

-- ─────────────────────────────────────────────
-- 4. Voucher Codes (generated from programs)
-- ─────────────────────────────────────────────
CREATE TABLE voucher_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id      UUID NOT NULL REFERENCES voucher_programs(id) ON DELETE CASCADE,
    code            VARCHAR(50) NOT NULL UNIQUE,
    used_by         UUID REFERENCES tenants(id) ON DELETE SET NULL,
    used_at         TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    is_redeemed     BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voucher_codes_program ON voucher_codes(program_id);
CREATE INDEX idx_voucher_codes_tenant ON voucher_codes(used_by);

-- ─────────────────────────────────────────────
-- 5. Subscription Tickets (auto-sent on purchase/voucher activation)
-- ─────────────────────────────────────────────
CREATE TABLE subscription_tickets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES tenant_subscriptions(tenant_id) ON DELETE SET NULL,
    ticket_number   VARCHAR(30) NOT NULL UNIQUE,  -- e.g., TKT-2026-0601-0001
    plan_id         VARCHAR(20) NOT NULL,
    plan_name       VARCHAR(50) NOT NULL,
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',  -- 'active', 'expired', 'cancelled'
    notify_wa       BOOLEAN NOT NULL DEFAULT false,
    notify_telegram BOOLEAN NOT NULL DEFAULT false,
    notify_email    BOOLEAN NOT NULL DEFAULT false,
    wa_sent_at      TIMESTAMPTZ,
    telegram_sent_at TIMESTAMPTZ,
    email_sent_at   TIMESTAMPTZ,
    ticket_payload  JSONB,  -- stores rendered message content
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_tickets_tenant ON subscription_tickets(tenant_id);
CREATE INDEX idx_subscription_tickets_status ON subscription_tickets(status);
CREATE INDEX idx_subscription_tickets_expires ON subscription_tickets(expires_at);

-- ─────────────────────────────────────────────
-- 6. Update tenant_subscriptions: add plan_tier, period_days, ticket_id
-- ─────────────────────────────────────────────
ALTER TABLE tenant_subscriptions
    ADD COLUMN IF NOT EXISTS plan_tier VARCHAR(20) DEFAULT 'lite',
    ADD COLUMN IF NOT EXISTS period_days INT DEFAULT 30,
    ADD COLUMN IF NOT EXISTS ticket_id UUID REFERENCES subscription_tickets(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS voucher_code_id UUID REFERENCES voucher_codes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS activated_by VARCHAR(50) DEFAULT 'payment';  -- 'payment', 'voucher', 'admin'

-- ─────────────────────────────────────────────
-- 7. Seed: Default SaaS Plans
-- ─────────────────────────────────────────────
INSERT INTO saas_plans (id, name, description, price_monthly, price_yearly, sort_order) VALUES
    ('lite',     'Lite',     'Untuk bisnis kecil yang baru memulai. Fitur dasar Accounting, POS, dan Chatbot AI.',     15000000, 150000000, 1),  -- 150k/month, 1.5M/year
    ('pro',      'Pro',      'Untuk bisnis berkembang. Semua fitur Lite + AI advanced, multi-user, priority support.', 45000000, 450000000, 2),  -- 450k/month, 4.5M/year
    ('ultimate', 'Ultimate', 'Untuk bisnis skala besar. Unlimited users, API access, custom branding, dedicated support.', 150000000, 1500000000, 3); -- 1.5M/month, 15M/year

-- ─────────────────────────────────────────────
-- 8. Seed: Default Features per Plan
-- ─────────────────────────────────────────────
-- LITE features
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled) VALUES
    ('lite', 'accounting',       'Double-Entry Accounting',         'true',   true),
    ('lite', 'pos',              'Point of Sale (POS)',              'true',   true),
    ('lite', 'chatbot',          'AI Chatbot WhatsApp',             '50',     true),
    ('lite', 'chatbot_messages', 'Chatbot pesan/bulan',             '50',     true),
    ('lite', 'products',         'Produk',                          '100',    true),
    ('lite', 'users',            'User seat',                       '2',      true),
    ('lite', 'ai_requests',      'AI request/bulan',               '100',    true),
    ('lite', 'reports',          'Laporan keuangan',               'basic',  true),
    ('lite', 'inventory',        'Inventory management',           'true',   true),
    ('lite', 'customer_db',      'Database pelanggan',             'true',   true),
    ('lite', 'api_access',       'API Access',                     'false',  false),
    ('lite', 'custom_branding',  'Custom Branding',                'false',  false),
    ('lite', 'priority_support', 'Priority Support',               'false',  false);

-- PRO features
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled) VALUES
    ('pro', 'accounting',       'Double-Entry Accounting',         'true',   true),
    ('pro', 'pos',              'Point of Sale (POS)',              'true',   true),
    ('pro', 'chatbot',          'AI Chatbot WhatsApp',             'unlimited', true),
    ('pro', 'chatbot_messages', 'Chatbot pesan/bulan',             '1000',   true),
    ('pro', 'products',         'Produk',                          '1000',   true),
    ('pro', 'users',            'User seat',                       '10',     true),
    ('pro', 'ai_requests',      'AI request/bulan',                '1000',   true),
    ('pro', 'reports',          'Laporan keuangan',                'advanced', true),
    ('pro', 'inventory',         'Inventory management',           'true',   true),
    ('pro', 'customer_db',       'Database pelanggan',             'true',   true),
    ('pro', 'api_access',        'API Access',                    'true',   true),
    ('pro', 'custom_branding',   'Custom Branding',               'false',  false),
    ('pro', 'priority_support',  'Priority Support',              'true',   true);

-- ULTIMATE features
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled) VALUES
    ('ultimate', 'accounting',       'Double-Entry Accounting',         'true',      true),
    ('ultimate', 'pos',              'Point of Sale (POS)',              'true',      true),
    ('ultimate', 'chatbot',          'AI Chatbot WhatsApp',             'unlimited', true),
    ('ultimate', 'chatbot_messages', 'Chatbot pesan/bulan',            'unlimited', true),
    ('ultimate', 'products',         'Produk',                          'unlimited', true),
    ('ultimate', 'users',            'User seat',                       'unlimited', true),
    ('ultimate', 'ai_requests',      'AI request/bulan',               'unlimited', true),
    ('ultimate', 'reports',          'Laporan keuangan',               'full',      true),
    ('ultimate', 'inventory',        'Inventory management',           'true',      true),
    ('ultimate', 'customer_db',      'Database pelanggan',             'true',      true),
    ('ultimate', 'api_access',        'API Access',                    'true',      true),
    ('ultimate', 'custom_branding',  'Custom Branding',               'true',      true),
    ('ultimate', 'priority_support', 'Priority Support',              'true',      true);

-- ─────────────────────────────────────────────
-- 9. Seed: Default Voucher Programs (global)
-- ─────────────────────────────────────────────
INSERT INTO voucher_programs (id, tenant_id, name, description, voucher_type, discount_value, target_plan_id, duration_months, max_uses, expires_at, is_active)
VALUES
    ('a1111111-1111-1111-1111-111111111111', NULL,
     'Diskon 50% Lite - Trial Promo', 'Diskon 50% untuk paket Lite selama 1 bulan',
     'discount_percent', 50, 'lite', 1, 100, NOW() + INTERVAL '90 days', true),

    ('a2222222-2222-2222-2222-222222222222', NULL,
     'Gratis 3 Bulan Pro', 'Gratis 3 bulan untuk paket Pro',
     'free_months', 3, 'pro', 3, 50, NOW() + INTERVAL '90 days', true),

    ('a3333333-3333-3333-3333-333333333333', NULL,
     'Diskon 30% Ultimate', 'Diskon 30% untuk paket Ultimate',
     'discount_percent', 30, 'ultimate', 1, 20, NOW() + INTERVAL '90 days', true),

    ('a4444444-4444-4444-4444-444444444444', NULL,
     'Gratis 1 Bulan All Plan', 'Gratis 1 bulan untuk semua paket',
     'free_months', 1, NULL, 1, 200, NOW() + INTERVAL '90 days', true);

-- ─────────────────────────────────────────────
-- 10. Seed: Voucher Codes for each program
-- ─────────────────────────────────────────────
INSERT INTO voucher_codes (program_id, code, expires_at) VALUES
    -- Diskon 50% Lite
    ('a1111111-1111-1111-1111-111111111111', 'LITE50-OFF',  NOW() + INTERVAL '90 days'),
    ('a1111111-1111-1111-1111-111111111111', 'PROMO-LITE50', NOW() + INTERVAL '90 days'),
    -- Gratis 3 Bulan Pro
    ('a2222222-2222-2222-2222-222222222222', 'PRO3-FREE',    NOW() + INTERVAL '90 days'),
    ('a2222222-2222-2222-2222-222222222222', 'PROMO-PRO90',  NOW() + INTERVAL '90 days'),
    -- Diskon 30% Ultimate
    ('a3333333-3333-3333-3333-333333333333', 'ULT30-OFF',    NOW() + INTERVAL '90 days'),
    -- Gratis 1 Bulan All Plan
    ('a4444444-4444-4444-4444-444444444444', 'FREE1MONTH',   NOW() + INTERVAL '90 days'),
    ('a4444444-4444-4444-4444-444444444444', 'ONEMONTHFREE', NOW() + INTERVAL '90 days');