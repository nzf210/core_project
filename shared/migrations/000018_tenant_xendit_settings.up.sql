ALTER TABLE tenants 
DROP COLUMN IF EXISTS qris_data,
ADD COLUMN xendit_api_key VARCHAR(255),
ADD COLUMN xendit_webhook_token VARCHAR(255);
