-- 000035_billing_cycle.up.sql
-- Add billing_cycle column to support monthly and yearly billing options

-- Create invoices table if it doesn't exist (schema.sql was never run via migration)
CREATE TABLE IF NOT EXISTS invoices (
    id          VARCHAR(100) PRIMARY KEY,
    tenant_id   VARCHAR(100) NOT NULL,
    plan_id     VARCHAR(50)  NOT NULL,
    amount      DECIMAL(15, 2) DEFAULT 0,
    status      VARCHAR(20)  DEFAULT 'pending',
    payment_url VARCHAR(255) DEFAULT '',
    voucher_code VARCHAR(100) DEFAULT '',
    paid_at     TIMESTAMPTZ,
    billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly'
);

-- Add billing_cycle to tenant_subscriptions if not already present
ALTER TABLE tenant_subscriptions
    ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly';
