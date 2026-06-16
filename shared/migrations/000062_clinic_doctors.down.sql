-- Rollback: remove patient medical records + doctor schedules
DROP TABLE IF EXISTS clinic_doctor_schedules CASCADE;
DROP TABLE IF EXISTS patient_medical_records CASCADE;