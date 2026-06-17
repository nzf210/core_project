package main

import (
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
	"github.com/google/uuid"
)

// handleClinicMedicalRecords - F047 Patient Medical Records
func handleClinicMedicalRecords(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, `
			SELECT id, patient_name, patient_phone, complaint, diagnosis, prescription, notes, follow_up_date, created_at, updated_at
			FROM patient_medical_records
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT 100`, tenantID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "DB error", nil)
			return
		}
		defer rows.Close()

		var records []MedicalRecord
		for rows.Next() {
			var rec MedicalRecord
			_ = rows.Scan(&rec.ID, &rec.PatientName, &rec.PatientPhone, &rec.Complaint,
				&rec.Diagnosis, &rec.Prescription, &rec.Notes, &rec.FollowUpDate, &rec.CreatedAt, &rec.UpdatedAt)
			records = append(records, rec)
		}
		response.JSON(w, http.StatusOK, "Success fetch records", records)
		return
	}

	if r.Method == http.MethodPost {
		var rec MedicalRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
			return
		}
		rec.ID = uuid.NewString()
		now := time.Now()
		rec.CreatedAt = now
		rec.UpdatedAt = now

		_, err := DB.Exec(ctx, `
			INSERT INTO patient_medical_records (id, tenant_id, patient_name, patient_phone, complaint, diagnosis, prescription, notes, follow_up_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			rec.ID, tenantID, rec.PatientName, rec.PatientPhone, rec.Complaint,
			rec.Diagnosis, rec.Prescription, rec.Notes, rec.FollowUpDate, rec.CreatedAt, rec.UpdatedAt)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save record", nil)
			return
		}
		response.JSON(w, http.StatusOK, "Record saved", rec)
		return
	}

	response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
}

// handleClinicDoctors - F047 Doctor Schedules
func handleClinicDoctors(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, `
			SELECT id, doctor_name, specialization, day_of_week, time_start, time_end, max_patients, is_active, created_at
			FROM clinic_doctor_schedules
			WHERE tenant_id = $1 AND is_active = true
			ORDER BY day_of_week, time_start`, tenantID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "DB error", nil)
			return
		}
		defer rows.Close()

		var doctors []DoctorSchedule
		for rows.Next() {
			var d DoctorSchedule
			_ = rows.Scan(&d.ID, &d.DoctorName, &d.Specialization, &d.DayOfWeek,
				&d.TimeStart, &d.TimeEnd, &d.MaxPatients, &d.IsActive, &d.CreatedAt)
			doctors = append(doctors, d)
		}
		response.JSON(w, http.StatusOK, "Success fetch doctors", doctors)
		return
	}

	if r.Method == http.MethodPost {
		var d DoctorSchedule
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
			return
		}
		d.ID = uuid.NewString()
		d.CreatedAt = time.Now()

		_, err := DB.Exec(ctx, `
			INSERT INTO clinic_doctor_schedules (id, tenant_id, doctor_name, specialization, day_of_week, time_start, time_end, max_patients, is_active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			d.ID, tenantID, d.DoctorName, d.Specialization, d.DayOfWeek,
			d.TimeStart, d.TimeEnd, d.MaxPatients, d.IsActive, d.CreatedAt)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save doctor schedule", nil)
			return
		}
		response.JSON(w, http.StatusOK, "Doctor schedule saved", d)
		return
	}

	response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
}

type MedicalRecord struct {
	ID           string     `json:"id"`
	PatientName  string     `json:"patient_name"`
	PatientPhone string     `json:"patient_phone,omitempty"`
	Complaint    string     `json:"complaint,omitempty"`
	Diagnosis    string     `json:"diagnosis,omitempty"`
	Prescription string     `json:"prescription,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	FollowUpDate *time.Time `json:"follow_up_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type DoctorSchedule struct {
	ID            string    `json:"id"`
	DoctorName    string    `json:"doctor_name"`
	Specialization string   `json:"specialization,omitempty"`
	DayOfWeek     string   `json:"day_of_week"` // monday, tuesday, ...
	TimeStart     string    `json:"time_start"`  // HH:MM:SS
	TimeEnd       string    `json:"time_end"`
	MaxPatients   int       `json:"max_patients"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}