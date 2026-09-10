ALTER TABLE pos_transactions
DROP COLUMN IF EXISTS customer_phone;

ALTER TABLE tenants
DROP COLUMN IF EXISTS static_qris_payload;
