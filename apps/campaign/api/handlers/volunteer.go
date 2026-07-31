package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/response"
)

type Volunteer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Rank  int    `json:"rank"`
}

func HandleVolunteers(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		listVolunteers(w, tenantID)
	case http.MethodPost:
		registerVolunteer(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func listVolunteers(w http.ResponseWriter, tenantID string) {
	rows, err := repository.DB.Query(context.Background(),
		"SELECT id, name, COALESCE(phone, ''), rank FROM volunteers WHERE tenant_id = $1", tenantID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	volunteers := []Volunteer{}
	for rows.Next() {
		var v Volunteer
		if err := rows.Scan(&v.ID, &v.Name, &v.Phone, &v.Rank); err == nil {
			volunteers = append(volunteers, v)
		}
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: volunteers})
}

func registerVolunteer(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	var id string
	err := repository.DB.QueryRow(context.Background(),
		"INSERT INTO volunteers (tenant_id, name, phone) VALUES ($1, $2, $3) RETURNING id",
		tenantID, req.Name, req.Phone).Scan(&id)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to register volunteer"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Volunteer registered", Data: map[string]string{"id": id}})
}

func HandleVolunteerStats(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var totalVolunteers int
	err := repository.DB.QueryRow(context.Background(), "SELECT count(*) FROM volunteers WHERE tenant_id = $1", tenantID).Scan(&totalVolunteers)
	if err != nil {
		totalVolunteers = 0
	}

	stats := map[string]any{
		"total_volunteers":  totalVolunteers,
		"active_volunteers": totalVolunteers,
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
}
