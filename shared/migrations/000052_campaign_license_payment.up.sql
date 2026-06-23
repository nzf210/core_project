-- F044: Campaign Modular License & Payment System

-- 1. Campaign License Keys (B2B Manual)
CREATE TABLE IF NOT EXISTS campaign_licenses (
    license_key VARCHAR(50) PRIMARY KEY,
    election_type VARCHAR(50),
    base_quota INT DEFAULT 5000,
    addons JSONB DEFAULT '[]'::jsonb,
    wargame_tokens INT DEFAULT 0,
    price_rupiah BIGINT DEFAULT 0,
    is_used BOOLEAN DEFAULT FALSE,
    used_by_tenant UUID REFERENCES tenants(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    used_at TIMESTAMP WITH TIME ZONE
);

-- 2. Campaign Billing Orders (Self-Service Xendit)
CREATE TABLE IF NOT EXISTS campaign_billing_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    order_type VARCHAR(50) NOT NULL, -- 'wargame_token', 'intelligence_pack'
    amount_rupiah BIGINT NOT NULL,
    quantity INT DEFAULT 1,
    invoice_url TEXT,
    xendit_invoice_id VARCHAR(100),
    status VARCHAR(50) DEFAULT 'PENDING', -- 'PENDING', 'PAID', 'EXPIRED'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    paid_at TIMESTAMP WITH TIME ZONE
);

-- 3. Extend Campaigns table for feature unlock
ALTER TABLE campaigns 
ADD COLUMN IF NOT EXISTS active_addons JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS wargame_tokens INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS max_voters INT DEFAULT 5000;
