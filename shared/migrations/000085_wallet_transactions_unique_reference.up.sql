-- Add UNIQUE constraint on wallet_transactions(reference) to prevent duplicate topup credits
-- from concurrent Xendit webhook retries. Enables idempotent INSERT via ON CONFLICT DO NOTHING.
CREATE UNIQUE INDEX idx_wallet_transactions_reference_unique ON wallet_transactions(reference);
