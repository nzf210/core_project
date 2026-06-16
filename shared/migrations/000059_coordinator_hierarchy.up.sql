-- Coordinator Hierarchy for Campaign (F046)
-- Stores the assignment of coordinators at each level: province, regency, district, village, tps

CREATE TABLE campaign_coordinators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    citizen_nik VARCHAR(16) NOT NULL, -- Must exist in citizens table
    coordinator_level VARCHAR(20) NOT NULL CHECK (coordinator_level IN ('korprov', 'korKab', 'korKec', 'korKades', 'saksi_tps')),
    region_id UUID NOT NULL, -- References provinces/regencies/districts/villages/tps
    assigned_by_user_id UUID REFERENCES users(id), -- Who made this assignment
    is_active BOOLEAN DEFAULT TRUE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(campaign_id, coordinator_level, region_id) -- One coordinator per level per region per campaign
);

-- Index for fast lookups by citizenship
CREATE INDEX idx_campaign_coordinators_nik ON campaign_coordinators(citizen_nik);
CREATE INDEX idx_campaign_coordinators_campaign ON campaign_coordinators(campaign_id);
CREATE INDEX idx_campaign_coordinators_level ON campaign_coordinators(coordinator_level);

-- Add max_coordinators column to campaigns for dynamic limits (optional)
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS coordinator_limits JSONB DEFAULT '{"korprov": 1, "korKab": 30, "korKec": 200, "korKades": 3000, "saksi_tps": -1}'; -- -1 = unlimited

-- Add foreign key constraint for citizen_nik (optional, requires citizens table)
-- ALTER TABLE campaign_coordinators ADD CONSTRAINT fk_citizen_nik FOREIGN KEY (citizen_nik) REFERENCES citizens(nik) ON DELETE CASCADE;