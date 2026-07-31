-- Migration: WhatsApp Cloud API Credentials
-- Description: Per-tenant Meta WhatsApp Cloud API credentials for hybrid WA architecture
-- Feature: Hybrid WhatsApp (Cloud API for transactional + whatsmeow for chatbot)
-- Includes: F048 — Verification tracking columns

-- Enum type untuk verification_status
CREATE TYPE IF NOT EXISTS verification_status_enum AS ENUM ('unverified', 'verified', 'rejected', 'expired', 'error');

CREATE TABLE IF NOT EXISTS wa_cloud_api_credentials (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    phone_number_id     TEXT NOT NULL,          -- Meta phone number ID (e.g., "123456789")
    waba_id             TEXT,                   -- WhatsApp Business Account ID
    access_token        TEXT NOT NULL,          -- Meta access token (encrypted)
    verify_token        TEXT,                   -- Webhook verify token (hub.verify_token)
    is_active           BOOLEAN DEFAULT true,
    verification_status verification_status_enum NOT NULL DEFAULT 'unverified',
    verified_at         TIMESTAMPTZ,
    last_checked_at     TIMESTAMPTZ,
    check_error         TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_cloud_credentials_tenant ON wa_cloud_api_credentials(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_cloud_credentials_phone ON wa_cloud_api_credentials(phone_number_id);
CREATE INDEX IF NOT EXISTS idx_wa_credentials_verification_status ON wa_cloud_api_credentials(verification_status);
