-- Migration: 000032_wa_tenant_sessions (down)
-- Reverses: creates proper tracked table for wa_tenant_sessions
DROP TABLE IF EXISTS wa_tenant_sessions;
