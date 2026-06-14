-- 000034_billing_cycle.up.sql
-- Add billing_cycle column to support monthly and yearly billing options

ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly';

ALTER TABLE tenant_subscriptions
    ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly';
