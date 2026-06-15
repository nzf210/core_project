-- F036: Dashboard Sentimen Isu Harian (AI NLP)

CREATE TABLE IF NOT EXISTS village_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    volunteer_id UUID REFERENCES volunteers(id) ON DELETE SET NULL,
    village_id UUID NOT NULL REFERENCES villages(id) ON DELETE CASCADE,
    raw_message TEXT NOT NULL,          -- Pesan asli WA dari relawan (contoh: "warga ngeluh pupuk mahal susah dicari")
    extracted_issue VARCHAR(255),       -- Ekstrak AI (contoh: "Kelangkaan Pupuk")
    sentiment_score DECIMAL(3, 2),      -- -1.00 (Sangat Negatif) sampai 1.00 (Sangat Positif)
    urgency_level VARCHAR(20),          -- 'high', 'medium', 'low'
    reported_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_village_issues_campaign ON village_issues(campaign_id);
CREATE INDEX IF NOT EXISTS idx_village_issues_village ON village_issues(village_id);
