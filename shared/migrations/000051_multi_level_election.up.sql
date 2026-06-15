-- F043: Multi-Level Election & Dapils

-- Tipe Pemilihan
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS election_type VARCHAR(50) DEFAULT 'pilkada';
-- pilkada, dpr_ri, dprd_prov, dprd_kab, dpd

-- Tabel Dapil
CREATE TABLE IF NOT EXISTS dapils (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    dapil_name VARCHAR(255) NOT NULL,
    total_seats INT NOT NULL DEFAULT 1,
    dpt_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabel Kompetitor Partai (Lawan)
CREATE TABLE IF NOT EXISTS competitor_parties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    dapil_id UUID NOT NULL REFERENCES dapils(id) ON DELETE CASCADE,
    party_name VARCHAR(255) NOT NULL,
    estimated_votes INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabel Kompetitor Internal (Teman Separtai tapi beda Caleg)
CREATE TABLE IF NOT EXISTS competitor_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    dapil_id UUID NOT NULL REFERENCES dapils(id) ON DELETE CASCADE,
    candidate_name VARCHAR(255) NOT NULL,
    estimated_votes INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
