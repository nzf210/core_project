-- F034: Extend addon_prices so superadmin can edit unit + is_active
-- Also backfill description for existing seeded rows

ALTER TABLE addon_prices
    ADD COLUMN IF NOT EXISTS unit VARCHAR(20) NOT NULL DEFAULT 'per_request',
    ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP;

-- Backfill description for seeded rows (idempotent — already empty)
UPDATE addon_prices SET description =
    CASE addon_key
        WHEN 'ai_vision'      THEN 'Analisis gambar (KTP, produk, dokumen) per request'
        WHEN 'ai_audio_stt'   THEN 'Konversi voice note ke teks per menit'
        WHEN 'wa_blast_api'   THEN 'Broadcast pesan massal via Meta Cloud API per request'
        WHEN 'wa_session_meta' THEN 'WhatsApp Cloud API per sesi aktif per bulan'
    END
WHERE description = '' OR description IS NULL;

-- Backfill unit from seed values (idempotent)
UPDATE addon_prices SET unit = 'per_request'
WHERE addon_key IN ('ai_vision', 'wa_blast_api')
  AND (unit = '' OR unit = 'per_request' AND description = '');

UPDATE addon_prices SET unit = 'per_minute'
WHERE addon_key = 'ai_audio_stt'
  AND (unit = '' OR unit = 'per_request' AND description = '');

UPDATE addon_prices SET unit = 'per_session'
WHERE addon_key = 'wa_session_meta'
  AND (unit = '' OR unit = 'per_request' AND description = '');

-- Ensure description not empty after backfill
UPDATE addon_prices SET description = 'Addon pricing' WHERE description = '' OR description IS NULL;
