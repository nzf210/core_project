-- Revert F044 campaign_licenses extensions
ALTER TABLE campaign_licenses DROP COLUMN IF EXISTS validity_days;
ALTER TABLE campaign_licenses DROP COLUMN IF EXISTS max_voters;
ALTER TABLE campaign_licenses DROP COLUMN IF EXISTS program_name;
ALTER TABLE campaign_licenses DROP COLUMN IF EXISTS id;
