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
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Login: failed to decode JSON body", "error", err)
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	role := ""
	if rolePtr != nil {
		role = *rolePtr
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
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
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// Also set in Redis for fast revocation checks
	Redis.Set(ctx, "refresh_token:"+tokenHash, userID, 7*24*time.Hour)

	// Cache tenant plan in Redis so feature gates (RequireFeature) resolve correctly.
	var plan string
	if err := DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan); err == nil && plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 30*24*time.Hour)
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
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
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
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
	_, err = Redis.Get(ctx, "refresh_token:"+tokenHash).Result()
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
	Redis.Del(ctx, "refresh_token:"+tokenHash)

	tokens, err := generateTokens(claims.UserID, claims.TenantID, claims.Role, claims.IsDataVerified)
	if err != nil {
		slog.Error("Failed to generate new tokens", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// Store new refresh token
	newTokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = DB.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", claims.UserID, newTokenHash, expiresAt)
	if err == nil {
		Redis.Set(ctx, "refresh_token:"+newTokenHash, claims.UserID, 7*24*time.Hour)
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Token refreshed successfully",
		Data:    tokens,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
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
		Redis.Del(ctx, "refresh_token:"+tokenHash)
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logged out successfully"})
}

func handleAddStaff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authorization required"})
		return
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	}
	tenantID := claims.TenantID
	// ONLY owner can add staff
	if claims.Role != "owner" && claims.Role != "admin" {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Hanya owner yang dapat menambah staff"})
		return
	}

	var req AddStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
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
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authorization token is missing or invalid"})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := validateToken(tokenStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Token is valid",
		Data: map[string]interface{}{
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
	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
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
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
			return
		}

		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
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

	case http.MethodPut:
		var req UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
			return
		}

		if req.NewPassword != "" {
			if req.OldPassword == "" {
				writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "old_password is required to change password"})
				return
			}
			var currentHash string
			err := DB.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", claims.UserID).Scan(&currentHash)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
				writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Old password is incorrect"})
				return
			}
			newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
			DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = false WHERE id = $2", string(newHash), claims.UserID)
		}
		if req.Username != "" {
			var exists bool
			err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)", req.Username, claims.UserID).Scan(&exists)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Terjadi kesalahan saat memeriksa username"})
				return
			}
			if exists {
				writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username sudah digunakan"})
				return
			}
			_, err = DB.Exec(ctx, "UPDATE users SET username = $1 WHERE id = $2 AND tenant_id = $3", req.Username, claims.UserID, claims.TenantID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal menyimpan username"})
				return
			}
		}
		if req.PhoneNumber != "" {
			DB.Exec(ctx, "UPDATE users SET phone_number = $1 WHERE id = $2 AND tenant_id = $3", req.PhoneNumber, claims.UserID, claims.TenantID)
		}
		tenantUpdates := []string{}
		tenantArgs := []interface{}{}
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
			tenantArgs = append(tenantArgs, claims.TenantID)
			query := fmt.Sprintf("UPDATE tenants SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(tenantUpdates, ", "), argIdx)
			DB.Exec(ctx, query, tenantArgs...)
		}
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Profile updated successfully"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
	}
}
