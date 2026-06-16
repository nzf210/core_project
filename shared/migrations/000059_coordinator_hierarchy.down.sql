-- Rollback: Drop coordinator hierarchy

DROP INDEX IF EXISTS idx_campaign_coordinators_campaign;
DROP INDEX IF EXISTS idx_campaign_coordinators_level;
DROP INDEX IF EXISTS idx_campaign_coordinators_nik;

ALTER TABLE campaigns DROP COLUMN IF EXISTS coordinator_limits;

DROP TABLE IF EXISTS campaign_coordinators CASCADE;