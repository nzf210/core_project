-- 000069_referral_system.down.sql
DROP TABLE IF EXISTS invoice_referrals;
DROP TABLE IF EXISTS affiliate_referrals;
ALTER TABLE referral_config DROP COLUMN IF EXISTS min_purchase_rupiah;
ALTER TABLE referral_config DROP COLUMN IF EXISTS max_commission_rupiah;
ALTER TABLE referral_config DROP COLUMN IF EXISTS is_active;
ALTER TABLE referral_config DROP COLUMN IF EXISTS referral_link_base;
ALTER TABLE affiliate_earnings DROP COLUMN IF EXISTS description;
