package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Login: failed to decode JSON body", "error", err)
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	slog.Info("Login attempt", "username", req.Username)

	ctx := context.Background()
	var userID, tenantID, passwordHash string
	var rolePtr *string
	var isDataVerified, mustChangePw bool

	err := DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, password_hash, is_phone_verified, must_change_password FROM users WHERE username = $1 OR email = $1",
		req.Username,
	).Scan(&userID, &tenantID, &rolePtr, &passwordHash, &isDataVerified, &mustChangePw)

	if err == pgx.ErrNoRows {
		slog.Error("Login: user not found", "username", req.Username)
		authLoginsTotal.WithLabelValues("password", "false").Inc()
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
		return
	} else if err != nil {
		slog.Error("DB query failed", "error", err)
		authLoginsTotal.WithLabelValues("password", "false").Inc()
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	role := ""
	if rolePtr != nil {
		role = *rolePtr
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		slog.Error("Login: wrong password", "username", req.Username)
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
		return
	}

	if role != "superadmin" && req.ExpectedTenantID != "" && req.ExpectedTenantID != tenantID {
		slog.Error("Login: user does not belong to expected tenant", "expected", req.ExpectedTenantID, "actual", tenantID)
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Anda tidak terdaftar di tenant ini. Harap periksa URL domain."})
		return
	}

	tokens, err := generateTokens(userID, tenantID, role, isDataVerified)
	if err != nil {
		slog.Error("Token generation failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to generate tokens"})
		return
	}

	tokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err = DB.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		slog.Error("Failed to store refresh token in DB", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	Redis.Set(ctx, redisKeyRefreshToken+tokenHash, userID, 7*24*time.Hour)

	var plan string
	if err := DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan); err == nil && plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 30*24*time.Hour)
	}

	authLoginsTotal.WithLabelValues("password", "true").Inc()

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]any{
			"accessToken":        tokens.AccessToken,
			"refreshToken":       tokens.RefreshToken,
			"tenantId":           tenantID,
			"role":               role,
			"mustChangePassword": mustChangePw,
		},
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	claims, err := validateToken(req.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid refresh token"})
		return
	}

	ctx := context.Background()
	tokenHash := hashToken(req.RefreshToken)

	_, err = Redis.Get(ctx, redisKeyRefreshToken+tokenHash).Result()
	if err != nil {
		var storedUserID string
		errDB := DB.QueryRow(ctx, "SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()", tokenHash).Scan(&storedUserID)
		if errDB != nil {
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Refresh token is invalid or expired"})
			return
		}
	}

	DB.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
	Redis.Del(ctx, redisKeyRefreshToken+tokenHash)

	tokens, err := generateTokens(claims.UserID, claims.TenantID, claims.Role, claims.IsDataVerified)
	if err != nil {
		slog.Error("Failed to generate new tokens", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	newTokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = DB.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", claims.UserID, newTokenHash, expiresAt)
	if err == nil {
		Redis.Set(ctx, redisKeyRefreshToken+newTokenHash, claims.UserID, 7*24*time.Hour)
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Token refreshed successfully",
		Data:    tokens,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()
	tokenHash := hashToken(req.RefreshToken)
	if DB != nil {
		DB.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
	}
	if Redis != nil {
		Redis.Del(ctx, redisKeyRefreshToken+tokenHash)
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logged out successfully"})
}

func validateStaffBody(req *AddStaffRequest) string {
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if strings.HasPrefix(req.PhoneNumber, "0") {
		req.PhoneNumber = "62" + req.PhoneNumber[1:]
	} else if strings.HasPrefix(req.PhoneNumber, "+") {
		req.PhoneNumber = req.PhoneNumber[1:]
	}
	if req.Username == "" || req.Password == "" || req.Role == "" || req.PhoneNumber == "" {
		return "Missing required fields"
	}
	if len(req.Username) < 3 {
		return "Username minimal 3 karakter"
	}
	if !usernameRE.MatchString(req.Username) {
		return "Username hanya boleh huruf, angka, dan underscore"
	}
	if req.Email != "" && !emailRE.MatchString(req.Email) {
		return "Format email tidak valid"
	}
	if len(req.Password) < 6 {
		return "Password minimal 6 karakter"
	}
	if !phoneRE.MatchString(req.PhoneNumber) {
		return "Nomor HP harus diawali 62, contoh: 62812..."
	}
	return ""
}

func handleAddStaff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: msgInvalidOrExpiredToken})
		return
	}
	tenantID := claims.TenantID
	if claims.Role != "owner" && claims.Role != "admin" {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Hanya owner yang dapat menambah staff"})
		return
	}

	var req AddStaffRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}
	if msg := validateStaffBody(&req); msg != "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msg})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}
	ctx := context.Background()
	_, err = DB.Exec(ctx,
		"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number) VALUES ($1, $2, $3, $4, $5, $6)",
		tenantID, req.Username, req.Email, string(hashedPassword), req.Role, req.PhoneNumber,
	)
	if err != nil {
		slog.Error("Failed to insert staff", "error", err)
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Username, email, or phone may already exist"})
		return
	}

	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "Staff added successfully"})
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authorization token is missing or invalid"})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, bearerPrefix)
	claims, err := validateToken(tokenStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Token is valid",
		Data: map[string]any{
			"userId":         claims.UserID,
			"tenantId":       claims.TenantID,
			"role":           claims.Role,
			"isDataVerified": claims.IsDataVerified,
		},
	})
}

func requireAuth(r *http.Request) (*Claims, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
	if err != nil {
		return nil, false
	}
	return claims, true
}
