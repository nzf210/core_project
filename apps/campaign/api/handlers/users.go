package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	PhoneNumber string `json:"phone_number"`
	CreatedAt   string `json:"created_at"`
}

func HandleUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := repository.DB.Query(context.Background(),
			"SELECT id, username, email, role, COALESCE(phone_number, ''), created_at::text FROM users WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.PhoneNumber, &u.CreatedAt); err == nil {
				users = append(users, u)
			}
		}

		if users == nil {
			users = []User{}
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: users})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Username    string `json:"username"`
			Email       string `json:"email"`
			Password    string `json:"password"`
			Role        string `json:"role"`
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
			return
		}

		if req.Role == "" {
			req.Role = "relawan" // default jenjang terendah
		}

		// Simple mock password hash for campaign context, auth service handles real auth
		hash := "hashed_" + req.Password

		var id string
		err := repository.DB.QueryRow(context.Background(),
			"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
			tenantID, req.Username, req.Email, hash, strings.ToLower(req.Role), req.PhoneNumber).Scan(&id)
		
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create user"})
			return
		}

		WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "User created", Data: map[string]string{"id": id}})
		return
	}

	WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
