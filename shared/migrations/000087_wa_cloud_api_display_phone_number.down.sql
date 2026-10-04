-- Migration 000087: Drop display_phone_number from wa_cloud_api_credentials

ALTER TABLE wa_cloud_api_credentials DROP COLUMN IF EXISTS display_phone_number;
