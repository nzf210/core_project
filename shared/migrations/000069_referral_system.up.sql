-- 000069_referral_system.up.sql
-- F054: Referral System — discount downline + commission upline

-- 1. affiliate_referrals: track who referred whom
CREATE TABLE IF NOT EXISTS affiliate_referrals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id        INT NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    referred_at         TIMESTAMPTZ DEFAULT NOW(),
    first_purchase_at   TIMESTAMPTZ,
    UNIQUE(affiliate_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_affiliate ON affiliate_referrals(affiliate_id);
CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_tenant   ON affiliate_referrals(tenant_id);

-- 2. invoice_referrals: track referral discount applied per invoice
CREATE TABLE IF NOT EXISTS invoice_referrals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      VARCHAR(100) NOT NULL,
    affiliate_id    INT NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    discount_amount BIGINT NOT NULL DEFAULT 0,  -- sen
    applied_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(invoice_id)
);
CREATE INDEX IF NOT EXISTS idx_invoice_referrals_affiliate ON invoice_referrals(affiliate_id);

-- 3. Extend referral_config with new columns
ALTER TABLE referral_config ADD COLUMN IF NOT EXISTS min_purchase_rupiah   BIGINT  DEFAULT 0;
ALTER TABLE referral_config ADD COLUMN IF NOT EXISTS max_commission_rupiah BIGINT  DEFAULT 0;
ALTER TABLE referral_config ADD COLUMN IF NOT EXISTS is_active             BOOLEAN DEFAULT true;
ALTER TABLE referral_config ADD COLUMN IF NOT EXISTS referral_link_base    VARCHAR(255) DEFAULT 'wch.id/r';

-- 4. Extend affiliate_earnings with transaction_type for addon purchases
-- (already has type col in 000056; ensure addon_purchase value is possible)
ALTER TABLE affiliate_earnings ADD COLUMN IF NOT EXISTS description TEXT;
