-- F050: AI Quota Per-Modalitas (Text/Vision/Image)
-- Add explicit `ai_image` plan_feature key. Previously `image_gen` was used directly.
-- Keep `image_gen` for backward compat but route through `ai_image` semantic key.

INSERT INTO plan_features (feature_key, description, category, is_addon, allowed_plans, default_limit, unit)
VALUES
    ('ai_text',   'AI Text generation (chat, embeddings, RAG)', 'ai', false, ARRAY['lite','pro','ultimate'], 0,  NULL),
    ('ai_vision', 'AI Vision analysis (KTP OCR, product photo)', 'ai', false, ARRAY['pro','ultimate'],       0,  NULL),
    ('ai_image',  'AI Image generation (banner, product)',     'ai', true,  ARRAY['pro','ultimate'],       NULL, 'per_request')
ON CONFLICT (feature_key) DO UPDATE
SET description = EXCLUDED.description,
    category    = EXCLUDED.category,
    is_addon    = EXCLUDED.is_addon,
    allowed_plans = EXCLUDED.allowed_plans;

-- Seed default limits if missing (mirror existing image_gen → ai_image)
UPDATE plan_features SET default_limit = 30
WHERE feature_key = 'ai_image' AND default_limit IS NULL;

-- Seed per-tier limits into plan_features_numeric-style table if present.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'plan_features_numeric') THEN
        INSERT INTO plan_features_numeric (plan_id, feature_key, max_value)
        SELECT 'lite', 'ai_image', 0
        WHERE NOT EXISTS (SELECT 1 FROM plan_features_numeric WHERE plan_id = 'lite' AND feature_key = 'ai_image');
        INSERT INTO plan_features_numeric (plan_id, feature_key, max_value)
        SELECT 'pro', 'ai_image', 10
        WHERE NOT EXISTS (SELECT 1 FROM plan_features_numeric WHERE plan_id = 'pro' AND feature_key = 'ai_image');
        INSERT INTO plan_features_numeric (plan_id, feature_key, max_value)
        SELECT 'ultimate', 'ai_image', 30
        WHERE NOT EXISTS (SELECT 1 FROM plan_features_numeric WHERE plan_id = 'ultimate' AND feature_key = 'ai_image');
    END IF;
END $$;