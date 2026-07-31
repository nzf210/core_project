package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type Survey struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
}

func HandleSurveys(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		listSurveys(w, tenantID)
	case http.MethodPost:
		createSurvey(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func listSurveys(w http.ResponseWriter, tenantID string) {
	rows, err := repository.DB.Query(context.Background(),
		"SELECT id, campaign_id, name FROM surveys WHERE tenant_id = $1", tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	surveys := []Survey{}
	for rows.Next() {
		var s Survey
		if err := rows.Scan(&s.ID, &s.CampaignID, &s.Name); err == nil {
			surveys = append(surveys, s)
		}
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: surveys})
}

func createSurvey(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		CampaignID string `json:"campaign_id"`
		Name       string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	var id string
	err := repository.DB.QueryRow(context.Background(),
		"INSERT INTO surveys (tenant_id, campaign_id, name) VALUES ($1, $2, $3) RETURNING id",
		tenantID, req.CampaignID, req.Name).Scan(&id)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create survey"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Survey created", Data: map[string]string{"id": id}})
}
