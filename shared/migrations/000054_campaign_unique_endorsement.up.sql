-- F031 (Anti-Double) Enforcement
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_endorsement ON endorsements(citizen_id, campaign_id);
