-- F047 Hardening: Down Migration
-- Order matters: RLS policies must be dropped before disabling RLS

-- Drop policies first
DROP POLICY IF EXISTS tenant_isolation_records ON patient_medical_records;
DROP POLICY IF EXISTS tenant_isolation_schedules ON clinic_doctor_schedules;
DROP POLICY IF EXISTS tenant_isolation_records_audit ON patient_records_audit;

-- Disable RLS
ALTER TABLE patient_medical_records DISABLE ROW LEVEL SECURITY;
ALTER TABLE clinic_doctor_schedules DISABLE ROW LEVEL SECURITY;
ALTER TABLE patient_records_audit DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_patient_records_audit_record;

-- Drop audit table
DROP TABLE IF EXISTS patient_records_audit;

-- Drop constraints
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS check_business_type;
ALTER TABLE clinic_doctor_schedules DROP CONSTRAINT IF EXISTS check_day_of_week;
ALTER TABLE clinic_doctor_schedules DROP CONSTRAINT IF EXISTS check_time_range;
ALTER TABLE clinic_doctor_schedules DROP CONSTRAINT IF EXISTS check_max_patients;

-- Drop updated_at column
ALTER TABLE clinic_doctor_schedules DROP COLUMN IF EXISTS updated_at;

-- Restore dead schema (re-add clinic_doctors column)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS clinic_doctors TEXT[] DEFAULT '{}';