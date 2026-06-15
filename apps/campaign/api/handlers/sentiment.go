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

	ctx := context.Background()

	// GET: Retrieve aggregated issues for Dashboard
	if r.Method == http.MethodGet {
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id required"})
			return
		}

		query := `
			SELECT 
				extracted_issue,
				AVG(sentiment_score) as avg_sentiment,
				COUNT(*) as report_count
			FROM village_issues
			WHERE tenant_id = $1 AND campaign_id = $2
			GROUP BY extracted_issue
			ORDER BY report_count DESC
			LIMIT 10
		`

		rows, err := repository.DB.Query(ctx, query, tenantID, campaignID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var stats []map[string]interface{}
		for rows.Next() {
			var issue string
			var avgSentiment float64
			var count int
			if err := rows.Scan(&issue, &avgSentiment, &count); err == nil {
				stats = append(stats, map[string]interface{}{
					"issue":          issue,
					"avg_sentiment":  avgSentiment,
					"report_count":   count,
					"urgency_status": "high", // mock derived from count
				})
			}
		}

		if stats == nil {
			stats = []map[string]interface{}{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
		return
	}

	// POST: Receive raw chat from Volunteer via WA Bot
	if r.Method == http.MethodPost {
		var req IssueReportPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid payload"})
			return
		}

		if req.RawMessage == "" {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "raw_message empty"})
			return
		}

		// AI NLP Processing via Gateway
		// TODO: Use internal Discovery URL
		chatURL := "http://localhost:8002/v1/chat"
		systemMsg := `Ekstrak keluhan utama dari pesan warga ini menjadi 1-3 kata (extracted_issue), beri nilai sentimen -1.0 (sangat buruk/marah) hingga 1.0 (sangat baik), dan urgency (high/medium/low). Kembalikan dalam JSON: {"extracted_issue": "...", "sentiment": -0.8, "urgency": "high"}`

		payload := map[string]interface{}{
			"message":    req.RawMessage,
			"system_msg": systemMsg,
			"tenant_id":  tenantID,
		}
		body, _ := json.Marshal(payload)

		httpReq, _ := http.NewRequest("POST", chatURL, bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Tenant-ID", tenantID)

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway unreachable"})
			return
		}
		defer resp.Body.Close()

		var aiResp struct {
			Success bool   `json:"success"`
			Text    string `json:"text"`
		}
		json.NewDecoder(resp.Body).Decode(&aiResp)

		extractedIssue := "Isu Tidak Diketahui"
		sentimentScore := 0.0
		urgencyLevel := "medium"

		if aiResp.Success && strings.Contains(aiResp.Text, "{") {
			// Extract JSON from AI text
			start := strings.Index(aiResp.Text, "{")
			end := strings.LastIndex(aiResp.Text, "}")
			if start != -1 && end != -1 && end > start {
				jsonStr := aiResp.Text[start : end+1]
				var nlpResult struct {
					ExtractedIssue string  `json:"extracted_issue"`
					Sentiment      float64 `json:"sentiment"`
					Urgency        string  `json:"urgency"`
				}
				if err := json.Unmarshal([]byte(jsonStr), &nlpResult); err == nil {
					extractedIssue = nlpResult.ExtractedIssue
					sentimentScore = nlpResult.Sentiment
					urgencyLevel = nlpResult.Urgency
				}
			}
		}

		// Insert to DB
		query := `
			INSERT INTO village_issues 
			(tenant_id, campaign_id, volunteer_id, village_id, raw_message, extracted_issue, sentiment_score, urgency_level)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
		`
		var newID string
		err = repository.DB.QueryRow(ctx, query,
			tenantID, req.CampaignID, req.VolunteerID, req.VillageID,
			req.RawMessage, extractedIssue, sentimentScore, urgencyLevel,
		).Scan(&newID)

		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to record issue"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Issue recorded and analyzed",
			Data: map[string]interface{}{
				"id":              newID,
				"extracted_issue": extractedIssue,
				"sentiment":       sentimentScore,
			},
		})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
