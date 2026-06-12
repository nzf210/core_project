-- Rollback: Telegram Auth
DROP INDEX IF EXISTS idx_users_telegram_chat_id;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_chat_id;
