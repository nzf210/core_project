-- Rollback: Remove queue_date column from clinic_settings
ALTER TABLE clinic_settings DROP COLUMN IF EXISTS queue_date;
DROP INDEX IF EXISTS idx_clinic_settings_queue_date;
