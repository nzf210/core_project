-- 000066_fix_voucher_codes_missing_columns.down.sql
-- Rollback: remove columns added by fix migration 000066

ALTER TABLE voucher_codes DROP COLUMN IF EXISTS validity_days;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS redeemed_at;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS is_system_generated;
ALTER TABLE voucher_codes DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS status;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS remaining_days;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS pending_expires_at;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS pending_timeout_hours;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS system_voucher_code;

ALTER TABLE tenants DROP COLUMN IF EXISTS plan_priority;

ALTER TABLE saas_plans DROP COLUMN IF EXISTS pending_timeout_hours;

DROP TABLE IF EXISTS voucher_subscriptions;
DROP TABLE IF EXISTS pending_tenant_cleanup_log;
