-- 000026_multi_store.down.sql
-- Rollback multi-store feature

DROP INDEX IF EXISTS idx_stores_active;
DROP INDEX IF EXISTS idx_stores_business_type;
DROP INDEX IF EXISTS idx_stores_tenant;
DROP INDEX IF EXISTS idx_stores_owner;

DROP TABLE IF EXISTS stores;

DELETE FROM plan_features WHERE feature_key = 'max_stores';
