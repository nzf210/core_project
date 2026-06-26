package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type Task struct {
	ID               string `json:"id"`
	CampaignID       string `json:"campaign_id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	CreatedBy        string `json:"created_by"`
	AssignedTo       string `json:"assigned_to"`
	VerificationType string `json:"verification_type"`
	ProofImage       string `json:"proof_image"`
	GpsLocation      string `json:"gps_location"`
	IsVerified       bool   `json:"is_verified"`
}

func HandleTasks(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	userID := ExtractUserID(r)
	userRole := ExtractUserRole(r)

	if tenantID == "" || userID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		listTasks(w, tenantID, userID, userRole)
	case http.MethodPost:
		createTask(w, r, tenantID, userID)
	case http.MethodPut:
		updateTask(w, r, tenantID)
	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func listTasks(w http.ResponseWriter, tenantID, userID, userRole string) {
	query := `SELECT id, campaign_id, title, COALESCE(description, ''), status, COALESCE(created_by::text, ''), COALESCE(assigned_to::text, ''), COALESCE(verification_type, 'auto'), COALESCE(proof_image, ''), COALESCE(gps_location, ''), COALESCE(is_verified, false) FROM tasks WHERE tenant_id = $1`
	args := []any{tenantID}

	if userRole != "admin" {
		query = repository.GetHierarchyCTE(2) + query + ` AND assigned_to IN (SELECT id FROM subordinates)`
		args = append(args, userID)
	}

	rows, err := repository.DB.Query(context.Background(), query, args...)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.CampaignID, &t.Title, &t.Description, &t.Status, &t.CreatedBy, &t.AssignedTo, &t.VerificationType, &t.ProofImage, &t.GpsLocation, &t.IsVerified); err == nil {
			tasks = append(tasks, t)
		}
	}
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: tasks})
}

func createTask(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	var req struct {
		CampaignID       string `json:"campaign_id"`
		Title            string `json:"title"`
		Description      string `json:"description"`
		AssignedTo       string `json:"assigned_to"`
		VerificationType string `json:"verification_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}
	if req.VerificationType == "" {
		req.VerificationType = "auto"
	}

	var assignedTo any = req.AssignedTo
	if req.AssignedTo == "" {
		assignedTo = nil
	}

	var id string
	err := repository.DB.QueryRow(context.Background(),
		"INSERT INTO tasks (tenant_id, campaign_id, title, description, created_by, assigned_to, verification_type) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		tenantID, req.CampaignID, req.Title, req.Description, userID, assignedTo, req.VerificationType).Scan(&id)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create task"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Task created", Data: map[string]string{"id": id}})
}

func updateTask(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		ID          string `json:"id"`
		ProofImage  string `json:"proof_image"`
		GpsLocation string `json:"gps_location"`
		Status      string `json:"status"`
		IsVerified  *bool  `json:"is_verified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	query := "UPDATE tasks SET updated_at = now()"
	args := []any{}
	argId := 1

	if req.ProofImage != "" {
		query += fmt.Sprintf(", proof_image = $%d", argId)
		args = append(args, req.ProofImage)
		argId++
	}
	if req.GpsLocation != "" {
		query += fmt.Sprintf(", gps_location = $%d", argId)
		args = append(args, req.GpsLocation)
		argId++
	}
	if req.Status != "" {
		query += fmt.Sprintf(", status = $%d", argId)
		args = append(args, req.Status)
		argId++
	}
	if req.IsVerified != nil {
		query += fmt.Sprintf(", is_verified = $%d", argId)
		args = append(args, *req.IsVerified)
		argId++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argId, argId+1)
	args = append(args, req.ID, tenantID)

	_, err := repository.DB.Exec(context.Background(), query, args...)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update task"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Task updated"})
}
