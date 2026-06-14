-- 000039_plan_features_numeric.down.sql
ALTER TABLE plan_features
    DROP COLUMN IF EXISTS max_users,
    DROP COLUMN IF EXISTS max_transactions,
    DROP COLUMN IF EXISTS max_ai_text,
    DROP COLUMN IF EXISTS max_ai_vision,
    DROP COLUMN IF EXISTS max_ai_audio_minutes,
    DROP COLUMN IF EXISTS max_image_gen,
    DROP COLUMN IF EXISTS max_products,
    DROP COLUMN IF EXISTS max_customers,
    DROP COLUMN IF EXISTS max_storage_mb,
    DROP COLUMN IF EXISTS api_rate_limit_per_min,
    DROP COLUMN IF EXISTS data_retention_months;
