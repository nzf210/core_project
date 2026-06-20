-- F050 rollback: remove ai_image plan_feature
DELETE FROM plan_features WHERE feature_key = 'ai_image';

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'plan_features_numeric') THEN
        DELETE FROM plan_features_numeric WHERE feature_key = 'ai_image';
    END IF;
END $$;