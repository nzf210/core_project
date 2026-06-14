ALTER TABLE tenant_chatbot_configs
  DROP COLUMN IF EXISTS enable_vision,
  DROP COLUMN IF EXISTS enable_voice_reply,
  DROP COLUMN IF EXISTS voice_model;
