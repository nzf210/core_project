-- Migration: WA Cloud API Credential Verification Tracking (rollback)

BEGIN;

ALTER TABLE wa_cloud_api_credentials
  DROP COLUMN IF EXISTS verification_status,
  DROP COLUMN IF EXISTS verified_at,
  DROP COLUMN IF EXISTS last_checked_at,
  DROP COLUMN IF EXISTS check_error;

DROP TYPE IF EXISTS verification_status_enum;
DROP INDEX IF EXISTS idx_wa_credentials_verification_status;

COMMIT;