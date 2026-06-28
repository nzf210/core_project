package main

import (
	"context"
	"encoding/json"
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
	for range 3 {
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
			slog.Warn("WA Gateway returned 409 (delegated), retrying...")
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
	slog.Info("handlePhoneLogin: looking for phone", "phone", req.PhoneNumber)

	// Normalize phone for DB lookup: try original first, then 62xx/08xx variants
	lookupPhone := req.PhoneNumber
	var userID string
	var err error
	err = DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", lookupPhone).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Try alternate formats: DB stores 08xx but request is 62xx, or vice versa
		normPhone := req.PhoneNumber
		if strings.HasPrefix(normPhone, "62") {
			normPhone = "0" + normPhone[2:]
		} else if strings.HasPrefix(normPhone, "0") {
			normPhone = "62" + normPhone[1:]
		}
		if normPhone != lookupPhone {
			slog.Info("handlePhoneLogin: not found, trying alternate format", "original", req.PhoneNumber, "alternate", normPhone)
			err = DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", normPhone).Scan(&userID)
			if err == nil {
				lookupPhone = normPhone
			}
		}
		if err == pgx.ErrNoRows {
			slog.Warn("handlePhoneLogin: phone not found in DB", "tried_original", req.PhoneNumber, "tried_alternate", normPhone)
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Phone number not registered"})
			return
		}
	}
	if err != nil && err != pgx.ErrNoRows {
		slog.Error("DB query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	var authWAProvider = "auto"
	var tenantIDForPref string
	if err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1", lookupPhone).Scan(&tenantIDForPref); err == nil {
		DB.QueryRow(ctx, "SELECT COALESCE(auth_wa_provider_preference::text, 'auto') FROM tenants WHERE id = $1", tenantIDForPref).Scan(&authWAProvider)
	}

	// Skip proactive OTP if platform uses whatsmeow (user must chat WA Center)
	waProvider, _ := getPlatformWAProvider(ctx)

	otpKey := "phone-login-otp:" + req.PhoneNumber
	// Skip reuse check when whatsmeow — OTP was never sent, always generate fresh code.
	if waProvider != "whatsmeow" {
		if existingOTP, otpErr := Redis.Get(ctx, otpKey).Result(); otpErr == nil && existingOTP != "" {
			ttl, _ := Redis.TTL(ctx, otpKey).Result()
			slog.Info("Login OTP still active, reusing existing", "phone", req.PhoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
			writeJSON(w, http.StatusOK, Response{
				Success: true,
				Message: "OTP sudah dikirim sebelumnya. Masih berlaku selama 1 jam. Silakan cek WhatsApp Anda.",
			})
			return
		}
	}

	// Normalize phone for auth:pending key.
	// extractPhoneFromJID in wa-gateway returns 0xxx format (62xx→0xx),
	// so auth:pending key must use 0xxx to match when user sends "OTP" via WA.
	redisPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, req.PhoneNumber)
	if strings.HasPrefix(redisPhone, "62") {
		redisPhone = "0" + redisPhone[2:] // 62xxx → 0xxx
	}

	// Set pending login state - OTP will be generated when user sends "OTP" to WA Center
	slog.Info("Setting auth:pending for OTP trigger",
		"original_phone", req.PhoneNumber,
		"normalized_phone", redisPhone,
		"redis_key", "auth:pending:"+redisPhone)
	// ponytail: 15 min TTL — user needs time to read message and switch to WA
	// Value stores the login phone so wa-gateway can match via sender phone even when they differ
	err = Redis.Set(ctx, "auth:pending:"+redisPhone, redisPhone, 15*time.Minute).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process login"})
		return
	}

	slog.Info("Phone login pending - awaiting WA Center OTP trigger", "phone", req.PhoneNumber)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Permintaan login diterima. Silakan kirim pesan 'OTP' ke WA Center untuk menerima kode.",
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
	otpPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, req.PhoneNumber)
	if strings.HasPrefix(otpPhone, "62") {
		otpPhone = "0" + otpPhone[2:]
	}

	storedOTP, err := Redis.Get(ctx, "phone-login-otp:"+otpPhone).Result()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "OTP expired or invalid"})
		return
	}

	// wa-gateway stores "otp|phone" format; extract just the OTP
	storedCode := storedOTP
	if idx := strings.Index(storedOTP, "|"); idx >= 0 {
		storedCode = storedOTP[:idx]
	}

	if storedCode != req.OTP {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	var userID, tenantID, role string
	var isDataVerified bool
	// Try original format first, then normalized (08xx↔62xx)
	lookupPhone := req.PhoneNumber
	err = DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, is_phone_verified FROM users WHERE phone_number = $1",
		lookupPhone,
	).Scan(&userID, &tenantID, &role, &isDataVerified)
	if err == pgx.ErrNoRows {
		normPhone := req.PhoneNumber
		if strings.HasPrefix(normPhone, "0") {
			normPhone = "62" + normPhone[1:]
		} else if strings.HasPrefix(normPhone, "62") {
			normPhone = "0" + normPhone[2:]
		}
		err = DB.QueryRow(ctx,
			"SELECT id, tenant_id, role, is_phone_verified FROM users WHERE phone_number = $1",
			normPhone,
		).Scan(&userID, &tenantID, &role, &isDataVerified)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Nomor tidak terdaftar."})
			return
		}
		lookupPhone = normPhone
	}
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
