-- Add business_type to tenants table (F047)
-- Restricts module access based on business type
-- NOTE: tenants.business_type already exists since 000011 (FK to business_types.id).
-- 000061 only adds columns for clinic-specific data + ensures 'clinic' is a valid business_type.

-- Ensure 'clinic' is registered as a business_type (idempotent)
INSERT INTO business_types (id, name, description, icon, default_modules, default_dashboard_widgets)
VALUES (
    'clinic',
    'Klinik / Praktik Dokter',
    'Klinik kesehatan, praktik dokter, bidan, dan layanan medis',
    'stethoscope',
    '["transactions", "customers", "reports", "patient_records", "doctor_schedules", "appointment_queue", "wa_notifications"]',
    '["todays_appointments", "patient_summary", "doctor_utilization", "queue_status", "revenue_summary"]'
)
ON CONFLICT (id) DO NOTHING;

-- Add clinic-specific columns
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS clinic_doctors TEXT[] DEFAULT '{}';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS clinic_services JSONB DEFAULT '[]'::JSONB;

-- Index already exists from 000011 — recreate idempotently to be safe
CREATE INDEX IF NOT EXISTS idx_tenants_business_type ON tenants(business_type);
