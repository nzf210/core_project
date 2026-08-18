package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func checkExistingQueue(ctx context.Context, tenantID, phone string) (string, error) {
	var existingQueue string
	err := DB.QueryRow(ctx, `
		SELECT queue_number FROM appointments
		WHERE tenant_id = $1 AND patient_phone = $2 AND status = 'waiting'
	`, tenantID, phone).Scan(&existingQueue)
	if err == nil && existingQueue != "" {
		return existingQueue, fmt.Errorf("existing queue found")
	}
	return "", nil
}

func getClinicSettings(ctx context.Context, tenantID string) (queueType string, slotDuration int, err error) {
	err = DB.QueryRow(ctx, `
		SELECT queue_type, slot_duration_minutes FROM clinic_settings WHERE tenant_id = $1 AND is_active = true
	`, tenantID).Scan(&queueType, &slotDuration)

	if err == sql.ErrNoRows {
		return "sequential", 30, nil
	}
	return queueType, slotDuration, err
}

func generateSequentialQueueNumber(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	var nextNum int
	err := tx.QueryRow(ctx, `
		INSERT INTO clinic_settings (tenant_id, current_queue_number, queue_date)
		VALUES ($1, 1, CURRENT_DATE)
		ON CONFLICT (tenant_id) DO UPDATE SET
			current_queue_number = CASE
				WHEN clinic_settings.queue_date = CURRENT_DATE
				THEN clinic_settings.current_queue_number + 1
				ELSE 1
			END,
			queue_date = CURRENT_DATE
		RETURNING current_queue_number
	`, tenantID).Scan(&nextNum)

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("A-%03d", nextNum), nil
}

func insertAppointment(ctx context.Context, tx pgx.Tx, tenantID, queueNum, patientName, patientPhone string, scheduledTime *time.Time) (string, error) {
	appointmentID := uuid.NewString()
	_, err := tx.Exec(ctx, `
		INSERT INTO appointments (id, tenant_id, queue_number, patient_name, patient_phone, scheduled_time, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'waiting')
	`, appointmentID, tenantID, queueNum, patientName, patientPhone, scheduledTime)
	return appointmentID, err
}

func logQueueHistory(ctx context.Context, tx pgx.Tx, tenantID, appointmentID, queueNum string) {
	tx.Exec(ctx, `
		INSERT INTO queue_history (tenant_id, appointment_id, action, performed_by, notes)
		VALUES ($1, $2, 'booked', 'patient', $3)
	`, tenantID, appointmentID, fmt.Sprintf("Patient booked queue %s", queueNum))
}
