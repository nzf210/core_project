-- Campaign DB changes
ALTER TABLE campaign_billing_orders RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE campaign_billing_orders RENAME COLUMN referral_discount_cents TO referral_discount_rupiah;
ALTER TABLE campaign_billing_orders RENAME COLUMN final_amount_cents TO final_amount_rupiah;

-- Billing DB changes
ALTER TABLE voucher_programs RENAME COLUMN price_cents TO price_rupiah;
ALTER TABLE plan_prices RENAME COLUMN price_cents TO price_rupiah;
ALTER TABLE subscriptions RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE invoices RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE invoices RENAME COLUMN referral_discount_cents TO referral_discount_rupiah;
ALTER TABLE invoices RENAME COLUMN voucher_discount_cents TO voucher_discount_rupiah;
ALTER TABLE invoices RENAME COLUMN total_amount_cents TO total_amount_rupiah;

-- Addon DB changes
ALTER TABLE addon_purchases RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE addon_purchases RENAME COLUMN price_cents TO price_rupiah;

-- Wallet DB changes
ALTER TABLE wallet_credits RENAME COLUMN balance_cents TO balance_rupiah;
ALTER TABLE wallet_transactions RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE wallet_transactions RENAME COLUMN final_balance_cents TO final_balance_rupiah;

-- Affiliate DB changes
ALTER TABLE affiliates RENAME COLUMN total_earnings_cents TO total_earnings_rupiah;
ALTER TABLE affiliates RENAME COLUMN cash_balance_cents TO cash_balance_rupiah;
ALTER TABLE affiliate_earnings RENAME COLUMN amount_cents TO amount_rupiah;
ALTER TABLE affiliate_payouts RENAME COLUMN amount_cents TO amount_rupiah;
