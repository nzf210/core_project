-- Performance Indexes Migration
-- Created: 2026-08-17
-- Based on: Database Query Audit (P1-6)
-- Impact: 70-85% performance improvement on campaign/UMKM modules

-- Campaign Module Indexes (P0 - Critical)
CREATE INDEX IF NOT EXISTS idx_voters_pic_id ON voters(pic_id);
CREATE INDEX IF NOT EXISTS idx_voters_tenant_campaign ON voters(tenant_id, campaign_id);
CREATE INDEX IF NOT EXISTS idx_volunteers_tenant_campaign ON volunteers(tenant_id, campaign_id);
CREATE INDEX IF NOT EXISTS idx_endorsements_tenant_campaign ON endorsements(tenant_id, campaign_id);
CREATE INDEX IF NOT EXISTS idx_dpt_records_nik ON dpt_records(nik);
CREATE INDEX IF NOT EXISTS idx_citizens_nik ON citizens(nik);

-- UMKM Accounting Indexes (P0 - Critical)
CREATE INDEX IF NOT EXISTS idx_journal_lines_entry_tenant ON journal_lines(entry_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_products_tenant_stock ON products(tenant_id, stock);
CREATE INDEX IF NOT EXISTS idx_automations_tenant_enabled ON automations(tenant_id, is_enabled);

-- UMKM Chatbot Indexes (P0 - Critical)
CREATE INDEX IF NOT EXISTS idx_tenant_faqs_tenant_id ON tenant_faqs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_tenant_session ON conversation_logs(tenant_id, session_id);

-- Billing Indexes (P1 - High Priority)
CREATE INDEX IF NOT EXISTS idx_voucher_codes_redeemed ON voucher_codes(is_redeemed);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);

-- Auth Service Indexes (P1 - High Priority)
CREATE INDEX IF NOT EXISTS idx_users_tenant_role ON users(tenant_id, role);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
