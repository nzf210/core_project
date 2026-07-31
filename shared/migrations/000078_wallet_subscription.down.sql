-- Rollback: F058 - Wallet subscription auto-renew
ALTER TABLE wallet_credits DROP COLUMN IF EXISTS auto_renew_subscription;
