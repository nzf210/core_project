ALTER TABLE tenants DROP COLUMN IF EXISTS auth_wa_provider_preference;

ALTER TABLE tenant_chatbot_configs DROP COLUMN IF EXISTS wa_provider_preference;

DROP TYPE IF EXISTS wa_provider_enum;
