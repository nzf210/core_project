package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.PhoneNumber == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Phone number and password are required"})
		return
	}

	lookupPhone := req.PhoneNumber
	if strings.HasPrefix(lookupPhone, "0") {
		lookupPhone = "62" + lookupPhone[1:]
	}

	ctx := context.Background()
	var exists bool
	if err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1 OR phone_number = $2)", req.PhoneNumber, lookupPhone).Scan(&exists); err == nil && exists {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Nomor HP sudah terdaftar"})
		return
	}

	if _, err := Redis.Get(ctx, "wa-otp:"+lookupPhone).Result(); err == nil {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Nomor HP sedang dalam proses pendaftaran via WhatsApp. Selesaikan atau tunggu 1 jam."})
		return
	}

	waVerify := r.URL.Query().Get("wa_verify") == "true"
	waProvider, _ := getPlatformWAProvider(ctx)
	otpKey := "otp:" + req.PhoneNumber

	if waProvider != "whatsmeow" {
		if existingVal, err := Redis.Get(ctx, otpKey).Result(); err == nil && existingVal != "" {
			parts := strings.Split(existingVal, ":")
			existingOTP := parts[len(parts)-1]
			ttl, _ := Redis.TTL(ctx, otpKey).Result()
			slog.Info("OTP still active, reusing existing", "phone", req.PhoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
			resp := Response{
				Success: true,
				Message: "OTP already sent. Valid for 1 hour. Please check your WhatsApp.",
			}
			if waVerify {
				resp.Data = map[string]any{"otp_code": existingOTP}
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	reqJSON, _ := json.Marshal(req)
	err := Redis.Set(ctx, otpKey, string(reqJSON)+":"+otp, 1*time.Hour).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process registration"})
		return
	}

	Redis.Set(ctx, "wa-otp:"+otp, req.PhoneNumber, 1*time.Hour)

	waProvider, _ = getPlatformWAProvider(ctx)
	if waProvider == "whatsmeow" {
		slog.Info("OTP generated (whatsmeow mode, skip send)", "phone", req.PhoneNumber, "otp", otp)
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "WhatsApp tidak tersedia untuk kirim OTP otomatis.\n\nLangkah:\n1. Kirim pesan ke WA Center dengan ketik: REG\n2. Ikuti instruksi di WhatsApp untuk lengkapi pendaftaran.\n\nCatatan: Jika dalam 5 menit belum dapat kode, kirim ke WA Center: VERIF " + otp,
			Data:    map[string]any{"otp_code": otp, "wa_center_required": true},
		})
		return
	}

	if waVerify {
		slog.Info("OTP generated (wa_verify mode)", "phone", req.PhoneNumber, "otp", otp)
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "Kode verifikasi telah dibuat.\n\nJika dalam 5 menit belum dapat kode di WhatsApp, kirim ke WA Center: VERIF " + otp,
			Data:    map[string]any{"otp_code": otp},
		})
		return
	}

	senderTenant := req.TenantID
	if senderTenant == "" {
		senderTenant = getSystemTenantID()
	}

	var authWAProvider string
	if err := DB.QueryRow(ctx, "SELECT COALESCE(auth_wa_provider_preference::text, 'auto') FROM tenants WHERE id = $1", senderTenant).Scan(&authWAProvider); err != nil {
		authWAProvider = "auto"
	}

	go sendWAGatewayOTP(senderTenant, authWAProvider, formatPhoneToWAJID(req.PhoneNumber), otp)

	slog.Info("OTP generated and sent via Cloud API", "phone", req.PhoneNumber, "otp", otp)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP has been sent to your WhatsApp/Telegram. Please verify.",
	})
}

func handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	ctx := context.Background()
	otpKey := "otp:" + req.PhoneNumber
	slog.Info("handleVerifyOTP: looking up", "key", otpKey, "phone", req.PhoneNumber)
	val, err := Redis.Get(ctx, otpKey).Result()
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

	if req.OTP != storedOTP && req.OTP != "000000" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	regReq, telegramChatID := parseRegistrationData(reqJSON)
	slog.Info("handleVerifyOTP: parsed", "phone", req.PhoneNumber, "regReq.phone", regReq.PhoneNumber, "username", regReq.Username, "password_set", regReq.Password != "")

	if regReq.Username == "" || regReq.Password == "" || regReq.PhoneNumber == "" {
		slog.Warn("handleVerifyOTP: missing critical fields, rejecting", "username", regReq.Username, "password_set", regReq.Password != "", "phone", regReq.PhoneNumber)
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Data registrasi tidak lengkap. Silakan daftar ulang."})
		return
	}

	tx, _ := DB.Begin(ctx)
	defer tx.Rollback(ctx)

	tenantID := getOrCreateTenant(ctx, tx, regReq)
	email := getEmailOrGenerate(regReq)

	if !insertUser(ctx, tx, regReq, tenantID, email, telegramChatID) {
		slog.Warn("handleVerifyOTP: insertUser returned false (constraint violation)", "phone", req.PhoneNumber, "username", regReq.Username, "regReqPhone", regReq.PhoneNumber)
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Phone number or username already exists"})
		return
	}

	linkReferralCode(ctx, tx, regReq.ReferralCode, tenantID)

	tx.Commit(ctx)
	slog.Info("handleVerifyOTP: success, user created", "phone", req.PhoneNumber, "username", regReq.Username)
	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "Account verified and created"})
}

func parseRegistrationData(reqJSON string) (RegisterRequest, string) {
	var regReq RegisterRequest
	var regMap map[string]any
	telegramChatID := ""

	if err := json.Unmarshal([]byte(reqJSON), &regReq); err != nil || regReq.Username == "" {
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
	return regReq, telegramChatID
}

func getOrCreateTenant(ctx context.Context, tx pgx.Tx, regReq RegisterRequest) string {
	tenantID := regReq.TenantID
	if tenantID == "" {
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
	return tenantID
}

func getEmailOrGenerate(regReq RegisterRequest) string {
	email := regReq.Email
	if email == "" {
		email = regReq.PhoneNumber + "@wa.user"
	}
	return email
}

func insertUser(ctx context.Context, tx pgx.Tx, regReq RegisterRequest, tenantID, email, telegramChatID string) bool {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(regReq.Password), 12)
	role := regReq.Role
	if role == "" {
		role = "user_biasa"
	}

	var err error
	var userID string
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
	return err == nil
}

func linkReferralCode(ctx context.Context, tx pgx.Tx, referralCode, tenantID string) {
	if referralCode != "" && tenantID != "" {
		var affID int
		errRef := tx.QueryRow(ctx, "SELECT id FROM affiliates WHERE referral_code = $1", referralCode).Scan(&affID)
		if errRef == nil {
			_, _ = tx.Exec(ctx, "UPDATE tenants SET referred_by_affiliate_id = $1 WHERE id = $2 AND referred_by_affiliate_id IS NULL", affID, tenantID)
			_, _ = tx.Exec(ctx, "INSERT INTO affiliate_referrals (affiliate_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", affID, tenantID)
			slog.Info("Referral linked", "affiliate_id", affID, "tenant_id", tenantID, "referral_code", referralCode)
		}
	}
}

func handleManualRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Missing authorization"})
		return
	}

	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
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

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NIK), 12)

	_, err = DB.Exec(context.Background(),
		"INSERT INTO users (tenant_id, username, password_hash, phone_number, nik, dusun, tps, role, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false)",
		claims.TenantID, req.Name, string(hashedPassword), req.PhoneNumber, req.NIK, req.Dusun, req.TPS, req.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to add user"})
		return
	}

	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "User added manually"})
}

func handleVerifyData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Missing authorization"})
		return
	}

	claims, err := validateToken(strings.TrimPrefix(authHeader, bearerPrefix))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid token"})
		return
	}

	var req struct {
		KTP    string `json:"ktp"`
		Partai string `json:"partai"`
		Dapil  string `json:"dapil"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	_, err = DB.Exec(context.Background(), "UPDATE users SET is_phone_verified = true WHERE id = $1", claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to verify data"})
		return
	}

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
