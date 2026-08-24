-- Remove UNIQUE constraint on wallet_transactions(reference)
DROP INDEX IF EXISTS idx_wallet_transactions_reference_unique;
