-- F054: Add referral_discount_rupiah and final_amount_rupiah to campaign_billing_orders
ALTER TABLE campaign_billing_orders
    ADD COLUMN IF NOT EXISTS referral_discount_rupiah BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS final_amount_rupiah BIGINT NOT NULL DEFAULT 0;