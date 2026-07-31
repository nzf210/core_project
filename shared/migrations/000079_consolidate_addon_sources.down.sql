-- Rollback: F057 - Consolidate addon sources
-- Remove addons that were inserted from addon_prices
DELETE FROM available_features WHERE feature_key IN ('ai_audio', 'wa_blast', 'wa_meta_session');
