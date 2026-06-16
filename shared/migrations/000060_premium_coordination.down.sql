-- Rollback: remove premium_coordination_view column
ALTER TABLE plan_features DROP COLUMN IF EXISTS premium_coordination_view;