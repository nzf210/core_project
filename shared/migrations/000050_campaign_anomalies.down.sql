DROP INDEX IF EXISTS idx_endorsements_anomaly;
ALTER TABLE endorsements
DROP COLUMN IF EXISTS anomaly_reason,
DROP COLUMN IF EXISTS is_anomaly;
