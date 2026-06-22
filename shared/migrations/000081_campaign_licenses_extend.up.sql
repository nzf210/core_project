-- F044: Extend campaign_licenses table to match frontend requirements
-- Adds: id (UUID PK), program_name, max_voters, validity_days

-- Add UUID primary key column
ALTER TABLE campaign_licenses DROP CONSTRAINT IF EXISTS campaign_licenses_pkey CASCADE;
ALTER TABLE campaign_licenses ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE campaign_licenses DROP CONSTRAINT IF EXISTS campaign_licenses_license_key_key;
ALTER TABLE campaign_licenses ADD CONSTRAINT campaign_licenses_license_key_key UNIQUE (license_key);

-- Add program_name (optional label for the campaign)
ALTER TABLE campaign_licenses ADD COLUMN IF NOT EXISTS program_name VARCHAR(255) DEFAULT '';

-- Add max_voters (alias for base_quota, clearer naming)
ALTER TABLE campaign_licenses ADD COLUMN IF NOT EXISTS max_voters INT DEFAULT 5000;

-- Add validity_days (how long the license is valid from creation)
ALTER TABLE campaign_licenses ADD COLUMN IF NOT EXISTS validity_days INT DEFAULT 365;

-- Backfill max_voters from base_quota for existing rows
UPDATE campaign_licenses SET max_voters = base_quota WHERE max_voters IS NULL OR max_voters = 0;
