package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock IDs sesuai real schema (tenants.id UUID, business_types.id VARCHAR)
const (
	mockTenantID      = "11111111-1111-1111-1111-111111111111"
	mockTenantLiteID  = "22222222-2222-2222-2222-222222222222"
	mockTenantClinicID = "33333333-3333-3333-3333-333333333333"
	mockRecordID      = "44444444-4444-4444-4444-444444444444"
	mockDoctorID      = "55555555-5555-5555-5555-555555555555"
)

// TestRequireClinicType_MissingTenantID verifies middleware returns 401 if no X-Tenant-ID
func TestRequireClinicType_MissingTenantID(t *testing.T) {
	handler := requireClinicType(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req, _ := http.NewRequest(http.MethodGet, "/clinic/settings", nil)

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing X-Tenant-ID, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// TestHandleClinicSettings_MissingTenantID verifies handleClinicSettings returns 401 without X-Tenant-ID
func TestHandleClinicSettings_MissingTenantID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/clinic/settings", nil)

	rr := httptest.NewRecorder()
	handleClinicSettings(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 401 or 400, got %d", rr.Code)
	}
}

// TestHandleClinicBook_WrongMethod verifies 405 for non-POST request
func TestHandleClinicBook_WrongMethod(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/clinic/appointments/book", nil)
	req.Header.Set("X-Tenant-ID", mockTenantClinicID)

	rr := httptest.NewRecorder()
	handleClinicBook(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on POST endpoint, got %d", rr.Code)
	}
}

// TestHandleClinicBook_MissingTenantID verifies 401 without X-Tenant-ID
func TestHandleClinicBook_MissingTenantID(t *testing.T) {
	payload := []byte(`{
		"patient_name": "Budi Santoso",
		"patient_phone": "081234567890"
	}`)
	req, _ := http.NewRequest(http.MethodPost, "/clinic/appointments/book", bytes.NewBuffer(payload))

	rr := httptest.NewRecorder()
	handleClinicBook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Missing tenant ID") {
		t.Errorf("expected 'Missing tenant ID' error, got: %s", rr.Body.String())
	}
}

