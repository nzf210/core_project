-- F033: Campaign Logistics Tracking Schema

CREATE TABLE IF NOT EXISTS logistic_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL, -- e.g. "Kaos Paslon", "Sembako"
    total_quantity INT NOT NULL,
    unit VARCHAR(50), -- e.g. "pcs", "paket"
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS logistic_distributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES logistic_items(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES volunteers(id) ON DELETE SET NULL, -- Who sent it
    receiver_id UUID REFERENCES volunteers(id) ON DELETE SET NULL, -- Who received it
    target_region_type VARCHAR(50), -- 'province', 'regency', 'district', 'village', 'tps'
    target_region_id UUID, -- References the specific region
    quantity INT NOT NULL,
    status VARCHAR(50) DEFAULT 'in_transit', -- in_transit, received, lost
    proof_image_url VARCHAR(255), -- Selfie proof of receipt
    location_lat DECIMAL(10, 8), -- Share location latitude
    location_lng DECIMAL(11, 8), -- Share location longitude
    notes TEXT,
    sent_at TIMESTAMPTZ DEFAULT NOW(),
    received_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_logistics_dist_item ON logistic_distributions(item_id);
CREATE INDEX IF NOT EXISTS idx_logistics_dist_receiver ON logistic_distributions(receiver_id);
