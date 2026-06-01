package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId"`
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	OTP         string `json:"otp"`
}

type LoginRequest struct {
	Username    string `json:"username"`    // Can be username or email
	Password    string `json:"password"`
	PhoneNumber      string `json:"phoneNumber"` // WA-only login
	ExpectedTenantID string `json:"expectedTenantId"`
}

type PhoneLoginRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

type VerifyPhoneLoginRequest struct {
	PhoneNumber      string `json:"phoneNumber"`
	OTP              string `json:"otp"`
	ExpectedTenantID string `json:"expectedTenantId"`
}

type SuperAdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AddStaffRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Role        string `json:"role"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type TenantProfileUpdateRequest struct {
	TenantID        string `json:"tenant_id"`
	Name            string `json:"name"`
	BusinessName    string `json:"business_name"`
	WaNumber        string `json:"wa_number"`
	OwnerPhone      string `json:"owner_phone"`
	BusinessAddress string `json:"business_address"`
	BusinessType    string `json:"business_type"`
	Plan            string `json:"plan"`
	NewPassword     string `json:"new_password"`
	CustomDomain    string `json:"custom_domain"`
	Subdomain       string `json:"subdomain"`
}

type UpdateProfileRequest struct {
	Username        string `json:"username"`
	PhoneNumber     string `json:"phone_number"`
	BusinessName    string `json:"business_name"`
	BusinessAddress string `json:"business_address"`
	BusinessType    string `json:"business_type"`
	WaNumber        string `json:"wa_number"`
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
}

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load shared configuration
	cfg := config.LoadConfig(".env")
	port := "8001" // Hardcoded to 8001 to match docker-compose and api-gateway

	// Initialize DB & Redis
	if err := initDB(cfg); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	if err := initRedis(cfg); err != nil {
		slog.Error("Failed to initialize redis", "error", err)
		os.Exit(1)
	}
	defer Redis.Close()

	// Run database migrations automatically
	if err := runMigrations(DB); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	if err := ensureSuperadmin(); err != nil {
		slog.Error("Failed to seed superadmin", "error", err)
		os.Exit(1)
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(filepath.Join(uploadDir, "logos"), 0755)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", handleRegister)
	mux.HandleFunc("/verify-otp", handleVerifyOTP)
	mux.HandleFunc("/manual-register", handleManualRegister)
	mux.HandleFunc("/verify-data", handleVerifyData)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/phone-login", handlePhoneLogin)
	mux.HandleFunc("/verify-phone-login", handleVerifyPhoneLogin)
	mux.HandleFunc("/superadmin/login", handleSuperAdminLogin)
	mux.HandleFunc("/superadmin/verifier/status", handleVerifierStatus)
	mux.HandleFunc("/superadmin/verifier/qr", handleVerifierQR)
	mux.HandleFunc("/superadmin/verifier/disconnect", handleVerifierDisconnect)
	mux.HandleFunc("/superadmin/tenants", handleSuperadminTenants)
	mux.HandleFunc("/superadmin/tenants/profile", handleSuperadminTenantProfile)
	mux.HandleFunc("/superadmin/tenants/profile/logo", handleUploadTenantLogo)
	mux.HandleFunc("/refresh", handleRefresh)
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc("/validate", handleValidate)
	mux.HandleFunc("/add-staff", handleAddStaff)
	mux.HandleFunc("/profile", handleProfile)
	mux.HandleFunc("/profile/logo", handleUploadProfileLogo)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/forgot-password", handleForgotPassword)
	mux.HandleFunc("/reset-password", handleResetPassword)
	mux.HandleFunc("/public/tenant/resolve", handleTenantResolve)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(uploadDir))))

	slog.Info(fmt.Sprintf("🔑 Starting Auth Service in %s mode on port %s...", cfg.Env, port))
	serverAddress := fmt.Sprintf(":%s", port)
	
	// Wrap mux with CORS then Logging
	handler := corsMiddleware(loggingMiddleware(mux))
	
	if err := http.ListenAndServe(serverAddress, handler); err != nil {
		slog.Error("Failed to start Auth Service", "error", err)
		os.Exit(1)
	}
}

func ensureSchema() error {
	ctx := context.Background()
	migrations := []string{
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS wa_number VARCHAR(50)`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS wa_provider VARCHAR(20) NOT NULL DEFAULT 'internal'`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_type VARCHAR(50) DEFAULT 'umum'`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_name VARCHAR(255)`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_address TEXT`,
		`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS logo_url VARCHAR(255)`,
	}
	for _, m := range migrations {
		if _, err := DB.Exec(ctx, m); err != nil {
			return fmt.Errorf("ensureSchema: %w", err)
		}
	}
	return nil
}

