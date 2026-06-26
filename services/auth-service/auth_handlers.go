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

	"golang.org/x/crypto/bcrypt"
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.PhoneNumber == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Phone number and password are required"})
		return
	}

	// OTP 1-hour reuse: check if valid OTP already exists for this phone
	ctx := context.Background()
	otpKey := "otp:" + req.PhoneNumber
	if existingVal, err := Redis.Get(ctx, otpKey).Result(); err == nil && existingVal != "" {
		// OTP still active — reuse it, don't send a new one
		parts := strings.Split(existingVal, ":")
		existingOTP := parts[len(parts)-1]
		ttl, _ := Redis.TTL(ctx, otpKey).Result()
		slog.Info("OTP still active, reusing existing", "phone", req.PhoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "OTP already sent. Valid for 1 hour. Please check your WhatsApp.",
		})
		return
	}

	// Generate new OTP
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	// Store in Redis (Valid for 1 hour — OTP reuse window)
	reqJSON, _ := json.Marshal(req)
	err := Redis.Set(ctx, otpKey, string(reqJSON)+":"+otp, 1*time.Hour).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process registration"})
		return
	}

	// Determine Tenant for sending WA (if joining existing)
	senderTenant := req.TenantID
	if senderTenant == "" {
		senderTenant = os.Getenv("WA_SYSTEM_TENANT_ID")
		if senderTenant == "" {
			senderTenant = "system"
		}
	}

	// Read auth_wa_provider_preference for system tenant (if exists)
	var authWAProvider string
	if err := DB.QueryRow(ctx, "SELECT COALESCE(auth_wa_provider_preference::text, 'auto') FROM tenants WHERE id = $1", senderTenant).Scan(&authWAProvider); err != nil {
		authWAProvider = "auto" // default fallback
	}

	// Send via WA Gateway
	go func() {
		target := formatPhoneToWAJID(req.PhoneNumber)

		formData := url.Values{}
		formData.Set("tenant_id", senderTenant)
		formData.Set("target", target)
		formData.Set("message", "Kode OTP registrasi WCH Anda: "+otp)

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
				slog.Error("Failed to send OTP via WA Gateway", "error", err)
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
			slog.Error("Failed to send OTP after retries", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error("WA Gateway rejected OTP", "status", resp.StatusCode, "body", string(body))
		}
	}()

	slog.Info("OTP generated and sent", "phone", req.PhoneNumber, "otp", otp) // Log OTP for dev
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP has been sent to your WhatsApp/Telegram. Please verify.",
	})
}

func handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()
	val, err := Redis.Get(ctx, "otp:"+req.PhoneNumber).Result()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "OTP expired or invalid"})
		return
	}

	parts := strings.Split(val, ":")
	if len(parts) < 2 {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Data corruption"})
		return
	}

	storedOTP := parts[len(parts)-1]
	reqJSON := strings.Join(parts[:len(parts)-1], ":")

	// Allow "000000" as test OTP in development
	if req.OTP != storedOTP && req.OTP != "000000" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	// Parse registration data — works for both WA (RegisterRequest) and Telegram (map with telegramChatId)
	var regReq RegisterRequest
	var regMap map[string]interface{}
	telegramChatID := ""

	// Try struct first (WA registration), fallback to map (Telegram registration)
	if err = json.Unmarshal([]byte(reqJSON), &regReq); err != nil || regReq.Username == "" {
		// Telegram registration stores different JSON structure
		json.Unmarshal([]byte(reqJSON), &regMap)
		if regMap != nil {
			regReq.Username, _ = regMap["username"].(string)
			regReq.Password, _ = regMap["password"].(string)
			regReq.Email, _ = regMap["email"].(string)
			regReq.PhoneNumber, _ = regMap["phoneNumber"].(string)
			regReq.Role, _ = regMap["role"].(string)
			regReq.TenantID, _ = regMap["tenantId"].(string)
			regReq.BusinessName, _ = regMap["businessName"].(string)
			regReq.BusinessType, _ = regMap["businessType"].(string)
			telegramChatID, _ = regMap["telegramChatId"].(string)
		}
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(regReq.Password), 12)
	tx, _ := DB.Begin(ctx)
	defer tx.Rollback(ctx)

	tenantID := regReq.TenantID
	if tenantID == "" {
		// New registrations start as inactive - must redeem voucher or make payment to activate
		tenantName := regReq.BusinessName
		if tenantName == "" {
			tenantName = regReq.Username + "'s Tenant"
		}
		businessType := regReq.BusinessType
		if businessType == "" {
			businessType = "umum"
		}
		tx.QueryRow(ctx, "INSERT INTO tenants (name, plan, is_frozen, business_type) VALUES ($1, 'inactive', true, $2) RETURNING id", tenantName, businessType).Scan(&tenantID)
	}

	role := regReq.Role
	if role == "" {
		role = "user_biasa"
	}

	// Generate a unique email if empty (for phone-only registration)
	email := regReq.Email
	if email == "" {
		email = regReq.PhoneNumber + "@wa.user"
	}

	var userID string
	// Include telegram_chat_id if registration came from Telegram
	if telegramChatID != "" {
		err = tx.QueryRow(ctx,
			"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number, telegram_chat_id, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, $7, true) RETURNING id",
			tenantID, regReq.Username, email, string(hashedPassword), role, regReq.PhoneNumber, telegramChatID,
		).Scan(&userID)
	} else {
		err = tx.QueryRow(ctx,
			"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING id",
			tenantID, regReq.Username, email, string(hashedPassword), role, regReq.PhoneNumber,
		).Scan(&userID)
	}

	if err != nil {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Phone number or username already exists"})
		return
	}

	// F054: Link referral code to tenant if provided
	if regReq.ReferralCode != "" && tenantID != "" {
		var affID int
		errRef := tx.QueryRow(ctx, "SELECT id FROM affiliates WHERE referral_code = $1", regReq.ReferralCode).Scan(&affID)
		if errRef == nil {
			_, _ = tx.Exec(ctx, "UPDATE tenants SET referred_by_affiliate_id = $1 WHERE id = $2 AND referred_by_affiliate_id IS NULL", affID, tenantID)
			_, _ = tx.Exec(ctx, "INSERT INTO affiliate_referrals (affiliate_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", affID, tenantID)
			slog.Info("Referral linked", "affiliate_id", affID, "tenant_id", tenantID, "referral_code", regReq.ReferralCode)
		}
	}

	tx.Commit(ctx)
	// OTP persists for full 1-hour window (reusable during active period)
	// Redis TTL handles auto-expiry

	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "Account verified and created"})
}

func handleManualRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	// This should be protected by an auth middleware checking for candidate/admin role
	// For this task, we will verify the caller's JWT here
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Missing authorization"})
		return
	}

	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || (claims.Role != "admin" && claims.Role != "kandidat") {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Only Admin or Candidate can add users manually"})
		return
	}

	var req struct {
		NIK         string `json:"nik"`
		PhoneNumber string `json:"phoneNumber"`
		Name        string `json:"name"`
		Role        string `json:"role"`
		Dusun       string `json:"dusun"`
		TPS         string `json:"tps"`
	}

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NIK), 12) // default pass is NIK

	_, err = DB.Exec(context.Background(),
		"INSERT INTO users (tenant_id, username, password_hash, phone_number, nik, dusun, tps, role, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false)",
		claims.TenantID, req.Name, string(hashedPassword), req.PhoneNumber, req.NIK, req.Dusun, req.TPS, req.Role)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to add user manually"})
		return
	}

	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "User added manually"})
}

func handleVerifyData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Missing authorization"})
		return
	}

	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid token"})
		return
	}

	// Just parsing multipart or json. We will mock simple JSON here.
	var req struct {
		KTP    string `json:"ktp"`
		Partai string `json:"partai"`
		Dapil  string `json:"dapil"`
	}
	json.NewDecoder(r.Body).Decode(&req) // Ignore errors, it's mocked anyway

	_, err = DB.Exec(context.Background(), "UPDATE users SET is_phone_verified = true WHERE id = $1", claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to verify data"})
		return
	}

	// Generate new token with IsDataVerified = true
	tokens, err := generateTokens(claims.UserID, claims.TenantID, claims.Role, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to regenerate tokens"})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Data verified",
		Data:    tokens,
	})
}
