-- Campaign Schema

CREATE TABLE candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    verification_document VARCHAR(255),
    is_verified BOOLEAN DEFAULT FALSE,
    suspended BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_candidates_tenant_id ON candidates(tenant_id);

CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, active, completed
    target_voters INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_campaigns_tenant_id ON campaigns(tenant_id);

CREATE TABLE provinces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL
    , total_voters INT NOT NULL DEFAULT 0
);

CREATE TABLE regencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    province_id UUID NOT NULL REFERENCES provinces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL
    , total_voters INT NOT NULL DEFAULT 0
);

CREATE TABLE districts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    regency_id UUID NOT NULL REFERENCES regencies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL
    , total_voters INT NOT NULL DEFAULT 0
);

CREATE TABLE villages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    district_id UUID NOT NULL REFERENCES districts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL
    , total_voters INT NOT NULL DEFAULT 0
);

CREATE TABLE tps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    village_id UUID NOT NULL REFERENCES villages(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    target_voters INT NOT NULL DEFAULT 0,
    total_voters INT NOT NULL DEFAULT 0
);

CREATE TABLE volunteers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    rank INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_volunteers_tenant_id ON volunteers(tenant_id);

CREATE TABLE volunteer_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    volunteer_id UUID NOT NULL REFERENCES volunteers(id) ON DELETE CASCADE,
    village_id UUID REFERENCES villages(id) ON DELETE SET NULL,
    tps_id UUID REFERENCES tps(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE voters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    village_id UUID REFERENCES villages(id) ON DELETE SET NULL,
    tps_id UUID REFERENCES tps(id) ON DELETE SET NULL,
    nik_encrypted VARCHAR(255) NOT NULL,
    name_encrypted VARCHAR(255) NOT NULL,
    address_encrypted VARCHAR(255),
    phone VARCHAR(50),
    status VARCHAR(50) DEFAULT 'uncontacted', -- uncontacted, strong_supporter, weak_supporter, opponent
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_voters_tenant_id ON voters(tenant_id);

CREATE TABLE voter_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voter_id UUID NOT NULL REFERENCES voters(id) ON DELETE CASCADE,
    volunteer_id UUID REFERENCES volunteers(id) ON DELETE SET NULL,
    interaction_type VARCHAR(50) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    event_date TIMESTAMPTZ NOT NULL,
    location VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_events_tenant_id ON events(tenant_id);

CREATE TABLE event_attendees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    voter_id UUID REFERENCES voters(id) ON DELETE SET NULL,
    scanned_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE surveys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE survey_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    question_type VARCHAR(50) NOT NULL, -- choice, text
    options JSONB
);

CREATE TABLE survey_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    voter_id UUID REFERENCES voters(id) ON DELETE SET NULL,
    volunteer_id UUID REFERENCES volunteers(id) ON DELETE SET NULL,
    answers JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