func ensureSuperadmin() error {
	ctx := context.Background()
	var exists bool
	err := DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'superadmin')`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check superadmin: %w", err)
	}
	if exists {
		slog.Info("Superadmin user already exists, skipping seed")
		return nil
	}

	var tenantID string
	err = DB.QueryRow(ctx, `INSERT INTO tenants (name, plan) VALUES ('Superadmin', 'superadmin') RETURNING id`).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("create superadmin tenant: %w", err)
	}

	pw := os.Getenv("SUPERADMIN_PASSWORD")
	if pw == "" {
		pw = "superadmin123"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = DB.Exec(ctx,
		`INSERT INTO users (tenant_id, username, email, password_hash, role) VALUES ($1, 'superadmin', 'superadmin@internal', $2, 'superadmin')`,
		tenantID, string(hash))
	if err != nil {
		return fmt.Errorf("insert superadmin user: %w", err)
	}

	slog.Info("Superadmin user seeded successfully", "username", "superadmin")
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		slog.Info("Incoming request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
		slog.Info("Request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

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

	// Generate OTP
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	
	// Store in Redis (Valid for 5 mins)
	reqJSON, _ := json.Marshal(req)
	err := Redis.Set(context.Background(), "otp:"+req.PhoneNumber, string(reqJSON)+":"+otp, 5*time.Minute).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process registration"})
		return
	}

	// Determine Tenant for sending WA (if joining existing)
	senderTenant := req.TenantID
	if senderTenant == "" {
		senderTenant = "system" // Or a default tenant ID if creating new
	}

	// Send via WA Gateway
	go func() {
		target := formatPhoneToWAJID(req.PhoneNumber)
		
		payload := strings.NewReader("tenant_id=" + senderTenant + "&target=" + target + "&message=Kode OTP Campaign Manager Anda: " + otp)
		http.Post("http://wa-gateway:8202/api/wa/send", "application/x-www-form-urlencoded", payload)
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

	if req.OTP != storedOTP {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Incorrect OTP"})
		return
	}

	var regReq RegisterRequest
	json.Unmarshal([]byte(reqJSON), &regReq)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(regReq.Password), 12)
	tx, _ := DB.Begin(ctx)
	defer tx.Rollback(ctx)

	tenantID := regReq.TenantID
	if tenantID == "" {
		tx.QueryRow(ctx, "INSERT INTO tenants (name, plan) VALUES ($1, 'free') RETURNING id", regReq.Username+"'s Tenant").Scan(&tenantID)
	}

	role := regReq.Role
	if role == "" { role = "user_biasa" }

	// Generate a unique email if empty (for phone-only registration)
	email := regReq.Email
	if email == "" {
		email = regReq.PhoneNumber + "@wa.user"
	}

	var userID string
	err = tx.QueryRow(ctx, 
		"INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number, is_phone_verified) VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING id",
		tenantID, regReq.Username, email, string(hashedPassword), role, regReq.PhoneNumber,
	).Scan(&userID)
	
	if err != nil {
		writeJSON(w, http.StatusConflict, Response{Success: false, Message: "Phone number or username already exists"})
		return
	}

	tx.Commit(ctx)
	Redis.Del(ctx, "otp:"+req.PhoneNumber)

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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		Data: tokens,
	})
}

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
	var isDataVerified bool

	err := DB.QueryRow(ctx, 
		"SELECT id, tenant_id, role, password_hash, is_phone_verified FROM users WHERE username = $1 OR email = $1", 
		req.Username,
	).Scan(&userID, &tenantID, &rolePtr, &passwordHash, &isDataVerified)

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
		Data: tokens,
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

	// In real life, we should extract TenantID from JWT here.
	// For simplicity in this demo, we accept TenantID from header if available, or just mock it.
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Tenant ID required"})
		return
	}

	var req AddStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.Username == "" || req.Password == "" || req.Role == "" || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Missing required fields"})
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
		err := DB.QueryRow(ctx, `
			SELECT u.username, u.email, u.phone_number, u.role,
			       COALESCE(t.business_name, t.name), t.wa_number, t.logo_url, t.business_address, t.business_type, t.plan, t.name
			FROM users u
			JOIN tenants t ON t.id = u.tenant_id
			WHERE u.id = $1 AND u.tenant_id = $2
		`, claims.UserID, claims.TenantID).Scan(
			&username, &email, &phoneNumber, &role,
			&businessName, &waNumber, &logoURL, &businessAddress, &businessType, &plan, &tenantName,
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
				"username":         username,
				"email":            derefStr(email),
				"phone_number":     derefStr(phoneNumber),
				"role":             role,
				"business_name":    derefStr(businessName),
				"wa_number":        derefStr(waNumber),
				"logo_url":         derefStr(logoURL),
				"business_address": derefStr(businessAddress),
				"business_type":    derefStr(businessType),
				"plan":             derefStr(plan),
				"tenant_id":        claims.TenantID,
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
			DB.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", string(newHash), claims.UserID)
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

func handleUploadProfileLogo(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuth(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authentication required"})
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "File terlalu besar (max 2MB)"})
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Logo file is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Format file tidak didukung (PNG, JPG, WebP)"})
		return
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(filepath.Join(uploadDir, "logos"), 0755)

	outExt := ".png"
	if ext == ".jpg" || ext == ".jpeg" {
		outExt = ".jpg"
	} else if ext == ".webp" {
		outExt = ".webp"
	}

	filename := claims.TenantID + outExt
	outPath := filepath.Join(uploadDir, "logos", filename)
	dst, err := os.Create(outPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}

	logoURL := "/uploads/logos/" + filename
	ctx := context.Background()
	_, err = DB.Exec(ctx, `UPDATE tenants SET logo_url = $1, updated_at = NOW() WHERE id = $2`, logoURL, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update logo URL"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logo uploaded successfully", Data: map[string]interface{}{"logo_url": logoURL}})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Auth service is healthy"})
}

func handleTenantResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Domain is required"})
		return
	}

	ctx := context.Background()
	var tenantID, businessName string
	var logoURL *string

	err := DB.QueryRow(ctx,
		"SELECT id, business_name, logo_url FROM tenants WHERE custom_domain = $1 OR subdomain = $1",
		domain,
	).Scan(&tenantID, &businessName, &logoURL)

	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusOK, Response{Success: false, Message: "Tenant not found"})
		return
	} else if err != nil {
		slog.Error("Failed to resolve tenant", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	logo := ""
	if logoURL != nil {
		logo = *logoURL
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"tenant_id":     tenantID,
			"business_name": businessName,
			"logo_url":      logo,
		},
	})
}

func handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid payload"})
		return
	}

	ctx := context.Background()
	var userID string
	err := DB.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Don't leak whether email exists
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "If the email is registered, a reset token will be sent."})
		return
	} else if err != nil {
		slog.Error("DB error in forgot password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	// Generate a mock random token
	tokenStr := fmt.Sprintf("%x", time.Now().UnixNano())
	
	expiresAt := time.Now().Add(1 * time.Hour)
	_, err = DB.Exec(ctx, "INSERT INTO password_resets (email, token, expires_at) VALUES ($1, $2, $3)", req.Email, tokenStr, expiresAt)
	if err != nil {
		slog.Error("Failed to insert reset token", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	// In a real application, send via SMTP here. For now, print to log.
	slog.Info("🔑 PASSWORD RESET TOKEN GENERATED (Simulating Email)", "email", req.Email, "token", tokenStr)

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "If the email is registered, a reset token will be sent."})
}

func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid payload"})
		return
	}

	if req.NewPassword == "" || req.Token == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Token and new password are required"})
		return
	}

	ctx := context.Background()
	var email string
	err := DB.QueryRow(ctx, "SELECT email FROM password_resets WHERE token = $1 AND expires_at > NOW()", req.Token).Scan(&email)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to hash password"})
		return
	}

	_, err = DB.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE email = $2", string(hashedPassword), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to reset password"})
		return
	}

	// Consume token
	DB.Exec(ctx, "DELETE FROM password_resets WHERE email = $1", email)

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Password has been successfully reset."})
}

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

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	err = Redis.Set(ctx, "phone-login-otp:"+req.PhoneNumber, otp, 5*time.Minute).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process login"})
		return
	}

	go func() {
		target := formatPhoneToWAJID(req.PhoneNumber)
		resp, err := http.Post("http://wa-gateway:8202/api/wa/send",
			"application/x-www-form-urlencoded",
			strings.NewReader("tenant_id=verifier&target="+target+"&message=Kode OTP login WCH Anda: "+otp))
		if err != nil {
			slog.Error("Failed to send login OTP via WA", "error", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
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
	Redis.Del(ctx, "phone-login-otp:"+req.PhoneNumber)

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

func handleSuperAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req SuperAdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()
	var userID, tenantID, passwordHash, role string
	var isDataVerified bool
	err := DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, password_hash, is_phone_verified FROM users WHERE username = $1 AND role = 'superadmin'",
		req.Username,
	).Scan(&userID, &tenantID, &role, &passwordHash, &isDataVerified)

	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials or not a super admin"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
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

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Superadmin login successful",
		Data: map[string]interface{}{
			"accessToken":  tokens.AccessToken,
			"refreshToken": tokens.RefreshToken,
			"tenantId":     tenantID,
			"role":         role,
		},
	})
}

func requireSuperAdmin(r *http.Request) (*Claims, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || claims.Role != "superadmin" {
		return nil, false
	}
	return claims, true
}

func handleVerifierStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8202/api/wa/status?tenant_id=verifier")
	if err != nil {
		slog.Warn("WA Gateway not reachable", "error", err)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status": "unavailable",
			"message": "WA Gateway tidak berjalan. Verifier WhatsApp tidak tersedia.",
		}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("WA Gateway returned non-200", "status", resp.StatusCode)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status": "unavailable",
			"message": "WA Gateway tidak merespon dengan benar.",
		}})
		return
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		slog.Warn("Failed to decode WA Gateway response", "error", err)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status": "unavailable",
			"message": "Gagal membaca status WA Gateway.",
		}})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: status})
}

func handleVerifierQR(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://localhost:8202/api/wa/qr?tenant_id=verifier")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to get QR code"})
		return
	}
	defer resp.Body.Close()

	var qrData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&qrData)

	writeJSON(w, http.StatusOK, Response{Success: true, Data: qrData})
}

func handleVerifierDisconnect(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	// msisdn parameter is no longer required as we logout using tenant_id=verifier

	// Use WA Gateway's logout endpoint
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, "http://localhost:8202/api/wa/logout?tenant_id=verifier", nil)
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to disconnect verifier"})
		return
	}
	defer resp.Body.Close()

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Verifier disconnected. Scan QR to reconnect."})
}

func handleSuperadminTenants(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		rows, err := DB.Query(ctx, `
			SELECT t.id, t.name, t.plan, t.created_at,
				COALESCE(u.username, '') as owner_username,
				COALESCE(u.phone_number, '') as owner_phone,
				(SELECT COUNT(*) FROM users WHERE tenant_id = t.id) as user_count
			FROM tenants t
			LEFT JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
			ORDER BY t.created_at DESC
		`)
		if err != nil {
			slog.Error("Failed to fetch tenants", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to fetch tenants"})
			return
		}
		defer rows.Close()

		var tenants []map[string]interface{}
		for rows.Next() {
			var id, name, plan, ownerUsername, ownerPhone string
			var userCount int
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &plan, &createdAt, &ownerUsername, &ownerPhone, &userCount); err != nil {
				continue
			}
			tenants = append(tenants, map[string]interface{}{
				"id":             id,
				"name":           name,
				"plan":           plan,
				"owner_username": ownerUsername,
				"owner_phone":    ownerPhone,
				"user_count":     userCount,
				"created_at":     createdAt,
			})
		}

		if tenants == nil {
			tenants = []map[string]interface{}{}
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Data: tenants})

	case http.MethodDelete:
		tenantID := r.URL.Query().Get("id")
		if tenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
			return
		}

		tx, err := DB.Begin(ctx)
		if err != nil {
			slog.Error("Failed to begin transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "DELETE FROM journal_lines WHERE entry_id IN (SELECT id FROM journal_entries WHERE tenant_id = $1)", tenantID)
		if err != nil {
			slog.Error("Failed to delete journal lines", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		_, err = tx.Exec(ctx, "DELETE FROM journal_entries WHERE tenant_id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete journal entries", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		_, err = tx.Exec(ctx, "DELETE FROM chart_of_accounts WHERE tenant_id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete chart of accounts", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		tag, err := tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete tenant", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("Failed to commit transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Tenant berhasil dihapus"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
	}
}

func handleSuperadminTenantProfile(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("id")
		if tenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
			return
		}

		var id, name, plan, ownerUsername, ownerID string
		var businessName, waNumber, logoURL, businessAddress, businessType, ownerPhone, customDomain, subdomain *string
		if err := DB.QueryRow(ctx, `
			SELECT t.id, t.name, t.plan, t.business_name, t.wa_number, t.logo_url, t.business_address, t.business_type,
			       t.custom_domain, t.subdomain, COALESCE(u.username, '') as owner_username, COALESCE(u.id::text, '') as owner_id, u.phone_number as owner_phone
			FROM tenants t
			LEFT JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
			WHERE t.id = $1
		`, tenantID).Scan(&id, &name, &plan, &businessName, &waNumber, &logoURL, &businessAddress, &businessType, &customDomain, &subdomain, &ownerUsername, &ownerID, &ownerPhone); err != nil {
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
				return
			}
			slog.Error("Failed to fetch tenant profile", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
			return
		}

		data := map[string]interface{}{
			"id":               id,
			"name":             name,
			"plan":             plan,
			"business_name":    derefStr(businessName),
			"wa_number":        derefStr(waNumber),
			"owner_phone":      derefStr(ownerPhone),
			"logo_url":         derefStr(logoURL),
			"business_address": derefStr(businessAddress),
			"business_type":    derefStr(businessType),
			"custom_domain":    derefStr(customDomain),
			"subdomain":        derefStr(subdomain),
			"owner_username":   ownerUsername,
			"owner_id":         ownerID,
		}
		writeJSON(w, http.StatusOK, Response{Success: true, Data: data})

	case http.MethodPut:
		var req TenantProfileUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
			return
		}
		slog.Info("Received update tenant profile request", "payload", req)
		if req.TenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "tenant_id is required"})
			return
		}

		tag, err := DB.Exec(ctx, `
			UPDATE tenants SET name=$1, business_name=$2, wa_number=$3, business_address=$4, business_type=$5, plan=$6, custom_domain=NULLIF($7, ''), subdomain=NULLIF($8, ''), updated_at=NOW()
			WHERE id=$9
		`, req.Name, req.BusinessName, req.WaNumber, req.BusinessAddress, req.BusinessType, req.Plan, req.CustomDomain, req.Subdomain, req.TenantID)
		if err != nil {
			slog.Error("Failed to update tenant profile", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update profile"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
			return
		}

		if req.NewPassword != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
			DB.Exec(ctx, `UPDATE users SET password_hash=$1 WHERE tenant_id=$2 AND role='owner'`, string(hash), req.TenantID)
		}

		if req.OwnerPhone != "" {
			_, err := DB.Exec(ctx, `UPDATE users SET phone_number=$1 WHERE tenant_id=$2 AND role='owner'`, req.OwnerPhone, req.TenantID)
			if err != nil {
				slog.Error("Failed to update owner phone", "error", err, "phone", req.OwnerPhone, "tenant", req.TenantID)
				writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal memperbarui nomor login owner. Pastikan nomor belum digunakan."})
				return
			}
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Profil tenant berhasil diperbarui"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
	}
}

func handleUploadTenantLogo(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	tenantID := r.URL.Query().Get("id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "File terlalu besar (max 2MB)"})
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Logo file is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Format file tidak didukung (PNG, JPG, WebP)"})
		return
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(filepath.Join(uploadDir, "logos"), 0755)

	outExt := ".png"
	if ext == ".jpg" || ext == ".jpeg" {
		outExt = ".jpg"
	} else if ext == ".webp" {
		outExt = ".webp"
	}

	filename := tenantID + outExt
	outPath := filepath.Join(uploadDir, "logos", filename)
	dst, err := os.Create(outPath)
	if err != nil {
		slog.Error("Failed to create logo file", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("Failed to write logo file", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}

	logoURL := "/uploads/logos/" + filename
	ctx := context.Background()
	_, err = DB.Exec(ctx, `UPDATE tenants SET logo_url=$1, updated_at=NOW() WHERE id=$2`, logoURL, tenantID)
	if err != nil {
		slog.Error("Failed to update logo_url", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update logo URL"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logo berhasil diupload", Data: map[string]interface{}{"logo_url": logoURL}})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatPhoneToWAJID(phone string) string {
	if strings.HasSuffix(phone, "@s.whatsapp.net") {
		return phone
	}
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	} else if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}
	return phone + "@s.whatsapp.net"
}
