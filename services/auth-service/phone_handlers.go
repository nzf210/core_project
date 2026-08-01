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
		req.Header.Set(headerContentType, contentTypeFormURLEncoded)
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

func lookupUserByPhone(ctx context.Context, phoneNumber string) (string, string, error) {
	var userID string
	err := DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", phoneNumber).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Try alternate format
		altPhone := phoneNumber
		if strings.HasPrefix(altPhone, "62") {
			altPhone = "0" + altPhone[2:]
		} else if strings.HasPrefix(altPhone, "0") {
			altPhone = "62" + altPhone[1:]
		}
		if altPhone != phoneNumber {
			slog.Info("handlePhoneLogin: trying alternate format", "original", phoneNumber, "alternate", altPhone)
			err = DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", altPhone).Scan(&userID)
			if err == nil {
				return userID, altPhone, nil
			}
		}
	}
	return userID, phoneNumber, err
}

func normalizePhoneForRedis(phoneNumber string) string {
	redisPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phoneNumber)
	if strings.HasPrefix(redisPhone, "62") {
		redisPhone = "0" + redisPhone[2:]
	}
	return redisPhone
}

func checkExistingOTP(ctx context.Context, otpKey, phoneNumber string) (bool, string) {
	existingOTP, err := Redis.Get(ctx, otpKey).Result()
	if err == nil && existingOTP != "" {
		ttl, _ := Redis.TTL(ctx, otpKey).Result()
		slog.Info("Login OTP still active, reusing existing", "phone", phoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
		return true, "OTP sudah dikirim sebelumnya. Masih berlaku selama 1 jam. Silakan cek WhatsApp Anda."
	}
	return false, ""
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

	_, _, err := lookupUserByPhone(ctx, req.PhoneNumber)
	if err == pgx.ErrNoRows {
		slog.Warn("handlePhoneLogin: phone not found in DB", "phone", req.PhoneNumber)
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Phone number not registered"})
		return
	}
	if err != nil {
		slog.Error("DB query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	waProvider, _ := getPlatformWAProvider(ctx)
	otpKey := redisKeyPhoneLoginOTP + req.PhoneNumber

	if waProvider != "whatsmeow" {
		if exists, msg := checkExistingOTP(ctx, otpKey, req.PhoneNumber); exists {
			writeJSON(w, http.StatusOK, Response{Success: true, Message: msg})
			return
		}
	}

	redisPhone := normalizePhoneForRedis(req.PhoneNumber)
	err = Redis.Set(ctx, redisKeyAuthPending+redisPhone, redisPhone, 15*time.Minute).Err()
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

func normalizePhoneForOTP(phoneNumber string) string {
	otpPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phoneNumber)
	if strings.HasPrefix(otpPhone, "62") {
		otpPhone = "0" + otpPhone[2:]
	}
	return otpPhone
}

func extractOTPCode(storedOTP string) string {
	if idx := strings.Index(storedOTP, "|"); idx >= 0 {
		return storedOTP[:idx]
	}
	return storedOTP
}

func lookupUserByPhoneForLogin(ctx context.Context, phoneNumber string) (string, string, string, bool, error) {
	var userID, tenantID, role string
	var isDataVerified bool

	err := DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, is_phone_verified FROM users WHERE phone_number = $1",
		phoneNumber,
	).Scan(&userID, &tenantID, &role, &isDataVerified)

	if err == pgx.ErrNoRows {
		altPhone := phoneNumber
		if strings.HasPrefix(altPhone, "0") {
			altPhone = "62" + altPhone[1:]
		} else if strings.HasPrefix(altPhone, "62") {
			altPhone = "0" + altPhone[2:]
		}
		err = DB.QueryRow(ctx,
			"SELECT id, tenant_id, role, is_phone_verified FROM users WHERE phone_number = $1",
			altPhone,
		).Scan(&userID, &tenantID, &role, &isDataVerified)
	}

	return userID, tenantID, role, isDataVerified, err
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
	otpPhone := normalizePhoneForOTP(req.PhoneNumber)

	storedOTP, err := Redis.Get(ctx, redisKeyPhoneLoginOTP+otpPhone).Result()
	slog.Info("handleVerifyPhoneLogin: OTP check",
		"phone", req.PhoneNumber,
		"normalizedPhone", otpPhone,
		"key", redisKeyPhoneLoginOTP+otpPhone,
		"storedOTP", storedOTP,
		"reqOTP", req.OTP,
		"redisErr", err)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "OTP expired or invalid"})
		return
	}

	storedCode := extractOTPCode(storedOTP)
	if storedCode != req.OTP {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	userID, tenantID, role, isDataVerified, err := lookupUserByPhoneForLogin(ctx, req.PhoneNumber)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Nomor tidak terdaftar."})
		} else {
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		}
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
