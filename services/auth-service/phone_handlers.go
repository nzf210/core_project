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

func handlePhoneLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req PhoneLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// OTP 1-hour reuse: check if valid OTP already exists for this phone
	otpKey := "phone-login-otp:" + req.PhoneNumber
	if existingOTP, err := Redis.Get(ctx, otpKey).Result(); err == nil && existingOTP != "" {
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

	// Read auth_wa_provider_preference from user's tenant (if registered)
	var authWAProvider string
	var tenantIDForPref string
	if err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1", req.PhoneNumber).Scan(&tenantIDForPref); err == nil {
		if err := DB.QueryRow(ctx, "SELECT COALESCE(auth_wa_provider_preference::text, 'auto') FROM tenants WHERE id = $1", tenantIDForPref).Scan(&authWAProvider); err != nil {
			authWAProvider = "auto"
		}
	} else {
		authWAProvider = "auto"
	}

	go func() {
		target := formatPhoneToWAJID(req.PhoneNumber)

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
			waURL = "http://wa-gateway:8202"
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
	}()

	slog.Info("Phone login OTP sent", "phone", req.PhoneNumber, "otp", otp)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP telah dikirim ke WhatsApp Anda. Silakan verifikasi.",
	})
}

func handleVerifyPhoneLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req VerifyPhoneLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "User lookup failed"})
		return
	}

	if role != "superadmin" && req.ExpectedTenantID != "" && req.ExpectedTenantID != tenantID {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Nomor WhatsApp Anda tidak terdaftar di tenant ini."})
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
	// OTP persists for full 1-hour window (reusable during active period)
	// Redis TTL handles auto-expiry

	// Cache tenant plan in Redis so feature gates (RequireFeature) resolve correctly.
	var plan string
	if err := DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan); err == nil && plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 30*24*time.Hour)
	}

	slog.Info("Phone login successful", "phone", req.PhoneNumber, "userId", userID)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"accessToken":  tokens.AccessToken,
			"refreshToken": tokens.RefreshToken,
			"tenantId":     tenantID,
			"role":         role,
		},
	})
}
