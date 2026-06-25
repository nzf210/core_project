package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"core_project/shared/sdk/config"

	"golang.org/x/crypto/bcrypt"
)

var (
	usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emailRE    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	phoneRE    = regexp.MustCompile(`^62[0-9]{6,15}$`)
)

var telegramBotToken string

type RegisterRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phoneNumber"`
	Role         string `json:"role"`
	TenantID     string `json:"tenantId"`
	BusinessName string `json:"businessName"`
	BusinessType string `json:"businessType"`
	ReferralCode string `json:"referralCode"`
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	OTP         string `json:"otp"`
}

type LoginRequest struct {
	Username         string `json:"username"` // Can be username or email
	Password         string `json:"password"`
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

// Telegram auth request types
type TelegramRegisterRequest struct {
	TelegramChatID string `json:"telegramChatId"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Email          string `json:"email"`
	PhoneNumber    string `json:"phoneNumber"`
	Role           string `json:"role"`
	TenantID       string `json:"tenantId"`
	BusinessName   string `json:"businessName"`
	BusinessType   string `json:"businessType"`
}

type TelegramLoginRequest struct {
	TelegramChatID string `json:"telegramChatId"`
	PhoneNumber    string `json:"phoneNumber"`
}

type SuperAdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ResetPasswordDefaultRequest struct {
	Username    string `json:"username"`
	PhoneNumber string `json:"phoneNumber"`
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
	TenantID         string `json:"tenant_id"`
	Name             string `json:"name"`
	BusinessName     string `json:"business_name"`
	WaNumber         string `json:"wa_number"`
	OwnerPhone       string `json:"owner_phone"`
	BusinessAddress  string `json:"business_address"`
	BusinessType     string `json:"business_type"`
	Plan             string `json:"plan"`
	NewPassword      string `json:"new_password"`
	CustomDomain     string `json:"custom_domain"`
	Subdomain        string `json:"subdomain"`
	XenditMerchantID string `json:"xendit_merchant_id"`
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
	mux.HandleFunc("/superadmin/tenants/", handleImpersonate) // F058: impersonate tenant owner
	mux.HandleFunc("/refresh", handleRefresh)
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc("/validate", handleValidate)
	mux.HandleFunc("/add-staff", handleAddStaff)
	mux.HandleFunc("/staff", handleStaffList)
	mux.HandleFunc("/staff/update", handleStaffUpdate)
	mux.HandleFunc("/staff/delete", handleStaffDelete)
	mux.HandleFunc("/profile", handleProfile)
	mux.HandleFunc("/profile/logo", handleUploadProfileLogo)
	mux.HandleFunc("/me", handleMe)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/forgot-password", handleForgotPassword)
	mux.HandleFunc("/reset-password", handleResetPassword)
	mux.HandleFunc("/reset-password-default", handleResetPasswordDefault)
	mux.HandleFunc("/force-change-password", handleForceChangePassword)
	mux.HandleFunc("/public/tenant/resolve", handleTenantResolve)
	mux.HandleFunc("/telegram/register", handleTelegramRegister)
	mux.HandleFunc("/telegram/login", handleTelegramLogin)
	mux.HandleFunc("/telegram/webhook", handleTelegramWebhook)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(uploadDir))))

	slog.Info(fmt.Sprintf("🔑 Starting Auth Service in %s mode on port %s...", cfg.Env, port))

	// Initialize Telegram bot token for package-level use
	telegramBotToken = cfg.Telegram.BotToken

	// Start Telegram polling goroutine if bot token is configured
	go startTelegramPolling(cfg)

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
	err = DB.QueryRow(ctx, `INSERT INTO tenants (name, plan, business_type) VALUES ('Superadmin', 'superadmin', 'umum') RETURNING id`).Scan(&tenantID)
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
		`INSERT INTO users (tenant_id, username, email, password_hash, role, phone_number) VALUES ($1, 'superadmin', 'superadmin@internal', $2, 'superadmin', '08210113344')`,
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
