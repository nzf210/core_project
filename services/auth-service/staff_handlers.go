package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func handleStaffList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: msgAuthorizationRequired})
		return
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	}

	ctx := context.Background()
	rows, err := DB.Query(ctx, "SELECT id, username, email, phone_number, role, created_at FROM users WHERE tenant_id = $1 AND role IN ('kasir', 'admin', 'staff') ORDER BY created_at DESC", claims.TenantID)
	if err != nil {
		slog.Error("Failed to list staff", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal mengambil data pegawai"})
		return
	}
	defer rows.Close()

	var staffs []map[string]any
	for rows.Next() {
		var id, username, email, role string
		var phone sql.NullString
		var created time.Time
		if err := rows.Scan(&id, &username, &email, &phone, &role, &created); err != nil {
			continue
		}

		staff := map[string]any{
			"id":           id,
			"username":     username,
			"email":        email,
			"role":         role,
			"phone_number": "",
			"created_at":   created,
		}
		if phone.Valid {
			staff["phone_number"] = phone.String
		}
		staffs = append(staffs, staff)
	}
	if staffs == nil {
		staffs = make([]map[string]any, 0)
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: staffs})
}

func handleStaffUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: msgAuthorizationRequired})
		return
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
	if err != nil || claims.Role != "owner" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Hanya owner yang dapat mengubah pegawai"})
		return
	}

	var req struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		PhoneNumber string `json:"phone_number"`
		Password    string `json:"password"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()

	// Base update (username, phone)
	_, err = DB.Exec(ctx, "UPDATE users SET username = $1, phone_number = $2 WHERE id = $3 AND tenant_id = $4",
		req.Username, req.PhoneNumber, req.ID, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal memperbarui pegawai"})
		return
	}

	// Reset password if provided
	if req.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = true WHERE id = $2 AND tenant_id = $3",
			string(hash), req.ID, claims.TenantID)
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Pegawai berhasil diperbarui"})
}

func handleStaffDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: msgAuthorizationRequired})
		return
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
	if err != nil || claims.Role != "owner" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Hanya owner yang dapat menghapus pegawai"})
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "ID pegawai diperlukan"})
		return
	}

	ctx := context.Background()
	_, err = DB.Exec(ctx, "DELETE FROM users WHERE id = $1 AND tenant_id = $2 AND role != 'owner'", id, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal menghapus pegawai"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Pegawai berhasil dihapus"})
}
