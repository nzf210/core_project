-- 000025_add_saas_plans.down.sql
-- Rollback: remove saas_plans, plan_features, subscription_tickets, voucher_programs, voucher_codes tables

DROP TABLE IF EXISTS subscription_tickets;
DROP TABLE IF EXISTS voucher_codes;
DROP TABLE IF EXISTS voucher_programs;
DROP TABLE IF EXISTS plan_features;
DROP TABLE IF EXISTS saas_plans;

DROP SEQUENCE IF EXISTS seq_saas_plan_id;