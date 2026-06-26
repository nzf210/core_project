package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type FraudReportPayload struct {
	CampaignID    string  `json:"campaign_id"`
	VolunteerID   string  `json:"volunteer_id"`
	ReporterName  string  `json:"reporter_name"`
	ViolationType string  `json:"violation_type"`
	Description   string  `json:"description"`
	ProofImageURL string  `json:"proof_image_url"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
}

func HandleFraudReports(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getFraudReports(w, r, tenantID)
	case http.MethodPost:
		createFraudReport(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func getFraudReports(w http.ResponseWriter, r *http.Request, tenantID string) {
	campaignID := r.URL.Query().Get("campaign_id")
	if campaignID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id is required"})
		return
	}

	query := `
		SELECT id, violation_type, description, proof_image_url, location_lat, location_lng, status, created_at
		FROM fraud_reports
		WHERE tenant_id = $1 AND campaign_id = $2
		ORDER BY created_at DESC
	`
	rows, err := repository.DB.Query(context.Background(), query, tenantID, campaignID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	var reports []map[string]interface{}
	for rows.Next() {
		var id, vType, desc, proof, status, createdAt string
		var lat, lng float64

		if err := rows.Scan(&id, &vType, &desc, &proof, &lat, &lng, &status, &createdAt); err == nil {
			reports = append(reports, map[string]interface{}{
				"id":              id,
				"violation_type":  vType,
				"description":     desc,
				"proof_image_url": proof,
				"lat":             lat,
				"lng":             lng,
				"status":          status,
				"created_at":      createdAt,
			})
		}
	}
	if reports == nil {
		reports = []map[string]interface{}{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: reports})
}

func createFraudReport(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req FraudReportPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
		return
	}

	if req.Lat == 0 || req.Lng == 0 || req.ProofImageURL == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Location (lat, lng) and proof_image_url are required"})
		return
	}

	query := `
		INSERT INTO fraud_reports
		(tenant_id, campaign_id, volunteer_id, reporter_name, violation_type, description, proof_image_url, location_lat, location_lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`
	var newID string
	err := repository.DB.QueryRow(context.Background(), query,
		tenantID, req.CampaignID, req.VolunteerID, req.ReporterName,
		req.ViolationType, req.Description, req.ProofImageURL, req.Lat, req.Lng,
	).Scan(&newID)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record fraud report"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Fraud report submitted successfully",
		Data:    map[string]string{"id": newID},
	})
}
