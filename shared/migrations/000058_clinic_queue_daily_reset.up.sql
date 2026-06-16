-- Add queue_date column to track daily reset of queue numbers
ALTER TABLE clinic_settings ADD COLUMN IF NOT EXISTS queue_date DATE DEFAULT CURRENT_DATE;

-- Backfill queue_date for existing rows so the reset logic works correctly
UPDATE clinic_settings SET queue_date = CURRENT_DATE WHERE queue_date IS NULL;

-- Add index for efficient date-based filtering
CREATE INDEX IF NOT EXISTS idx_clinic_settings_queue_date ON clinic_settings(queue_date);
