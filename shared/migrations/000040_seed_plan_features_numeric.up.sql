-- 000040_seed_plan_features_numeric.up.sql
-- Single source of truth for all tier quotas (post F024)

-- LITE (Rp 150rb/bln)
UPDATE plan_features SET
    max_users = 3, max_transactions = 1000, max_ai_text = 250,
    max_ai_vision = 0, max_ai_audio_minutes = 0, max_image_gen = 0,
    max_products = 100, max_customers = 500, max_storage_mb = 1000,
    api_rate_limit_per_min = 60, data_retention_months = 12
WHERE plan_id = 'lite';

-- PRO (Rp 450rb/bln)
UPDATE plan_features SET
    max_users = 10, max_transactions = 10000, max_ai_text = 5000,
    max_ai_vision = 50, max_ai_audio_minutes = 0, max_image_gen = 0,
    max_products = 1000, max_customers = 5000, max_storage_mb = 10000,
    api_rate_limit_per_min = 300, data_retention_months = 36
WHERE plan_id = 'pro';

-- ULTIMATE (Rp 1.5jt/bln)
UPDATE plan_features SET
    max_users = -1, max_transactions = -1, max_ai_text = -1,
    max_ai_vision = 500, max_ai_audio_minutes = 60, max_image_gen = 30,
    max_products = -1, max_customers = -1, max_storage_mb = -1,
    api_rate_limit_per_min = 1000, data_retention_months = 60
WHERE plan_id = 'ultimate';
