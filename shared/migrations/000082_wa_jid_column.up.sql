-- Adds wa_jid column to users table for WA registration flow (F048 AC-X)
-- Stores WhatsApp JID (e.g. 62812xxxx@s.whatsapp.net) of user who registered via WA
ALTER TABLE users ADD COLUMN IF NOT EXISTS wa_jid VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_users_wa_jid ON users(wa_jid);