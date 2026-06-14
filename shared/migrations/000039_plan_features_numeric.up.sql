-- 000039_plan_features_numeric.up.sql
-- Add numeric quota columns to plan_features for runtime enforcement
-- (Previously: feature_value was free-form VARCHAR(50), now structured)

ALTER TABLE plan_features
    ADD COLUMN IF NOT EXISTS max_users INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_transactions INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_text INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_vision INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_audio_minutes INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_image_gen INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_products INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_customers INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_storage_mb INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS api_rate_limit_per_min INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS data_retention_months INT DEFAULT 0;
