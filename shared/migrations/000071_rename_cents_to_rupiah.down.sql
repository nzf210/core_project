-- Campaign DB changes
ALTER TABLE campaign_billing_orders RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE campaign_billing_orders RENAME COLUMN referral_discount_rupiah TO referral_discount_cents;
ALTER TABLE campaign_billing_orders RENAME COLUMN final_amount_rupiah TO final_amount_cents;

-- Billing DB changes
ALTER TABLE voucher_programs RENAME COLUMN price_rupiah TO price_cents;
ALTER TABLE plan_prices RENAME COLUMN price_rupiah TO price_cents;
ALTER TABLE subscriptions RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE invoices RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE invoices RENAME COLUMN referral_discount_rupiah TO referral_discount_cents;
ALTER TABLE invoices RENAME COLUMN voucher_discount_rupiah TO voucher_discount_cents;
ALTER TABLE invoices RENAME COLUMN total_amount_rupiah TO total_amount_cents;

-- Addon DB changes
ALTER TABLE addon_purchases RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE addon_purchases RENAME COLUMN price_rupiah TO price_cents;

-- Wallet DB changes
ALTER TABLE wallet_credits RENAME COLUMN balance_rupiah TO balance_cents;
ALTER TABLE wallet_transactions RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE wallet_transactions RENAME COLUMN final_balance_rupiah TO final_balance_cents;

-- Affiliate DB changes
ALTER TABLE affiliates RENAME COLUMN total_earnings_rupiah TO total_earnings_cents;
ALTER TABLE affiliates RENAME COLUMN cash_balance_rupiah TO cash_balance_cents;
ALTER TABLE affiliate_earnings RENAME COLUMN amount_rupiah TO amount_cents;
ALTER TABLE affiliate_payouts RENAME COLUMN amount_rupiah TO amount_cents;
