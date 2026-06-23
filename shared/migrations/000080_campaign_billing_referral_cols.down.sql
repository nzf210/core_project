-- F054: Remove referral columns from campaign_billing_orders (rollback)
ALTER TABLE campaign_billing_orders
    DROP COLUMN IF EXISTS referral_discount_rupiah,
    DROP COLUMN IF EXISTS final_amount_rupiah;