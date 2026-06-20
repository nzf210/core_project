-- F052: Rollback
DROP TABLE IF EXISTS tenant_addons;
DROP TABLE IF EXISTS available_features;
ALTER TABLE plan_features DROP COLUMN IF EXISTS min_tier;
