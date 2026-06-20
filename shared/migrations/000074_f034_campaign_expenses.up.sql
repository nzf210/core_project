-- F034: Cost-per-Vote — tabel campaign_expenses
CREATE TABLE IF NOT EXISTS campaign_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    expense_category VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL, -- satuan sen
    target_region_type VARCHAR(50),
    target_region_id UUID,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_expenses_tenant ON campaign_expenses(tenant_id);
CREATE INDEX IF NOT EXISTS idx_campaign_expenses_campaign ON campaign_expenses(campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_expenses_category ON campaign_expenses(expense_category);