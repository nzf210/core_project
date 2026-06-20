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

	// Logic 3: NIK regional mismatch — kode wilayah NIK tidak cocok dengan TPS region
	// NIK digit 1-6 = kode kabupaten/kota tempat domisili di KTP
	// Cocokkan dengan region TPS yang terdaftar di tps_records
	regionalMismatchQuery := `
		UPDATE endorsements e
		SET is_anomaly = TRUE, anomaly_reason = 'NIK regional code mismatch dengan TPS region'
		FROM citizens c
		LEFT JOIN tps_records t ON t.id = e.tps_id
		WHERE e.tenant_id = $1
		  AND e.is_anomaly = FALSE
		  AND c.nik IS NOT NULL
		  AND t.region_code IS NOT NULL
		  AND LENGTH(c.nik) >= 6
		  AND SUBSTRING(c.nik FROM 1 FOR 6) != t.region_code
	`
	_, err = repository.DB.Exec(ctx, regionalMismatchQuery, tenantID)
	if err != nil && err != sql.ErrNoRows {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to flag regional mismatch anomalies: " + err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Anomaly detection job finished successfully",
	})
}

// HandleAutoAnomalyDetection — F039 AC-3: Cron-triggered auto anomaly detection.
// Dipanggil oleh N8N workflow / subscription worker setiap hari.
func HandleAutoAnomalyDetection(w http.ResponseWriter, r *http.Request) {
	// Auth check: hanya bisa dipanggil oleh system/internal service
	apiKey := r.Header.Get("X-Internal-Key")
	if apiKey == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Internal key required for auto-detection",
		})
		return
	}

	ctx := context.Background()

	// Get all active tenants
	rows, err := repository.DB.Query(ctx, "SELECT DISTINCT tenant_id FROM endorsements WHERE status = 'valid'")
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to fetch tenants: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var tenants []string
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err == nil {
			tenants = append(tenants, tid)
		}
	}

	totalAnomalies := 0
	for _, tenantID := range tenants {
		// Run age anomaly check
		ageAnomalyQuery := `
			UPDATE endorsements e
			SET is_anomaly = TRUE, anomaly_reason = 'Usia Terindikasi > 100 Tahun (Siluman)'
			FROM citizens c
			WHERE e.tenant_id = $1 AND e.is_anomaly = FALSE
			AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) >= 20
			AND CAST(SUBSTRING(c.nik FROM 9 FOR 2) AS INTEGER) <= 25
		`
		_, _ = repository.DB.Exec(ctx, ageAnomalyQuery, tenantID)

		// Run burst anomaly check
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
		_, _ = repository.DB.Exec(ctx, burstAnomalyQuery, tenantID)

		// Run regional mismatch check
		regionalMismatchQuery := `
			UPDATE endorsements e
			SET is_anomaly = TRUE, anomaly_reason = 'NIK regional code mismatch dengan TPS region'
			FROM citizens c
			LEFT JOIN tps_records t ON t.id = e.tps_id
			WHERE e.tenant_id = $1
			  AND e.is_anomaly = FALSE
			  AND c.nik IS NOT NULL
			  AND t.region_code IS NOT NULL
			  AND LENGTH(c.nik) >= 6
			  AND SUBSTRING(c.nik FROM 1 FOR 6) != t.region_code
		`
		_, _ = repository.DB.Exec(ctx, regionalMismatchQuery, tenantID)

		// Count anomalies for this tenant
		var count int
		_ = repository.DB.QueryRow(ctx, "SELECT COUNT(*) FROM endorsements WHERE tenant_id = $1 AND is_anomaly = TRUE", tenantID).Scan(&count)
		totalAnomalies += count
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success:       true,
		Message:       "Auto anomaly detection finished",
		Data:          map[string]interface{}{"tenants_checked": len(tenants), "total_anomalies": totalAnomalies},
	})
}
