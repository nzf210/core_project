-- F058: Wallet Payment for Subscription
-- 1. Add auto_renew_subscription flag per tenant
ALTER TABLE wallet_credits
  ADD COLUMN IF NOT EXISTS auto_renew_subscription BOOLEAN DEFAULT false;

-- 2. Ensure wallet_transactions accepts 'subscription' type
-- (PostgreSQL VARCHAR has no enum constraint, no migration needed)
-- The code just inserts 'subscription' as a VARCHAR — safe to add at any time.

-- 3. Index for auto-renew scan (used by cron)
CREATE INDEX IF NOT EXISTS idx_wallet_auto_renew ON wallet_credits(tenant_id)
  WHERE auto_renew_subscription = true;
