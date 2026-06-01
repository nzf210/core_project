-- 000027_voucher_links_and_freeze.down.sql
-- Rollback voucher-link + freeze support

DROP INDEX IF EXISTS idx_voucher_gen_logs_created;
DROP INDEX IF EXISTS idx_voucher_gen_logs_program;
DROP TABLE IF EXISTS voucher_generation_logs;

DROP INDEX IF EXISTS idx_tenants_frozen;
ALTER TABLE tenants DROP COLUMN IF EXISTS current_plan_expires_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS frozen_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS is_frozen;

DROP INDEX IF EXISTS idx_tenant_subs_status_expires;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS frozen_reason;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS frozen_at;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS status;

DROP INDEX IF EXISTS idx_voucher_links_expires;
DROP INDEX IF EXISTS idx_voucher_links_active;
DROP INDEX IF EXISTS idx_voucher_links_redeemed;
DROP INDEX IF EXISTS idx_voucher_links_program;
DROP TABLE IF EXISTS voucher_links;
