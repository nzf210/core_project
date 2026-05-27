package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

type Notification struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Status  string `json:"status"`
}

func HandleNotifications(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, title, message, type, status FROM notifications WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var notifications []Notification
		for rows.Next() {
			var n Notification
			if err := rows.Scan(&n.ID, &n.Title, &n.Message, &n.Type, &n.Status); err == nil {
				notifications = append(notifications, n)
			}
		}

		if notifications == nil {
			notifications = []Notification{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: notifications})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Title   string `json:"title"`
			Message string `json:"message"`
			Type    string `json:"type"` // in_app, email, broadcast
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO notifications (tenant_id, title, message, type) VALUES ($1, $2, $3, $4) RETURNING id",
			tenantID, req.Title, req.Message, req.Type).Scan(&id)
		
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create notification"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Notification sent", Data: map[string]string{"id": id}})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
