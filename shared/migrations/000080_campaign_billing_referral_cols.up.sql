-- F054: Add referral_discount_cents and final_amount_cents to campaign_billing_orders
ALTER TABLE campaign_billing_orders
    ADD COLUMN IF NOT EXISTS referral_discount_cents BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS final_amount_cents BIGINT NOT NULL DEFAULT 0;