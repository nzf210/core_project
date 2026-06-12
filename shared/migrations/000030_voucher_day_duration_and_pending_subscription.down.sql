-- 000030_voucher_day_duration_and_pending_subscription.down.sql

DROP TABLE IF EXISTS pending_tenant_cleanup_log;
DROP TABLE IF EXISTS voucher_subscriptions;
DROP INDEX IF EXISTS idx_voucher_subs_tenant;
DROP INDEX IF EXISTS idx_voucher_subs_tenant_plan;

ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS system_voucher_code;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS pending_timeout_hours;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS pending_expires_at;

ALTER TABLE tenants DROP COLUMN IF EXISTS plan_priority;

ALTER TABLE voucher_codes DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS is_system_generated;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS redeemed_at;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS remaining_days;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS validity_days;

ALTER TABLE voucher_codes ADD COLUMN valid_until TIMESTAMPTZ;
ALTER TABLE tenant_subscriptions ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

ALTER TABLE saas_plans DROP COLUMN IF EXISTS pending_timeout_hours;