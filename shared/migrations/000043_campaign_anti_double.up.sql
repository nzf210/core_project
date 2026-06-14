-- Campaign Data Integrity (Anti-Double) Schema

-- 1. Create citizens table (Master Data / Single Source of Truth for NIK)
-- NIK acts as the natural unique key.
CREATE TABLE IF NOT EXISTS citizens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nik VARCHAR(16) UNIQUE NOT NULL, -- The 16-digit KTP number
    name VARCHAR(255) NOT NULL,
    address TEXT,
    gender VARCHAR(10),
    age INT,
    tps_id UUID REFERENCES tps(id) ON DELETE SET NULL, -- Default TPS (bisa ditarik dari DPT jika cocok)
    is_dpt_verified BOOLEAN DEFAULT FALSE, -- True jika NIK ini ditemukan di dpt_records
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_citizens_nik ON citizens(nik);

-- 2. Create dpt_records table (Official KPU Data)
-- Used for reconciliation against citizens/endorsements
CREATE TABLE IF NOT EXISTS dpt_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nik VARCHAR(16) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    province_id UUID REFERENCES provinces(id) ON DELETE SET NULL,
    regency_id UUID REFERENCES regencies(id) ON DELETE SET NULL,
    district_id UUID REFERENCES districts(id) ON DELETE SET NULL,
    village_id UUID REFERENCES villages(id) ON DELETE SET NULL,
    tps_name VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dpt_records_nik ON dpt_records(nik);

-- 3. Create endorsements table (The Relational / Campaign Claims table)
-- This replaces the "voters" table paradigm, mapping citizens to campaigns/tenants.
CREATE TABLE IF NOT EXISTS endorsements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    citizen_id UUID NOT NULL REFERENCES citizens(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    recruiter_id UUID REFERENCES volunteers(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'valid', -- valid, invalid_nik, conflict_internal, conflict_external
    proof_image_url VARCHAR(255), -- KTP Photo URL (from N8N/WA)
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_endorsements_tenant_id ON endorsements(tenant_id);
CREATE INDEX IF NOT EXISTS idx_endorsements_citizen_id ON endorsements(citizen_id);

-- Optional: If you want to strictly prevent the exact same recruiter from submitting the exact same NIK twice:
-- CREATE UNIQUE INDEX idx_unique_endorsement ON endorsements(citizen_id, tenant_id, recruiter_id);