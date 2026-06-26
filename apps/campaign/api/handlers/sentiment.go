package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

type IssueReportPayload struct {
	CampaignID  string `json:"campaign_id"`
	VolunteerID string `json:"volunteer_id"`
	VillageID   string `json:"village_id"`
	RawMessage  string `json:"raw_message"`
}

func HandleSentimentIssues(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getSentimentStats(w, r, tenantID)
	case http.MethodPost:
		reportIssue(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func getSentimentStats(w http.ResponseWriter, r *http.Request, tenantID string) {
	campaignID := r.URL.Query().Get("campaign_id")
	if campaignID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id required"})
		return
	}

	query := `
		SELECT extracted_issue, AVG(sentiment_score) as avg_sentiment, COUNT(*) as report_count
		FROM village_issues
		WHERE tenant_id = $1 AND campaign_id = $2
		GROUP BY extracted_issue
		ORDER BY report_count DESC
		LIMIT 10
	`
	rows, err := repository.DB.Query(context.Background(), query, tenantID, campaignID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	stats := []map[string]any{}
	for rows.Next() {
		var issue string
		var avgSentiment float64
		var count int
		if err := rows.Scan(&issue, &avgSentiment, &count); err == nil {
			stats = append(stats, map[string]any{
				"issue":          issue,
				"avg_sentiment":  avgSentiment,
				"report_count":   count,
				"urgency_status": "high",
			})
		}
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
}

func reportIssue(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req IssueReportPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
		return
	}
	if req.RawMessage == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "raw_message empty"})
		return
	}

	nlp := performNLP(tenantID, req.RawMessage)

	query := `
		INSERT INTO village_issues
		(tenant_id, campaign_id, volunteer_id, village_id, raw_message, extracted_issue, sentiment_score, urgency_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`
	var newID string
	err := repository.DB.QueryRow(context.Background(), query,
		tenantID, req.CampaignID, req.VolunteerID, req.VillageID,
		req.RawMessage, nlp.Issue, nlp.Sentiment, nlp.Urgency,
	).Scan(&newID)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record issue"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]any{"id": newID, "extracted_issue": nlp.Issue, "sentiment": nlp.Sentiment},
	})
}

type nlpResult struct {
	Issue     string  `json:"extracted_issue"`
	Sentiment float64 `json:"sentiment"`
	Urgency   string  `json:"urgency"`
}

func performNLP(tenantID, message string) nlpResult {
	payload := map[string]any{
		"message":    message,
		"system_msg": "Ekstrak keluhan utama (1-3 kata), sentimen (-1 s/d 1), dan urgency (high/medium/low). JSON: {extracted_issue, sentiment, urgency}",
		"tenant_id":  tenantID,
	}
	body, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("POST", chatGatewayURL, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil || resp.StatusCode != 200 {
		return nlpResult{"Isu Tidak Diketahui", 0.0, "medium"}
	}
	defer resp.Body.Close()

	var aiResp struct{ Text string `json:"text"` }
	json.NewDecoder(resp.Body).Decode(&aiResp)

	var res nlpResult
	if start := strings.Index(aiResp.Text, "{"); start != -1 {
		json.Unmarshal([]byte(aiResp.Text[start:]), &res)
	}
	return res
}