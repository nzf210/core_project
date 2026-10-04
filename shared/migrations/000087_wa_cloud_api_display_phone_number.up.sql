-- Migration 000087: Add display_phone_number to wa_cloud_api_credentials
-- Adds human-readable WhatsApp phone number returned by Meta Graph API

ALTER TABLE wa_cloud_api_credentials ADD COLUMN IF NOT EXISTS display_phone_number TEXT;
