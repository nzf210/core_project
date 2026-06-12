-- Migration: Telegram Auth (telegram_chat_id for users)
-- Feature: Register & login via Telegram Bot OTP

ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_users_telegram_chat_id ON users(telegram_chat_id);
