package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type Voter struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	PotentialLevel    string `json:"potential_level"`
	CompetitorSupport string `json:"competitor_support"`
	PicID             string `json:"pic_id"`
}

func HandleVoters(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	userID := ExtractUserID(r)
	userRole := ExtractUserRole(r)

	if tenantID == "" || userID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		listVoters(w, tenantID, userID, userRole)
	case http.MethodPut:
		updateVoter(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func listVoters(w http.ResponseWriter, tenantID, userID, userRole string) {
	query := `SELECT id, COALESCE(status, 'uncontacted'), COALESCE(potential_level, ''), COALESCE(competitor_support, ''), COALESCE(pic_id::text, '') FROM voters WHERE tenant_id = $1`
	args := []any{tenantID}

	if userRole != "admin" {
		query = repository.GetHierarchyCTE(2) + query + ` AND pic_id IN (SELECT id FROM subordinates)`
		args = append(args, userID)
	}

	rows, err := repository.DB.Query(context.Background(), query, args...)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	voters := []Voter{}
	for rows.Next() {
		var v Voter
		if err := rows.Scan(&v.ID, &v.Status, &v.PotentialLevel, &v.CompetitorSupport, &v.PicID); err == nil {
			voters = append(voters, v)
		}
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: voters})
}

func updateVoter(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		PotentialLevel    string `json:"potential_level"`
		CompetitorSupport string `json:"competitor_support"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	query := "UPDATE voters SET updated_at = now()"
	args := []any{}
	argId := 1

	if req.Status != "" {
		query += fmt.Sprintf(", status = $%d", argId)
		args = append(args, req.Status)
		argId++
	}
	if req.PotentialLevel != "" {
		query += fmt.Sprintf(", potential_level = $%d", argId)
		args = append(args, req.PotentialLevel)
		argId++
	}
	if req.CompetitorSupport != "" {
		query += fmt.Sprintf(", competitor_support = $%d", argId)
		args = append(args, req.CompetitorSupport)
		argId++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argId, argId+1)
	args = append(args, req.ID, tenantID)

	_, err := repository.DB.Exec(context.Background(), query, args...)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update voter"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Voter updated"})
}
