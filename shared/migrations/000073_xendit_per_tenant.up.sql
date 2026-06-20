-- +migrate Up
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS xendit_merchant_id VARCHAR(255);

-- Index for fast lookup by merchant ID (useful for webhook routing)
CREATE INDEX IF NOT EXISTS idx_tenants_xendit_merchant_id
    ON tenants(xendit_merchant_id)
    WHERE xendit_merchant_id IS NOT NULL;

COMMENT ON COLUMN tenants.xendit_merchant_id IS 'Xendit merchant account ID for per-tenant payment routing';