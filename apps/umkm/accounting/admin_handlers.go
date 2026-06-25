package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)


func handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createAdminTenant(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" || DB == nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
			return
		}

		ctx := r.Context()
		tx, err := DB.Begin(ctx)
		if err != nil {
			slog.Error("Failed to start transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
			return
		}
		defer tx.Rollback(ctx)

		// 1. Manual cascade for journal_lines because it references chart_of_accounts without ON DELETE CASCADE
		_, err = tx.Exec(ctx, "DELETE FROM journal_lines WHERE entry_id IN (SELECT id FROM journal_entries WHERE tenant_id = $1)", id)
		if err != nil {
			slog.Error("Failed to delete journal_lines", "error", err, "tenant_id", id)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to clean up tenant journal data"})
			return
		}

		// 2. Delete tenant (will cascade to journal_entries, chart_of_accounts, users, etc.)
		_, err = tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
		if err != nil {
			slog.Error("Failed to delete tenant row", "error", err, "tenant_id", id)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to delete tenant"})
			return
		}

		tx.Commit(ctx)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant deleted"})
		return
	}

	if r.Method == http.MethodPut {
		if DB == nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
			return
		}
		var req struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Plan        string `json:"plan"`
			Username    string `json:"username"`
			Email       string `json:"email"`
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		ctx := r.Context()
		tx, err := DB.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
			return
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "UPDATE tenants SET name = $1, plan = $2, updated_at = NOW() WHERE id = $3", req.Name, req.Plan, req.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update tenant"})
			return
		}

		_, err = tx.Exec(ctx, "UPDATE users SET username = $1, email = $2, phone_number = $3, updated_at = NOW() WHERE tenant_id = $4", req.Username, req.Email, req.PhoneNumber, req.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update user"})
			return
		}

		tx.Commit(ctx)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant updated"})
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{}})
		return
	}

	rows, err := DB.Query(r.Context(), `
		SELECT t.id, t.name, t.plan, t.created_at, 
		       COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.phone_number, '')
		FROM tenants t
		LEFT JOIN (
			SELECT DISTINCT ON (tenant_id) tenant_id, username, email, phone_number 
			FROM users ORDER BY tenant_id, created_at ASC
		) u ON u.tenant_id = t.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id, name, plan, username, email, phone string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &plan, &createdAt, &username, &email, &phone); err == nil {
			result = append(result, map[string]interface{}{
				"id":           id,
				"name":         name,
				"plan":         plan,
				"username":     username,
				"email":        email,
				"phone_number": phone,
				"expiry":       "2027-12-31", // Mock expiry for now
			})
		}
	}
	if result == nil {
		result = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

func createAdminTenant(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	var req struct {
		Name         string `json:"name"`
		Username     string `json:"username"`
		Email        string `json:"email"`
		PhoneNumber  string `json:"phone_number"`
		Plan         string `json:"plan"`
		Password     string `json:"password"`
		Subdomain    string `json:"subdomain"`
		CustomDomain string `json:"custom_domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	if req.Name == "" || req.Username == "" || req.Email == "" || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "All fields are required"})
		return
	}
	if req.Plan == "" {
		req.Plan = "lite"
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
		return
	}
	defer tx.Rollback(ctx)

	// 1. Create Tenant
	var tenantID string
	err = tx.QueryRow(ctx, "INSERT INTO tenants (name, plan, subdomain, custom_domain) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, '')) RETURNING id", req.Name, req.Plan, req.Subdomain, req.CustomDomain).Scan(&tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create tenant"})
		return
	}

	// Use provided password or fallback to default
	password := req.Password
	if password == "" {
		password = "password123"
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	passwordHash := string(hashBytes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to hash password"})
		return
	}

	var userID string
	err = tx.QueryRow(ctx, "INSERT INTO users (tenant_id, username, email, password_hash, phone_number) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, req.Username, req.Email, passwordHash, req.PhoneNumber).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create user (phone/email/username might exist)"})
		return
	}

	// 3. Seed Accounting (Default COA)
	accounts := []struct{ Code, Name, Type string }{
		{"100", "Kas", "asset"},
		{"101", "Bank / QRIS", "asset"},
		{"110", "Piutang Usaha", "asset"},
		{"120", "Persediaan", "asset"},
		{"200", "Hutang Usaha", "liability"},
		{"300", "Modal", "equity"},
		{"400", "Pendapatan Usaha", "revenue"},
		{"500", "Beban Operasional", "expense"},
	}
	for _, acc := range accounts {
		_, err = tx.Exec(ctx,
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type) VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, code) DO NOTHING",
			tenantID, acc.Code, acc.Name, acc.Type,
		)
		if err != nil {
			slog.Error("Failed to seed account during admin tenant creation", "error", err)
		}
	}

	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant and user created successfully!"})
}