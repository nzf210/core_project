ALTER TABLE usage_quotas ALTER COLUMN plan_tier SET DEFAULT 'free';
UPDATE usage_quotas SET plan_tier = 'free' WHERE plan_tier = 'lite';
UPDATE tenants SET plan = 'free' WHERE plan = 'lite';

DROP TABLE IF EXISTS tenant_coupons;
DROP TABLE IF EXISTS coupons;
