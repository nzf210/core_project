-- Rollback Performance Indexes Migration

DROP INDEX IF EXISTS idx_voters_pic_id;
DROP INDEX IF EXISTS idx_voters_tenant_campaign;
DROP INDEX IF EXISTS idx_volunteers_tenant_campaign;
DROP INDEX IF EXISTS idx_endorsements_tenant_campaign;
DROP INDEX IF EXISTS idx_dpt_records_nik;
DROP INDEX IF EXISTS idx_citizens_nik;
DROP INDEX IF EXISTS idx_journal_lines_entry_tenant;
DROP INDEX IF EXISTS idx_products_tenant_stock;
DROP INDEX IF EXISTS idx_automations_tenant_enabled;
DROP INDEX IF EXISTS idx_tenant_faqs_tenant_id;
DROP INDEX IF EXISTS idx_conversation_logs_tenant_session;
DROP INDEX IF EXISTS idx_voucher_codes_redeemed;
DROP INDEX IF EXISTS idx_subscriptions_tenant_status;
DROP INDEX IF EXISTS idx_users_tenant_role;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
