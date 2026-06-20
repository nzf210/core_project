-- F042: Campaign RAG Documents — FAQ/guide storage for WA Bot vision-misi retrieval
-- Reuses vector_embeddings table from migration 029 (source_type = 'campaign_document').

CREATE TABLE IF NOT EXISTS campaign_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    source VARCHAR(100) DEFAULT 'manual',  -- 'manual' | 'uploaded' | 'auto_extracted'
    indexed BOOLEAN DEFAULT FALSE,         -- true setelah content sudah di-embed ke vector_embeddings
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_documents_tenant ON campaign_documents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_campaign_documents_campaign ON campaign_documents(campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_documents_indexed ON campaign_documents(indexed) WHERE indexed = FALSE;