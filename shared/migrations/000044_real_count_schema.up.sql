-- F032: Modul Saksi & Real Count Schema

-- 1. Saksi Assignment (Extend volunteers if needed, or use a specific relation)
-- We add 'tps_id_assigned' to the existing volunteers table via ALTER,
-- but since we're in a fresh migration context for this module, let's create a dedicated table
-- to track their attendance and C1 submissions clearly.

CREATE TABLE IF NOT EXISTS saksi_attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    volunteer_id UUID NOT NULL REFERENCES volunteers(id) ON DELETE CASCADE,
    tps_id UUID NOT NULL REFERENCES tps(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    selfie_image_url VARCHAR(255),
    status VARCHAR(50) DEFAULT 'present', -- present, late, absent
    verified_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_saksi_att_tps ON saksi_attendances(tps_id);

-- 2. Real Count Records
CREATE TABLE IF NOT EXISTS real_count_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    tps_id UUID NOT NULL REFERENCES tps(id) ON DELETE CASCADE,
    volunteer_id UUID NOT NULL REFERENCES volunteers(id) ON DELETE CASCADE,
    
    -- The numbers reported by the saksi
    reported_candidate_votes INT NOT NULL DEFAULT 0,
    reported_opponent_votes INT NOT NULL DEFAULT 0,
    reported_invalid_votes INT NOT NULL DEFAULT 0,
    
    -- The numbers extracted by AI Vision
    ai_candidate_votes INT,
    ai_opponent_votes INT,
    ai_invalid_votes INT,
    
    c1_image_url VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending_review', -- auto_verified, pending_review, rejected, human_verified
    notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure one TPS only has one valid Real Count record per campaign
    CONSTRAINT unique_tps_real_count UNIQUE (campaign_id, tps_id)
);
CREATE INDEX IF NOT EXISTS idx_real_count_campaign ON real_count_records(campaign_id);
