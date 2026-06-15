ALTER TABLE campaigns 
DROP COLUMN IF EXISTS max_voters,
DROP COLUMN IF EXISTS wargame_tokens,
DROP COLUMN IF EXISTS active_addons;

DROP TABLE IF EXISTS campaign_billing_orders;
DROP TABLE IF EXISTS campaign_licenses;
