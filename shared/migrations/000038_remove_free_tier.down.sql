-- 000038_remove_free_tier.down.sql
-- Rollback: perkenalkan kembali tier "free" + migrate lite → free.

UPDATE tenants SET plan = 'free' WHERE plan = 'lite';
UPDATE usage_quotas SET plan_tier = 'free' WHERE plan_tier = 'lite';

ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'free';
ALTER TABLE usage_quotas ALTER COLUMN plan_tier SET DEFAULT 'free';
