-- 000030_voucher_day_duration_and_pending_subscription.up.sql
-- F015: Onboarding Activation Flow
-- Adds day-duration voucher system and pending subscription status

-- 1. Drop valid_until from voucher_codes (replace with validity_days)
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS valid_until;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS remaining_days;

ALTER TABLE voucher_codes ADD COLUMN validity_days INTEGER NOT NULL DEFAULT 30;
ALTER TABLE voucher_codes ADD COLUMN redeemed_at TIMESTAMPTZ;
ALTER TABLE voucher_codes ADD COLUMN is_system_generated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE voucher_codes ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;

-- 2. Add pending status to tenant_subscriptions
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS status;
ALTER TABLE tenant_subscriptions ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tenant_subscriptions ADD COLUMN pending_expires_at TIMESTAMPTZ;
ALTER TABLE tenant_subscriptions ADD COLUMN pending_timeout_hours INTEGER NOT NULL DEFAULT 24;

-- 3. Add plan priority column for query optimization
ALTER TABLE tenants ADD COLUMN plan_priority INTEGER NOT NULL DEFAULT 0;

-- 4. Create voucher_subscriptions table for day-duration accumulation
-- This tracks per-(tenant, plan_id) the active days
CREATE TABLE IF NOT EXISTS voucher_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL,
    validity_days INTEGER NOT NULL,
    remaining_days INTEGER NOT NULL,
    redeemed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_system_generated BOOLEAN NOT NULL DEFAULT false,
    source_voucher_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voucher_subs_tenant ON voucher_subscriptions(tenant_id);
CREATE UNIQUE INDEX idx_voucher_subs_tenant_plan ON voucher_subscriptions(tenant_id, plan_id);

-- 5. Create pending_tenants cleanup log
CREATE TABLE IF NOT EXISTS pending_tenant_cleanup_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    email TEXT,
    phone TEXT,
    cleaned_up_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT NOT NULL DEFAULT 'pending_timeout'
);

-- 6. Add pending_timeout_hours config to saas_plans
ALTER TABLE saas_plans ADD COLUMN pending_timeout_hours INTEGER NOT NULL DEFAULT 24;

-- 7. Add system_generated_voucher_code to tenant_subscriptions
ALTER TABLE tenant_subscriptions ADD COLUMN system_voucher_code TEXT;

-- 8. Backfill plan_priority for existing plans (Lite=1, Pro=2, Business=3)
UPDATE tenants SET plan_priority = 1 WHERE plan = 'lite';
UPDATE tenants SET plan_priority = 2 WHERE plan = 'pro';
UPDATE tenants SET plan_priority = 3 WHERE plan = 'ultimate';

-- 9. Backfill existing voucher_codes with default validity_days
UPDATE voucher_codes SET redeemed_at = used_at WHERE used_at IS NOT NULL AND redeemed_at IS NULL;
UPDATE voucher_codes SET redeemed_at = used_at WHERE used_at IS NULL AND redeemed_at IS NULL;