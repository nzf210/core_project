ALTER TABLE tenants DROP COLUMN IF EXISTS referred_by_affiliate_id;
DROP TABLE IF EXISTS affiliate_withdrawals CASCADE;
DROP TABLE IF EXISTS affiliate_earnings CASCADE;
DROP TABLE IF EXISTS affiliates CASCADE;
