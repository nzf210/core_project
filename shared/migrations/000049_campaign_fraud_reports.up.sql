-- F038: Peta Kerawanan & Pelaporan Pelanggaran Schema

CREATE TABLE IF NOT EXISTS fraud_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    volunteer_id UUID REFERENCES volunteers(id) ON DELETE SET NULL,
    reporter_name VARCHAR(255) NOT NULL,
    violation_type VARCHAR(100) NOT NULL, -- e.g. "money_politics", "banner_destruction", "black_campaign"
    description TEXT,
    proof_image_url VARCHAR(255),
    location_lat DECIMAL(10, 8) NOT NULL,
    location_lng DECIMAL(11, 8) NOT NULL,
    status VARCHAR(50) DEFAULT 'reported', -- reported, verified_by_admin, submitted_to_bawaslu
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fraud_reports_campaign ON fraud_reports(campaign_id);
CREATE INDEX IF NOT EXISTS idx_fraud_reports_location ON fraud_reports(location_lat, location_lng);
