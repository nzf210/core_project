package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
		return
	} else if err != nil {
		slog.Error("DB query failed", "error", err)
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

	// Store refresh token
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

	// Also set in Redis for fast revocation checks
	Redis.Set(ctx, redisKeyRefreshToken+tokenHash, userID, 7*24*time.Hour)

	// Cache tenant plan in Redis so feature gates (RequireFeature) resolve correctly.
	var plan string
	if err := DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan); err == nil && plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 30*24*time.Hour)
	}

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

	// Check Redis first
	_, err = Redis.Get(ctx, redisKeyRefreshToken+tokenHash).Result()
	if err != nil {
		// Try DB if not in redis
		var storedUserID string
		errDB := DB.QueryRow(ctx, "SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()", tokenHash).Scan(&storedUserID)
		if errDB != nil {
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Refresh token is invalid or expired"})
			return
		}
	}

	// Token is valid. Rotate it.
	DB.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
	Redis.Del(ctx, redisKeyRefreshToken+tokenHash)

	tokens, err := generateTokens(claims.UserID, claims.TenantID, claims.Role, claims.IsDataVerified)
	if err != nil {
		slog.Error("Failed to generate new tokens", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	// Store new refresh token
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
	// ONLY owner can add staff
	if claims.Role != "owner" && claims.Role != "admin" {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Hanya owner yang dapat menambah staff"})
		return
	}

	var req AddStaffRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	// Auto-format phone number
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if strings.HasPrefix(req.PhoneNumber, "0") {
		req.PhoneNumber = "62" + req.PhoneNumber[1:]
	} else if strings.HasPrefix(req.PhoneNumber, "+") {
		req.PhoneNumber = req.PhoneNumber[1:]
	}

	if req.Username == "" || req.Password == "" || req.Role == "" || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Missing required fields"})
		return
	}
	if len(req.Username) < 3 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username minimal 3 karakter"})
		return
	}
	if !usernameRE.MatchString(req.Username) {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username hanya boleh huruf, angka, dan underscore"})
		return
	}
	if req.Email != "" && !emailRE.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Format email tidak valid"})
		return
	}
	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Password minimal 6 karakter"})
		return
	}
	if !phoneRE.MatchString(req.PhoneNumber) {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Nomor HP harus diawali 62, contoh: 62812..."})
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

func handleProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuth(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authentication required"})
		return
	}

	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		getProfileData(ctx, w, claims)
	case http.MethodPut:
		updateProfileData(ctx, w, r, claims)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
	}
}

func getProfileData(ctx context.Context, w http.ResponseWriter, claims *Claims) {
	var username, role string
	var phoneNumber, email, businessName, waNumber, logoURL, businessAddress, businessType, plan, tenantName *string
	var isFrozen, onboardingCompleted, mustChangePw bool
	err := DB.QueryRow(ctx, `
		SELECT u.username, u.email, u.phone_number, u.role,
		       COALESCE(t.business_name, t.name), t.wa_number, t.logo_url, t.business_address, t.business_type, t.plan, t.name, COALESCE(t.is_frozen, false), COALESCE(t.onboarding_completed, false), u.must_change_password
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.id = $1 AND u.tenant_id = $2
	`, claims.UserID, claims.TenantID).Scan(
		&username, &email, &phoneNumber, &role,
		&businessName, &waNumber, &logoURL, &businessAddress, &businessType, &plan, &tenantName, &isFrozen, &onboardingCompleted, &mustChangePw,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "User not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"username":             username,
			"email":                derefStr(email),
			"phone_number":         derefStr(phoneNumber),
			"role":                 role,
			"business_name":        derefStr(businessName),
			"wa_number":            derefStr(waNumber),
			"logo_url":             derefStr(logoURL),
			"business_address":     derefStr(businessAddress),
			"business_type":        derefStr(businessType),
			"plan":                 derefStr(plan),
			"tenant_id":            claims.TenantID,
			"is_frozen":            isFrozen,
			"onboarding_completed": onboardingCompleted,
			"must_change_password": mustChangePw,
		},
	})
}

func updateProfileData(ctx context.Context, w http.ResponseWriter, r *http.Request, claims *Claims) {
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.NewPassword != "" && !updatePassword(ctx, w, req, claims.UserID) {
		return
	}

	if req.Username != "" && !updateUsername(ctx, w, req.Username, claims) {
		return
	}

	if req.PhoneNumber != "" {
		DB.Exec(ctx, "UPDATE users SET phone_number = $1 WHERE id = $2 AND tenant_id = $3", req.PhoneNumber, claims.UserID, claims.TenantID)
	}

	updateTenantFields(ctx, req, claims.TenantID)
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Profile updated successfully"})
}

func updatePassword(ctx context.Context, w http.ResponseWriter, req UpdateProfileRequest, userID string) bool {
	if req.OldPassword == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "old_password is required to change password"})
		return false
	}
	var currentHash string
	err := DB.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Old password is incorrect"})
		return false
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = false WHERE id = $2", string(newHash), userID)
	return true
}

func updateUsername(ctx context.Context, w http.ResponseWriter, username string, claims *Claims) bool {
	var exists bool
	err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)", username, claims.UserID).Scan(&exists)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Terjadi kesalahan saat memeriksa username"})
		return false
	}
	if exists {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username sudah digunakan"})
		return false
	}
	_, err = DB.Exec(ctx, "UPDATE users SET username = $1 WHERE id = $2 AND tenant_id = $3", username, claims.UserID, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal menyimpan username"})
		return false
	}
	return true
}

func updateTenantFields(ctx context.Context, req UpdateProfileRequest, tenantID string) {
	tenantUpdates := []string{}
	tenantArgs := []any{}
	argIdx := 1
	if req.BusinessName != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_name = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessName)
		argIdx++
	}
	if req.BusinessAddress != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_address = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessAddress)
		argIdx++
	}
	if req.BusinessType != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_type = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessType)
		argIdx++
	}
	if req.WaNumber != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("wa_number = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.WaNumber)
		argIdx++
	}
	if len(tenantUpdates) > 0 {
		tenantArgs = append(tenantArgs, tenantID)
		query := fmt.Sprintf("UPDATE tenants SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(tenantUpdates, ", "), argIdx)
		DB.Exec(ctx, query, tenantArgs...)
	}
}
