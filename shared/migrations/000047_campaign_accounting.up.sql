-- F034: Campaign Cost-per-Vote / Finance Schema

CREATE TABLE IF NOT EXISTS campaign_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    expense_category VARCHAR(100) NOT NULL, -- e.g. "logistik", "operasional_saksi", "marketing"
    amount DECIMAL(15, 2) NOT NULL,
    target_region_type VARCHAR(50), -- To calculate cost per region
    target_region_id UUID, 
    accounting_transaction_id UUID, -- References UMKM Accounting engine transaction if linked
    description TEXT,
    date TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_expenses_campaign ON campaign_expenses(campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_expenses_region ON campaign_expenses(target_region_type, target_region_id);
