-- F050: AI Quota Per-Modalitas (Text/Vision/Image)
-- Add explicit ai_text, ai_vision, ai_image plan_feature keys.
-- Previously `image_gen` was used directly as a quota key; now we use structured
-- feature keys (ai_text, ai_vision, ai_image) that match quota.go / CheckQuota.

INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
VALUES
    -- Lite: no AI features beyond basic chatbot
    -- Pro: bundled AI text + limited vision/image
    ('pro',      'ai_text',  'AI Text generation (chat, embeddings, RAG)', 'unlimited', true),
    ('pro',      'ai_vision','AI Vision analysis (KTP OCR, product photo)', '50',       true),
    ('pro',      'ai_image', 'AI Image generation (banner, product)',       '10',       true),
    -- Ultimate: higher limits
    ('ultimate', 'ai_text',  'AI Text generation (chat, embeddings, RAG)', 'unlimited', true),
    ('ultimate', 'ai_vision','AI Vision analysis (KTP OCR, product photo)', '500',      true),
    ('ultimate', 'ai_image', 'AI Image generation (banner, product)',       '30',       true)
ON CONFLICT (plan_id, feature_key) DO UPDATE
SET feature_name    = EXCLUDED.feature_name,
    feature_value   = EXCLUDED.feature_value;

-- Seed per-tier limits into plan_features numeric columns if the table has them
-- (added in migration 000039). Safe no-op if columns don't exist yet.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'plan_features' AND column_name = 'max_image_gen') THEN
        UPDATE plan_features SET max_image_gen = 0  WHERE plan_id = 'lite'  AND feature_key = 'ai_image';
        UPDATE plan_features SET max_image_gen = 10 WHERE plan_id = 'pro'   AND feature_key = 'ai_image';
        UPDATE plan_features SET max_image_gen = 30 WHERE plan_id = 'ultimate' AND feature_key = 'ai_image';
    END IF;
END $$;