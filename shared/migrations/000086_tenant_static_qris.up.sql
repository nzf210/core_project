ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS static_qris_payload TEXT;

ALTER TABLE pos_transactions
ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(50);
