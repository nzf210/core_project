package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleSuperAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: errMethodNotAllowed})
		return
	}

	var req SuperAdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()
	var userID, tenantID, passwordHash, role string
	var isDataVerified bool
	err := DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, password_hash, is_phone_verified FROM users WHERE username = $1 AND role = 'superadmin'",
		req.Username,
	).Scan(&userID, &tenantID, &role, &passwordHash, &isDataVerified)

	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials or not a super admin"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
		return
	}

	tokens, err := generateTokens(userID, tenantID, role, isDataVerified)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to generate tokens"})
		return
	}

	tokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	DB.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", userID, tokenHash, expiresAt)
	Redis.Set(ctx, "refresh_token:"+tokenHash, userID, 7*24*time.Hour)

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Super admin login successful",
		Data: map[string]any{
			"accessToken":  tokens.AccessToken,
			"refreshToken": tokens.RefreshToken,
			"role":         role,
		},
	})
}
