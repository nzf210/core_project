-- Migration: async_jobs table for RabbitMQ job tracking
-- Purpose: Track status dan result dari async jobs yang diproses via RabbitMQ
-- Related: docs/RABBITMQ_GUIDE.md

CREATE TABLE IF NOT EXISTS async_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    data JSONB,
    result JSONB,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- Indexes untuk query performa
CREATE INDEX idx_async_jobs_tenant_id ON async_jobs(tenant_id);
CREATE INDEX idx_async_jobs_status ON async_jobs(status);
CREATE INDEX idx_async_jobs_type ON async_jobs(type);
CREATE INDEX idx_async_jobs_created_at ON async_jobs(created_at DESC);
CREATE INDEX idx_async_jobs_job_id ON async_jobs(job_id);

-- Composite index untuk query paging per tenant
CREATE INDEX idx_async_jobs_tenant_status_created ON async_jobs(tenant_id, status, created_at DESC);

COMMENT ON TABLE async_jobs IS 'Job tracking untuk async operations via RabbitMQ';
COMMENT ON COLUMN async_jobs.job_id IS 'UUID job yang di-generate saat enqueue';
COMMENT ON COLUMN async_jobs.type IS 'Job type: accounting.transactions, voucher.distribution, etc';
COMMENT ON COLUMN async_jobs.status IS 'Job status: pending → processing → completed/failed';
COMMENT ON COLUMN async_jobs.data IS 'Input data untuk job processing';
COMMENT ON COLUMN async_jobs.result IS 'Output result setelah job selesai';
COMMENT ON COLUMN async_jobs.error IS 'Error message jika job failed';
