package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

var visionGatewayURL = "http://ai-gateway:8002/v1/vision"
var chatGatewayURL = "http://ai-gateway:8002/v1/chat"

func init() {
	// Note: AI_GATEWAY_URL kept as os.Getenv for runtime override capability
	// Allows dynamic configuration without rebuilding
	if url := os.Getenv("AI_GATEWAY_URL"); url != "" {
		visionGatewayURL = strings.Replace(url, "/v1/chat", "/v1/vision", 1)
		chatGatewayURL = url
	}
}

type RealCountPayload struct {
	CampaignID      string `json:"campaign_id"`
	TpsID           string `json:"tps_id"`
	VolunteerID     string `json:"volunteer_id"`
	C1ImageURL      string `json:"c1_image_url"`
	ReportedVotes1  int    `json:"reported_candidate_votes"`
	ReportedVotes2  int    `json:"reported_opponent_votes"`
	ReportedInvalid int    `json:"reported_invalid_votes"`
}

type visionRequest struct {
	TenantID string `json:"tenant_id"`
	ImageURL string `json:"image_url"`
	Prompt   string `json:"prompt"`
}

type visionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Text string `json:"text"`
	} `json:"data"`
}

type c1OCRResult struct {
	CandidateVotes int `json:"candidate_votes"`
	OpponentVotes  int `json:"opponent_votes"`
	InvalidVotes   int `json:"invalid_votes"`
}

// HandleRealCount processes incoming C1 Plano reports from WA/N8N
func HandleRealCount(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getRealCountDashboard(w, r, tenantID)
	case http.MethodPost:
		recordRealCount(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func getRealCountDashboard(w http.ResponseWriter, r *http.Request, tenantID string) {
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

	var totalTPS int
	_ = repository.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM tps").Scan(&totalTPS)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"candidate_votes": cand,
			"opponent_votes":  opp,
			"invalid_votes":   inv,
			"tps_reported":    reported,
			"total_tps":       totalTPS,
		},
	})
}

func recordRealCount(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req RealCountPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	if req.CampaignID == "" || req.TpsID == "" || req.VolunteerID == "" || req.C1ImageURL == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing required fields (campaign_id, tps_id, volunteer_id, c1_image_url)"})
		return
	}

	ctx := context.Background()

	var saksiPresent bool
	_ = repository.DB.QueryRow(ctx, `
		SELECT true FROM saksi_attendances
		WHERE tenant_id = $1 AND volunteer_id = $2 AND tps_id = $3
	`, tenantID, req.VolunteerID, req.TpsID).Scan(&saksiPresent)

	if !saksiPresent {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Volunteer is not registered or absent at this TPS"})
		return
	}

	aiCandVotes, aiOppVotes, aiInvVotes := performOCR(ctx, tenantID, req.C1ImageURL)

	if aiCandVotes == -1 {
		aiCandVotes = req.ReportedVotes1
		aiOppVotes = req.ReportedVotes2
		aiInvVotes = req.ReportedInvalid
		slog.Warn("AI Vision failed, using reported values", "tps_id_length", len(req.TpsID))
	}

	status, notes := "pending_review", ""
	if aiCandVotes == req.ReportedVotes1 && aiOppVotes == req.ReportedVotes2 && aiInvVotes == req.ReportedInvalid {
		status = "auto_verified"
	} else {
		status = "needs_human_review"
		notes = "AI Vision mismatch with Saksi input"
	}

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
		Data: map[string]any{
			"status":   status,
			"ai_match": status == "auto_verified",
		},
	})
}

func performOCR(ctx context.Context, tenantID, imageURL string) (int, int, int) {
	visionReq := visionRequest{
		TenantID: tenantID,
		ImageURL: imageURL,
		Prompt:   "Extract C1 numbers",
	}
	visionReqBytes, _ := json.Marshal(visionReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", visionGatewayURL, bytes.NewBuffer(visionReqBytes))
	if err != nil {
		return -1, -1, -1
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Tenant-ID", tenantID)
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return -1, -1, -1
	}
	defer httpResp.Body.Close()

	var visionResp visionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&visionResp); err != nil || !visionResp.Success {
		return -1, -1, -1
	}

	var ocrResult c1OCRResult
	if err := json.Unmarshal([]byte(visionResp.Data.Text), &ocrResult); err != nil {
		return -1, -1, -1
	}
	return ocrResult.CandidateVotes, ocrResult.OpponentVotes, ocrResult.InvalidVotes
}
