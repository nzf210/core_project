DROP TABLE IF EXISTS tenant_module_config;
ALTER TABLE tenants DROP COLUMN IF EXISTS business_address;
ALTER TABLE tenants DROP COLUMN IF EXISTS business_name;
ALTER TABLE tenants DROP COLUMN IF EXISTS onboarding_completed;
ALTER TABLE tenants DROP COLUMN IF EXISTS business_type;
DROP TABLE IF EXISTS usage_quotas;
DROP TABLE IF EXISTS business_types;
