-- 000027_voucher_links_and_freeze.up.sql
-- Voucher-driven subscription: link-based + freeze worker + status enum
--
-- Tujuan: voucher jadi primary activation. Xendit fallback (B2B corporate only).
-- Akun freeze otomatis saat current_period_end lewat (0 hari grace, read-only).

-- ─────────────────────────────────────────────
-- 1. Tabel voucher_links (link-based redemption)
-- 1 link = 1 klaim, signed token (JWT) di URL
-- ─────────────────────────────────────────────
CREATE TABLE voucher_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id      UUID NOT NULL REFERENCES voucher_programs(id) ON DELETE CASCADE,
    token_hash      VARCHAR(128) NOT NULL UNIQUE,  -- SHA-256 dari signed token
    token_prefix    VARCHAR(20) NOT NULL,           -- 8 char pertama untuk display
    created_by      UUID,                            -- superadmin user_id
    redeemed_by     UUID REFERENCES tenants(id) ON DELETE SET NULL,
    redeemed_at     TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,            -- copy dari program.expires_at
    ip_address      INET,
    user_agent      TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voucher_links_program ON voucher_links(program_id);
CREATE INDEX idx_voucher_links_redeemed ON voucher_links(redeemed_by);
CREATE INDEX idx_voucher_links_active ON voucher_links(is_active) WHERE is_active = true;
CREATE INDEX idx_voucher_links_expires ON voucher_links(expires_at);

-- ─────────────────────────────────────────────
-- 2. Subscription status: 'active' | 'frozen' | 'cancelled'
-- Default 'active' (existing rows tetap 'active')
-- ─────────────────────────────────────────────
ALTER TABLE tenant_subscriptions
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS frozen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS frozen_reason TEXT;

-- Backfill: kalau ada row tanpa status, set 'active'
UPDATE tenant_subscriptions SET status = 'active' WHERE status IS NULL OR status = '';

-- Index untuk freeze worker
CREATE INDEX idx_tenant_subs_status_expires
    ON tenant_subscriptions(status, current_period_end)
    WHERE status = 'active';

-- ─────────────────────────────────────────────
-- 3. Tenant: is_frozen flag untuk fast read di middleware
-- (denormalized dari tenant_subscriptions.status, di-update oleh freeze worker)
-- ─────────────────────────────────────────────
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS is_frozen BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS frozen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_plan_expires_at TIMESTAMPTZ;

CREATE INDEX idx_tenants_frozen ON tenants(is_frozen) WHERE is_frozen = true;

-- ─────────────────────────────────────────────
-- 4. Voucher generation audit log
-- (siapa generate berapa link, kapan)
-- ─────────────────────────────────────────────
CREATE TABLE voucher_generation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id      UUID NOT NULL REFERENCES voucher_programs(id) ON DELETE CASCADE,
    generated_by    UUID,                            -- superadmin user_id
    count           INT NOT NULL,
    prefix          VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voucher_gen_logs_program ON voucher_generation_logs(program_id);
CREATE INDEX idx_voucher_gen_logs_created ON voucher_generation_logs(created_at);

-- ─────────────────────────────────────────────
-- 5. Update saas_plans: tambah description untuk fallback
-- ─────────────────────────────────────────────
COMMENT ON TABLE saas_plans IS 'SaaS plans. Primary activation: voucher (link-based). Xendit kept as fallback for B2B corporate billing.';
COMMENT ON TABLE voucher_links IS 'Link-based voucher redemption. 1 link = 1 tenant. Token signed with HMAC, only SHA-256 hash stored.';
