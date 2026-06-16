-- Migration F048: Add WA Provider Preference to tenant_chatbot_configs
-- Enum: auto (default), whatsmeow, cloud_api

CREATE TYPE wa_provider_enum AS ENUM ('auto', 'whatsmeow', 'cloud_api');

ALTER TABLE tenant_chatbot_configs 
ADD COLUMN IF NOT EXISTS wa_provider_preference wa_provider_enum NOT NULL DEFAULT 'auto';

-- Migration for Auth/Verif preference? 
-- Auth/Verif setting biasanya di tabel 'tenants' atau global config,
-- tapi karena ini untuk verifikasi pendaftaran, kita simpan di tabel 'tenants'
-- karena tenant belum punya 'tenant_chatbot_configs' saat pendaftaran awal.

ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS auth_wa_provider_preference wa_provider_enum NOT NULL DEFAULT 'auto';

-- For internal/superadmin notifications, let's keep it in a global config table or app config,
-- tapi untuk sekarang kita bisa pakai tenant setting superadmin.
