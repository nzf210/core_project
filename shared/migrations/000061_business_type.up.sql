-- Add business_type to tenants table (F047)
-- Restricts module access based on business type

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_type VARCHAR(20) DEFAULT 'general';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS clinic_doctors TEXT[] DEFAULT '{}';

-- Add clinic_services JSONB for specializations
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS clinic_services JSONB DEFAULT '[]'::JSONB;

-- Add index for fast lookup
CREATE INDEX IF NOT EXISTS idx_tenants_business_type ON tenants(business_type);