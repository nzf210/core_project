package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func sendLoginOTP(phoneNumber, authWAProvider, otp string) {
	target := formatPhoneToWAJID(phoneNumber)
	formData := url.Values{}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "system"
	}
	formData.Set("tenant_id", verifierTenant)
	formData.Set("target", target)
	formData.Set("message", "Kode OTP login WCH Anda: "+otp)

	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = waGatewayDefault
	}

	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		payload := strings.NewReader(formData.Encode())
		req, _ := http.NewRequest("POST", waURL+"/api/wa/send", payload)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Message-Type", "otp")
		req.Header.Set("X-Source", "auth-service")
		req.Header.Set("X-WA-Provider-Override", authWAProvider)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to send login OTP via WA Gateway", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if resp.StatusCode == 409 {
			resp.Body.Close()
			slog.Warn("WA Gateway returned 409 (delegated), retrying...", "attempt", i+1)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		slog.Error("Failed to send login OTP after retries", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("WA Gateway rejected login OTP", "status", resp.StatusCode, "body", string(body))
	}
}

func handlePhoneLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req PhoneLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Phone number is required"})
		return
	}

	ctx := context.Background()
	var userID string
	err := DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", req.PhoneNumber).Scan(&userID)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Phone number not registered"})
		return
	} else if err != nil {
		slog.Error("DB query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	otpKey := "phone-login-otp:" + req.PhoneNumber
	if existingOTP, otpErr := Redis.Get(ctx, otpKey).Result(); otpErr == nil && existingOTP != "" {
		ttl, _ := Redis.TTL(ctx, otpKey).Result()
		slog.Info("Login OTP still active, reusing existing", "phone", req.PhoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "OTP sudah dikirim sebelumnya. Masih berlaku selama 1 jam. Silakan cek WhatsApp Anda.",
		})
		return
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	err = Redis.Set(ctx, otpKey, otp, 1*time.Hour).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process login"})
		return
	}

	var authWAProvider = "auto"
	var tenantIDForPref string
	if err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1", req.PhoneNumber).Scan(&tenantIDForPref); err == nil {
		DB.QueryRow(ctx, "SELECT COALESCE(auth_wa_provider_preference::text, 'auto') FROM tenants WHERE id = $1", tenantIDForPref).Scan(&authWAProvider)
	}

	go sendLoginOTP(req.PhoneNumber, authWAProvider, otp)

	slog.Info("Phone login OTP sent", "phone", req.PhoneNumber, "otp", otp)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP telah dikirim ke WhatsApp Anda. Silakan verifikasi.",
	})
}

func handleVerifyPhoneLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req VerifyPhoneLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	ctx := context.Background()
	storedOTP, err := Redis.Get(ctx, "phone-login-otp:"+req.PhoneNumber).Result()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "OTP expired or invalid"})
		return
	}

	if storedOTP != req.OTP {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	var userID, tenantID, role string
	var isDataVerified bool
	err = DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, is_phone_verified FROM users WHERE phone_number = $1",
		req.PhoneNumber,
	).Scan(&userID, &tenantID, &role, &isDataVerified)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	if role != "superadmin" && req.ExpectedTenantID != "" && req.ExpectedTenantID != tenantID {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Nomor WhatsApp Anda tidak terdaftar di tenant ini."})
		return
	}

	tokens, err := generateTokens(userID, tenantID, role, isDataVerified)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	tokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	DB.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", userID, tokenHash, expiresAt)
	Redis.Set(ctx, redisKeyRefreshToken+tokenHash, userID, 7*24*time.Hour)

	var plan string
	if err := DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan); err == nil && plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 30*24*time.Hour)
	}

	slog.Info("Phone login successful", "phone", req.PhoneNumber, "userId", userID)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]any{
			"accessToken":  tokens.AccessToken,
			"refreshToken": tokens.RefreshToken,
			"tenantId":     tenantID,
			"role":         role,
		},
	})
}
