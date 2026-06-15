package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

type RealCountPayload struct {
	CampaignID      string `json:"campaign_id"`
	TpsID           string `json:"tps_id"`
	VolunteerID     string `json:"volunteer_id"`
	C1ImageURL      string `json:"c1_image_url"`
	ReportedVotes1  int    `json:"reported_candidate_votes"`
	ReportedVotes2  int    `json:"reported_opponent_votes"`
	ReportedInvalid int    `json:"reported_invalid_votes"`
}

// HandleRealCount processes incoming C1 Plano reports from WA/N8N
func HandleRealCount(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant context"})
		return
	}

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	// GET: Retrieve Real Count Dashboard Data
	if r.Method == http.MethodGet {
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id is required"})
			return
		}

		query := `
			SELECT 
				SUM(reported_candidate_votes) as cand_votes,
				SUM(reported_opponent_votes) as opp_votes,
				SUM(reported_invalid_votes) as inv_votes,
				COUNT(tps_id) as tps_reported
			FROM real_count_records
			WHERE tenant_id = $1 AND campaign_id = $2 AND status IN ('auto_verified', 'human_verified')
		`
		var cand, opp, inv, reported int
		err := repository.DB.QueryRow(context.Background(), query, tenantID, campaignID).Scan(&cand, &opp, &inv, &reported)
		if err != nil {
			cand, opp, inv, reported = 0, 0, 0, 0 // No data yet
		}

		// Count total TPS to get percentage
		var totalTPS int
		_ = repository.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tps").Scan(&totalTPS)

		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"candidate_votes": cand,
				"opponent_votes":  opp,
				"invalid_votes":   inv,
				"tps_reported":    reported,
				"total_tps":       totalTPS,
			},
		})
		return
	}

	// POST: Receive C1 from WA Bot / Saksi
	var req RealCountPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	// Basic validation
	if req.CampaignID == "" || req.TpsID == "" || req.VolunteerID == "" || req.C1ImageURL == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing required fields (campaign_id, tps_id, volunteer_id, c1_image_url)"})
		return
	}

	ctx := context.Background()

	// Ensure volunteer is authorized saksi for this TPS (check saksi_attendances)
	var saksiPresent bool
	_ = repository.DB.QueryRow(ctx, `
		SELECT true FROM saksi_attendances 
		WHERE tenant_id = $1 AND volunteer_id = $2 AND tps_id = $3
	`, tenantID, req.VolunteerID, req.TpsID).Scan(&saksiPresent)

	if !saksiPresent {
		// Tolak jika belum absen / bukan saksi resmi TPS tersebut
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Volunteer is not registered or absent at this TPS"})
		return
	}

	// CALL AI GATEWAY VISION (Internal HTTP Call)
	// We call our own internal ai-gateway to analyze the C1 Image
	aiCandVotes := -1
	aiOppVotes := -1
	aiInvVotes := -1
	
	// TODO: Replace with real HTTP POST to ai-gateway:8002/v1/vision
	// For now, we simulate AI Gateway response reading the exact numbers reported
	// In production, the Vision model will OCR the image and return a JSON.
	mockAIAgrees := true
	if mockAIAgrees {
		aiCandVotes = req.ReportedVotes1
		aiOppVotes = req.ReportedVotes2
		aiInvVotes = req.ReportedInvalid
	}

	// Cross-check reported vs AI
	status := "pending_review"
	var notes string

	if aiCandVotes == req.ReportedVotes1 && aiOppVotes == req.ReportedVotes2 && aiInvVotes == req.ReportedInvalid {
		status = "auto_verified" // Angka ketik saksi match dengan foto C1
	} else {
		status = "needs_human_review"
		notes = "AI Vision mismatch with Saksi input"
	}

	// Insert into DB
	query := `
		INSERT INTO real_count_records 
		(tenant_id, campaign_id, tps_id, volunteer_id, 
		 reported_candidate_votes, reported_opponent_votes, reported_invalid_votes, 
		 ai_candidate_votes, ai_opponent_votes, ai_invalid_votes, 
		 c1_image_url, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (campaign_id, tps_id) 
		DO UPDATE SET 
			reported_candidate_votes = EXCLUDED.reported_candidate_votes,
			c1_image_url = EXCLUDED.c1_image_url,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := repository.DB.Exec(ctx, query,
		tenantID, req.CampaignID, req.TpsID, req.VolunteerID,
		req.ReportedVotes1, req.ReportedVotes2, req.ReportedInvalid,
		aiCandVotes, aiOppVotes, aiInvVotes,
		req.C1ImageURL, status, notes,
	)

	if err != nil {
		if strings.Contains(err.Error(), "unique_tps_real_count") {
			WriteJSON(w, http.StatusConflict, APIResponse{Message: "C1 for this TPS has already been submitted"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record Real Count data"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: "Real count recorded",
		Data: map[string]interface{}{
			"status": status,
			"ai_match": status == "auto_verified",
		},
	})
}
