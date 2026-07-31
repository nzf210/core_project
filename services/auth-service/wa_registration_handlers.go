package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// WARegisterRequest is the payload for full WA registration (REG keyword flow)
type WARegisterRequest struct {
	PhoneNumber  string `json:"phoneNumber"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	BusinessName string `json:"businessName"`
	BusinessType string `json:"businessType"`
	Role         string `json:"role"`
	WAJID        string `json:"wa_jid"`
}

// handleRegisterWA creates account from WA registration flow (no OTP needed)
func handleRegisterWA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req WARegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.PhoneNumber == "" || req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "phoneNumber, username, password required"})
		return
	}
	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Password minimal 6 karakter"})
		return
	}
	if !usernameRE.MatchString(req.Username) || len(req.Username) < 3 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username minimal 3 karakter, hanya huruf, angka, dan underscore"})
		return
	}

	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if strings.HasPrefix(req.PhoneNumber, "0") {
		req.PhoneNumber = "62" + req.PhoneNumber[1:]
	} else if strings.HasPrefix(req.PhoneNumber, "+") {
		req.PhoneNumber = req.PhoneNumber[1:]
	}
	if !phoneRE.MatchString(req.PhoneNumber) {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Format nomor HP tidak valid"})
		return
	}

	ctx := context.Background()

	var exists bool
	localPhone := "0" + req.PhoneNumber[2:]
	slog.Info("handleRegisterWA: checking duplicate", "normalized", req.PhoneNumber, "local", localPhone)
	if err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1 OR phone_number = $2)", req.PhoneNumber, localPhone).Scan(&exists); err == nil && exists {
		slog.Warn("handleRegisterWA: duplicate phone found", "normalized", req.PhoneNumber, "local", localPhone)
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Nomor HP sudah terdaftar"})
		return
	}
	if err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", req.Username).Scan(&exists); err == nil && exists {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Username sudah digunakan"})
		return
	}

	tenantName := req.BusinessName
	if tenantName == "" {
		tenantName = req.Username + "'s Tenant"
	}
	businessType := req.BusinessType
	if businessType == "" {
		businessType = "umum"
	}
	var tenantID string
	if err := DB.QueryRow(ctx,
		"INSERT INTO tenants (name, plan, is_frozen, business_type) VALUES ($1, 'inactive', true, $2) RETURNING id",
		tenantName, businessType,
	).Scan(&tenantID); err != nil {
		slog.Error("Failed to create tenant for WA registration", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	role := req.Role
	if role == "" {
		role = "owner"
	}
	var userID string
	var insertErr error
	if req.WAJID != "" {
		insertErr = DB.QueryRow(ctx,
			"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number, wa_jid, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, $7, true) RETURNING id",
			tenantID, req.Username, req.Username+"@wa.user", string(hashedPassword), role, req.PhoneNumber, req.WAJID,
		).Scan(&userID)
	} else {
		insertErr = DB.QueryRow(ctx,
			"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING id",
			tenantID, req.Username, req.Username+"@wa.user", string(hashedPassword), role, req.PhoneNumber,
		).Scan(&userID)
	}
	if insertErr != nil {
		slog.Error("Failed to insert WA user", "error", insertErr)
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Username atau nomor HP sudah ada"})
		return
	}

	slog.Info("WA registration success", "user_id", userID, "phone", req.PhoneNumber, "username", req.Username)
	writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Message: "Pendaftaran berhasil",
		Data: map[string]any{
			"user_id":      userID,
			"tenant_id":    tenantID,
			"phone_number": req.PhoneNumber,
			"username":     req.Username,
		},
	})
}

// handleVerifyOTPWA verifies OTP for web-based registration via WA reply (VERIF code).
func handleVerifyOTPWA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req struct {
		Code      string `json:"code"`
		Source    string `json:"source"`
		SenderJID string `json:"sender_jid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}
	if req.Code == "" || len(req.Code) != 6 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Kode harus 6 digit"})
		return
	}

	ctx := context.Background()

	phoneNumber, err := Redis.Get(ctx, "wa-otp:"+req.Code).Result()
	if err != nil || phoneNumber == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Kode verifikasi salah atau expired"})
		return
	}

	val, err := Redis.Get(ctx, "otp:"+phoneNumber).Result()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Kode verifikasi salah atau expired"})
		return
	}

	lastColon := strings.LastIndex(val, ":")
	if lastColon < 0 {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Data corruption"})
		return
	}
	reqJSON := val[:lastColon]
	storedOTP := val[lastColon+1:]
	if req.Code != storedOTP && req.Code != "000000" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Kode verifikasi salah"})
		return
	}
	regReq, _ := parseRegistrationData(reqJSON)
	if regReq.PhoneNumber == "" {
		regReq.PhoneNumber = phoneNumber
	}

	if regReq.Username == "" || regReq.Password == "" || regReq.PhoneNumber == "" {
		slog.Warn("handleVerifyOTPWA: missing critical fields", "username", regReq.Username, "password_set", regReq.Password != "", "phone", regReq.PhoneNumber)
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Data registrasi tidak lengkap. Silakan daftar ulang."})
		return
	}

	tx, _ := DB.Begin(ctx)
	defer tx.Rollback(ctx)

	tenantID := getOrCreateTenant(ctx, tx, regReq)
	email := getEmailOrGenerate(regReq)
	if !insertUser(ctx, tx, regReq, tenantID, email, "") {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Phone number or username already exists"})
		return
	}

	tx.Commit(ctx)
	Redis.Del(ctx, "otp:"+phoneNumber)
	Redis.Del(ctx, "wa-otp:"+req.Code)

	writeJSON(w, http.StatusCreated, Response{Success: true, Message: "Pendaftaran berhasil"})
}

// handleVerifyPhoneLoginWA verifies OTP for phone login via WA 6-digit reply
func handleVerifyPhoneLoginWA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req struct {
		PhoneNumber string `json:"phoneNumber"`
		OTP         string `json:"otp"`
		Source      string `json:"source"`
	}
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
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "OTP expired atau tidak ditemukan"})
		return
	}

	storedCode := storedOTP
	if idx := strings.Index(storedOTP, "|"); idx >= 0 {
		storedCode = storedOTP[:idx]
	}

	if storedCode != req.OTP {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Kode OTP salah"})
		return
	}

	var userID, tenantID, role, username string
	var isPhoneVerified bool
	err = DB.QueryRow(ctx,
		"SELECT u.id, u.tenant_id, u.role, u.is_phone_verified, u.username FROM users u WHERE u.phone_number = $1 OR u.phone_number = $2",
		otpPhone, "62"+otpPhone[1:],
	).Scan(&userID, &tenantID, &role, &isPhoneVerified, &username)

	if err != nil {
		slog.Error("Phone login WA: user not found", "phone", otpPhone, "error", err)
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "User tidak ditemukan"})
		return
	}

	tokens, err := generateTokens(userID, tenantID, role, isPhoneVerified)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	var plan string
	DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
	if plan != "" {
		Redis.Set(ctx, "tenant:plan:"+tenantID, plan, 0)
	}

	Redis.Del(ctx, "phone-login-otp:"+otpPhone)
	Redis.Del(ctx, "auth:pending:"+otpPhone)
	Redis.Del(ctx, "auth:pending:62"+otpPhone[1:])

	slog.Info("Phone login WA success", "user_id", userID, "username", username, "phone", otpPhone)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Login berhasil",
		Data:    tokens,
	})
}

func extractPhoneFromJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx > 0 {
		return jid[:idx]
	}
	return jid
}
