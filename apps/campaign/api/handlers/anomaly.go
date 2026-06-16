package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

// HandleAnomalyDetection run jobs to flag anomalies
func HandleAnomalyDetection(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	ctx := context.Background()

	// Logic 1: Umur > 100 Tahun siluman. NIK digits 7-12 is date of birth (DDMMYY).
	// Position 7-8 = day (01-31), 9-10 = month (01-12), 11-12 = year (00-99).
	// If month is 20-25 → indicates "siluman" (ktp element造假).
	ageAnomalyQuery := `
		UPDATE endorsements e
		SET is_anomaly = TRUE, anomaly_reason = 'Usia Terindikasi > 100 Tahun (Siluman)'
		FROM citizens c
		WHERE e.citizen_id = c.id AND e.tenant_id = $1 AND e.is_anomaly = FALSE
		AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) >= 20
		AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) <= 25
	`
	_, err := repository.DB.Exec(ctx, ageAnomalyQuery, tenantID)
	if err != nil && err != sql.ErrNoRows {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to flag age anomalies: " + err.Error(),
		})
		return
	}

	// Logic 2: Burst detect. Relawan input > 500 KTP dalam 1 jam = joki / bot.
	burstAnomalyQuery := `
		WITH burst_recruiters AS (
			SELECT recruiter_id
			FROM endorsements
			WHERE tenant_id = $1
			GROUP BY recruiter_id, date_trunc('hour', created_at)
			HAVING count(id) > 500
		)
		UPDATE endorsements
		SET is_anomaly = TRUE, anomaly_reason = 'Terdeteksi Joki/Bot: Burst Insert > 500/jam'
		WHERE tenant_id = $1 AND recruiter_id IN (SELECT recruiter_id FROM burst_recruiters)
		AND is_anomaly = FALSE
	`
	_, err = repository.DB.Exec(ctx, burstAnomalyQuery, tenantID)
	if err != nil && err != sql.ErrNoRows {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to flag burst anomalies: " + err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Anomaly detection job finished successfully",
	})
}
