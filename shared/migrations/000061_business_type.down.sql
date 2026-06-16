-- Rollback: remove business_type columns from tenants
ALTER TABLE tenants DROP COLUMN IF EXISTS clinic_services;
ALTER TABLE tenants DROP COLUMN IF NOT EXISTS clinic_doctors;
ALTER TABLE tenants DROP COLUMN IF EXISTS business_type;
DROP INDEX IF EXISTS idx_tenants_business_type;