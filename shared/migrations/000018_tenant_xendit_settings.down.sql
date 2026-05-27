ALTER TABLE tenants 
DROP COLUMN IF EXISTS xendit_api_key,
DROP COLUMN IF EXISTS xendit_webhook_token,
ADD COLUMN qris_data TEXT;
