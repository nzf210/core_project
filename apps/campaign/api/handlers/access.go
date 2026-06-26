package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AuditLog struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	CreatedAt string `json:"created_at"`
}

func HandleRoles(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, name, COALESCE(description, '') FROM roles WHERE tenant_id = $1", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var roles []Role
		for rows.Next() {
			var role Role
			if err := rows.Scan(&role.ID, &role.Name, &role.Description); err == nil {
				roles = append(roles, role)
			}
		}

		if roles == nil {
			roles = []Role{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: roles})

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO roles (tenant_id, name, description) VALUES ($1, $2, $3) RETURNING id",
			tenantID, req.Name, req.Description).Scan(&id)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create role"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Role created", Data: map[string]string{"id": id}})

	default:
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, action, resource, created_at::text FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var logs []AuditLog
		for rows.Next() {
			var log AuditLog
			if err := rows.Scan(&log.ID, &log.Action, &log.Resource, &log.CreatedAt); err == nil {
				logs = append(logs, log)
			}
		}

		if logs == nil {
			logs = []AuditLog{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: logs})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
