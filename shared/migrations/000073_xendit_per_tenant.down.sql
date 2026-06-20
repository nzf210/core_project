-- +migrate Down
ALTER TABLE tenants
DROP COLUMN IF EXISTS xendit_merchant_id;