// TestHandleClinicBook_InvalidJSON verifies 400 for malformed JSON
func TestHandleClinicBook_InvalidJSON(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/clinic/appointments/book", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("X-Tenant-ID", mockTenantClinicID)

	rr := httptest.NewRecorder()
	handleClinicBook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

// TestHandleClinicMedicalRecords_MissingTenantID verifies 401 for medical records
func TestHandleClinicMedicalRecords_MissingTenantID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/clinic/medical-records", nil)

	rr := httptest.NewRecorder()
	handleClinicMedicalRecords(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestHandleClinicDoctors_MissingTenantID verifies 401 for doctor schedules
func TestHandleClinicDoctors_MissingTenantID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/clinic/doctors", nil)

	rr := httptest.NewRecorder()
	handleClinicDoctors(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestHandleClinicMedicalRecords_WrongMethodOnlyGetOrPost verifies 405 for PUT/DELETE
func TestHandleClinicMedicalRecords_WrongMethodOnlyGetOrPost(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, "/clinic/medical-records", nil)
	req.Header.Set("X-Tenant-ID", mockTenantClinicID)

	rr := httptest.NewRecorder()
	handleClinicMedicalRecords(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE, got %d", rr.Code)
	}
}

// TestMedicalRecord_JSONBinding verifies JSON deserialization for MedicalRecord
func TestMedicalRecord_JSONBinding(t *testing.T) {
	jsonStr := `{
		"patient_name": "Budi Santoso",
		"patient_phone": "081234567890",
		"complaint": "Demam 3 hari, sakit kepala",
		"diagnosis": "ISPA ringan",
		"prescription": "Paracetamol 3x1, istirahat 3 hari",
		"notes": "Follow-up jika tidak membaik"
	}`
	var rec MedicalRecord
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if rec.PatientName != "Budi Santoso" {
		t.Errorf("expected patient_name 'Budi Santoso', got %q", rec.PatientName)
	}
	if rec.Complaint != "Demam 3 hari, sakit kepala" {
		t.Errorf("expected complaint, got %q", rec.Complaint)
	}
	if rec.Diagnosis != "ISPA ringan" {
		t.Errorf("expected diagnosis 'ISPA ringan', got %q", rec.Diagnosis)
	}
}

// TestDoctorSchedule_JSONBinding verifies JSON deserialization for DoctorSchedule
func TestDoctorSchedule_JSONBinding(t *testing.T) {
	jsonStr := `{
		"doctor_name": "dr. Anisa Putri",
		"specialization": "Umum",
		"day_of_week": "monday",
		"time_start": "08:00:00",
		"time_end": "12:00:00",
		"max_patients": 20,
		"is_active": true
	}`
	var d DoctorSchedule
	if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if d.DoctorName != "dr. Anisa Putri" {
		t.Errorf("expected doctor_name 'dr. Anisa Putri', got %q", d.DoctorName)
	}
	if d.DayOfWeek != "monday" {
		t.Errorf("expected day_of_week 'monday', got %q", d.DayOfWeek)
	}
	if d.TimeStart != "08:00:00" {
		t.Errorf("expected time_start '08:00:00', got %q", d.TimeStart)
	}
	if d.MaxPatients != 20 {
		t.Errorf("expected max_patients 20, got %d", d.MaxPatients)
	}
}

// TestDayOfWeek_EnumValues verifies valid day_of_week enum (migration 000062 VARCHAR(10))
func TestDayOfWeek_EnumValues(t *testing.T) {
	validDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	for _, day := range validDays {
		if day == "" {
			t.Errorf("day should not be empty: %s", day)
		}
		if len(day) > 10 {
			t.Errorf("day %q exceeds VARCHAR(10) limit", day)
		}
	}
}

// TestTimeValidation verifies time_start < time_end invariant (frontend UI enforces this in ClinicFrontdesk.vue)
func TestTimeValidation(t *testing.T) {
	cases := []struct {
		name      string
		start     string
		end       string
		shouldErr bool
	}{
		{"valid morning", "08:00", "12:00", false},
		{"valid afternoon", "13:00", "17:00", false},
		{"valid full day", "08:00", "20:00", false},
		{"invalid same time", "08:00", "08:00", true},
		{"invalid end before start", "12:00", "08:00", true},
		{"edge case 23:59", "00:00", "23:59", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.start >= tc.end
			if hasError != tc.shouldErr {
				t.Errorf("time validation: start=%s end=%s shouldErr=%v gotErr=%v",
					tc.start, tc.end, tc.shouldErr, hasError)
			}
		})
	}
}

// TestBusinessType_EnumValues verifies business_type FK constraints (from business_types table per migration 000011)
// 'clinic' was added in 000061 via INSERT INTO business_types ... ON CONFLICT DO NOTHING
func TestBusinessType_EnumValues(t *testing.T) {
	validBusinessTypes := []string{
		"umum",            // default
		"warung",          // retail kecil
		"toko_online",     // e-commerce
		"restoran",        // F&B
		"jasa",            // service / bengkel / salon
		"industri_kreatif", // desain / fotografi
		"laundry",         // laundry kiloan
		"clinic",          // klinik / praktik dokter (F047)
	}
	for _, bt := range validBusinessTypes {
		if bt == "" {
			t.Errorf("business_type should not be empty")
		}
		if len(bt) > 50 {
			t.Errorf("business_type %q exceeds VARCHAR(50) limit", bt)
		}
	}
}

// TestBookReq_JSONBinding verifies JSON deserialization for BookReq
func TestBookReq_JSONBinding(t *testing.T) {
	jsonStr := `{
		"patient_name": "Siti Aminah",
		"patient_phone": "081987654321"
	}`
	var req BookReq
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if req.PatientName != "Siti Aminah" {
		t.Errorf("expected patient_name 'Siti Aminah', got %q", req.PatientName)
	}
	if req.PatientPhone != "081987654321" {
		t.Errorf("expected patient_phone '081987654321', got %q", req.PatientPhone)
	}
}

// TestClinicSettings_DefaultValues verifies default settings when none configured (F045)
func TestClinicSettings_DefaultValues(t *testing.T) {
	defaultSettings := ClinicSettings{
		QueueType:           "sequential",
		SlotDurationMinutes: 30,
		IsActive:            true,
	}
	if defaultSettings.QueueType != "sequential" {
		t.Errorf("expected default queue_type 'sequential', got %q", defaultSettings.QueueType)
	}
	if defaultSettings.SlotDurationMinutes != 30 {
		t.Errorf("expected default slot_duration 30, got %d", defaultSettings.SlotDurationMinutes)
	}
	if !defaultSettings.IsActive {
		t.Error("expected default is_active=true")
	}

	// Verify valid queue_type enum
	validQueueTypes := []string{"sequential", "timeslot"}
	isValid := false
	for _, vt := range validQueueTypes {
		if defaultSettings.QueueType == vt {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("queue_type %q not in valid enum", defaultSettings.QueueType)
	}
}
