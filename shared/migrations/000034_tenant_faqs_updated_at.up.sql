-- Add updated_at column to tenant_faqs for PUT /faqs handler
-- The PUT handler in apps/umkm/accounting/main.go references updated_at = NOW()
-- but the original migration 000014 only created created_at.
ALTER TABLE tenant_faqs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
