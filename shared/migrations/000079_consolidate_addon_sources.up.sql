-- F057 AC-4: Consolidate legacy addon_prices into available_features
-- addon_prices (F034 legacy) keys: ai_audio_stt, wa_blast_api, wa_session_meta
-- available_features (F052/F057 primary) — consolidate to single source of truth
--
-- ai_vision: already exists in available_features (from F052 seed)
-- ai_audio_stt: INSERT as ai_audio (available_features key = 'ai_audio')
-- wa_blast_api: INSERT as wa_blast (available_features already has wa_blast)
-- wa_session_meta: INSERT as new 'wa_meta_session' addon
--
-- After this migration:
--   - addon_prices table is READ-ONLY (no code should INSERT/UPDATE it)
--   - All purchase flows read from available_features only

-- 1. Upsert ai_audio — from legacy ai_audio_stt (per_minute)
INSERT INTO available_features (feature_key, feature_name, description, category, is_addon, default_enabled, addon_price_rupiah, addon_unit)
SELECT 'ai_audio', 'AI Audio (Voice Note)', ap.description,
       'ai', true, NULL, ap.price_rupiah, ap.unit
FROM addon_prices ap
WHERE ap.addon_key = 'ai_audio_stt'
ON CONFLICT (feature_key) DO UPDATE
  SET addon_price_rupiah = EXCLUDED.addon_price_rupiah,
      addon_unit = EXCLUDED.addon_unit,
      description = COALESCE(NULLIF(EXCLUDED.description, ''), available_features.description);

-- 2. wa_blast — available_features already has 'wa_blast', sync price
UPDATE available_features af
SET addon_price_rupiah = ap.price_rupiah,
    addon_unit = ap.unit,
    description = COALESCE(ap.description, af.description)
FROM addon_prices ap
WHERE af.feature_key = 'wa_blast'
  AND ap.addon_key = 'wa_blast_api';

-- 3. Upsert wa_meta_session — from legacy wa_session_meta (per_session)
INSERT INTO available_features (feature_key, feature_name, description, category, is_addon, default_enabled, addon_price_rupiah, addon_unit)
SELECT 'wa_meta_session', 'WA Meta Session (Cloud API)', ap.description,
       'wa', true, NULL, ap.price_rupiah, ap.unit
FROM addon_prices ap
WHERE ap.addon_key = 'wa_session_meta'
ON CONFLICT (feature_key) DO UPDATE
  SET addon_price_rupiah = EXCLUDED.addon_price_rupiah,
      addon_unit = EXCLUDED.addon_unit,
      description = COALESCE(NULLIF(EXCLUDED.description, ''), available_features.description);

-- 4. ai_vision — sync from addon_prices to available_features if needed
UPDATE available_features af
SET addon_price_rupiah = ap.price_rupiah,
    description = COALESCE(ap.description, af.description)
FROM addon_prices ap
WHERE af.feature_key = 'ai_vision'
  AND ap.addon_key = 'ai_vision';
