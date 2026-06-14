-- 000034_billing_cycle.down.sql

ALTER TABLE invoices DROP COLUMN IF EXISTS billing_cycle;
ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS billing_cycle;
