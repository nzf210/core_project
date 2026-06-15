ALTER TABLE endorsements
ADD COLUMN IF NOT EXISTS is_anomaly BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS anomaly_reason TEXT;

CREATE INDEX idx_endorsements_anomaly ON endorsements(tenant_id, is_anomaly) WHERE is_anomaly = TRUE;
