-- F048: Seed wa_cloud_api plan_features for each plan
-- Lock Cloud API access via plan_features feature_key='wa_cloud_api'

-- Ensure unique constraint allows idempotency
INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
SELECT id, 'wa_cloud_api', 'WA Cloud API Access (Meta)', 'boolean', false
FROM saas_plans WHERE id = 'lite'
ON CONFLICT (plan_id, feature_key) DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
SELECT id, 'wa_cloud_api', 'WA Cloud API Access (Meta)', 'boolean', true
FROM saas_plans WHERE id = 'pro'
ON CONFLICT (plan_id, feature_key) DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
SELECT id, 'wa_cloud_api', 'WA Cloud API Access (Meta)', 'boolean', true
FROM saas_plans WHERE id = 'ultimate'
ON CONFLICT (plan_id, feature_key) DO NOTHING;

-- If plans use different naming, the SELECT WHERE id IN (...) will simply find no match.
-- Plans: 'lite', 'pro', 'ultimate' (verified via 000025 seed data).
