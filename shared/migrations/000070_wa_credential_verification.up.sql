-- Migration: WA Cloud API Credential Verification Tracking
-- Feature: F048 AC-6 — Hybrid WA Setup Wizard with Meta Business verification
-- Tambahkan kolom verifikasi status ke wa_cloud_api_credentials

BEGIN;

-- Enum type untuk verification_status (buat dulu sebelum digunakan)
DO $$ BEGIN
  CREATE TYPE verification_status_enum AS ENUM ('unverified', 'verified', 'rejected', 'expired', 'error');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Tambahkan kolom verification_status dan verified_at
ALTER TABLE wa_cloud_api_credentials
  ADD COLUMN IF NOT EXISTS verification_status verification_status_enum NOT NULL DEFAULT 'unverified',
  ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS check_error TEXT;

-- Index untuk filtering berdasarkan status verifikasi
CREATE INDEX IF NOT EXISTS idx_wa_credentials_verification_status
  ON wa_cloud_api_credentials(verification_status);

-- Convert existing boolean is_active to use verification_status
-- Jika is_active=true dan verification_status masih 'unverified', set ke 'verified' (backward compat)
UPDATE wa_cloud_api_credentials
SET verification_status = 'verified',
    verified_at = updated_at
WHERE is_active = true AND verification_status = 'unverified';

COMMIT;