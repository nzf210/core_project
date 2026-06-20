-- F050 rollback: remove ai_text, ai_vision, ai_image plan_features
DELETE FROM plan_features WHERE feature_key IN ('ai_text', 'ai_vision', 'ai_image');