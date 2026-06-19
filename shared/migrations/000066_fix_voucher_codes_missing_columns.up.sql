-- 000066_fix_voucher_codes_missing_columns.up.sql
-- Fix: voucher_codes is missing validity_days, redeemed_at, is_system_generated, tenant_id
-- Root cause: migration 000030 was recorded as applied but columns were never created
-- (likely a partially-failed migration that was incorrectly marked as success)

-- 1. Add missing columns to voucher_codes
ALTER TABLE voucher_codes ADD COLUMN IF NOT EXISTS validity_days INTEGER NOT NULL DEFAULT 30;
ALTER TABLE voucher_codes ADD COLUMN IF NOT EXISTS redeemed_at TIMESTAMPTZ;
ALTER TABLE voucher_codes ADD COLUMN IF NOT EXISTS is_system_generated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE voucher_codes ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;

-- 2. Add missing columns to tenant_subscriptions
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS remaining_days INTEGER NOT NULL DEFAULT 30;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS pending_expires_at TIMESTAMPTZ;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS pending_timeout_hours INTEGER NOT NULL DEFAULT 24;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS system_voucher_code TEXT;

-- 3. Add missing column to tenants
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan_priority INTEGER NOT NULL DEFAULT 0;

-- 4. Add missing column to saas_plans
ALTER TABLE saas_plans ADD COLUMN IF NOT EXISTS pending_timeout_hours INTEGER NOT NULL DEFAULT 24;

-- 5. Backfill redeemed_at for existing voucher_codes
UPDATE voucher_codes SET redeemed_at = used_at WHERE used_at IS NOT NULL AND redeemed_at IS NULL;

-- 6. Backfill plan_priority for existing tenants
UPDATE tenants SET plan_priority = 1 WHERE plan = 'lite' AND plan_priority = 0;
UPDATE tenants SET plan_priority = 2 WHERE plan = 'pro' AND plan_priority = 0;
UPDATE tenants SET plan_priority = 3 WHERE plan = 'ultimate' AND plan_priority = 0;

-- 7. Create voucher_subscriptions table if not exists (needed for day-duration accumulation)
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

CREATE INDEX IF NOT EXISTS idx_voucher_subs_tenant ON voucher_subscriptions(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_voucher_subs_tenant_plan ON voucher_subscriptions(tenant_id, plan_id);

-- 8. Create pending_tenant_cleanup_log table if not exists
CREATE TABLE IF NOT EXISTS pending_tenant_cleanup_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    email TEXT,
    phone TEXT,
    cleaned_up_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT NOT NULL DEFAULT 'pending_timeout'
);
