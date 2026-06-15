-- Down migration for Target Voters Stats

DROP TABLE IF EXISTS campaign_target_stats CASCADE;

ALTER TABLE tps DROP COLUMN IF EXISTS total_voters;
ALTER TABLE villages DROP COLUMN IF EXISTS total_voters;
ALTER TABLE districts DROP COLUMN IF EXISTS total_voters;
ALTER TABLE regencies DROP COLUMN IF EXISTS total_voters;
ALTER TABLE provinces DROP COLUMN IF EXISTS total_voters;