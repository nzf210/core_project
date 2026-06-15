package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"core_project/shared/sdk/response"
)

func getTenantID(r *http.Request) string {
	return r.Header.Get("X-Tenant-ID")
}

type ClinicSettings struct {
	QueueType           string `json:"queue_type"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
	IsActive            bool   `json:"is_active"`
}

type BookReq struct {
	PatientName  string     `json:"patient_name"`
	PatientPhone string     `json:"patient_phone"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
}

type CancelReq struct {
	AppointmentID string `json:"appointment_id"`
	PerformedBy   string `json:"performed_by"`
}

func handleClinicSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()

	if r.Method == http.MethodGet {
		var settings ClinicSettings
		err := DB.QueryRow(ctx, `
			SELECT queue_type, slot_duration_minutes, is_active 
			FROM clinic_settings WHERE tenant_id = $1
		`, tenantID).Scan(&settings.QueueType, &settings.SlotDurationMinutes, &settings.IsActive)

		if err == sql.ErrNoRows {
			settings = ClinicSettings{QueueType: "sequential", SlotDurationMinutes: 30, IsActive: true}
			_, _ = DB.Exec(ctx, `
				INSERT INTO clinic_settings (tenant_id, queue_type, slot_duration_minutes, is_active)
				VALUES ($1, $2, $3, $4)
			`, tenantID, settings.QueueType, settings.SlotDurationMinutes, settings.IsActive)
		} else if err != nil {
			response.Error(w, http.StatusInternalServerError, "DB error", nil)
			return
		}

		response.JSON(w, http.StatusOK, "Success fetch settings", settings)
		return
	}

	if r.Method == http.MethodPut {
		var req ClinicSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
			return
		}

		if req.QueueType != "sequential" && req.QueueType != "timeslot" {
			response.Error(w, http.StatusBadRequest, "Invalid queue type", nil)
			return
		}

		_, err := DB.Exec(ctx, `
			INSERT INTO clinic_settings (tenant_id, queue_type, slot_duration_minutes, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id) DO UPDATE SET
				queue_type = EXCLUDED.queue_type,
				slot_duration_minutes = EXCLUDED.slot_duration_minutes,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, tenantID, req.QueueType, req.SlotDurationMinutes, req.IsActive)

		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save settings", nil)
			return
		}

		response.JSON(w, http.StatusOK, "Settings updated", req)
		return
	}

	response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
}

func handleClinicBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	var req BookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}

	ctx := r.Context()

	var queueType string
	var slotDuration int
	err := DB.QueryRow(ctx, `
		SELECT queue_type, slot_duration_minutes FROM clinic_settings WHERE tenant_id = $1 AND is_active = true
	`, tenantID).Scan(&queueType, &slotDuration)

	if err == sql.ErrNoRows {
		queueType = "sequential"
		slotDuration = 30
	} else if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", nil)
		return
	}

	var queueNum string
	var scheduledTime *time.Time

	tx, err := DB.Begin(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tx start failed", nil)
		return
	}
	defer tx.Rollback(ctx)

	if queueType == "sequential" {
		var nextNum int
		err = tx.QueryRow(ctx, `
			INSERT INTO clinic_settings (tenant_id, current_queue_number)
			VALUES ($1, 1)
			ON CONFLICT (tenant_id) DO UPDATE SET
				current_queue_number = clinic_settings.current_queue_number + 1
			RETURNING current_queue_number
		`, tenantID).Scan(&nextNum)

		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed generating queue number", nil)
			return
		}
		queueNum = fmt.Sprintf("A-%03d", nextNum)
	} else {
		if req.ScheduledAt == nil {
			response.Error(w, http.StatusBadRequest, "scheduled_at is required for timeslot mode", nil)
			return
		}
		scheduledTime = req.ScheduledAt
		queueNum = req.ScheduledAt.Format("15:04")
	}

	appointmentID := uuid.NewString()

	_, err = tx.Exec(ctx, `
		INSERT INTO appointments (id, tenant_id, queue_number, patient_name, patient_phone, scheduled_time, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'waiting')
	`, appointmentID, tenantID, queueNum, req.PatientName, req.PatientPhone, scheduledTime)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to book appointment: "+err.Error(), nil)
		return
	}

	_, _ = tx.Exec(ctx, `
		INSERT INTO queue_history (tenant_id, appointment_id, action, performed_by, notes)
		VALUES ($1, $2, 'booked', 'patient', $3)
	`, tenantID, appointmentID, fmt.Sprintf("Patient booked queue %s", queueNum))

	tx.Commit(ctx)

	response.JSON(w, http.StatusOK, "Booking success", map[string]interface{}{
		"appointment_id":   appointmentID,
		"queue_number":     queueNum,
		"scheduled_time":   scheduledTime,
		"estimated_minute": slotDuration,
	})
}

func handleClinicCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	var req CancelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}

	ctx := r.Context()

	res, err := DB.Exec(ctx, `
		UPDATE appointments 
		SET status = 'cancelled', cancelled_by = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND status = 'waiting'
	`, req.PerformedBy, req.AppointmentID, tenantID)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", nil)
		return
	}

	rows := res.RowsAffected()
	if rows == 0 {
		response.Error(w, http.StatusNotFound, "Appointment not found or not in waiting status", nil)
		return
	}

	_, _ = DB.Exec(ctx, `
		INSERT INTO queue_history (tenant_id, appointment_id, action, performed_by, notes)
		VALUES ($1, $2, 'cancelled', $3, 'Cancelled by request')
	`, tenantID, req.AppointmentID, req.PerformedBy)

	response.JSON(w, http.StatusOK, "Appointment cancelled", nil)
}

func handleClinicQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, queue_number, patient_name, patient_phone, scheduled_time, status 
		FROM appointments 
		WHERE tenant_id = $1 AND (status = 'waiting' OR status = 'in_progress')
		ORDER BY created_at ASC
	`, tenantID)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", nil)
		return
	}
	defer rows.Close()

	type QueueItem struct {
		ID            string     `json:"id"`
		QueueNumber   string     `json:"queue_number"`
		PatientName   string     `json:"patient_name"`
		PatientPhone  string     `json:"patient_phone"`
		ScheduledTime *time.Time `json:"scheduled_time"`
		Status        string     `json:"status"`
	}

	var list []QueueItem
	for rows.Next() {
		var item QueueItem
		_ = rows.Scan(&item.ID, &item.QueueNumber, &item.PatientName, &item.PatientPhone, &item.ScheduledTime, &item.Status)
		list = append(list, item)
	}

	response.JSON(w, http.StatusOK, "Success fetch queue list", list)
}

func handleClinicCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	type CallReq struct {
		AppointmentID string `json:"appointment_id"`
	}
	var req CallReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tx start failed", nil)
		return
	}
	defer tx.Rollback(ctx)

	_, _ = tx.Exec(ctx, `
		UPDATE appointments SET status = 'completed', updated_at = NOW()
		WHERE tenant_id = $1 AND status = 'in_progress'
	`, tenantID)

	res, err := tx.Exec(ctx, `
		UPDATE appointments SET status = 'in_progress', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status = 'waiting'
	`, req.AppointmentID, tenantID)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", nil)
		return
	}

	rows := res.RowsAffected()
	if rows == 0 {
		response.Error(w, http.StatusNotFound, "Appointment not found or not in waiting status", nil)
		return
	}

	_, _ = tx.Exec(ctx, `
		INSERT INTO queue_history (tenant_id, appointment_id, action, performed_by, notes)
		VALUES ($1, $2, 'called', 'clinic', 'Called to counter')
	`, tenantID, req.AppointmentID)

	tx.Commit(ctx)

	response.JSON(w, http.StatusOK, "Called successfully", nil)
}
