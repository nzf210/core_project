-- Patient Medical Records + Doctor Schedules for F047

CREATE TABLE IF NOT EXISTS patient_medical_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_name VARCHAR(255) NOT NULL,
    patient_phone VARCHAR(50),
    complaint TEXT,
    diagnosis TEXT,
    prescription TEXT,
    notes TEXT,
    follow_up_date DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_medical_records_tenant ON patient_medical_records(tenant_id);
CREATE INDEX idx_medical_records_phone ON patient_medical_records(patient_phone);

CREATE TABLE IF NOT EXISTS clinic_doctor_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    doctor_name VARCHAR(255) NOT NULL,
    specialization VARCHAR(100),
    day_of_week VARCHAR(10) NOT NULL, -- monday, tuesday, ...
    time_start TIME NOT NULL,
    time_end TIME NOT NULL,
    max_patients INT DEFAULT 20,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_doctor_schedules_tenant ON clinic_doctor_schedules(tenant_id);
CREATE INDEX idx_doctor_schedules_day ON clinic_doctor_schedules(day_of_week);