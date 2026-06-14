-- 000040_seed_plan_features_numeric.down.sql
-- Rollback: reset numeric quotas to 0 for all plans

UPDATE plan_features SET
    max_users = 0, max_transactions = 0, max_ai_text = 0,
    max_ai_vision = 0, max_ai_audio_minutes = 0, max_image_gen = 0,
    max_products = 0, max_customers = 0, max_storage_mb = 0,
    api_rate_limit_per_min = 0, data_retention_months = 0
WHERE plan_id IN ('lite', 'pro', 'ultimate');
