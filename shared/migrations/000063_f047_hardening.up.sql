-- F047: Add RLS, Constraints, and Audit Log for Clinic Module

-- 1. Backfill missing business_type values
UPDATE tenants SET business_type = 'umum' WHERE business_type IS NULL;

-- 2. Add CHECK Constraints
-- Values must match business_types.id FK (umum/warung/laundry/industri_kreatif/toko_online/restoran/jasa/hotel)
ALTER TABLE tenants ADD CONSTRAINT check_business_type
    CHECK (business_type IN ('umum', 'warung', 'laundry', 'industri_kreatif', 'toko_online', 'restoran', 'jasa', 'hotel'));

ALTER TABLE clinic_doctor_schedules ADD CONSTRAINT check_day_of_week 
    CHECK (day_of_week IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'));

ALTER TABLE clinic_doctor_schedules ADD CONSTRAINT check_time_range 
    CHECK (time_end > time_start);

ALTER TABLE clinic_doctor_schedules ADD CONSTRAINT check_max_patients 
    CHECK (max_patients > 0);

-- 3. Audit Log for Patient Records
CREATE TABLE IF NOT EXISTS patient_records_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id UUID NOT NULL REFERENCES patient_medical_records(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL, -- User ID
    action VARCHAR(20) NOT NULL, -- VIEW, UPDATE, DELETE
    performed_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_patient_records_audit_record ON patient_records_audit(record_id);

-- 4. Enable RLS
ALTER TABLE patient_medical_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE clinic_doctor_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE patient_records_audit ENABLE ROW LEVEL SECURITY;

-- 5. RLS Policies
CREATE POLICY tenant_isolation_records ON patient_medical_records
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

CREATE POLICY tenant_isolation_schedules ON clinic_doctor_schedules
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

CREATE POLICY tenant_isolation_records_audit ON patient_records_audit
    USING (record_id IN (SELECT id FROM patient_medical_records WHERE tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID));

-- 6. Cleanup Dead Schema + Add updated_at
ALTER TABLE tenants DROP COLUMN IF EXISTS clinic_doctors;

ALTER TABLE clinic_doctor_schedules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

ALTER TABLE patient_records_audit ADD CONSTRAINT check_action_valid
    CHECK (action IN ('VIEW', 'UPDATE', 'DELETE'));
