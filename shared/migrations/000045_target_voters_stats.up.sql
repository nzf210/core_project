-- F032 Supplement: Target vs DPT Statistics Tracking

-- Extend the existing location tables to include target and total voters
ALTER TABLE provinces ADD COLUMN IF NOT EXISTS total_voters INT NOT NULL DEFAULT 0;
ALTER TABLE regencies ADD COLUMN IF NOT EXISTS total_voters INT NOT NULL DEFAULT 0;
ALTER TABLE districts ADD COLUMN IF NOT EXISTS total_voters INT NOT NULL DEFAULT 0;
ALTER TABLE villages ADD COLUMN IF NOT EXISTS total_voters INT NOT NULL DEFAULT 0;

ALTER TABLE tps ADD COLUMN IF NOT EXISTS total_voters INT NOT NULL DEFAULT 0;

-- Optional: Add a materialized view or aggregated table for fast dashboard querying
CREATE TABLE IF NOT EXISTS campaign_target_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    region_type VARCHAR(50) NOT NULL, -- 'province', 'regency', 'district', 'village', 'tps'
    region_id UUID NOT NULL,          -- ID of the respective region
    total_dpt_voters INT NOT NULL DEFAULT 0, -- Sourced from DPT
    target_voters INT NOT NULL DEFAULT 0,    -- Our campaign target
    achieved_voters INT NOT NULL DEFAULT 0,  -- Count of valid endorsements
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_campaign_region UNIQUE (campaign_id, region_type, region_id)
);
CREATE INDEX IF NOT EXISTS idx_campaign_target_stats ON campaign_target_stats(campaign_id, region_type);