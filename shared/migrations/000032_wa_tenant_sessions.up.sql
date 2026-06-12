-- Migration: 000032_wa_tenant_sessions
-- Purpose: Proper tracked migration for wa_tenant_sessions table.
-- Previously created in-code by wa-gateway/main.go without migration tracking.
-- Schema matches the in-code creation: TEXT tenant_id (not UUID),
-- because wa-gateway stores tenant_id as plain string throughout.

CREATE TABLE IF NOT EXISTS wa_tenant_sessions (
    tenant_id   TEXT PRIMARY KEY,
    jid         VARCHAR NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Index on jid for fast lookups during session restore
CREATE INDEX IF NOT EXISTS idx_wa_tenant_sessions_jid ON wa_tenant_sessions(jid);

COMMENT ON TABLE wa_tenant_sessions IS 'Maps tenant_id to WhatsApp JID for wa-gateway (whatsmeow). Created in-code previously.';
COMMENT ON COLUMN wa_tenant_sessions.jid IS 'WhatsApp JID, e.g. 6281234567890@s.whatsapp.net';
