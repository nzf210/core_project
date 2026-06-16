-- Add premium_coordination_view feature for F046
ALTER TABLE plan_features ADD COLUMN IF NOT EXISTS premium_coordination_view BOOLEAN DEFAULT FALSE;

-- Enable for 'ultimate' tier only (premium feature)
UPDATE plan_features SET premium_coordination_view = TRUE WHERE plan_id = 'ultimate';
UPDATE plan_features SET premium_coordination_view = FALSE WHERE plan_id IN ('lite', 'pro');