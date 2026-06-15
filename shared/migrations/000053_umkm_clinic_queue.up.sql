-- F045: UMKM Healthcare Clinic Queue System

-- 1. Clinic Queue Settings (per tenant)
CREATE TABLE IF NOT EXISTS clinic_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    queue_type VARCHAR(20) NOT NULL DEFAULT 'sequential',  -- 'sequential' atau 'timeslot'
    slot_duration_minutes INT NOT NULL DEFAULT 30,
    is_active BOOLEAN NOT NULL DEFAULT true,
    current_queue_number INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Appointments / Queue
CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    queue_number VARCHAR(20) NOT NULL,              -- 'A-001' untuk sequential, null untuk timeslot
    patient_name VARCHAR(255) NOT NULL,
    patient_phone VARCHAR(50) NOT NULL,
    scheduled_time TIMESTAMP WITH TIME ZONE,         -- null untuk sequential, di-set pas dipanggil
    status VARCHAR(20) NOT NULL DEFAULT 'waiting',   -- 'waiting', 'in_progress', 'completed', 'cancelled'
    cancelled_by VARCHAR(20),                        -- 'patient', 'clinic'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_appointments_tenant_status ON appointments(tenant_id, status);
CREATE INDEX idx_appointments_phone ON appointments(patient_phone);

-- 3. Queue History (audit log)
CREATE TABLE IF NOT EXISTS queue_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    appointment_id UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,                     -- 'booked', 'called', 'completed', 'cancelled'
    performed_by VARCHAR(20) NOT NULL,               -- 'patient', 'clinic', 'system'
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
