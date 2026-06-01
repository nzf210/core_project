CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    discount_type VARCHAR(20) NOT NULL, -- 'percentage', 'fixed', 'free_months'
    discount_value INT NOT NULL,
    duration_months INT DEFAULT 1,
    max_uses INT DEFAULT 0, -- 0 means unlimited
    uses_count INT DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_coupons (
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    coupon_id UUID REFERENCES coupons(id) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, coupon_id)
);

-- Seed default promo coupon for 3 months free Lite tier
INSERT INTO coupons (code, discount_type, discount_value, duration_months, max_uses, expires_at)
VALUES ('PROMO-LITE-90', 'free_months', 3, 3, 0, NOW() + INTERVAL '1 year')
ON CONFLICT (code) DO NOTHING;

-- Change usage_quotas defaults
ALTER TABLE usage_quotas ALTER COLUMN plan_tier SET DEFAULT 'lite';
UPDATE usage_quotas SET plan_tier = 'lite' WHERE plan_tier = 'free';
UPDATE tenants SET plan = 'lite' WHERE plan = 'free';
