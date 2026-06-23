package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

const (
	walletEndpoint         = "/wallet"
	querySelectAffiliateID = "SELECT referred_by_affiliate_id FROM tenants WHERE id = $1"
)

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────

func isSuperadmin(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(response.XUserRole) != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return false
	}
	return true
}


type SubscribeReq struct {
	PlanID      string `json:"plan_id"`
	VoucherCode string `json:"voucher_code,omitempty"`
	PayViaWallet bool   `json:"pay_via_wallet"` // F058: pay subscription via wallet
}

type VoucherRedeemReq struct {
	Code string `json:"code"`
}

type TicketPayload struct {
	TicketNumber  string `json:"ticket_number"`
	TenantName    string `json:"tenant_name"`
	PlanName      string `json:"plan_name"`
	PlanID        string `json:"plan_id"`
	ActivatedAt   string `json:"activated_at"`
	ExpiresAt     string `json:"expires_at"`
	AmountPaid    int64  `json:"amount_paid"`
	PaymentMethod string `json:"payment_method"`
	VoucherCode   string `json:"voucher_code,omitempty"`
}

type planRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceMonthly int64  `json:"price_monthly"`
	PriceYearly  int64  `json:"price_yearly"`
	IsActive     bool   `json:"is_active"`
	SortOrder    int    `json:"sort_order"`
}

type featureRow struct {
	FeatureKey   string `json:"feature_key"`
	FeatureName  string `json:"feature_name"`
	FeatureValue string `json:"feature_value"`
	IsEnabled    bool   `json:"is_enabled"`
}

type planWithFeatures struct {
	planRow
	Features []featureRow `json:"features"`
}

// N8N status types
type N8NStatus struct {
	Status       string `json:"status"`         // "connected", "disconnected", "unknown"
	Version      string `json:"version"`        // N8N version
	ActiveWorkflows int    `json:"active_workflows"`
	QueueMode    bool   `json:"queue_mode"`
	LastHealthCheck string `json:"last_health_check"`
}

// ─────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────

var (
	tenantCache = make(map[string]cachedTenant)
)

type xenditClientCacheEntry struct {
	client    *xendit.APIClient
	createdAt time.Time
}

var (
	xenditClientMu    sync.RWMutex
	xenditClientCache = make(map[string]*xenditClientCacheEntry) // tenantID → client
)

// getTenantXenditClient returns a cached Xendit API client for the given tenant.
// Caches for 5 minutes, then re-creates (in case key was rotated).
func getTenantXenditClient(ctx context.Context, tenantID string) (*xendit.APIClient, error) {
	xenditClientMu.RLock()
	entry, ok := xenditClientCache[tenantID]
	xenditClientMu.RUnlock()

	if ok && time.Since(entry.createdAt) < 5*time.Minute {
		return entry.client, nil
	}

	// Fetch tenant's Xendit API key from DB
	var apiKey string
	err := DB.QueryRow(ctx, "SELECT xendit_api_key FROM tenants WHERE id = $1", tenantID).Scan(&apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get xendit_api_key for tenant %s: %w", tenantID, err)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("tenant %s has no xendit_api_key configured", tenantID)
	}

	client := xendit.NewClient(apiKey)

	xenditClientMu.Lock()
	xenditClientCache[tenantID] = &xenditClientCacheEntry{
		client:    client,
		createdAt: time.Now(),
	}
	xenditClientMu.Unlock()

	return client, nil
}

// getTenantXenditMerchantID returns the tenant's Xendit merchant ID.
func getTenantXenditMerchantID(ctx context.Context, tenantID string) (string, error) {
	var merchantID string
	err := DB.QueryRow(ctx, "SELECT xendit_merchant_id FROM tenants WHERE id = $1", tenantID).Scan(&merchantID)
	if err != nil {
		return "", fmt.Errorf("failed to get xendit_merchant_id for tenant %s: %w", tenantID, err)
	}
	return merchantID, nil
}

// getTenantXenditWebhookToken returns the tenant's Xendit webhook token for verification.
func getTenantXenditWebhookToken(ctx context.Context, tenantID string) (string, error) {
	var token string
	err := DB.QueryRow(ctx, "SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to get xendit_webhook_token for tenant %s: %w", tenantID, err)
	}
	return token, nil
}

type cachedTenant struct {
	name      string
	email     string
	waNumber  string
	telegram  string
	expiresAt time.Time
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleN8NStatus(w http.ResponseWriter, r *http.Request) {
	status := N8NStatus{
		Status:         "unknown",
		Version:        "unknown",
		ActiveWorkflows: 0,
		QueueMode:      false,
		LastHealthCheck: time.Now().Format(time.RFC3339),
	}

	// Check N8N health via HTTP request
	n8nURL := "http://localhost:5678/rest/healthz"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(n8nURL)
	if err != nil {
		status.Status = "disconnected"
		response.JSON(w, http.StatusOK, "ok", status)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Status = "connected"

		// Parse N8N response to get version
		var n8nResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&n8nResp); err == nil {
			if v, ok := n8nResp["version"].(string); ok {
				status.Version = v
			}
			if qm, ok := n8nResp["queueMode"].(bool); ok {
				status.QueueMode = qm
			}
		}
	} else {
		status.Status = "disconnected"
	}

	// Get active workflows count
	workflowsURL := "http://localhost:5678/rest/workflows?active=true"
	wfResp, err := client.Get(workflowsURL)
	if err == nil {
		defer wfResp.Body.Close()
		if wfResp.StatusCode == http.StatusOK {
			var workflows []map[string]interface{}
			if err := json.NewDecoder(wfResp.Body).Decode(&workflows); err == nil {
				status.ActiveWorkflows = len(workflows)
			}
		}
	}

	response.JSON(w, http.StatusOK, "ok", status)
}

func handleN8NExecutions(w http.ResponseWriter, r *http.Request) {
	// Proxy to N8N API to get recent executions
	n8nURL := "http://localhost:5678/rest/executions?limit=10&includeData=true"
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET", n8nURL, nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:n8n_secure_admin_password_123")))

	resp, err := client.Do(req)
	if err != nil {
		response.JSON(w, http.StatusServiceUnavailable, "N8N unavailable", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.JSON(w, http.StatusServiceUnavailable, "Failed to fetch executions", nil)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		response.JSON(w, http.StatusInternalServerError, "Failed to parse response", nil)
		return
	}

	// Extract data from N8N response
	data := result["data"]
	if data == nil {
		data = []interface{}{}
	}

	response.JSON(w, http.StatusOK, "ok", data)
}

// handleHealthStatus returns aggregated health status of all platform services
func handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	type svcHealth struct {
		Name    string `json:"name"`
		Port    string `json:"port"`
		Status  string `json:"status"`
		Metrics string `json:"metrics,omitempty"`
	}

	services := []struct {
		name string
		port string
	}{
		{"wa-gateway", "8202"},
		{"umkm-chatbot", "8203"},
		{"auth-service", "8001"},
		{"ai-gateway", "8002"},
		{"umkm-accounting", "8201"},
		{"campaign-api", "9002"},
		{"billing-service", "8003"},
		{"notification-service", "8005"},
	}

	getTargetURL := func(svcName, port, endpoint string) string {
		if os.Getenv("APP_ENV") == "production" {
			return fmt.Sprintf("http://%s:%s%s", svcName, port, endpoint)
		}
		return fmt.Sprintf("http://localhost:%s%s", port, endpoint)
	}

	_, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	results := make([]svcHealth, 0, len(services))
	allUp := true

	for _, svc := range services {
		sh := svcHealth{Name: svc.name, Port: svc.port}

		// Try metrics endpoint first (WA Gateway & Chatbot have it)
		metricsURL := getTargetURL(svc.name, svc.port, "/metrics")
		client := &http.Client{Timeout: 3 * time.Second}

		if resp, err := client.Get(metricsURL); err == nil && resp.StatusCode < 500 {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				sh.Status = "up"
				sh.Metrics = string(body)
			}
		} else {
			// Fallback to health endpoint
			healthURL := getTargetURL(svc.name, svc.port, "/health")
			if resp, err := client.Get(healthURL); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					sh.Status = "up"
				} else {
					sh.Status = "degraded"
					allUp = false
				}
			} else {
				sh.Status = "down"
				allUp = false
			}
		}

		results = append(results, sh)
	}

	overall := "healthy"
	if !allUp {
		overall = "degraded"
	}

	response.JSON(w, http.StatusOK, "ok", map[string]interface{}{
		"status":  overall,
		"env":     config.LoadConfig(".env").Env,
		"services": results,
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")
	if err := initDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	// Run database migrations automatically
	if err := runMigrations(DB); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Start pending-cleanup background worker (F015)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startPendingCleanupWorker(ctx)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", handleHealth)

	// Public routes
	mux.HandleFunc("/plans", handleListPlans)
	mux.HandleFunc("/vouchers/validate", handleValidateVoucher)

	// Protected routes
	mux.Handle("/subscribe", auth.Middleware(http.HandlerFunc(handleSubscribe)))
	mux.Handle("/subscription", auth.Middleware(http.HandlerFunc(handleGetSubscription)))
	mux.Handle("/voucher/redeem", auth.Middleware(http.HandlerFunc(handleRedeemVoucher)))
	mux.Handle("/tickets", auth.Middleware(http.HandlerFunc(handleListTickets)))

	// Webhook (public with token auth)
	mux.HandleFunc("/webhook/payment", handlePaymentWebhook)

	// Superadmin routes (role checked in handler via X-User-Role header)
	mux.Handle("/admin/plans", auth.Middleware(http.HandlerFunc(handleAdminListPlans)))
	mux.Handle("/admin/plans/", auth.Middleware(http.HandlerFunc(handleAdminUpdatePlan)))
	mux.Handle("/admin/plan-features", auth.Middleware(http.HandlerFunc(handleAdminPlanFeaturesCollection)))
	mux.Handle("/admin/plan-features/", auth.Middleware(http.HandlerFunc(handleAdminPlanFeaturesItem)))
	mux.Handle("/admin/plan-features-matrix/", auth.Middleware(http.HandlerFunc(handleAdminPlanFeaturesMatrix)))
	mux.Handle("/admin/available-features", auth.Middleware(http.HandlerFunc(handleAdminAvailableFeaturesCollection)))
	mux.Handle("/admin/available-features/", auth.Middleware(http.HandlerFunc(handleAdminAvailableFeaturesItem)))
	mux.Handle("/admin/feature-matrix", auth.Middleware(http.HandlerFunc(handleAdminFeatureMatrix)))
	mux.Handle("/admin/addon-gating", auth.Middleware(http.HandlerFunc(handleAdminAddonGating)))
	mux.Handle("/admin/voucher-programs", auth.Middleware(http.HandlerFunc(handleAdminVoucherProgramsCollection)))
	mux.Handle("/admin/voucher-analytics", auth.Middleware(http.HandlerFunc(handleAdminVoucherAnalytics)))

	// Voucher link routes (public redeem + superadmin generate)
	mux.HandleFunc("/voucher/redeem-link", handleRedeemVoucherLink) // public, via signed token
	mux.Handle("/admin/voucher-links/generate", auth.Middleware(http.HandlerFunc(handleAdminGenerateVoucherLinks)))
	mux.Handle("/admin/voucher-links", auth.Middleware(http.HandlerFunc(handleAdminListVoucherLinks)))

	// Superadmin batch voucher codes (F015)
	mux.Handle("/admin/vouchers/generate", auth.Middleware(http.HandlerFunc(handleAdminGenerateVouchers)))
	mux.Handle("/admin/vouchers", auth.Middleware(http.HandlerFunc(handleAdminVouchers)))
	mux.Handle("/admin/tenants/", auth.Middleware(http.HandlerFunc(handleAdminTenantItem)))

	// Cleanup expired pending subscriptions (F015)
	mux.Handle("/admin/cleanup/pending", auth.Middleware(http.HandlerFunc(handleAdminCleanupPending)))

	// Superadmin dashboard (single endpoint, aggregated)
	mux.Handle("/admin/dashboard", auth.Middleware(http.HandlerFunc(handleAdminDashboard)))
	mux.Handle("/admin/n8n-status", auth.Middleware(http.HandlerFunc(handleN8NStatus)))
	mux.Handle("/admin/n8n-executions", auth.Middleware(http.HandlerFunc(handleN8NExecutions)))
	mux.Handle("/admin/health-status", auth.Middleware(http.HandlerFunc(handleHealthStatus)))

	// Superadmin: per-tenant quota dashboard (Task 2.8 — F025)
	mux.Handle("/admin/quota/", auth.Middleware(http.HandlerFunc(handleAdminQuotaUsage)))

	// F034: Add-on Wallet & Pricing
	mux.Handle("/admin/addon-prices", auth.Middleware(http.HandlerFunc(handleAdminAddonPrices)))
	mux.Handle("/admin/addon-prices/", auth.Middleware(http.HandlerFunc(handleAdminAddonPricesItem)))
	mux.Handle(walletEndpoint, auth.Middleware(http.HandlerFunc(handleWallet)))
	mux.Handle("/wallet/topup", auth.Middleware(http.HandlerFunc(handleWalletTopup)))

	// F053: Addon Purchase Flow
	mux.Handle("/addon-marketplace", auth.Middleware(http.HandlerFunc(handleAddonMarketplace)))
	mux.Handle("/addons/purchase", auth.Middleware(http.HandlerFunc(handlePurchaseAddon)))
	mux.Handle("/addons", auth.Middleware(http.HandlerFunc(handleMyAddons)))

	// F036: Lifetime Affiliate
	mux.HandleFunc("/api/public/affiliate-leaderboard", handleAffiliateLeaderboard)
	mux.Handle("/profile", auth.Middleware(http.HandlerFunc(handleAffiliateProfile)))
	mux.Handle("/register", auth.Middleware(http.HandlerFunc(handleAffiliateRegister)))
	mux.Handle("/withdraw", auth.Middleware(http.HandlerFunc(handleAffiliateWithdraw)))
	mux.Handle("/redeem-referral", auth.Middleware(http.HandlerFunc(handleAffiliateRedeemReferral)))
	mux.Handle("/referrals", auth.Middleware(http.HandlerFunc(handleAffiliateReferrals)))
	mux.Handle("/earnings", auth.Middleware(http.HandlerFunc(handleAffiliateEarnings)))

	// F037: Referral Config (Superadmin)
	mux.Handle("/admin/referral-config", auth.Middleware(http.HandlerFunc(handleAdminReferralConfig)))

	// F044: Campaign Licenses (B2B)
	mux.Handle("/admin/licenses", auth.Middleware(http.HandlerFunc(handleAdminLicenses)))
	mux.Handle("/admin/licenses/generate", auth.Middleware(http.HandlerFunc(handleAdminGenerateLicenses)))

	server := &http.Server{
		Addr:    ":8003",
		Handler: mux,
	}

	slog.Info("Billing Service listening", "port", 8003)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start Billing Service", "error", err)
	}
}

// ─────────────────────────────────────────────
// Public: List Plans
// ─────────────────────────────────────────────

func handleListPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT p.id, p.name, p.description, p.price_monthly, p.price_yearly, p.is_active, p.sort_order
		FROM saas_plans p
		WHERE p.is_active = true
		ORDER BY p.sort_order ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch plans", err)
		return
	}
	defer rows.Close()

	var plans []planWithFeatures
	for rows.Next() {
		var p planRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.IsActive, &p.SortOrder); err != nil {
			continue
		}

		featRows, err := DB.Query(ctx, `
			SELECT feature_key, feature_name, feature_value, is_enabled
			FROM plan_features WHERE plan_id = $1 AND is_enabled = true
			ORDER BY feature_name ASC
		`, p.ID)
		if err == nil {
			var feats []featureRow
			for featRows.Next() {
				var f featureRow
				if featRows.Scan(&f.FeatureKey, &f.FeatureName, &f.FeatureValue, &f.IsEnabled) == nil {
					feats = append(feats, f)
				}
			}
			featRows.Close()
			plans = append(plans, planWithFeatures{planRow: p, Features: feats})
		} else {
			plans = append(plans, planWithFeatures{planRow: p})
		}
	}

	response.JSON(w, http.StatusOK, "Plans retrieved", plans)
}

// ─────────────────────────────────────────────
// Public: Validate Voucher
// ─────────────────────────────────────────────

func handleValidateVoucher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "Missing voucher code", nil)
		return
	}

	ctx := r.Context()

	var programID, programName, voucherType string
	var discountValue, durationMonths int
	var targetPlanID *string
	var expiresAt *time.Time
	var maxUses, usesCount int
	var isActive bool

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.name, vp.voucher_type, vp.discount_value, vp.duration_months,
		       vp.target_plan_id, vp.expires_at, vp.max_uses, vp.uses_count, vp.is_active
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.starts_at IS NULL OR vp.starts_at <= NOW())
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		LIMIT 1
	`, code).Scan(&programID, &programName, &voucherType, &discountValue, &durationMonths,
		&targetPlanID, &expiresAt, &maxUses, &usesCount, &isActive)

	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher invalid or expired", nil)
		return
	}

	if maxUses > 0 && usesCount >= maxUses {
		response.Error(w, http.StatusBadRequest, "Voucher quota exceeded", nil)
		return
	}

	// Check plan name
	planName := "All Plans"
	if targetPlanID != nil && *targetPlanID != "" {
		var pn string
		if err := DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", *targetPlanID).Scan(&pn); err == nil {
			planName = pn
		}
	}

	var expStr *string
	if expiresAt != nil {
		s := expiresAt.Format(time.RFC3339)
		expStr = &s
	}

	response.JSON(w, http.StatusOK, "Voucher valid", map[string]interface{}{
		"program_id":      programID,
		"program_name":    programName,
		"voucher_type":    voucherType,
		"discount_value":  discountValue,
		"target_plan":     planName,
		"target_plan_id":  targetPlanID,
		"duration_months": durationMonths,
		"expires_at":      expStr,
	})
}

// ─────────────────────────────────────────────
// Protected: Subscribe
// ─────────────────────────────────────────────

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req SubscribeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	ctx := r.Context()

	// Validate plan
	var priceMonthly int64
	var planName string
	err := DB.QueryRow(ctx, `
		SELECT name, price_monthly FROM saas_plans WHERE id = $1 AND is_active = true
	`, req.PlanID).Scan(&planName, &priceMonthly)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid plan", nil)
		return
	}

	// Check if voucher is provided — apply discount
	var voucherApplied bool
	if req.VoucherCode != "" {
		voucherApplied = validateVoucherOnly(ctx, req.VoucherCode, req.PlanID)
		if voucherApplied {
			slog.Info("Voucher applied", "tenant_id", tenantID, "code", req.VoucherCode)
		}
	}

	// Calculate final price
	finalPrice := priceMonthly

	// ── Voucher discount (applied first) ──
	if voucherApplied {
		var discountType string
		var discountValue int
		DB.QueryRow(ctx, `
			SELECT voucher_type, discount_value FROM voucher_programs vp
			JOIN voucher_codes vc ON vc.program_id = vp.id WHERE vc.code = $1
		`, req.VoucherCode).Scan(&discountType, &discountValue)

		switch discountType {
		case "discount_percent":
			finalPrice = priceMonthly * int64(100-discountValue) / 100
		case "discount_fixed":
			finalPrice = maxInt64(0, priceMonthly-int64(discountValue))
		// free_months: finalPrice stays same, handled elsewhere
		}
	}

	// ── F054: Referral discount (stacked on post-voucher price) ──
	var referredByAffiliateID *int
	var referralDiscountAmount int64
	DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&referredByAffiliateID)
	if referredByAffiliateID != nil {
		var discountPct float64
		_ = DB.QueryRow(ctx, `SELECT COALESCE(discount_percent,0) FROM referral_config WHERE id=1`).Scan(&discountPct)
		if discountPct > 0 {
			referralDiscountAmount = finalPrice * int64(discountPct) / 100
			finalPrice = maxInt64(0, finalPrice-referralDiscountAmount)
		}
	}

	// ── F058: Pay via wallet ──
	if req.PayViaWallet && finalPrice > 0 {
		if !auth.CheckWalletBalance(ctx, tenantID, finalPrice) {
			var balance int64
			_ = DB.QueryRow(ctx, "SELECT COALESCE(balance_cents,0) FROM wallet_credits WHERE tenant_id=$1", tenantID).Scan(&balance)
			response.JSON(w, http.StatusPaymentRequired, "Saldo wallet tidak cukup", map[string]interface{}{
				"required_cents": finalPrice,
				"balance_cents":  balance,
				"topup_url":     walletEndpoint,
			})
			return
		}

		// Deduct wallet
		ref := fmt.Sprintf("subscription:%s:%d", req.PlanID, time.Now().Unix())
		desc := fmt.Sprintf("Pembayaran langganan %s via Wallet", planName)
		if err := auth.DeductWalletBalance(ctx, tenantID, finalPrice, ref, desc); err != nil {
			slog.Error("Wallet deduct failed for subscription", "tenant_id", tenantID, "error", err)
			response.Error(w, http.StatusInternalServerError, "Gagal memproses pembayaran wallet", nil)
			return
		}

		// Activate directly
		validityDays := 30
		if req.VoucherCode != "" {
			_ = DB.QueryRow(ctx, "SELECT validity_days FROM voucher_codes WHERE code=$1", req.VoucherCode).Scan(&validityDays)
		}
		activateSubscription(ctx, tenantID, req.PlanID, planName, validityDays, "wallet", nil, "")

		// F054: Affiliate commission (from final_price)
		if referredByAffiliateID != nil && referralDiscountAmount >= 0 {
			var commRate float64
			_ = DB.QueryRow(ctx, `SELECT COALESCE(commission_rate_percent,0) FROM referral_config WHERE id=1`).Scan(&commRate)
			if commRate > 0 {
				comm := float64(finalPrice) * commRate / 100
				_, _ = DB.Exec(ctx, `
					INSERT INTO affiliate_earnings (affiliate_id,tenant_id,invoice_id,amount_cents,commission_rate_percent,transaction_type,description)
					VALUES ($1,$2,$3,$4,$5,'subscription',$6)
				`, *referredByAffiliateID, tenantID, ref, int64(comm), commRate, desc)
			}
		}

		slog.Info("Subscription paid via wallet", "tenant_id", tenantID, "plan", req.PlanID, "amount", finalPrice)
		response.JSON(w, http.StatusOK, "Subscription activated via wallet", map[string]interface{}{
			"status":        "activated",
			"payment_method": "wallet",
			"plan_id":       req.PlanID,
			"amount_charged": finalPrice,
		})
		return
	}

	// Generate Xendit invoice
	invoiceID := uuid.NewString()
	externalID := fmt.Sprintf("INV-%s|%s", invoiceID, tenantID)

	createReq := invoice.NewCreateInvoiceRequest(externalID, float64(finalPrice)/100) // Xendit uses float but we convert sen→rupiah
	desc := fmt.Sprintf("Langganan %s — WCH Platform", planName)
	createReq.Description = &desc
	createReq.PaymentMethods = []string{"BANK_TRANSFER", "EWALLET", "QRIS"}

	var paymentURL string
	if finalPrice == 0 {
		slog.Info("FREE TRANSACTION DETECTED: Bypassing Xendit", "tenant_id", tenantID)
		// 1. Direct activation
		var codeValidityDays int
		if req.VoucherCode != "" {
			DB.QueryRow(ctx, "SELECT validity_days FROM voucher_codes WHERE code = $1", req.VoucherCode).Scan(&codeValidityDays)
			applyVoucher(ctx, req.VoucherCode, req.PlanID, tenantID) // Actually redeem it since it is zero-rupiah
		}
		if codeValidityDays == 0 { codeValidityDays = 30 }
		
		activateSubscription(ctx, tenantID, req.PlanID, planName, codeValidityDays, "voucher_direct", nil, "")
		
		// 2. Mark invoice as paid instantly
		extID := fmt.Sprintf("FREE-%s|%s", uuid.NewString()[:8], tenantID)
		_, _ = DB.Exec(ctx, "INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code, paid_at) VALUES ($1, $2, $3, 0, 'paid', 'free_bypass', $4, NOW())", extID, tenantID, req.PlanID, req.VoucherCode)
		
		response.JSON(w, http.StatusOK, "Success: Free subscription activated", map[string]string{"status": "activated", "payment_url": ""})
		return
	}

	xClient, errXc := getTenantXenditClient(ctx, tenantID)
	if errXc != nil {
		slog.Error("Failed to get xendit client for tenant", "tenant_id", tenantID, "error", errXc)
		response.Error(w, http.StatusInternalServerError, "Payment provider not configured", nil)
		return
	}
	resp, _, err := xClient.InvoiceApi.CreateInvoice(r.Context()).
		CreateInvoiceRequest(*createReq).Execute()
	paymentURL = ""
	if err != nil {
		slog.Error("Failed to create xendit invoice", "error", err)

		// In development mode, mock the invoice creation
		if os.Getenv("ENV") == "development" || os.Getenv("APP_ENV") == "development" || os.Getenv("ENV") == "" {
			slog.Warn("DEV mode: Mocking Xendit invoice creation")
			mockInvoiceUrl := fmt.Sprintf("https://checkout.xendit.co/web/%s", externalID)
			_, err = DB.Exec(ctx, `
				INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code)
				VALUES ($1, $2, $3, $4, 'pending', $5, $6)
				ON CONFLICT (id) DO UPDATE SET
					status = EXCLUDED.status, payment_url = EXCLUDED.payment_url
			`, externalID, tenantID, req.PlanID, finalPrice, mockInvoiceUrl, req.VoucherCode)
			if err != nil {
				slog.Warn("Failed to save mock invoice", "error", err)
			}
			paymentURL = mockInvoiceUrl
		} else {
			response.Error(w, http.StatusInternalServerError, "Failed to create invoice", nil)
			return
		}
	} else {
		paymentURL = resp.InvoiceUrl
	}

	// Persist invoice
	_, err = DB.Exec(ctx, `
		INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, payment_url = EXCLUDED.payment_url
	`, externalID, tenantID, req.PlanID, finalPrice, paymentURL, req.VoucherCode)
	if err != nil {
		slog.Warn("Failed to save invoice", "error", err)
	}

	// F054: Record referral discount applied to this invoice
	if referredByAffiliateID != nil && referralDiscountAmount > 0 {
		_, _ = DB.Exec(ctx, `
			INSERT INTO invoice_referrals (invoice_id, affiliate_id, discount_amount)
			VALUES ($1, $2, $3)
			ON CONFLICT (invoice_id) DO NOTHING
		`, externalID, *referredByAffiliateID, referralDiscountAmount)
	}

	// Create pending subscription — activates only after Xendit webhook confirms payment
	pendingHours := int64(24)
	if v := os.Getenv("SUBSCRIPTION_PENDING_TIMEOUT_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			pendingHours = int64(h)
		}
	}
	_, _ = DB.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, period_days, remaining_days, activated_by, pending_expires_at, updated_at)
		VALUES ($1, $2, $2, 'pending', 0, 0, 'pending', NOW() + ($3 || ' hours')::interval, NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_tier = EXCLUDED.plan_tier,
			status = 'pending',
			period_days = 0,
			remaining_days = 0,
			pending_expires_at = EXCLUDED.pending_expires_at,
			updated_at = NOW()
	`, tenantID, req.PlanID, pendingHours)

	slog.Info("Subscription pending created", "tenant_id", tenantID, "plan", req.PlanID, "invoice", externalID)

	response.JSON(w, http.StatusOK, "Invoice created", map[string]interface{}{
		"invoice_id":       externalID,
		"payment_url":     paymentURL,
		"plan_id":         req.PlanID,
		"plan_name":       planName,
		"original_price":  priceMonthly,
		"final_price":     finalPrice,
		"voucher_applied": voucherApplied,
		"status":          "pending",
	})
}

// ─────────────────────────────────────────────
// Protected: Get Current Subscription
// ─────────────────────────────────────────────

func handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	ctx := r.Context()
	var planID, status string
	var currentPeriodEnd *time.Time
	var planTier string
	var activatedBy string

	err := DB.QueryRow(ctx, `
		SELECT ts.plan_id, ts.status, ts.current_period_end, ts.plan_tier, ts.activated_by
		FROM tenant_subscriptions ts
		WHERE ts.tenant_id = $1
	`, tenantID).Scan(&planID, &status, &currentPeriodEnd, &planTier, &activatedBy)

	if err != nil || planID == "" {
		response.JSON(w, http.StatusOK, "No active subscription", map[string]interface{}{
			"has_subscription": false,
		})
		return
	}

	// Get plan details
	var planName string
	var priceMonthly int64
	DB.QueryRow(ctx, "SELECT name, price_monthly FROM saas_plans WHERE id = $1", planID).Scan(&planName, &priceMonthly)

	// Get features
	rows, _ := DB.Query(ctx, `
		SELECT feature_key, feature_name, feature_value
		FROM plan_features WHERE plan_id = $1 AND is_enabled = true
	`, planID)
	var features []map[string]string
	if rows != nil {
		for rows.Next() {
			var key, name, val string
			if rows.Scan(&key, &name, &val) == nil {
				features = append(features, map[string]string{
					"key":   key,
					"name":  name,
					"value": val,
				})
			}
		}
		rows.Close()
	}

	var periodEndStr *string
	if currentPeriodEnd != nil {
		s := currentPeriodEnd.Format(time.RFC3339)
		periodEndStr = &s
	}

	response.JSON(w, http.StatusOK, "Subscription retrieved", map[string]interface{}{
		"has_subscription": true,
		"plan_id":          planID,
		"plan_name":        planName,
		"plan_tier":        planTier,
		"price_monthly":    priceMonthly,
		"status":           status,
		"period_end":       periodEndStr,
		"features":         features,
		"activated_by":     activatedBy,
	})
}

// ─────────────────────────────────────────────
// Protected: Redeem Voucher
// ─────────────────────────────────────────────

func handleRedeemVoucher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req VoucherRedeemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	ctx := r.Context()

	// Lookup voucher — include validity_days from code row (set at generate time)
	var programID, programName, voucherType string
	var discountValue, programDurationMonths int
	var targetPlanID *string
	var expiresAt *time.Time
	var maxUses, usesCount int
	var codeValidityDays int

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.name, vp.voucher_type, vp.discount_value, vp.duration_months,
		       vp.target_plan_id, vp.expires_at, vp.max_uses, vp.uses_count,
		       vc.validity_days
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		LIMIT 1
	`, req.Code).Scan(&programID, &programName, &voucherType, &discountValue, &programDurationMonths,
		&targetPlanID, &expiresAt, &maxUses, &usesCount, &codeValidityDays)

	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher invalid or already used", nil)
		return
	}

	if maxUses > 0 && usesCount >= maxUses {
		response.Error(w, http.StatusBadRequest, "Voucher quota exceeded", nil)
		return
	}

	// Determine target plan
	planID := "lite"
	if targetPlanID != nil && *targetPlanID != "" {
		planID = *targetPlanID
	}

	// Get plan name + price
	var planName string
	var priceMonthly int64
	DB.QueryRow(ctx, "SELECT name, price_monthly FROM saas_plans WHERE id = $1", planID).Scan(&planName, &priceMonthly)

	// Activate subscription based on voucher type
	amountToCharge := priceMonthly
	switch voucherType {
	case "free_months":
		amountToCharge = 0
	case "discount_percent":
		amountToCharge = priceMonthly * int64(100-discountValue) / 100
	case "discount_fixed":
		amountToCharge = maxInt64(0, priceMonthly-int64(discountValue))
	}

	// Mark voucher as redeemed
	_, err = DB.Exec(ctx, `
		UPDATE voucher_codes SET is_redeemed = true, used_by = $1, used_at = NOW()
		WHERE code = $2 AND is_redeemed = false
	`, tenantID, req.Code)
	if err != nil {
		slog.Warn("Failed to mark voucher redeemed", "error", err)
	}

	// Increment usage count
	_, _ = DB.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

	// Activate subscription — use validity_days from the code row (set at generate time)
	ticketID := activateSubscription(ctx, tenantID, planID, planName, codeValidityDays, "voucher", nil, "")

	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", map[string]interface{}{
		"program_name":    programName,
		"voucher_type":    voucherType,
		"discount_value":  discountValue,
		"target_plan":     planName,
		"validity_days":   codeValidityDays,
		"amount_charged":  amountToCharge,
		"ticket_id":       ticketID,
	})
}

// ─────────────────────────────────────────────
// Protected: List Tickets
// ─────────────────────────────────────────────

func handleListTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, ticket_number, plan_id, plan_name, status,
		       activated_at, expires_at,
		       notify_wa, notify_telegram, notify_email,
		       wa_sent_at, telegram_sent_at, email_sent_at
		FROM subscription_tickets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
		return
	}
	defer rows.Close()

	var tickets []map[string]interface{}
	for rows.Next() {
		var id, ticketNum, planID, planName, status string
		var activatedAt, expiresAt *time.Time
		var notifyWA, notifyTelegram, notifyEmail bool
		var waSentAt, tgSentAt, emailSentAt *time.Time

		if rows.Scan(&id, &ticketNum, &planID, &planName, &status, &activatedAt, &expiresAt,
			&notifyWA, &notifyTelegram, &notifyEmail, &waSentAt, &tgSentAt, &emailSentAt) != nil {
			continue
		}

		tickets = append(tickets, map[string]interface{}{
			"id":             id,
			"ticket_number":  ticketNum,
			"plan_id":        planID,
			"plan_name":      planName,
			"status":         status,
			"activated_at":   formatTime(activatedAt),
			"expires_at":     formatTime(expiresAt),
			"notify_wa":      notifyWA,
			"notify_telegram": notifyTelegram,
			"notify_email":   notifyEmail,
			"wa_sent_at":     formatTime(waSentAt),
			"telegram_sent_at": formatTime(tgSentAt),
			"email_sent_at":  formatTime(emailSentAt),
		})
	}

	response.JSON(w, http.StatusOK, "Tickets retrieved", tickets)
}

// ─────────────────────────────────────────────
// Payment Webhook (Xendit callback)
// ─────────────────────────────────────────────

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status, _ := payload["status"].(string)
	externalID, _ := payload["external_id"].(string)

	// ── Per-tenant webhook token verification ──
	// Extract tenantID from external_id format:
	//   Invoice: "INV-{uuid}|{tenantID}"
	//   Topup:   "{uuid}-wallet-topup-{tenantID}"
	var tenantID string
	if strings.Contains(externalID, "-wallet-topup-") {
		parts := strings.Split(externalID, "-wallet-topup-")
		if len(parts) == 2 {
			tenantID = parts[1]
		}
	} else {
		parts := strings.Split(externalID, "|")
		if len(parts) >= 2 {
			tenantID = parts[1]
		}
	}

	if tenantID == "" {
		slog.Warn("Malformed external_id in webhook", "external_id", externalID)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify per-tenant webhook token
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken != "" {
		dbToken, err := getTenantXenditWebhookToken(r.Context(), tenantID)
		if err == nil && dbToken != "" {
			// Per-tenant token takes precedence
			if callbackToken != dbToken {
				slog.Warn("Unauthorized webhook: token mismatch", "tenant_id", tenantID,
					"expected", dbToken, "got", callbackToken)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else if envToken := os.Getenv("XENDIT_WEBHOOK_TOKEN"); envToken != "" {
			// Fallback to global env token (backward compat)
			if callbackToken != envToken {
				slog.Warn("Unauthorized webhook attempt", "token", callbackToken)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	paidAmountFloat, _ := payload["paid_amount"].(float64)
	paidAmountCents := int64(paidAmountFloat)

	// --- HANDLE WALLET TOPUP INVOICE ---
	if strings.Contains(externalID, "-wallet-topup-") {
		topupParts := strings.Split(externalID, "-wallet-topup-")
		if len(topupParts) == 2 {
			wTenantID := topupParts[1]
			if status == "EXPIRED" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if status != "PAID" && status != "SETTLED" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			// Get topup amount from xendit payload
			amtFloat, _ := payload["paid_amount"].(float64)
			amountCents := int64(amtFloat)
			
			// Deduplication check: have we processed this topup externalID?
			var existing int
			DB.QueryRow(r.Context(), "SELECT count(id) FROM wallet_transactions WHERE reference = $1", externalID).Scan(&existing)
			if existing > 0 {
				w.WriteHeader(http.StatusOK) // Already processed
				return
			}

			// Process topup transaction
			tx, err := DB.Begin(r.Context())
			if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(r.Context())
			
			// Upsert wallet_credits
			_, err = tx.Exec(r.Context(), `
				INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at) 
				VALUES ($1, $2, NOW()) 
				ON CONFLICT (tenant_id) 
				DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
			`, wTenantID, amountCents)
			if err != nil {
				slog.Error("Topup: Failed to update wallet_credits", "tenant", wTenantID, "err", err)
				return
			}
			
			// Insert transaction log
			_, err = tx.Exec(r.Context(), `
				INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description) 
				VALUES ($1, $2, 'topup', $3, 'Top-up via Xendit invoice')
			`, wTenantID, amountCents, externalID)
			if err != nil {
				return
			}
			
			tx.Commit(r.Context())
			slog.Info("Wallet topup successful", "tenant_id", wTenantID, "amount_cents", amountCents)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	ctx := r.Context()

	// IDEMPOTENCY CHECK + ROW LOCK: SELECT FOR UPDATE prevents concurrent webhooks
	// from both passing the PAID check and double-granting subscription.
	var currentStatus string
	var planID, voucherCode string
	var invoiceAmount int64
	err := DB.QueryRow(ctx, `
		SELECT status, COALESCE(plan_id,''), COALESCE(voucher_code,''), amount
		FROM invoices WHERE id = $1
		FOR UPDATE
	`, externalID).Scan(&currentStatus, &planID, &voucherCode, &invoiceAmount)
	if err != nil {
		slog.Warn("Invoice not found", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}
	if currentStatus == "paid" {
		slog.Info("Invoice already paid, ignoring duplicate webhook", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if status == "PAID" || status == "SETTLED" {
		if paidAmountCents < invoiceAmount {
			slog.Warn("PAYMENT AMOUNT MISMATCH", "external_id", externalID, "expected", invoiceAmount, "paid", paidAmountCents)
			// Still process but log as anomaly, or block activation. Let's block for safety.
			w.WriteHeader(http.StatusOK) 
			return
		}
	}
	if planID == "" {
		planID = "lite"
	}

	if status == "EXPIRED" {
		// Refund voucher so it can be reused
		if voucherCode != "" {
			_, err := DB.Exec(ctx, `
				UPDATE voucher_codes 
				SET is_redeemed = false, used_by = NULL, used_at = NULL 
				WHERE code = $1
			`, voucherCode)
			if err == nil {
				slog.Info("Voucher refunded due to expired invoice", "code", voucherCode, "tenant_id", tenantID)
			}
		}
		_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'expired' WHERE id = $1", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if status != "PAID" && status != "SETTLED" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get plan name
	var planName string
	DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", planID).Scan(&planName)

	// Update invoice
	_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", externalID)

	// HANDLE OVERPAYMENT: If paid > invoice, put difference in wallet
	if paidAmountCents > invoiceAmount {
		excess := paidAmountCents - invoiceAmount
		slog.Info("Overpayment detected, crediting wallet", "tenant_id", tenantID, "excess_cents", excess)
		_, errW := DB.Exec(ctx, `
			INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at) 
			VALUES ($1, $2, NOW()) 
			ON CONFLICT (tenant_id) 
			DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
		`, tenantID, excess)
		if errW == nil {
			_, _ = DB.Exec(ctx, `
				INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description) 
				VALUES ($1, $2, 'topup', $3, 'Excess payment from invoice')
			`, tenantID, excess, externalID)
		}
	}

	// Apply Voucher (Mark as REDEEMED permanently on actual payment)
	if voucherCode != "" {
		ok, _ := applyVoucher(ctx, voucherCode, planID, tenantID)
		if !ok {
			slog.Warn("Failed to apply voucher on payment webhook", "code", voucherCode, "tenant_id", tenantID)
		}
	}

	// ─────────────────────────────────────────────
	// F054: AFFILIATE COMMISSION LOGIC (lifetime)
	// ─────────────────────────────────────────────
	var referredByID *int
	DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&referredByID)
	if referredByID != nil {
		var commissionPct float64
		var minPurchaseCents, maxCommissionCents int64
		var isActive bool
		errCfg := DB.QueryRow(ctx, `
			SELECT COALESCE(commission_percent,10), COALESCE(min_purchase_cents,0),
			       COALESCE(max_commission_cents,0), COALESCE(is_active,true)
			FROM referral_config WHERE id = 1
		`).Scan(&commissionPct, &minPurchaseCents, &maxCommissionCents, &isActive)
		if errCfg != nil {
			commissionPct = 10
			minPurchaseCents = 0
			maxCommissionCents = 0
			isActive = true
		}

		if isActive && commissionPct > 0 && invoiceAmount >= minPurchaseCents {
			commission := invoiceAmount * int64(commissionPct) / 100
			if maxCommissionCents > 0 && commission > maxCommissionCents {
				commission = maxCommissionCents
			}
			if commission > 0 {
				_, errAff := DB.Exec(ctx, `
					UPDATE affiliates
					SET cash_balance_cents = cash_balance_cents + $1,
						total_earnings_cents = total_earnings_cents + $1,
						updated_at = NOW()
					WHERE id = $2
				`, commission, *referredByID)

				if errAff == nil {
					_, _ = DB.Exec(ctx, `
						INSERT INTO affiliate_earnings (affiliate_id, tenant_id, invoice_id, amount_cents, commission_rate_percent, transaction_type, description)
						VALUES ($1, $2, $3, $4, $5, 'subscription_renewal', 'Subscription renewal')
					`, *referredByID, tenantID, externalID, commission, int(commissionPct))

					// Mark first_purchase_at in affiliate_referrals on first commission
					_, _ = DB.Exec(ctx, `
						UPDATE affiliate_referrals
						SET first_purchase_at = NOW()
						WHERE affiliate_id = $1 AND tenant_id = $2 AND first_purchase_at IS NULL
					`, *referredByID, tenantID)

					slog.Info("Affiliate commission granted", "affiliate_id", *referredByID, "tenant_id", tenantID, "amount_cents", commission, "rate", commissionPct)
				}
			}
		}
	}

	// Activate subscription
	periodDays := 30
	activatedBy := "payment"
	if voucherCode != "" {
		activatedBy = "voucher"
		var duration int
		DB.QueryRow(ctx, `
			SELECT duration_months FROM voucher_programs vp
			JOIN voucher_codes vc ON vc.program_id = vp.id WHERE vc.code = $1
		`, voucherCode).Scan(&duration)
		if duration > 0 {
			periodDays = duration * 30
		}
	}

	// Auto-generate voucher code for this payment (for tenant's records and future reference)
	autoVoucherCode := generateVoucherCode(planID, tenantID)
	var voucherCodeID *string
	if autoVoucherCode != "" {
		var vid string
		DB.QueryRow(ctx, `
			INSERT INTO voucher_codes (program_id, code, is_redeemed, used_by, used_at, created_at)
			SELECT id, $1, true, $2, NOW(), NOW()
			FROM voucher_programs WHERE target_plan_id = $3 AND is_active = true
			LIMIT 1
			RETURNING id
		`, autoVoucherCode, tenantID, planID).Scan(&vid)
		if vid != "" {
			voucherCodeID = &vid
		}
	}

	ticketID := activateSubscription(ctx, tenantID, planID, planName, periodDays, activatedBy, voucherCodeID, autoVoucherCode)

	slog.Info("Payment processed", "tenant_id", tenantID, "plan", planID, "ticket_id", ticketID, "voucher_code", autoVoucherCode)

	w.WriteHeader(http.StatusOK)
}

// ─────────────────────────────────────────────
// Core: Activate Subscription + Generate Ticket
// ─────────────────────────────────────────────

func activateSubscription(ctx context.Context, tenantID, planID, planName string, validityDays int, activatedBy string, voucherCodeID *string, systemVoucherCode string) string {
	now := time.Now()
	ticketNumber := generateTicketNumber()

	// F027 AC-4: Proration — preserve remaining days from previous active subscription
	var proratedDays int
	row := DB.QueryRow(ctx, `
		SELECT GREATEST(0,
			EXTRACT(EPOCH FROM (current_plan_expires_at - NOW())) / 86400
		)::INTEGER
		FROM tenant_subscriptions
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err := row.Scan(&proratedDays); err == nil && proratedDays > 0 {
		validityDays += proratedDays
		slog.Info("prorated subscription", "tenant_id", tenantID, "prorated_days", proratedDays, "new_validity", validityDays)
	}

	// 1. Day-duration accumulator logic via voucher_subscriptions
	// If same plan already exists → accumulate remaining_days
	// If different plan → create new row (both co-exist; priority decides which is active)
	var newOrUpdatedVoucherSubID string
	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_subscriptions (tenant_id, plan_id, validity_days, remaining_days, is_system_generated, source_voucher_code)
		VALUES ($1, $2, $3, $3, $4, $5)
		ON CONFLICT (tenant_id, plan_id) DO UPDATE SET
			validity_days = voucher_subscriptions.validity_days + EXCLUDED.validity_days,
			remaining_days = voucher_subscriptions.remaining_days + EXCLUDED.remaining_days,
			updated_at = NOW(),
			is_system_generated = COALESCE($4, voucher_subscriptions.is_system_generated),
			source_voucher_code = COALESCE($5, voucher_subscriptions.source_voucher_code)
		RETURNING id
	`, tenantID, planID, validityDays, systemVoucherCode != "", systemVoucherCode).Scan(&newOrUpdatedVoucherSubID)
	if err != nil {
		slog.Error("Failed to upsert voucher_subscription", "error", err)
	}

	// 2. Recalculate effective plan_priority and update tenants
	// Priority: business=4, pro=3, lite=2, free=1 (highest wins)
	var effectivePlanID string
	var maxPriority int
	err = DB.QueryRow(ctx, `
		SELECT vs.plan_id,
			   CASE vs.plan_id
				   WHEN 'ultimate' THEN 4 WHEN 'pro' THEN 3 WHEN 'lite' THEN 2
				   ELSE 1
			   END AS priority,
			   vs.remaining_days
		FROM voucher_subscriptions vs
		WHERE vs.tenant_id = $1 AND vs.remaining_days > 0
		ORDER BY priority DESC, remaining_days DESC
		LIMIT 1
	`, tenantID).Scan(&effectivePlanID, &maxPriority, new(int))
	if err != nil || effectivePlanID == "" {
		effectivePlanID = "inactive"
		maxPriority = 0
	}

	// 3. Upsert subscription record
	_, err = DB.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, period_days, remaining_days, activated_by, voucher_code_id, system_voucher_code, updated_at)
		VALUES ($1, $2, $2, 'active', $3, $3, $4, $5, $6, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			plan_id = $2,
			plan_tier = $2,
			status = 'active',
			period_days = $3,
			remaining_days = $3,
			activated_by = $4,
			voucher_code_id = COALESCE($5, tenant_subscriptions.voucher_code_id),
			system_voucher_code = COALESCE($6, tenant_subscriptions.system_voucher_code),
			updated_at = NOW()
	`, tenantID, effectivePlanID, validityDays, activatedBy, voucherCodeID, systemVoucherCode)
	if err != nil {
		slog.Error("Failed to upsert subscription", "error", err)
	}

	// 4. Update tenant (un-freeze, set plan + priority)
	_, _ = DB.Exec(ctx, `
		UPDATE tenants SET
			plan = $1,
			plan_priority = $2,
			is_frozen = false,
			frozen_at = NULL,
			current_plan_expires_at = NOW() + (SELECT COALESCE(SUM(remaining_days), 0) || ' days'::interval FROM voucher_subscriptions WHERE tenant_id = $3)
		WHERE id = $3
	`, effectivePlanID, maxPriority, tenantID)

	// Sync Redis cache for quota gates
	auth.SetTenantPlan(ctx, tenantID, effectivePlanID)

	// 5. Create subscription ticket
	var ticketID string
	err = DB.QueryRow(ctx, `
		INSERT INTO subscription_tickets (tenant_id, plan_id, plan_name, ticket_number, expires_at, activated_by, notify_wa, notify_telegram, notify_email)
		VALUES ($1, $2, $3, $4, GREATEST(COALESCE((SELECT expires_at FROM subscription_tickets WHERE tenant_id = $1), NOW()), NOW()) + ($5 || ' days')::interval, $6, true, true, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_name = EXCLUDED.plan_name,
			ticket_number = EXCLUDED.ticket_number,
			status = 'active',
			expires_at = GREATEST(COALESCE(subscription_tickets.expires_at, NOW()), NOW()) + ($5 || ' days')::interval,
			activated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, tenantID, effectivePlanID, planName, ticketNumber, fmt.Sprintf("%d", validityDays), activatedBy).Scan(&ticketID)
	if err != nil {
		slog.Error("Failed to create ticket", "error", err)
		return ""
	}

	// 6. Update subscription with ticket ID
	_, _ = DB.Exec(ctx, "UPDATE tenant_subscriptions SET ticket_id = $1 WHERE tenant_id = $2", ticketID, tenantID)

	// 7. Async notification
	go sendTicketNotifications(tenantID, TicketPayload{
		TicketNumber:  ticketNumber,
		TenantName:    "",
		PlanName:      planName,
		PlanID:        planID,
		ActivatedAt:   now.Format("02 Jan 2006, 15:04 WIB"),
		ExpiresAt:     fmt.Sprintf("%d hari dari sekarang", validityDays),
		AmountPaid:    0,
		PaymentMethod: activatedBy,
		VoucherCode:   systemVoucherCode,
	})

	slog.Info("Subscription activated (day-duration)", "tenant_id", tenantID, "plan", effectivePlanID, "validity_days", validityDays, "ticket", ticketNumber, "system_voucher", systemVoucherCode)
	return ticketID
}

// ─────────────────────────────────────────────
// Voucher Application
// ─────────────────────────────────────────────

// validateVoucherOnly checks if a voucher code is valid without redeeming it.
func validateVoucherOnly(ctx context.Context, code, planID string) bool {
	var id string
	err := DB.QueryRow(ctx, `
		SELECT vc.code FROM voucher_codes vc
		JOIN voucher_programs vp ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		  AND (vp.target_plan_id IS NULL OR vp.target_plan_id = '' OR vp.target_plan_id = $2)
	`, code, planID).Scan(&id)
	return err == nil
}

func applyVoucher(ctx context.Context, code, planID, tenantID string) (bool, *string) {
	// Atomically mark voucher as redeemed and fetch its program rules
	var programID string
	var targetPlanID *string

	err := DB.QueryRow(ctx, `
		WITH valid_voucher AS (
			SELECT vc.code, vp.id as program_id, vp.target_plan_id
			FROM voucher_codes vc
			JOIN voucher_programs vp ON vc.program_id = vp.id
			WHERE vc.code = $1 AND vc.is_redeemed = false
			  AND vp.is_active = true
			  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
			  AND (vp.target_plan_id IS NULL OR vp.target_plan_id = '' OR vp.target_plan_id = $3)
		),
		updated_voucher AS (
			UPDATE voucher_codes
			SET is_redeemed = true, used_by = $2, used_at = NOW()
			FROM valid_voucher
			WHERE voucher_codes.code = valid_voucher.code
			RETURNING valid_voucher.program_id, valid_voucher.target_plan_id
		)
		SELECT program_id, target_plan_id FROM updated_voucher
	`, code, tenantID, planID).Scan(&programID, &targetPlanID)

	if err != nil { // No rows = invalid, expired, or already used (Race Condition prevented)
		return false, nil
	}

	// Increment usage count
	DB.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

	return true, nil
}

// ─────────────────────────────────────────────
// Notification Dispatcher
// ─────────────────────────────────────────────

func sendTicketNotifications(tenantID string, payload TicketPayload) {
	// Get tenant contact info
	waNumber := os.Getenv("TEST_WA_NUMBER")
	telegramChatID := os.Getenv("TEST_TELEGRAM_CHAT_ID")
	email := os.Getenv("TEST_EMAIL")

	// Try to fetch from DB
	var tenantName string
	DB.QueryRow(context.Background(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
	if tenantName == "" {
		tenantName = "Pelanggan"
	}
	payload.TenantName = tenantName

	// Get notification preferences
	var notifyWA, notifyTelegram, notifyEmail bool
	DB.QueryRow(context.Background(), `
		SELECT COALESCE(notify_wa, true), COALESCE(notify_telegram, false), COALESCE(notify_email, true)
		FROM tenant_notification_settings WHERE tenant_id = $1
	`, tenantID).Scan(&notifyWA, &notifyTelegram, &notifyEmail)

	if notifyWA && waNumber != "" {
		msg := buildTicketMessage(payload)
		sendWANotification(tenantID, waNumber, msg)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET wa_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}

	if notifyTelegram && telegramChatID != "" {
		msg := buildTicketMessage(payload)
		go sendTelegramNotification(telegramChatID, msg)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET telegram_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}

	if notifyEmail && email != "" {
		go sendEmailNotification(email, payload)
		DB.Exec(context.Background(), `
			UPDATE subscription_tickets SET email_sent_at = NOW() WHERE ticket_number = $1
		`, payload.TicketNumber)
	}
}

func buildTicketMessage(p TicketPayload) string {
	return fmt.Sprintf(`🎫 *Tiket Langganan WCH Platform*

Halo %s!

Langganan kamu telah aktif:

📋 *Detail Tiket:*
• Nomor Tiket: %s
• Paket: %s
• Status: ✅ Aktif
• Aktivasi: %s
• Kadaluarsa: %s

Terima kasih sudah percaya WCH Platform! 🚀`, p.TenantName, p.TicketNumber, p.PlanName, p.ActivatedAt, p.ExpiresAt)
}

// ─────────────────────────────────────────────
// Send WA Notification
// ─────────────────────────────────────────────

func sendWANotification(tenantID, target, message string) {
	targetJID := target
	if !strings.Contains(targetJID, "@s.whatsapp.net") {
		if strings.HasPrefix(targetJID, "0") {
			targetJID = "62" + targetJID[1:]
		}
		targetJID = strings.TrimPrefix(targetJID, "+")
		targetJID = targetJID + "@s.whatsapp.net"
	}

	waURL := "http://localhost:8202/api/wa/send"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		waURL = "http://wa-gateway:8202/api/wa/send"
	}

	data := url.Values{}
	data.Set("tenant_id", tenantID)
	data.Set("target", targetJID)
	data.Set("message", message)

	req, err := http.NewRequest("POST", waURL, strings.NewReader(data.Encode()))
	if err != nil {
		slog.Error("WA notification: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Message-Type", "invoice")
	req.Header.Set("X-Source", "billing-service")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("WA notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("WA ticket notification sent", "tenant_id", tenantID, "target", targetJID)
	}
}

// ─────────────────────────────────────────────
// Send Telegram Notification
// ─────────────────────────────────────────────

func sendTelegramNotification(chatID, message string) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping")
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    message,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Telegram notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Telegram ticket notification sent", "chat_id", chatID)
	}
}

// ─────────────────────────────────────────────
// Send Email Notification
// ─────────────────────────────────────────────

func sendEmailNotification(email string, payload TicketPayload) {
	// Simple SMTP email via notification-service
	notifURL := "http://localhost:8005/api/notification/send"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		notifURL = "http://notification-service:8005/api/notification/send"
	}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"type":    "email",
		"target":  email,
		"message": buildTicketMessage(payload),
		"subject": fmt.Sprintf("Tiket Langganan WCH Platform — %s", payload.TicketNumber),
	})

	resp, err := http.Post(notifURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		slog.Error("Email notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Email ticket notification sent", "email", email)
	}
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func generateTicketNumber() string {
	now := time.Now()
	return fmt.Sprintf("TKT-%d-%s-%04d",
		now.Year(),
		now.Format("0102"),
		now.UnixNano()%10000)
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────
// Superadmin: List All Plans (including inactive)
// ─────────────────────────────────────────────

func handleAdminListPlans(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, name, description, price_monthly, price_yearly, is_active, sort_order
		FROM saas_plans ORDER BY sort_order ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch plans", err)
		return
	}
	defer rows.Close()

	var plans []planRow
	for rows.Next() {
		var p planRow
		if rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.IsActive, &p.SortOrder) == nil {
			plans = append(plans, p)
		}
	}

	response.JSON(w, http.StatusOK, "Plans retrieved", plans)
}

// ─────────────────────────────────────────────
// Superadmin: Update Plan Price
// ─────────────────────────────────────────────

type UpdatePlanReq struct {
	PriceMonthly int64 `json:"price_monthly"`
	PriceYearly  int64 `json:"price_yearly"`
	IsActive     *bool `json:"is_active"`
	SortOrder    *int  `json:"sort_order"`
}

func handleAdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	planID := strings.TrimPrefix(r.URL.Path, "/admin/plans/")
	if planID == "" {
		response.Error(w, http.StatusBadRequest, "Missing plan ID", nil)
		return
	}

	var req UpdatePlanReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	// Validate: price must be non-negative (0 = free plan)
	if req.PriceMonthly < 0 || req.PriceYearly < 0 {
		response.Error(w, http.StatusBadRequest, "Price cannot be negative", nil)
		return
	}

	ctx := r.Context()

	// Verify plan exists
	var existingID string
	if err := DB.QueryRow(ctx, "SELECT id FROM saas_plans WHERE id = $1", planID).Scan(&existingID); err != nil {
		response.Error(w, http.StatusNotFound, "Plan not found", nil)
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []any{}
	argIdx := 1

	updates = append(updates, fmt.Sprintf("price_monthly = $%d", argIdx))
	args = append(args, req.PriceMonthly)
	argIdx++

	updates = append(updates, fmt.Sprintf("price_yearly = $%d", argIdx))
	args = append(args, req.PriceYearly)
	argIdx++

	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, planID)

	query := fmt.Sprintf("UPDATE saas_plans SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	_, err := DB.Exec(ctx, query, args...)
	if err != nil {
		slog.Error("Failed to update plan", "plan_id", planID, "error", err)
		response.Error(w, http.StatusInternalServerError, "Failed to update plan", err)
		return
	}

	slog.Info("Plan updated by superadmin", "plan_id", planID, "price_monthly", req.PriceMonthly, "price_yearly", req.PriceYearly)

	// Fetch updated plan
	var updated planRow
	DB.QueryRow(ctx, `
		SELECT id, name, description, price_monthly, price_yearly, is_active, sort_order
		FROM saas_plans WHERE id = $1
	`, planID).Scan(&updated.ID, &updated.Name, &updated.Description, &updated.PriceMonthly, &updated.PriceYearly, &updated.IsActive, &updated.SortOrder)

	response.JSON(w, http.StatusOK, "Plan updated", updated)
}

// ─────────────────────────────────────────────
// Superadmin: Plan Features CRUD (Dynamic per tier)
// Superadmin bisa add/edit/toggle feature per plan kapan saja
// tanpa harus tulis migration baru.
// ─────────────────────────────────────────────

type PlanFeatureReq struct {
	PlanID      string `json:"plan_id"`
	FeatureKey  string `json:"feature_key"`
	FeatureName string `json:"feature_name"`
	FeatureValue string `json:"feature_value"`
	IsEnabled   *bool  `json:"is_enabled"`
}

func handleAdminPlanFeaturesCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listPlanFeatures(w, r)
	case http.MethodPost:
		createPlanFeature(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func handleAdminPlanFeaturesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/admin/plan-features/")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Feature ID required", nil)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		updatePlanFeature(w, r, id)
	case http.MethodDelete:
		deletePlanFeature(w, r, id)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func handleAdminPlanFeaturesMatrix(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	planID := strings.TrimPrefix(r.URL.Path, "/admin/plan-features-matrix/")
	if planID == "" {
		response.Error(w, http.StatusBadRequest, "Plan ID required", nil)
		return
	}

	// GET — return current numeric limits for this plan
	if r.Method == http.MethodGet {
		row := DB.QueryRow(r.Context(), `
			SELECT COALESCE(MAX(max_users), 0), COALESCE(MAX(max_transactions), 0), COALESCE(MAX(max_ai_text), 0), COALESCE(MAX(max_ai_vision), 0),
				   COALESCE(MAX(max_ai_audio_minutes), 0), COALESCE(MAX(max_image_gen), 0), COALESCE(MAX(max_products), 0), COALESCE(MAX(max_customers), 0),
				   COALESCE(MAX(max_storage_mb), 0), COALESCE(MAX(api_rate_limit_per_min), 0), COALESCE(MAX(data_retention_months), 0)
			FROM plan_features WHERE plan_id = $1
		`, planID)
		var m struct {
			MaxUsers            int `json:"max_users"`
			MaxTransactions     int `json:"max_transactions"`
			MaxAIText           int `json:"max_ai_text"`
			MaxAIVision         int `json:"max_ai_vision"`
			MaxAIAudioMinutes   int `json:"max_ai_audio_minutes"`
			MaxImageGen         int `json:"max_image_gen"`
			MaxProducts         int `json:"max_products"`
			MaxCustomers        int `json:"max_customers"`
			MaxStorageMB        int `json:"max_storage_mb"`
			APIRateLimitPerMin  int `json:"api_rate_limit_per_min"`
			DataRetentionMonths int `json:"data_retention_months"`
		}
		if err := row.Scan(&m.MaxUsers, &m.MaxTransactions, &m.MaxAIText, &m.MaxAIVision,
			&m.MaxAIAudioMinutes, &m.MaxImageGen, &m.MaxProducts, &m.MaxCustomers,
			&m.MaxStorageMB, &m.APIRateLimitPerMin, &m.DataRetentionMonths); err != nil {
			response.Error(w, http.StatusNotFound, "Plan not found", err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", m)
		return
	}

	// PATCH — update numeric limits (columns are in saas_plans table)
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req map[string]int
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	updates := []string{}
	args := []any{}
	idx := 1

	// We map exact columns allowed to be updated to prevent SQL injection
	allowedColumns := map[string]bool{
		"max_users": true, "max_transactions": true, "max_ai_text": true,
		"max_ai_vision": true, "max_ai_audio_minutes": true, "max_image_gen": true,
		"max_products": true, "max_customers": true, "max_storage_mb": true,
		"api_rate_limit_per_min": true, "data_retention_months": true,
	}

	for key, val := range req {
		if allowedColumns[key] {
			updates = append(updates, fmt.Sprintf("%s = $%d", key, idx))
			args = append(args, val)
			idx++
		}
	}

	if len(updates) == 0 {
		response.JSON(w, http.StatusOK, "No updates applied", nil)
		return
	}

	args = append(args, planID)
	query := fmt.Sprintf("UPDATE plan_features SET %s WHERE plan_id = $%d", strings.Join(updates, ", "), idx)
	if _, err := DB.Exec(r.Context(), query, args...); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update matrix", err)
		return
	}
	// Invalidate the cache across all services so the new limits take effect immediately
	if cache.Client != nil {
		cache.Client.Del(r.Context(), "plan_features:"+planID)
	}
	response.JSON(w, http.StatusOK, "Matrix updated", nil)
}

func listPlanFeatures(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("plan_id")
	var (
		rows interface {
			Next() bool
			Close()
			Scan(...any) error
		}
		err error
	)

	if planID != "" {
		rows, err = DB.Query(r.Context(), `
			SELECT id, plan_id, feature_key, feature_name, feature_value, is_enabled, created_at
			FROM plan_features WHERE plan_id = $1 ORDER BY feature_key ASC
		`, planID)
	} else {
		rows, err = DB.Query(r.Context(), `
			SELECT id, plan_id, feature_key, feature_name, feature_value, is_enabled, created_at
			FROM plan_features ORDER BY plan_id ASC, feature_key ASC
		`)
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list features", err)
		return
	}
	defer rows.Close()

	features := []map[string]interface{}{}
	for rows.Next() {
		var id, planID, key, name, value string
		var enabled bool
		var createdAt time.Time
		if rows.Scan(&id, &planID, &key, &name, &value, &enabled, &createdAt) == nil {
			features = append(features, map[string]interface{}{
				"id":            id,
				"plan_id":       planID,
				"feature_key":   key,
				"feature_name":  name,
				"feature_value": value,
				"is_enabled":    enabled,
				"created_at":    createdAt.Format(time.RFC3339),
			})
		}
	}

	response.JSON(w, http.StatusOK, "Features retrieved", features)
}

func createPlanFeature(w http.ResponseWriter, r *http.Request) {
	var req PlanFeatureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}
	if req.PlanID == "" || req.FeatureKey == "" || req.FeatureName == "" {
		response.Error(w, http.StatusBadRequest, "plan_id, feature_key, and feature_name are required", nil)
		return
	}

	var planExists bool
	if err := DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM saas_plans WHERE id=$1)", req.PlanID).Scan(&planExists); err != nil || !planExists {
		response.Error(w, http.StatusBadRequest, "Invalid plan_id", nil)
		return
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	var newID string
	err := DB.QueryRow(r.Context(), `
		INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (plan_id, feature_key) DO UPDATE SET
			feature_name = EXCLUDED.feature_name,
			feature_value = EXCLUDED.feature_value,
			is_enabled = EXCLUDED.is_enabled
		RETURNING id
	`, req.PlanID, req.FeatureKey, req.FeatureName, req.FeatureValue, enabled).Scan(&newID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create/update feature", err)
		return
	}

	slog.Info("Plan feature upserted", "plan_id", req.PlanID, "feature_key", req.FeatureKey, "value", req.FeatureValue)

	response.JSON(w, http.StatusOK, "Feature saved", map[string]interface{}{
		"id":            newID,
		"plan_id":       req.PlanID,
		"feature_key":   req.FeatureKey,
		"feature_name":  req.FeatureName,
		"feature_value": req.FeatureValue,
		"is_enabled":    enabled,
	})
}

func updatePlanFeature(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		FeatureName  *string `json:"feature_name,omitempty"`
		FeatureValue *string `json:"feature_value,omitempty"`
		IsEnabled    *bool   `json:"is_enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := []string{}
	args := []any{}
	idx := 1
	if req.FeatureName != nil {
		updates = append(updates, fmt.Sprintf("feature_name = $%d", idx))
		args = append(args, *req.FeatureName)
		idx++
	}
	if req.FeatureValue != nil {
		updates = append(updates, fmt.Sprintf("feature_value = $%d", idx))
		args = append(args, *req.FeatureValue)
		idx++
	}
	if req.IsEnabled != nil {
		updates = append(updates, fmt.Sprintf("is_enabled = $%d", idx))
		args = append(args, *req.IsEnabled)
		idx++
	}

	if len(updates) == 0 {
		response.Error(w, http.StatusBadRequest, "No fields to update", nil)
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE plan_features SET %s WHERE id = $%d", strings.Join(updates, ", "), idx)
	res, err := DB.Exec(r.Context(), query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update feature", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Feature not found", nil)
		return
	}

	slog.Info("Plan feature updated", "id", id)
	response.JSON(w, http.StatusOK, "Feature updated", nil)
}

func deletePlanFeature(w http.ResponseWriter, r *http.Request, id string) {
	res, err := DB.Exec(r.Context(), "DELETE FROM plan_features WHERE id = $1", id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete feature", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Feature not found", nil)
		return
	}
	slog.Info("Plan feature deleted", "id", id)
	response.JSON(w, http.StatusOK, "Feature deleted", nil)
}

// ─────────────────────────────────────────────
// F057: Superadmin Feature & Addon Matrix UI
// ─────────────────────────────────────────────

// Routes — add after existing admin routes
// mux.Handle("/admin/available-features", ...)
// mux.Handle("/admin/available-features/", ...)
// mux.Handle("/admin/feature-matrix", ...)
// mux.Handle("/admin/addon-gating", ...)

func handleAdminAvailableFeaturesCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), `
			SELECT feature_key, feature_name, description, category,
			       is_addon, default_enabled, addon_price_cents, addon_unit
			FROM available_features ORDER BY is_addon, feature_key
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to list features", err)
			return
		}
		defer rows.Close()
		var items []map[string]interface{}
		for rows.Next() {
			var key, name, desc, cat, unit string
			var isAddon bool
			var defaultEnabled []string
			var price int64
			if rows.Scan(&key, &name, &desc, &cat, &isAddon, &defaultEnabled, &price, &unit) == nil {
				items = append(items, map[string]interface{}{
					"feature_key":       key,
					"feature_name":      name,
					"description":       desc,
					"category":          cat,
					"is_addon":          isAddon,
					"default_enabled":   defaultEnabled,
					"addon_price_cents": price,
					"addon_unit":        unit,
				})
			}
		}
		response.JSON(w, http.StatusOK, "ok", items)
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			FeatureKey      string   `json:"feature_key"`
			FeatureName     string   `json:"feature_name"`
			Description     string   `json:"description"`
			Category        string   `json:"category"`
			IsAddon         bool     `json:"is_addon"`
			DefaultEnabled  []string `json:"default_enabled"`
			AddonPriceCents int64    `json:"addon_price_cents"`
			AddonUnit       string   `json:"addon_unit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request", err)
			return
		}
		if req.FeatureKey == "" || req.FeatureName == "" || req.Category == "" {
			response.Error(w, http.StatusBadRequest, "feature_key, feature_name, category required", nil)
			return
		}
		_, err := DB.Exec(r.Context(), `
			INSERT INTO available_features (feature_key, feature_name, description, category, is_addon, default_enabled, addon_price_cents, addon_unit)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (feature_key) DO UPDATE SET
				feature_name=EXCLUDED.feature_name, description=EXCLUDED.description,
				category=EXCLUDED.category, is_addon=EXCLUDED.is_addon,
				default_enabled=EXCLUDED.default_enabled,
				addon_price_cents=EXCLUDED.addon_price_cents, addon_unit=EXCLUDED.addon_unit
		`, req.FeatureKey, req.FeatureName, req.Description, req.Category,
			req.IsAddon, req.DefaultEnabled, req.AddonPriceCents, req.AddonUnit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save feature", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		response.JSON(w, http.StatusOK, "Feature saved", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

func handleAdminAvailableFeaturesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/available-features/")
	if key == "" {
		response.Error(w, http.StatusBadRequest, "Feature key required", nil)
		return
	}
	if r.Method == http.MethodPatch {
		var req struct {
			FeatureName     *string  `json:"feature_name,omitempty"`
			Description     *string  `json:"description,omitempty"`
			Category        *string  `json:"category,omitempty"`
			IsAddon         *bool    `json:"is_addon,omitempty"`
			DefaultEnabled  []string `json:"default_enabled"`
			AddonPriceCents *int64   `json:"addon_price_cents,omitempty"`
			AddonUnit       *string  `json:"addon_unit,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request", err)
			return
		}
		updates, args, idx := []string{}, []any{}, 1
		if req.FeatureName != nil {
			updates = append(updates, fmt.Sprintf("feature_name=$%d", idx)); args = append(args, *req.FeatureName); idx++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description=$%d", idx)); args = append(args, *req.Description); idx++
		}
		if req.Category != nil {
			updates = append(updates, fmt.Sprintf("category=$%d", idx)); args = append(args, *req.Category); idx++
		}
		if req.IsAddon != nil {
			updates = append(updates, fmt.Sprintf("is_addon=$%d", idx)); args = append(args, *req.IsAddon); idx++
		}
		if req.DefaultEnabled != nil {
			updates = append(updates, fmt.Sprintf("default_enabled=$%d", idx)); args = append(args, req.DefaultEnabled); idx++
		}
		if req.AddonPriceCents != nil {
			updates = append(updates, fmt.Sprintf("addon_price_cents=$%d", idx)); args = append(args, *req.AddonPriceCents); idx++
		}
		if req.AddonUnit != nil {
			updates = append(updates, fmt.Sprintf("addon_unit=$%d", idx)); args = append(args, *req.AddonUnit); idx++
		}
		if len(updates) == 0 {
			response.JSON(w, http.StatusOK, "No updates applied", nil)
			return
		}
		args = append(args, key)
		query := fmt.Sprintf("UPDATE available_features SET %s WHERE feature_key=$%d", strings.Join(updates, ", "), idx)
		_, err := DB.Exec(r.Context(), query, args...)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), key)
		response.JSON(w, http.StatusOK, "Feature updated", nil)
		return
	}
	if r.Method == http.MethodDelete {
		_, err := DB.Exec(r.Context(), "DELETE FROM available_features WHERE feature_key=$1", key)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to delete", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), key)
		response.JSON(w, http.StatusOK, "Feature deleted", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

// handleAdminFeatureMatrix returns all plans × all features as a toggle matrix.
func handleAdminFeatureMatrix(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), `
			SELECT pf.plan_id, pf.feature_key, pf.feature_name, pf.is_enabled,
			       pf.feature_value, pf.min_tier,
			       sp.name as plan_name, sp.sort_order
			FROM plan_features pf
			JOIN saas_plans sp ON sp.id = pf.plan_id
			ORDER BY sp.sort_order, pf.feature_key
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load matrix", err)
			return
		}
		defer rows.Close()

		type planRow struct {
			PlanID   string `json:"plan_id"`
			PlanName string `json:"plan_name"`
		}
		planMap := map[string]planRow{}
		// matrix: plan_id → feature_key → feature row
		matrix := map[string]map[string]map[string]interface{}{}

		featureOrder := []string{} // dedup preserve order
		seenFeature := map[string]bool{}

		for rows.Next() {
			var planID, key, name, value, minTier, planName string
			var enabled bool
			var sortOrder int
			if rows.Scan(&planID, &key, &name, &enabled, &value, &minTier, &planName, &sortOrder) == nil {
				if _, ok := matrix[planID]; !ok {
					matrix[planID] = map[string]map[string]interface{}{}
					planMap[planID] = planRow{PlanID: planID, PlanName: planName}
				}
				matrix[planID][key] = map[string]interface{}{
					"feature_key":   key,
					"feature_name":  name,
					"is_enabled":    enabled,
					"feature_value": value,
					"min_tier":      minTier,
				}
				if !seenFeature[key] {
					seenFeature[key] = true
					featureOrder = append(featureOrder, key)
				}
			}
		}

		// Build ordered plans list
		planIDs := []string{}
		rows2, _ := DB.Query(r.Context(), "SELECT id, name FROM saas_plans WHERE is_active=true ORDER BY sort_order")
		if rows2 != nil {
			defer rows2.Close()
			for rows2.Next() {
				var id, nm string
				rows2.Scan(&id, &nm)
				planIDs = append(planIDs, id)
			}
		}

		response.JSON(w, http.StatusOK, "ok", map[string]interface{}{
			"plans":          planMap,
			"plan_ids":       planIDs,
			"feature_order":  featureOrder,
			"matrix":         matrix,
		})
		return
	}

	if r.Method == http.MethodPatch {
		// Body: { plan_id, feature_key, is_enabled }
		var req struct {
			PlanID     string `json:"plan_id"`
			FeatureKey string `json:"feature_key"`
			IsEnabled  bool   `json:"is_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request", err)
			return
		}
		if req.PlanID == "" || req.FeatureKey == "" {
			response.Error(w, http.StatusBadRequest, "plan_id and feature_key required", nil)
			return
		}
		_, err := DB.Exec(r.Context(), `
			INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
			VALUES ($1, $2, $2, 'yes', $3)
			ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled=EXCLUDED.is_enabled
		`, req.PlanID, req.FeatureKey, req.IsEnabled)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to toggle feature", err)
			return
		}
		// Invalidate all plan feature caches for this tier
		if cache.Client != nil {
			cache.Client.Del(r.Context(), "plan_features:"+req.PlanID)
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		response.JSON(w, http.StatusOK, "Feature toggled", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

// handleAdminAddonGating GET/PATCH: manage per-addon min_tier requirement.
func handleAdminAddonGating(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), `
			SELECT af.feature_key, af.feature_name, af.is_addon,
			       af.default_enabled, af.addon_price_cents, af.addon_unit,
			       pf.min_tier
			FROM available_features af
			LEFT JOIN plan_features pf ON pf.plan_id = 'lite' AND pf.feature_key = af.feature_key
			WHERE af.is_addon = true
			ORDER BY af.feature_key
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load addon gating", err)
			return
		}
		defer rows.Close()
		var items []map[string]interface{}
		for rows.Next() {
			var key, name, unit, minTier string
			var isAddon bool
			var defaultEnabled []string
			var price int64
			if rows.Scan(&key, &name, &isAddon, &defaultEnabled, &price, &unit, &minTier) == nil {
				items = append(items, map[string]interface{}{
					"feature_key":       key,
					"feature_name":      name,
					"is_addon":          isAddon,
					"default_enabled":   defaultEnabled,
					"addon_price_cents": price,
					"addon_unit":        unit,
					"min_tier":          minTier,
				})
			}
		}
		response.JSON(w, http.StatusOK, "ok", items)
		return
	}

	if r.Method == http.MethodPatch {
		var req struct {
			FeatureKey     string  `json:"feature_key"`
			MinTier        *string `json:"min_tier"`        // nil = no minimum (any tier can buy)
			DefaultEnabled []string `json:"default_enabled"` // tiers that bundle this addon by default
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid request", err)
			return
		}
		if req.FeatureKey == "" {
			response.Error(w, http.StatusBadRequest, "feature_key required", nil)
			return
		}
		// Update available_features.default_enabled
		_, err := DB.Exec(r.Context(), `
			UPDATE available_features SET default_enabled=$1 WHERE feature_key=$2
		`, req.DefaultEnabled, req.FeatureKey)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update default_enabled", err)
			return
		}
		// Set min_tier in plan_features for all plans that have this addon row
		if req.MinTier != nil && *req.MinTier != "" {
			_, err = DB.Exec(r.Context(), `
				UPDATE plan_features SET min_tier=$1 WHERE feature_key=$2
			`, *req.MinTier, req.FeatureKey)
		} else {
			_, err = DB.Exec(r.Context(), `
				UPDATE plan_features SET min_tier=NULL WHERE feature_key=$1
			`, req.FeatureKey)
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update min_tier", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		if cache.Client != nil {
			for _, p := range []string{"lite", "pro", "ultimate"} {
				cache.Client.Del(r.Context(), "plan_features:"+p)
			}
		}
		response.JSON(w, http.StatusOK, "Addon gating updated", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

// ─────────────────────────────────────────────
// Voucher Link Redemption (link-based, signed token)
// 1 link = 1 klaim. Token = JWT signed with cfg.JWTSecret.
// Hanya SHA-256 hash yang disimpan di DB (bukan token mentah).
// ─────────────────────────────────────────────

type VoucherLinkRedeemReq struct {
	Token    string `json:"token"`
	TenantID string `json:"tenant_id"`
}

type VoucherLinkRedeemResp struct {
	PlanID         string `json:"plan_id"`
	PlanName       string `json:"plan_name"`
	ActivatedAt    string `json:"activated_at"`
	ExpiresAt      string `json:"expires_at"`
	DurationMonths int    `json:"duration_months"`
	TicketNumber   string `json:"ticket_number"`
}

func signVoucherToken(programID, planID string, durationMonths int, expiresAt time.Time) (string, error) {
	cfg := config.GlobalConfig
	secret := []byte(cfg.JWTSecret)
	claims := jwt.MapClaims{
		"program_id":      programID,
		"plan_id":         planID,
		"duration_months": durationMonths,
		"exp":             expiresAt.Unix(),
		"iat":             time.Now().Unix(),
		"jti":             uuid.NewString(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func handleRedeemVoucherLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req VoucherLinkRedeemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}
	if req.Token == "" || req.TenantID == "" {
		response.Error(w, http.StatusBadRequest, "token and tenant_id are required", nil)
		return
	}

	// 1. Verify JWT signature & extract claims
	cfg := config.GlobalConfig
	secret := []byte(cfg.JWTSecret)
	parsed, err := jwt.Parse(req.Token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		response.Error(w, http.StatusBadRequest, "Invalid or expired voucher link", nil)
		return
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	programID, _ := claims["program_id"].(string)
	planID, _ := claims["plan_id"].(string)
	durationMonthsF, _ := claims["duration_months"].(float64)
	durationMonths := int(durationMonthsF)
	if programID == "" || planID == "" {
		response.Error(w, http.StatusBadRequest, "Malformed voucher link", nil)
		return
	}

	ctx := r.Context()
	tokenHash := hashToken(req.Token)

	// 2. Lookup link in DB
	var (
		linkID string
		linkProgramID string
		linkExpiresAt time.Time
		redeemedBy *string
		redeemedAt *time.Time
		isActive bool
	)
	err = DB.QueryRow(ctx, `
		SELECT id, program_id, expires_at, redeemed_by, redeemed_at, is_active
		FROM voucher_links WHERE token_hash = $1
	`, tokenHash).Scan(&linkID, &linkProgramID, &linkExpiresAt, &redeemedBy, &redeemedAt, &isActive)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher link not found", nil)
		return
	}
	if !isActive {
		response.Error(w, http.StatusBadRequest, "Voucher link has been deactivated", nil)
		return
	}
	if redeemedBy != nil {
		response.Error(w, http.StatusBadRequest, "Voucher link already redeemed", nil)
		return
	}
	if time.Now().After(linkExpiresAt) {
		response.Error(w, http.StatusBadRequest, "Voucher link has expired", nil)
		return
	}
	if linkProgramID != programID {
		response.Error(w, http.StatusBadRequest, "Token/program mismatch", nil)
		return
	}

	// 3. Lookup program & plan
	var (
		planName string
		programDuration int
		maxUsesPerTenant int
	)
	err = DB.QueryRow(ctx, `
		SELECT sp.name, COALESCE(vp.duration_months, 1), COALESCE(vp.max_uses_per_tenant, 1)
		FROM voucher_programs vp
		JOIN saas_plans sp ON sp.id = $1
		WHERE vp.id = $2 AND vp.is_active = true
	`, planID, programID).Scan(&planName, &programDuration, &maxUsesPerTenant)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Program inactive or not found", nil)
		return
	}
	if durationMonths == 0 {
		durationMonths = programDuration
	}

	// 4. Check max_uses_per_tenant (default 1)
	var usesByTenant int
	DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM voucher_links
		WHERE program_id = $1 AND redeemed_by = $2
	`, programID, req.TenantID).Scan(&usesByTenant)
	if usesByTenant >= maxUsesPerTenant {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Voucher quota per tenant exceeded",
			"data":    map[string]interface{}{"max_uses_per_tenant": maxUsesPerTenant},
		})
		return
	}

	// 5. Begin tx: mark link redeemed, activate/extend subscription
	tx, err := DB.Begin(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to start tx", err)
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE voucher_links
		SET redeemed_by = $1, redeemed_at = NOW(), is_active = false, ip_address = $2, user_agent = $3
		WHERE id = $4 AND is_active = true AND redeemed_by IS NULL
	`, req.TenantID, r.RemoteAddr, r.UserAgent(), linkID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to redeem link", err)
		return
	}

	// Increment uses_count on program
	_, _ = tx.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

	// Extend or activate subscription
	// If existing subscription active, extend period_end. Otherwise create new.
	var existingPeriodEnd *time.Time
	var existingStatus string
	err = tx.QueryRow(ctx, `
		SELECT current_period_end, status FROM tenant_subscriptions WHERE tenant_id = $1
	`, req.TenantID).Scan(&existingPeriodEnd, &existingStatus)

	now := time.Now()
	var newPeriodEnd time.Time
	if existingPeriodEnd != nil && existingStatus == "active" && existingPeriodEnd.After(now) {
		// Extend
		newPeriodEnd = existingPeriodEnd.AddDate(0, durationMonths, 0)
	} else {
		newPeriodEnd = now.AddDate(0, durationMonths, 0)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, current_period_end, period_days, activated_by, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $5, 'voucher', NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_tier = EXCLUDED.plan_tier,
			status = 'active',
			current_period_end = EXCLUDED.current_period_end,
			period_days = EXCLUDED.period_days,
			frozen_at = NULL,
			frozen_reason = NULL,
			updated_at = NOW()
	`, req.TenantID, planID, planID, newPeriodEnd, durationMonths*30)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update subscription", err)
		return
	}

	// Update tenants table (un-freeze, set plan, set expires)
	_, err = tx.Exec(ctx, `
		UPDATE tenants SET plan = $1, is_frozen = false, frozen_at = NULL, current_plan_expires_at = $2
		WHERE id = $3
	`, planID, newPeriodEnd, req.TenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update tenant", err)
		return
	}

	// Generate ticket
	ticketNumber := generateTicketNumber()
	var ticketID string
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_tickets (tenant_id, plan_id, plan_name, ticket_number, expires_at, activated_by, notify_wa, notify_telegram, notify_email)
		VALUES ($1, $2, $3, $4, $5, 'voucher', true, true, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_name = EXCLUDED.plan_name,
			ticket_number = EXCLUDED.ticket_number,
			status = 'active',
			expires_at = EXCLUDED.expires_at,
			activated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, req.TenantID, planID, planName, ticketNumber, newPeriodEnd).Scan(&ticketID)
	if err != nil {
		slog.Warn("Failed to create ticket", "error", err)
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to commit", err)
		return
	}

	// Sync Redis cache so quota gates read the correct plan tier.
	auth.SetTenantPlan(ctx, req.TenantID, planID)

	// Async notification
	go sendTicketNotifications(req.TenantID, TicketPayload{
		TicketNumber:  ticketNumber,
		PlanName:      planName,
		PlanID:        planID,
		ActivatedAt:   now.Format("02 Jan 2006, 15:04 WIB"),
		ExpiresAt:     newPeriodEnd.Format("02 Jan 2006, 15:04 WIB"),
		AmountPaid:    0,
		PaymentMethod: "voucher",
	})

	slog.Info("Voucher link redeemed", "tenant_id", req.TenantID, "plan", planID, "duration_months", durationMonths, "new_expires", newPeriodEnd)

	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", VoucherLinkRedeemResp{
		PlanID:         planID,
		PlanName:       planName,
		ActivatedAt:    now.Format(time.RFC3339),
		ExpiresAt:      newPeriodEnd.Format(time.RFC3339),
		DurationMonths: durationMonths,
		TicketNumber:   ticketNumber,
	})
}

// ─────────────────────────────────────────────
// Superadmin: Generate Voucher Links (bulk)
// Returns list of {token, url} for distribution
// ─────────────────────────────────────────────

type GenerateVoucherLinksReq struct {
	ProgramID      string `json:"program_id"`
	Count          int    `json:"count"`
	ValidDays      int    `json:"valid_days"`      // override program.expires_at
	BaseURL        string `json:"base_url"`        // e.g. https://app.wch.id
}

func handleAdminGenerateVoucherLinks(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req GenerateVoucherLinksReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}
	if req.ProgramID == "" || req.Count <= 0 || req.Count > 1000 {
		response.Error(w, http.StatusBadRequest, "program_id and count (1-1000) required", nil)
		return
	}

	// Lookup program
	var (
		planID string
		durationMonths int
		programExpiresAt *time.Time
	)
	err := DB.QueryRow(r.Context(), `
		SELECT target_plan_id, COALESCE(duration_months, 1), expires_at
		FROM voucher_programs WHERE id = $1 AND is_active = true
	`, req.ProgramID).Scan(&planID, &durationMonths, &programExpiresAt)
	if err != nil || planID == nilStr() {
		response.Error(w, http.StatusBadRequest, "Program not found or inactive", nil)
		return
	}

	// Determine link expiry
	var linkExpiresAt time.Time
	if req.ValidDays > 0 {
		linkExpiresAt = time.Now().AddDate(0, 0, req.ValidDays)
	} else if programExpiresAt != nil {
		linkExpiresAt = *programExpiresAt
	} else {
		linkExpiresAt = time.Now().AddDate(1, 0, 0) // 1 year default
	}

	// Find current superadmin user_id (best effort from header — actual auth user comes from JWT)
	creatorID, _ := r.Context().Value(auth.UserIDKey).(string)

	baseURL := req.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("PUBLIC_BASE_URL")
		if baseURL == "" {
			baseURL = "https://app.wch.id"
		}
	}

	type linkOut struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	links := make([]linkOut, 0, req.Count)

	for i := 0; i < req.Count; i++ {
		tok, err := signVoucherToken(req.ProgramID, planID, durationMonths, linkExpiresAt)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to sign token", err)
			return
		}
		hash := hashToken(tok)
		prefix := tok[:8]
		_, err = DB.Exec(r.Context(), `
			INSERT INTO voucher_links (program_id, token_hash, token_prefix, created_by, expires_at, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
		`, req.ProgramID, hash, prefix, creatorID, linkExpiresAt)
		if err != nil {
			slog.Warn("Failed to persist link", "error", err)
			continue
		}
		links = append(links, linkOut{
			Token: tok,
			URL:   fmt.Sprintf("%s/redeem?token=%s", baseURL, tok),
		})
	}

	// Log generation
	_, _ = DB.Exec(r.Context(), `
		INSERT INTO voucher_generation_logs (program_id, generated_by, count, prefix)
		VALUES ($1, $2, $3, $4)
	`, req.ProgramID, creatorID, len(links), "")

	slog.Info("Voucher links generated", "program_id", req.ProgramID, "count", len(links), "by", creatorID)

	response.JSON(w, http.StatusOK, "Links generated", map[string]interface{}{
		"program_id":      req.ProgramID,
		"count":           len(links),
		"expires_at":      linkExpiresAt.Format(time.RFC3339),
		"links":           links,
	})
}

func handleAdminListVoucherLinks(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	programID := r.URL.Query().Get("program_id")
	redeemedOnly := r.URL.Query().Get("redeemed") == "true"

	var (
		rows pgxRows
		err error
	)
	if programID != "" {
		rows, err = DB.Query(r.Context(), `
			SELECT id, program_id, token_prefix, redeemed_by, redeemed_at, expires_at, is_active, created_at
			FROM voucher_links WHERE program_id = $1
			  AND ($2 = false OR redeemed_by IS NOT NULL)
			ORDER BY created_at DESC LIMIT 200
		`, programID, redeemedOnly)
	} else {
		rows, err = DB.Query(r.Context(), `
			SELECT id, program_id, token_prefix, redeemed_by, redeemed_at, expires_at, is_active, created_at
			FROM voucher_links
			WHERE ($1 = false OR redeemed_by IS NOT NULL)
			ORDER BY created_at DESC LIMIT 200
		`, redeemedOnly)
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list links", err)
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, progID, prefix string
			redeemedBy *string
			redeemedAt, expiresAt, createdAt time.Time
			isActive bool
		)
		if rows.Scan(&id, &progID, &prefix, &redeemedBy, &redeemedAt, &expiresAt, &isActive, &createdAt) == nil {
			entry := map[string]interface{}{
				"id":          id,
				"program_id":  progID,
				"prefix":      prefix,
				"redeemed_by": redeemedBy,
				"redeemed_at": redeemedAt.Format(time.RFC3339),
				"expires_at":  expiresAt.Format(time.RFC3339),
				"is_active":   isActive,
				"created_at":  createdAt.Format(time.RFC3339),
			}
			out = append(out, entry)
		}
	}
	response.JSON(w, http.StatusOK, "Links retrieved", out)
}

// pgxRows is a minimal interface for both *pgx.Rows (used in helper) and *sql.Rows.
// Since pgx.Rows doesn't directly satisfy sql-scanner, we use a small adapter.
type pgxRows interface {
	Next() bool
	Close()
	Scan(...any) error
}

// ─────────────────────────────────────────────
// Superadmin Dashboard (single endpoint, aggregated)
// Overview: tenants, vouchers, revenue, frozen accounts
// ─────────────────────────────────────────────

func handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()

	// Tenant counts
	var (
		totalTenants, activeTenants, frozenTenants int
	)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&totalTenants)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE is_frozen = false`).Scan(&activeTenants)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE is_frozen = true`).Scan(&frozenTenants)

	// Voucher stats (last 30 days)
	var (
		linksGenerated30d, linksRedeemed30d int
	)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_links WHERE created_at >= NOW() - INTERVAL '30 days'`).Scan(&linksGenerated30d)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_links WHERE redeemed_at >= NOW() - INTERVAL '30 days'`).Scan(&linksRedeemed30d)

	// Active programs
	var activePrograms int
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_programs WHERE is_active = true`).Scan(&activePrograms)

	// Revenue (Xendit fallback, last 30 days, in sen)
	var revenue30d int64
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM invoices
		WHERE status = 'paid' AND paid_at >= NOW() - INTERVAL '30 days'
	`).Scan(&revenue30d)

	// Recent frozen accounts (top 10)
	rows, _ := DB.Query(ctx, `
		SELECT t.id, t.name, t.plan, t.frozen_at, t.current_plan_expires_at
		FROM tenants t WHERE t.is_frozen = true
		ORDER BY t.frozen_at DESC NULLS LAST LIMIT 10
	`)
	recentFrozen := []map[string]interface{}{}
	if rows != nil {
		for rows.Next() {
			var id, name, plan string
			var frozenAt, expiresAt *time.Time
			if rows.Scan(&id, &name, &plan, &frozenAt, &expiresAt) == nil {
				entry := map[string]interface{}{
					"id":   id,
					"name": name,
					"plan": plan,
				}
				if frozenAt != nil {
					entry["frozen_at"] = frozenAt.Format(time.RFC3339)
				}
				if expiresAt != nil {
					entry["expired_at"] = expiresAt.Format(time.RFC3339)
				}
				recentFrozen = append(recentFrozen, entry)
			}
		}
		rows.Close()
	}

	// Active subscriptions by plan (using tenants table for source of truth of active plans)
	planRows, _ := DB.Query(ctx, `
		SELECT plan, COUNT(*) FROM tenants
		WHERE is_frozen = false GROUP BY plan
	`)
	subsByPlan := map[string]int{}
	if planRows != nil {
		for planRows.Next() {
			var pid string
			var cnt int
			if planRows.Scan(&pid, &cnt) == nil {
				subsByPlan[pid] = cnt
			}
		}
		planRows.Close()
	}

	response.JSON(w, http.StatusOK, "Dashboard data", map[string]interface{}{
		"tenants": map[string]interface{}{
			"total":  totalTenants,
			"active": activeTenants,
			"frozen": frozenTenants,
		},
		"vouchers_30d": map[string]interface{}{
			"links_generated": linksGenerated30d,
			"links_redeemed":  linksRedeemed30d,
			"active_programs": activePrograms,
		},
		"revenue_30d_sen": revenue30d,
		"recent_frozen":   recentFrozen,
		"subs_by_plan":    subsByPlan,
	})
}

// nilStr returns "" for any string (helper for null target_plan_id)
func nilStr() string { return "" }

// generateVoucherCode creates an auto-generated voucher code for a payment
// Format: {PLAN_PREFIX}-{TIMESTAMP}-{RANDOM}
func generateVoucherCode(planID, tenantID string) string {
	planPrefix := map[string]string{
		"lite": "LITE",
		"pro":  "PRO",
		"enterprise": "ENT",
	}
	prefix := planPrefix[planID]
	if prefix == "" {
		prefix = "WCH"
	}

	// Use timestamp + tenant hash for uniqueness
	timestamp := time.Now().Unix() % 100000
	shortTenant := tenantID
	if len(shortTenant) > 4 {
		shortTenant = shortTenant[len(shortTenant)-4:]
	}

	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, strings.ToUpper(shortTenant))
}

// ─────────────────────────────────────────────
// Superadmin: Voucher Program Handlers
// ─────────────────────────────────────────────

type CreateVoucherProgramReq struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	VoucherType     string `json:"voucher_type"`
	DiscountValue   int    `json:"discount_value"`
	TargetPlanID    string `json:"target_plan_id"`
	DurationMonths  int    `json:"duration_months"`
	MaxUses         int    `json:"max_uses"`
	StartsAt        string `json:"starts_at"`
	ExpiresAt       string `json:"expires_at"`
}

func handleAdminVoucherProgramsCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listVoucherPrograms(w, r)
	case http.MethodPost:
		createVoucherProgram(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func listVoucherPrograms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT id, name, description, voucher_type, discount_value, COALESCE(target_plan_id, ''), duration_months, max_uses, uses_count, starts_at, expires_at, is_active
		FROM voucher_programs ORDER BY created_at DESC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list voucher programs", err)
		return
	}
	defer rows.Close()

	type programRow struct {
		ID             string     `json:"id"`
		Name           string     `json:"name"`
		Description    string     `json:"description"`
		VoucherType    string     `json:"voucher_type"`
		DiscountValue  int        `json:"discount_value"`
		TargetPlanID   string     `json:"target_plan_id"`
		DurationMonths int        `json:"duration_months"`
		MaxUses        int        `json:"max_uses"`
		UsesCount      int        `json:"uses_count"`
		StartsAt       time.Time  `json:"starts_at"`
		ExpiresAt      *time.Time `json:"expires_at"`
		IsActive       bool       `json:"is_active"`
	}

	var programs []programRow
	for rows.Next() {
		var p programRow
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.VoucherType, &p.DiscountValue,
			&p.TargetPlanID, &p.DurationMonths, &p.MaxUses, &p.UsesCount,
			&p.StartsAt, &p.ExpiresAt, &p.IsActive,
		)
		if err == nil {
			programs = append(programs, p)
		}
	}

	response.JSON(w, http.StatusOK, "Voucher programs retrieved", programs)
}

func createVoucherProgram(w http.ResponseWriter, r *http.Request) {
	var req CreateVoucherProgramReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	if req.Name == "" || req.VoucherType == "" {
		response.Error(w, http.StatusBadRequest, "name and voucher_type are required", nil)
		return
	}

	startsAt := time.Now()
	if req.StartsAt != "" {
		if t, err := time.Parse(time.RFC3339, req.StartsAt); err == nil {
			startsAt = t
		}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			expiresAt = &t
		}
	}

	ctx := r.Context()
	var id string
	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_programs (name, description, voucher_type, discount_value, target_plan_id, duration_months, max_uses, starts_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true)
		RETURNING id
	`, req.Name, req.Description, req.VoucherType, req.DiscountValue, req.TargetPlanID, req.DurationMonths, req.MaxUses, startsAt, expiresAt).Scan(&id)

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create voucher program", err)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher program created", map[string]interface{}{
		"id": id,
	})
}

func handleAdminVoucherAnalytics(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	programID := r.URL.Query().Get("program_id")

	var (
		totalGenerated int
		totalRedeemed  int
	)

	if programID != "" {
		// Specific program analytics
		err := DB.QueryRow(ctx, `
			SELECT 
				COUNT(*), 
				COUNT(CASE WHEN redeemed_by IS NOT NULL THEN 1 END)
			FROM voucher_links
			WHERE program_id = $1
		`, programID).Scan(&totalGenerated, &totalRedeemed)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch program analytics", err)
			return
		}

		response.JSON(w, http.StatusOK, "Program analytics retrieved", map[string]interface{}{
			"program_id":             programID,
			"total_links_generated":  totalGenerated,
			"total_links_redeemed":   totalRedeemed,
			"redemption_rate_percent": calculateRate(totalGenerated, totalRedeemed),
		})
	} else {
		// Global analytics
		var totalPrograms, activePrograms int
		err := DB.QueryRow(ctx, `
			SELECT 
				COUNT(*), 
				COUNT(CASE WHEN is_active = true THEN 1 END)
			FROM voucher_programs
		`).Scan(&totalPrograms, &activePrograms)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch global program stats", err)
			return
		}

		err = DB.QueryRow(ctx, `
			SELECT 
				COUNT(*), 
				COUNT(CASE WHEN redeemed_by IS NOT NULL THEN 1 END)
			FROM voucher_links
		`).Scan(&totalGenerated, &totalRedeemed)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to fetch global links stats", err)
			return
		}

		response.JSON(w, http.StatusOK, "Global voucher analytics retrieved", map[string]interface{}{
			"total_programs":          totalPrograms,
			"active_programs":         activePrograms,
			"total_links_generated":  totalGenerated,
			"total_links_redeemed":   totalRedeemed,
			"redemption_rate_percent": calculateRate(totalGenerated, totalRedeemed),
		})
	}
}

func calculateRate(total, redeemed int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(redeemed) / float64(total) * 100.0
}

// ─────────────────────────────────────────────
// Superadmin: Batch Generate Voucher Codes (F015)
// POST /admin/vouchers/generate
// ─────────────────────────────────────────────

type GenerateVouchersReq struct {
	PlanID        string `json:"plan_id"`
	ValidityDays  int    `json:"validity_days"`
	Quantity      int    `json:"quantity"`
	ProgramName   string `json:"program_name"`
	MaxUses       int    `json:"max_uses"`
	VoucherType   string `json:"voucher_type"`   // e.g. "bonus_months", "discount_percent", "discount_fixed"
	DiscountValue int    `json:"discount_value"`
}

func handleAdminGenerateVouchers(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req GenerateVouchersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}
	if req.PlanID == "" || req.ValidityDays <= 0 || req.Quantity <= 0 || req.Quantity > 1000 {
		response.Error(w, http.StatusBadRequest, "plan_id, validity_days (>0), and quantity (1-1000) required", nil)
		return
	}

	ctx := r.Context()
	creatorID, _ := r.Context().Value(auth.UserIDKey).(string)

	// Find or create program
	var programID string
	programName := req.ProgramName
	if programName == "" {
		programName = "Ad-hoc Voucher - " + req.PlanID
	}
	
	vType := req.VoucherType
	if vType == "" {
		vType = "bonus_months" // Backward compatibility fallback
	}

	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_programs (name, voucher_type, discount_value, target_plan_id, duration_months, max_uses, is_active)
		VALUES ($1, $2, $3, $4, 0, $5, true)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, programName, vType, req.DiscountValue, req.PlanID, req.MaxUses).Scan(&programID)
	if err != nil {
		// Try to get existing
		DB.QueryRow(ctx, `SELECT id FROM voucher_programs WHERE name = $1`, programName).Scan(&programID)
	}

	type codeOut struct {
		Code  string `json:"code"`
		Days  int    `json:"validity_days"`
	}
	codes := make([]codeOut, 0, req.Quantity)

	for i := 0; i < req.Quantity; i++ {
		code := generateVoucherCode(req.PlanID, uuid.NewString()[:8])
		_, err := DB.Exec(ctx, `
			INSERT INTO voucher_codes (program_id, code, is_redeemed, validity_days, created_at)
			VALUES ($1, $2, false, $3, NOW())
		`, programID, code, req.ValidityDays)
		if err != nil {
			slog.Warn("Failed to create voucher code", "error", err)
			continue
		}
		codes = append(codes, codeOut{Code: code, Days: req.ValidityDays})
	}

	// Log generation
	if programID != "" {
		_, _ = DB.Exec(ctx, `
			INSERT INTO voucher_generation_logs (program_id, generated_by, count, prefix)
			VALUES ($1, $2, $3, $4)
		`, programID, creatorID, len(codes), "")
	}

	slog.Info("Batch voucher codes generated", "plan", req.PlanID, "count", len(codes), "by", creatorID)

	response.JSON(w, http.StatusOK, "Voucher codes generated", map[string]interface{}{
		"plan_id":       req.PlanID,
		"validity_days": req.ValidityDays,
		"count":         len(codes),
		"codes":         codes,
	})
}

// ─────────────────────────────────────────────
// Superadmin: Voucher Codes (GET list / DELETE remove)
// ─────────────────────────────────────────────

func handleAdminVouchers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminListVouchers(w, r)
	case http.MethodDelete:
		handleAdminDeleteVoucher(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

// ─────────────────────────────────────────────
// Superadmin: List All Voucher Codes (F015)
// GET /admin/vouchers?plan_id=&used=&limit=
// ─────────────────────────────────────────────

func handleAdminListVouchers(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	planID := r.URL.Query().Get("plan_id")
	usedStr := r.URL.Query().Get("used")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	query := `
		SELECT vc.id, vc.code, vc.program_id, COALESCE(vp.name, ''),
		       vc.is_redeemed, COALESCE(vc.used_by::text, ''), vc.used_at, vc.created_at,
		       COALESCE(vp.target_plan_id, ''), COALESCE(vp.is_active, false)
		FROM voucher_codes vc
		LEFT JOIN voucher_programs vp ON vp.id = vc.program_id
		WHERE ($1 = '' OR vp.target_plan_id = $1)
		  AND ($2 = '' OR ($2 = 'true' AND vc.is_redeemed = true) OR ($2 = 'false' AND vc.is_redeemed = false))
		ORDER BY vc.created_at DESC
		LIMIT $3
	`

	rows, err := DB.Query(ctx, query, planID, usedStr, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list vouchers", err)
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id, code, progID, progName, usedBy string
		var isRedeemed bool
		var usedAt *time.Time
		var createdAt time.Time
		var targetPlan string
		var progActive bool
		if rows.Scan(&id, &code, &progID, &progName, &isRedeemed, &usedBy, &usedAt, &createdAt, &targetPlan, &progActive) == nil {
			entry := map[string]interface{}{
				"id":           id,
				"code":         code,
				"program_id":   progID,
				"program_name": progName,
				"is_redeemed":  isRedeemed,
				"used_by":      usedBy,
				"used_at":      formatTime(usedAt),
				"created_at":   createdAt.Format(time.RFC3339),
				"target_plan":  targetPlan,
				"program_active": progActive,
			}
			out = append(out, entry)
		}
	}

	// Count totals
	var total, used, unused int
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes`).Scan(&total)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes WHERE is_redeemed = true`).Scan(&used)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes WHERE is_redeemed = false`).Scan(&unused)

	response.JSON(w, http.StatusOK, "Vouchers retrieved", map[string]interface{}{
		"total":  total,
		"used":   used,
		"unused": unused,
		"codes":  out,
	})
}

// ─────────────────────────────────────────────
// Superadmin: Delete Voucher Code
// DELETE /admin/vouchers?id=<voucher_id>
// Only unused vouchers can be deleted
// ─────────────────────────────────────────────

func handleAdminDeleteVoucher(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	voucherID := r.URL.Query().Get("id")
	if voucherID == "" {
		response.Error(w, http.StatusBadRequest, "Missing voucher id", nil)
		return
	}

	ctx := r.Context()

	var isRedeemed bool
	err := DB.QueryRow(ctx, `SELECT is_redeemed FROM voucher_codes WHERE id = $1`, voucherID).Scan(&isRedeemed)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Voucher not found", err)
		return
	}
	if isRedeemed {
		response.Error(w, http.StatusBadRequest, "Cannot delete a voucher that has already been used", nil)
		return
	}

	_, err = DB.Exec(ctx, `DELETE FROM voucher_codes WHERE id = $1 AND is_redeemed = false`, voucherID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete voucher", err)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher deleted successfully", nil)
}

// ─────────────────────────────────────────────
// Superadmin: Tenant Item (GET /admin/tenants/{id}, PATCH) (F015)
// GET: returns tenant info + active vouchers
// PATCH: activate/freeze/delete tenant
// ─────────────────────────────────────────────

func handleAdminTenantItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/admin/tenants/")
	parts := strings.Split(path, "/")
	tenantID := parts[0]
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		// Tenant info
		var tenant map[string]interface{}
		var tID, name, plan string
		var isFrozen bool
		var createdAt time.Time
		var frozenAt, expiresAt *time.Time
		var planPriority int
		err := DB.QueryRow(ctx, `
			SELECT id, name, plan, is_frozen, created_at, frozen_at, current_plan_expires_at, plan_priority
			FROM tenants WHERE id = $1
		`, tenantID).Scan(&tID, &name, &plan, &isFrozen, &createdAt, &frozenAt, &expiresAt, &planPriority)
		if err != nil {
			response.Error(w, http.StatusNotFound, "Tenant not found", nil)
			return
		}

		tenant = map[string]interface{}{
			"id":              tID,
			"name":            name,
			"plan":            plan,
			"is_frozen":       isFrozen,
			"created_at":      createdAt.Format(time.RFC3339),
			"frozen_at":       formatTime(frozenAt),
			"expires_at":      formatTime(expiresAt),
			"plan_priority":   planPriority,
			"xendit_merchant_id": "",
		}
		// Fetch xendit_merchant_id separately (may be NULL)
		var merchantID *string
		DB.QueryRow(ctx, `SELECT xendit_merchant_id FROM tenants WHERE id = $1`, tenantID).Scan(&merchantID)
		if merchantID != nil {
			tenant["xendit_merchant_id"] = *merchantID
		}

		// Active vouchers for this tenant
		vrows, _ := DB.Query(ctx, `
			SELECT id, plan_id, validity_days, remaining_days, redeemed_at, is_system_generated, source_voucher_code
			FROM voucher_subscriptions WHERE tenant_id = $1 AND remaining_days > 0
			ORDER BY remaining_days DESC
		`, tenantID)
		vouchers := []map[string]interface{}{}
		if vrows != nil {
			for vrows.Next() {
				var vid, pid, srcCode string
				var vd, rd int
				var redeemedAt time.Time
				var isSysGen bool
				if vrows.Scan(&vid, &pid, &vd, &rd, &redeemedAt, &isSysGen, &srcCode) == nil {
					vouchers = append(vouchers, map[string]interface{}{
						"id":                   vid,
						"plan_id":              pid,
						"validity_days":        vd,
						"remaining_days":       rd,
						"redeemed_at":          redeemedAt.Format(time.RFC3339),
						"is_system_generated":  isSysGen,
						"source_voucher_code":  srcCode,
					})
				}
			}
			vrows.Close()
		}

		// Subscription info
		var subStatus string
		DB.QueryRow(ctx, `SELECT status FROM tenant_subscriptions WHERE tenant_id = $1`, tenantID).Scan(&subStatus)
		tenant["subscription_status"] = subStatus
		tenant["active_vouchers"] = vouchers

		response.JSON(w, http.StatusOK, "Tenant retrieved", tenant)

	case http.MethodPatch:
		var req struct {
			Action string `json:"action"` // "activate", "freeze", "delete"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}

		switch req.Action {
		case "activate":
			_, _ = DB.Exec(ctx, `UPDATE tenants SET is_frozen = false, frozen_at = NULL WHERE id = $1`, tenantID)
			response.JSON(w, http.StatusOK, "Tenant activated", nil)
		case "freeze":
			_, _ = DB.Exec(ctx, `UPDATE tenants SET is_frozen = true, frozen_at = NOW() WHERE id = $1`, tenantID)
			response.JSON(w, http.StatusOK, "Tenant frozen", nil)
		case "delete":
			// Delete tenant + cascade (users, subscriptions, etc)
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "Failed to delete tenant", err)
				return
			}
			slog.Warn("Tenant deleted by superadmin", "tenant_id", tenantID)
			response.JSON(w, http.StatusOK, "Tenant deleted", nil)
		default:
			response.Error(w, http.StatusBadRequest, "action must be: activate, freeze, or delete", nil)
		}

	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

// ─────────────────────────────────────────────
// Superadmin: Cleanup Expired Pending Subscriptions (F015)
// POST /admin/cleanup/pending — manual trigger
// GET  /admin/cleanup/pending — list what would be cleaned
// Runs automatically via subscription-worker cron
// ─────────────────────────────────────────────

func handleAdminCleanupPending(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	ctx := r.Context()

	// List pending tenants past timeout
	rows, err := DB.Query(ctx, `
		SELECT t.id, t.name, t.email, t.plan, ts.pending_expires_at, t.created_at
		FROM tenants t
		JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE ts.status = 'pending'
		  AND ts.pending_expires_at < NOW()
		ORDER BY ts.pending_expires_at ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query pending tenants", err)
		return
	}
	defer rows.Close()

	type pendingTenant struct {
		ID              string     `json:"id"`
		Name            string     `json:"name"`
		Email           string     `json:"email"`
		Plan            string     `json:"plan"`
		PendingExpiredAt *time.Time `json:"pending_expires_at"`
		CreatedAt       time.Time  `json:"created_at"`
	}

	pending := []pendingTenant{}
	for rows.Next() {
		var pt pendingTenant
		if rows.Scan(&pt.ID, &pt.Name, &pt.Email, &pt.Plan, &pt.PendingExpiredAt, &pt.CreatedAt) == nil {
			pending = append(pending, pt)
		}
	}

	if r.Method == http.MethodGet {
		response.JSON(w, http.StatusOK, "Pending tenants (would be cleaned)", map[string]interface{}{
			"count":   len(pending),
			"tenants": pending,
		})
		return
	}

	if r.Method == http.MethodPost {
		// Execute cleanup: delete expired pending tenants + users
		deleted := 0
		for _, pt := range pending {
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, pt.ID)
			if err == nil {
				_, _ = DB.Exec(ctx, `
					INSERT INTO pending_tenant_cleanup_log (tenant_id, user_id, email, phone, reason)
					SELECT $1, id, email, phone_number, 'pending_timeout'
					FROM users WHERE tenant_id = $1
				`, pt.ID)
				deleted++
				slog.Info("Expired pending tenant cleaned up", "tenant_id", pt.ID, "name", pt.Name)
			} else {
				slog.Error("Failed to cleanup pending tenant", "tenant_id", pt.ID, "error", err)
			}
		}

		// Also delete any tenant_subscriptions rows that are stuck in pending
		res, _ := DB.Exec(ctx, `
			DELETE FROM tenant_subscriptions
			WHERE status = 'pending' AND pending_expires_at < NOW()
		`)
		_, _ = DB.Exec(ctx, `DELETE FROM tenant_subscriptions WHERE status = 'pending' AND pending_expires_at IS NULL AND updated_at < NOW() - INTERVAL '48 hours'`)

		response.JSON(w, http.StatusOK, "Cleanup completed", map[string]interface{}{
			"deleted_tenants":   deleted,
			"cleaned_pending_subs": res.RowsAffected(),
		})
	}
}

// ─────────────────────────────────────────────
// Start pending-cleanup background goroutine (F015)
// Runs every 15 minutes while service is up
// ─────────────────────────────────────────────

func startPendingCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				cleanupPendingTenants(context.Background())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	slog.Info("Pending cleanup worker started (every 15 min)")
}

// ─────────────────────────────────────────────
// F034: Add-on Wallet & Pricing
// ─────────────────────────────────────────────

// GET /admin/addon-prices — list all addon price configs (superadmin)
// handleAdminAddonPrices GET: F057 AC-4 consolidated — reads from available_features.
// Legacy addon_prices table is kept for backward compat but no longer written to.
func handleAdminAddonPrices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT feature_key, feature_name, description, category,
		       is_addon, addon_price_cents, addon_unit, default_enabled
		FROM available_features
		WHERE is_addon = true
		ORDER BY feature_key
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query addon prices", err)
		return
	}
	defer rows.Close()
	type ap struct {
		Key         string   `json:"addon_key"`
		Name        string   `json:"feature_name"`
		Price       int64    `json:"price_cents"`
		Unit        string   `json:"unit"`
		Description string   `json:"description"`
		IsActive    bool     `json:"is_active"`
		DefaultEna  []string `json:"default_enabled"`
	}
	var list []ap
	for rows.Next() {
		var a ap
		var price int64
		var unit, desc string
		var defaultEna []string
		var category string
		var isAddon bool
		if rows.Scan(&a.Key, &a.Name, &desc, &category, &isAddon, &price, &unit, &defaultEna) == nil {
			a.Price = price
			a.Unit = unit
			a.Description = desc
			a.DefaultEna = defaultEna
			list = append(list, a)
		}
	}
	response.JSON(w, http.StatusOK, "Addon prices retrieved", list)
}

// PATCH /admin/addon-prices/{key} — update one addon price (superadmin)
func handleAdminAddonPricesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/addon-prices/")
	if key == "" || strings.Contains(key, "/") {
		response.Error(w, http.StatusBadRequest, "Missing addon_key", nil)
		return
	}
	var req struct {
		Price       *int64  `json:"price_cents"`
		Unit        *string `json:"unit"`
		IsActive    *bool   `json:"is_active"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}
	ctx := r.Context()
	setParts := "updated_at = NOW()"
	args := []any{}
	argIdx := 1
	if req.Price != nil {
		setParts += fmt.Sprintf(", price_cents = $%d", argIdx)
		args = append(args, *req.Price)
		argIdx++
	}
	if req.Unit != nil {
		setParts += fmt.Sprintf(", unit = $%d", argIdx)
		args = append(args, *req.Unit)
		argIdx++
	}
	if req.IsActive != nil {
		setParts += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Description != nil {
		setParts += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *req.Description)
		argIdx++
	}
	args = append(args, key)
	query := fmt.Sprintf("UPDATE available_features SET %s WHERE feature_key = $%d", setParts, argIdx)
	_, err := DB.Exec(ctx, query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Update failed", err)
		return
	}
	response.JSON(w, http.StatusOK, "Addon price updated", map[string]interface{}{"addon_key": key})
}

// GET /wallet — get current tenant wallet balance + transactions
func handleWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant", nil)
		return
	}
	ctx := r.Context()
	var balance int64
	err := DB.QueryRow(ctx, `SELECT balance_cents FROM wallet_credits WHERE tenant_id = $1`, tenantID).Scan(&balance)
	if err != nil {
		balance = 0
	}
	rows, err := DB.Query(ctx, `SELECT id, amount_cents, transaction_type, reference, description, created_at FROM wallet_transactions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 20`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query transactions", err)
		return
	}
	defer rows.Close()
	type tx struct {
		ID       int    `json:"id"`
		Amount   int64  `json:"amount_cents"`
		Type     string `json:"transaction_type"`
		Ref      string `json:"reference"`
		Desc     string `json:"description,omitempty"`
		Time     string `json:"created_at"`
	}
	var txs []tx
	for rows.Next() {
		var t tx
		var t2 time.Time
		if rows.Scan(&t.ID, &t.Amount, &t.Type, &t.Ref, &t.Desc, &t2) == nil {
			t.Time = t2.Format(time.RFC3339)
			txs = append(txs, t)
		}
	}
	response.JSON(w, http.StatusOK, "Wallet retrieved", map[string]interface{}{
		"balance_cents": balance,
		"transactions":  txs,
	})
}

// POST /wallet/topup — create Xendit invoice for wallet topup
func handleWalletTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	// FIX #3: Use X-Tenant-ID (consistent with all other handlers), not X-User-Id
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant", nil)
		return
	}
	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountCents < 10000 {
		response.Error(w, http.StatusBadRequest, "Invalid amount (min Rp 10.000)", nil)
		return
	}
	desc := "Top-up Wallet Credit"
	curr := "IDR"
	ctx := r.Context()
	// FIX #4: Use UUID for external_id (unpredictable, not UnixNano)
	invoiceReq := invoice.CreateInvoiceRequest{
		ExternalId:     uuid.NewString() + "-wallet-topup-" + tenantID,
		Amount:         float64(req.AmountCents),
		Description:    &desc,
		Currency:       &curr,
	}
	xClient, errXc := getTenantXenditClient(ctx, tenantID)
	if errXc != nil {
		slog.Error("Failed to get xendit client for tenant", "tenant_id", tenantID, "error", errXc)
		response.Error(w, http.StatusInternalServerError, "Payment provider not configured", nil)
		return
	}
	invoiceResp, _, err := xClient.InvoiceApi.CreateInvoice(context.Background()).CreateInvoiceRequest(invoiceReq).Execute()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create invoice", err)
		return
	}
	response.JSON(w, http.StatusOK, "Topup invoice created", map[string]interface{}{
		"invoice_url": invoiceResp.InvoiceUrl,
		"external_id": invoiceResp.ExternalId,
	})
}

func cleanupPendingTenants(ctx context.Context) {
	rows, err := DB.Query(ctx, `
		SELECT t.id FROM tenants t
		JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE ts.status = 'pending' AND ts.pending_expires_at < NOW()
	`)
	if err != nil {
		slog.Error("Pending cleanup: failed to query", "error", err)
		return
	}
	defer rows.Close()

	deleted := 0
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
			if err == nil {
				deleted++
			}
		}
	}
	if deleted > 0 {
		slog.Info("Pending cleanup worker ran", "deleted", deleted)
	}
}

// ─────────────────────────────────────────────
// Superadmin: Per-Tenant Quota Dashboard (F025, Task 2.8)
// GET /admin/quota/{tenant_id}
// Returns current period quota usage + plan limits for a tenant.
// Superadmin-only (caller responsible — API Gateway enforces).
// ─────────────────────────────────────────────

func handleAdminQuotaUsage(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID := strings.TrimPrefix(r.URL.Path, "/admin/quota/")
	if tenantID == "" || strings.Contains(tenantID, "/") {
		response.Error(w, http.StatusBadRequest, "tenant_id required", nil)
		return
	}

	ctx := r.Context()

	// Get plan features (limits) for tenant
	plan, _ := auth.GetPlanFeatures(ctx, tenantID)

	// Get current period counters (YYYYMM, e.g. "202606")
	period := time.Now().UTC().Format("200601")

	rows, err := DB.Query(ctx,
		`SELECT feature_key, count, reset_at
		 FROM quota_counters
		 WHERE tenant_id = $1 AND period_yyyymm = $2
		 ORDER BY feature_key ASC`,
		tenantID, period)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query counters", err)
		return
	}
	defer rows.Close()

	type counter struct {
		Feature string `json:"feature"`
		Used    int64  `json:"used"`
		ResetAt string `json:"reset_at"`
	}
	counters := []counter{}
	for rows.Next() {
		var c counter
		var resetAt time.Time
		if err := rows.Scan(&c.Feature, &c.Used, &resetAt); err != nil {
			continue
		}
		c.ResetAt = resetAt.Format(time.RFC3339)
		counters = append(counters, c)
	}

	response.JSON(w, http.StatusOK, "Quota usage retrieved", map[string]interface{}{
		"tenant_id": tenantID,
		"tier":      plan.Tier,
		"plan_name": plan.PlanName,
		"period":    period,
		"limits":    plan, // full struct: max_users, max_ai_text, etc.
		"usage":     counters,
	})
}



// ─────────────────────────────────────────────
// F053: Addon Purchase Flow
// GET /addon-marketplace — all available addons with price + has_addon for this tenant
// POST /addons/purchase — buy an addon (wallet deducted)
// GET /addons — list this tenant's active addons
// ─────────────────────────────────────────────

// GET /addon-marketplace
func handleAddonMarketplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	// Get all addon features
	rows, err := DB.Query(ctx,
		`SELECT af.feature_key, af.feature_name, af.description, af.category,
		        af.addon_price_cents, af.addon_unit, af.is_addon,
		        ta.status, ta.expires_at, ta.purchase_price_cents
		 FROM available_features af
		 LEFT JOIN tenant_addons ta ON ta.addon_key = af.feature_key
		        AND ta.tenant_id = $1
		 WHERE af.is_addon = true
		 ORDER BY af.category, af.feature_key`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query marketplace", err)
		return
	}
	defer rows.Close()

	type marketplaceItem struct {
		Key                string  `json:"addon_key"`
		Name               string  `json:"feature_name"`
		Description        string  `json:"description"`
		Category           string  `json:"category"`
		PriceCents        int64   `json:"price_cents"`
		Unit               string  `json:"addon_unit"`
		HasAddon           bool    `json:"has_addon"`
		AddonStatus        *string `json:"addon_status,omitempty"`
		ExpiresAt          *string `json:"expires_at,omitempty"`
		PurchasePriceCents *int64  `json:"purchase_price_cents,omitempty"`
	}

	var items []marketplaceItem
	for rows.Next() {
		var m marketplaceItem
		var expiresAt *time.Time
		var addonStatus *string
		var purchasePrice *int64
		if err := rows.Scan(&m.Key, &m.Name, &m.Description, &m.Category,
			&m.PriceCents, &m.Unit, &m.HasAddon,
			&addonStatus, &expiresAt, &purchasePrice); err != nil {
			continue
		}
		if addonStatus != nil && *addonStatus != "" {
			m.HasAddon = true
			m.AddonStatus = addonStatus
			m.PurchasePriceCents = purchasePrice
			if expiresAt != nil {
				ea := expiresAt.Format(time.RFC3339)
				m.ExpiresAt = &ea
			}
		}
		items = append(items, m)
	}

	response.JSON(w, http.StatusOK, "Addon marketplace", map[string]any{"addons": items})
}

// POST /addons/purchase
func handlePurchaseAddon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	var req struct {
		AddonKey string `json:"addon_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AddonKey == "" {
		response.Error(w, http.StatusBadRequest, "addon_key required", nil)
		return
	}

	// 1. Verify addon exists and is active
	var price int64
	var unit string
	err := DB.QueryRow(ctx,
		`SELECT addon_price_cents, addon_unit FROM available_features
		 WHERE feature_key = $1 AND is_addon = true`,
		req.AddonKey).Scan(&price, &unit)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Addon not found", nil)
		return
	}

	// 2. Check if already has active addon
	var existingStatus string
	err = DB.QueryRow(ctx,
		`SELECT status FROM tenant_addons
		 WHERE tenant_id = $1 AND addon_key = $2
		   AND status = 'active'
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		tenantID, req.AddonKey).Scan(&existingStatus)
	if err == nil {
		response.Error(w, http.StatusConflict, "Addon already active", nil)
		return
	}

	// 3. Deduct wallet (after referral discount)
	var addonFinalPrice = price
	var refAid *int
	_ = DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&refAid)
	if refAid != nil {
		var dpct float64
		_ = DB.QueryRow(ctx, `SELECT COALESCE(discount_percent,0) FROM referral_config WHERE id=1`).Scan(&dpct)
		if dpct > 0 {
			disc := price * int64(dpct) / 100
			addonFinalPrice = maxInt64(0, price-disc)
		}
	}
	if addonFinalPrice > 0 {
		if !auth.CheckWalletBalance(ctx, tenantID, addonFinalPrice) {
			response.JSON(w, http.StatusPaymentRequired, "Insufficient wallet balance. Please top up.", map[string]any{
				"wallet_url": walletEndpoint,
			})
			return
		}
		if err := auth.DeductWalletBalance(ctx, tenantID, addonFinalPrice,
			"addon_purchase:"+req.AddonKey,
			"Pembelian addon: "+req.AddonKey); err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to deduct wallet", err)
			return
		}
	}

	// 4. Calculate expiry (1 month from now)
	expiresAt := time.Now().AddDate(0, 1, 0)

	// 5. Upsert tenant_addons
	_, err = DB.Exec(ctx,
		`INSERT INTO tenant_addons (tenant_id, addon_key, status, purchased_at, expires_at, purchase_price_cents)
		 VALUES ($1, $2, 'active', NOW(), $3, $4)
		 ON CONFLICT (tenant_id, addon_key)
		 DO UPDATE SET status='active', purchased_at=NOW(), expires_at=$3, purchase_price_cents=$4`,
		tenantID, req.AddonKey, expiresAt, addonFinalPrice)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to activate addon", err)
		return
	}

	// 7. F054: Affiliate commission for addon purchases — from actual paid amount (addonFinalPrice)
	if refAid != nil && addonFinalPrice > 0 {
		var cpct float64
		_ = DB.QueryRow(ctx, `SELECT COALESCE(commission_percent,0) FROM referral_config WHERE id=1`).Scan(&cpct)
		if cpct > 0 {
			comm := addonFinalPrice * int64(cpct) / 100
			if comm > 0 {
				txID := "addon:" + req.AddonKey + ":" + uuid.NewString()[:8]
				_, _ = DB.Exec(ctx,
					`INSERT INTO affiliate_earnings (affiliate_id,tenant_id,invoice_id,amount_cents,commission_rate_percent,transaction_type,description)
					 VALUES ($1,$2,$3,$4,$5,'addon_purchase',$6)`,
					*refAid, tenantID, txID, comm, int(cpct), "Addon: "+req.AddonKey)
				_, _ = DB.Exec(ctx, `UPDATE affiliates SET cash_balance_cents = cash_balance_cents + $1 WHERE id = $2`, comm, *refAid)
			}
		}
	}

	// 6. Invalidate addon cache
	auth.InvalidateAddonCache(ctx, tenantID, req.AddonKey)

	response.JSON(w, http.StatusOK, "Addon purchased", map[string]any{
		"addon_key":  req.AddonKey,
		"expires_at": expiresAt.Format(time.RFC3339),
		"price":      price,
	})
}

// GET /addons — list tenant's active (and recent) addons
func handleMyAddons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	rows, err := DB.Query(ctx,
		`SELECT ta.addon_key, af.feature_name, ta.status,
		        ta.purchased_at, ta.expires_at, ta.auto_renew,
		        ta.purchase_price_cents, af.addon_unit
		 FROM tenant_addons ta
		 JOIN available_features af ON af.feature_key = ta.addon_key
		 WHERE ta.tenant_id = $1
		 ORDER BY ta.created_at DESC`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query addons", err)
		return
	}
	defer rows.Close()

	type addonItem struct {
		Key                string  `json:"addon_key"`
		Name               string  `json:"feature_name"`
		Status             string  `json:"status"`
		PurchasedAt        string  `json:"purchased_at"`
		ExpiresAt          *string `json:"expires_at,omitempty"`
		AutoRenew          bool    `json:"auto_renew"`
		PurchasePriceCents int64   `json:"purchase_price_cents"`
		Unit               string  `json:"addon_unit"`
	}
	var items []addonItem
	for rows.Next() {
		var a addonItem
		var expiresAt *time.Time
		if err := rows.Scan(&a.Key, &a.Name, &a.Status,
			&a.PurchasedAt, &expiresAt, &a.AutoRenew,
			&a.PurchasePriceCents, &a.Unit); err != nil {
			continue
		}
		if expiresAt != nil {
			ea := expiresAt.Format(time.RFC3339)
			a.ExpiresAt = &ea
		}
		items = append(items, a)
	}

	response.JSON(w, http.StatusOK, "My addons", map[string]any{"addons": items})
}

// ─────────────────────────────────────────────
// F036: Lifetime Affiliate & Leaderboard
// ─────────────────────────────────────────────

func handleAffiliateLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	rows, err := DB.Query(context.Background(), `
		SELECT 
			u.full_name, 
			COUNT(DISTINCT ae.tenant_id) as total_closing, 
			SUM(ae.amount_cents) as total_revenue
		FROM affiliate_earnings ae
		JOIN affiliates a ON ae.affiliate_id = a.id
		JOIN users u ON a.user_id = u.id
		GROUP BY a.id, u.full_name
		ORDER BY total_revenue DESC
		LIMIT 10
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query leaderboard", err)
		return
	}
	defer rows.Close()

	type leader struct {
		Name         string `json:"name"`
		TotalClosing int    `json:"total_closing"`
		TotalRevenue int64  `json:"total_revenue_cents"`
	}
	var leaders []leader
	for rows.Next() {
		var l leader
		var rawName string
		if err := rows.Scan(&rawName, &l.TotalClosing, &l.TotalRevenue); err == nil {
			// Masking name (e.g. "Budi Santoso" -> "Budi S.")
			parts := strings.Split(rawName, " ")
			if len(parts) > 1 {
				l.Name = parts[0] + " " + string(parts[1][0]) + "."
			} else {
				l.Name = parts[0]
			}
			leaders = append(leaders, l)
		}
	}

	response.JSON(w, http.StatusOK, "Leaderboard retrieved", leaders)
}

func handleAffiliateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get("X-User-Id")
	
	var affID int
	var refCode string
	var balance, earnings int64
	err := DB.QueryRow(context.Background(), "SELECT id, referral_code, cash_balance_cents, total_earnings_cents FROM affiliates WHERE user_id = $1", userID).Scan(&affID, &refCode, &balance, &earnings)
	if err != nil {
		response.JSON(w, http.StatusOK, "Not an affiliate", map[string]interface{}{"is_affiliate": false})
		return
	}

	// Fetch recent earnings (last 50)
	rows, err2 := DB.Query(context.Background(),
		`SELECT ae.id, ae.tenant_id, ae.invoice_id, ae.amount_cents, ae.commission_rate_percent, ae.transaction_type, ae.created_at,
		        t.name as tenant_name
		 FROM affiliate_earnings ae
		 LEFT JOIN tenants t ON t.id = ae.tenant_id
		 WHERE ae.affiliate_id = $1
		 ORDER BY ae.created_at DESC
		 LIMIT 50`, affID)
	if err2 != nil {
		response.JSON(w, http.StatusOK, "Profile retrieved", map[string]interface{}{
			"is_affiliate": true,
			"affiliate_id": affID,
			"referral_code": refCode,
			"cash_balance_cents": balance,
			"total_earnings_cents": earnings,
			"earnings": []interface{}{},
		})
		return
	}
	defer rows.Close()

	type earning struct {
		ID              string    `json:"id"`
		TenantID        string    `json:"tenant_id"`
		TenantName      string    `json:"tenant_name"`
		InvoiceID       string    `json:"invoice_id"`
		AmountCents     int64     `json:"amount_cents"`
		CommissionRate  int       `json:"commission_rate_percent"`
		TransactionType string    `json:"transaction_type"`
		CreatedAt       time.Time `json:"created_at"`
	}
	var earningsList []earning
	for rows.Next() {
		var e earning
		var eid int
		if err2 := rows.Scan(&eid, &e.TenantID, &e.InvoiceID, &e.AmountCents, &e.CommissionRate, &e.TransactionType, &e.CreatedAt, &e.TenantName); err2 == nil {
			e.ID = strconv.Itoa(eid)
			earningsList = append(earningsList, e)
		}
	}

	response.JSON(w, http.StatusOK, "Affiliate profile retrieved", map[string]interface{}{
		"is_affiliate":        true,
		"affiliate_id":        affID,
		"referral_code":       refCode,
		"referral_link":       "https://wch.id/r/" + refCode,
		"cash_balance_cents":  balance,
		"total_earnings_cents": earnings,
		"earnings":            earningsList,
	})
}

func handleAffiliateRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get("X-User-Id")

	var existing int
	DB.QueryRow(context.Background(), "SELECT id FROM affiliates WHERE user_id = $1", userID).Scan(&existing)
	if existing > 0 {
		response.Error(w, http.StatusBadRequest, "Already registered", nil)
		return
	}

	refCode := "AGEN-" + strings.ToUpper(uuid.NewString()[:6])
	_, err := DB.Exec(context.Background(), "INSERT INTO affiliates (user_id, referral_code) VALUES ($1, $2)", userID, refCode)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Registration failed", err)
		return
	}

	response.JSON(w, http.StatusOK, "Registered as affiliate", map[string]string{"referral_code": refCode})
}

func handleAffiliateWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get("X-User-Id")

	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountCents < 10000000 { // min Rp 100.000
		response.Error(w, http.StatusBadRequest, "Minimum withdraw Rp 100.000", nil)
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", err)
		return
	}
	defer tx.Rollback(ctx)

	var affID int
	err = tx.QueryRow(ctx, "UPDATE affiliates SET cash_balance_cents = cash_balance_cents - $1 WHERE user_id = $2 AND cash_balance_cents >= $1 RETURNING id", req.AmountCents, userID).Scan(&affID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Insufficient balance", err)
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO affiliate_withdrawals (affiliate_id, amount_cents, status) VALUES ($1, $2, 'pending')", affID, req.AmountCents)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to record withdrawal", err)
		return
	}

	tx.Commit(ctx)
	response.JSON(w, http.StatusOK, "Withdrawal requested", nil)
}

func handleAffiliateRedeemReferral(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req struct {
		ReferralCode string `json:"referral_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ReferralCode == "" {
		response.Error(w, http.StatusBadRequest, "Referral code required", nil)
		return
	}

	var affID int
	err := DB.QueryRow(r.Context(), "SELECT id FROM affiliates WHERE referral_code = $1", req.ReferralCode).Scan(&affID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid referral code", nil)
		return
	}

	tx, err := DB.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB error", err)
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), "UPDATE tenants SET referred_by_affiliate_id = $1 WHERE id = $2 AND referred_by_affiliate_id IS NULL", affID, tenantID)
	if err != nil {
		tx.Rollback(r.Context())
		response.Error(w, http.StatusInternalServerError, "Failed to apply referral", err)
		return
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO affiliate_referrals (affiliate_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		affID, tenantID)
	if err != nil {
		tx.Rollback(r.Context())
		response.Error(w, http.StatusInternalServerError, "Failed to record referral", err)
		return
	}

	tx.Commit(r.Context())
	response.JSON(w, http.StatusOK, "Referral applied successfully", nil)
}

func handleAffiliateReferrals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get("X-User-Id")
	ctx := r.Context()

	var affID int
	err := DB.QueryRow(ctx, "SELECT id FROM affiliates WHERE user_id = $1", userID).Scan(&affID)
	if err != nil {
		response.JSON(w, http.StatusOK, "Not an affiliate", map[string]interface{}{"referrals": []interface{}{}})
		return
	}

	rows, err2 := DB.Query(ctx,
		`SELECT ar.id, ar.tenant_id, ar.referred_at, ar.first_purchase_at, t.name as tenant_name
		 FROM affiliate_referrals ar
		 LEFT JOIN tenants t ON t.id = ar.tenant_id
		 WHERE ar.affiliate_id = $1
		 ORDER BY ar.referred_at DESC`, affID)
	if err2 != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query referrals", err2)
		return
	}
	defer rows.Close()

	type referral struct {
		ID            string     `json:"id"`
		TenantID      string     `json:"tenant_id"`
		TenantName    string     `json:"tenant_name"`
		ReferredAt    time.Time  `json:"referred_at"`
		FirstPurchase *time.Time `json:"first_purchase_at"`
	}
	var referrals []referral
	for rows.Next() {
		var r referral
		var tid string
		if err := rows.Scan(&r.ID, &tid, &r.ReferredAt, &r.FirstPurchase, &r.TenantName); err == nil {
			r.TenantID = tid
			referrals = append(referrals, r)
		}
	}
	response.JSON(w, http.StatusOK, "Referrals retrieved", referrals)
}

func handleAffiliateEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get("X-User-Id")
	ctx := r.Context()

	var affID int
	err := DB.QueryRow(ctx, "SELECT id FROM affiliates WHERE user_id = $1", userID).Scan(&affID)
	if err != nil {
		response.JSON(w, http.StatusOK, "Not an affiliate", map[string]interface{}{"earnings": []interface{}{}})
		return
	}

	rows, err2 := DB.Query(ctx,
		`SELECT ae.id, ae.tenant_id, ae.invoice_id, ae.amount_cents, ae.commission_rate_percent,
		        ae.transaction_type, ae.description, ae.created_at, t.name as tenant_name
		 FROM affiliate_earnings ae
		 LEFT JOIN tenants t ON t.id = ae.tenant_id
		 WHERE ae.affiliate_id = $1
		 ORDER BY ae.created_at DESC`, affID)
	if err2 != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query earnings", err2)
		return
	}
	defer rows.Close()

	type earning struct {
		ID              string    `json:"id"`
		TenantID        string    `json:"tenant_id"`
		TenantName      string    `json:"tenant_name"`
		InvoiceID       string    `json:"invoice_id"`
		AmountCents     int64     `json:"amount_cents"`
		CommissionRate  int       `json:"commission_rate_percent"`
		TransactionType string    `json:"transaction_type"`
		Description     string    `json:"description"`
		CreatedAt       time.Time `json:"created_at"`
	}
	var earningsList []earning
	for rows.Next() {
		var e earning
		var eid int
		if err := rows.Scan(&eid, &e.TenantID, &e.InvoiceID, &e.AmountCents, &e.CommissionRate, &e.TransactionType, &e.Description, &e.CreatedAt, &e.TenantName); err == nil {
			e.ID = strconv.Itoa(eid)
			earningsList = append(earningsList, e)
		}
	}
	response.JSON(w, http.StatusOK, "Earnings retrieved", earningsList)
}

func handleAdminReferralConfig(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		var discountPct, commissionPct, minPurchase, maxCommission float64
		var isActive bool
		var linkBase string
		err := DB.QueryRow(ctx, `
			SELECT COALESCE(discount_percent,10), COALESCE(commission_percent,10),
			       COALESCE(min_purchase_cents,0), COALESCE(max_commission_cents,0),
			       COALESCE(is_active,true), COALESCE(referral_link_base,'wch.id/r')
			FROM referral_config WHERE id = 1
		`).Scan(&discountPct, &commissionPct, &minPurchase, &maxCommission, &isActive, &linkBase)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load config", err)
			return
		}
		response.JSON(w, http.StatusOK, "Referral config loaded", map[string]interface{}{
			"discount_percent":     discountPct,
			"commission_percent":   commissionPct,
			"min_purchase_cents":   minPurchase,
			"max_commission_cents": maxCommission,
			"is_active":            isActive,
			"referral_link_base":   linkBase,
		})

	case http.MethodPut, http.MethodPost:
		var req struct {
			DiscountPercent    float64 `json:"discount_percent"`
			CommissionPercent  float64 `json:"commission_percent"`
			MinPurchaseCents   int64   `json:"min_purchase_cents"`
			MaxCommissionCents int64   `json:"max_commission_cents"`
			IsActive           bool    `json:"is_active"`
			ReferralLinkBase   string  `json:"referral_link_base"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid body", err)
			return
		}
		if req.DiscountPercent < 0 || req.DiscountPercent > 100 || req.CommissionPercent < 0 || req.CommissionPercent > 100 {
			response.Error(w, http.StatusBadRequest, "Percentage must be 0-100", nil)
			return
		}

		_, err := DB.Exec(ctx, `
			INSERT INTO referral_config (id, discount_percent, commission_percent, min_purchase_cents, max_commission_cents, is_active, referral_link_base, updated_at)
			VALUES (1, $1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id)
			DO UPDATE SET discount_percent = EXCLUDED.discount_percent,
			              commission_percent = EXCLUDED.commission_percent,
			              min_purchase_cents = EXCLUDED.min_purchase_cents,
			              max_commission_cents = EXCLUDED.max_commission_cents,
			              is_active = EXCLUDED.is_active,
			              referral_link_base = EXCLUDED.referral_link_base,
			              updated_at = NOW()
		`, req.DiscountPercent, req.CommissionPercent, req.MinPurchaseCents, req.MaxCommissionCents, req.IsActive, req.ReferralLinkBase)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update config", err)
			return
		}
		slog.Info("Referral config updated", "discount", req.DiscountPercent, "commission", req.CommissionPercent)
		response.JSON(w, http.StatusOK, "Referral config updated", map[string]interface{}{
			"discount_percent":     req.DiscountPercent,
			"commission_percent":   req.CommissionPercent,
			"min_purchase_cents":   req.MinPurchaseCents,
			"max_commission_cents": req.MaxCommissionCents,
			"is_active":            req.IsActive,
			"referral_link_base":   req.ReferralLinkBase,
		})

	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

// ─────────────────────────────────────────────
// F044: Campaign Licenses (B2B)
// ─────────────────────────────────────────────

func handleAdminLicenses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	usedFilter := r.URL.Query().Get("used")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	query := `
		SELECT id::text, license_key, COALESCE(program_name, ''), COALESCE(election_type, ''),
		       COALESCE(max_voters, base_quota, 5000), COALESCE(wargame_tokens, 0),
		       COALESCE(validity_days, 365), is_used, used_by_tenant::text, created_at, used_at
		FROM campaign_licenses
		WHERE 1=1
	`

	if usedFilter == "true" {
		query += " AND is_used = true"
	} else if usedFilter == "false" {
		query += " AND is_used = false"
	}

	query += " ORDER BY created_at DESC LIMIT $1"

	rows, err := DB.Query(ctx, query, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch licenses", err)
		return
	}
	defer rows.Close()

	var licenses []map[string]interface{}
	for rows.Next() {
		var id, key, programName, electionType string
		var maxVoters, wargameTokens, validityDays int
		var isUsed bool
		var usedByTenantID *string
		var createdAt time.Time
		var usedAt *time.Time

		if err := rows.Scan(&id, &key, &programName, &electionType, &maxVoters, &wargameTokens, &validityDays, &isUsed, &usedByTenantID, &createdAt, &usedAt); err != nil {
			continue
		}

		licenses = append(licenses, map[string]interface{}{
			"id":                id,
			"license_key":       key,
			"program_name":      programName,
			"election_type":     electionType,
			"max_voters":        maxVoters,
			"wargame_tokens":    wargameTokens,
			"validity_days":     validityDays,
			"is_used":          isUsed,
			"used_by_tenant_id": usedByTenantID,
			"created_at":        createdAt.Format(time.RFC3339),
			"used_at":           formatTimePtr(usedAt),
		})
	}

	if licenses == nil {
		licenses = []map[string]interface{}{}
	}

	response.JSON(w, http.StatusOK, "Licenses retrieved", licenses)
}

func handleAdminGenerateLicenses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, "Superadmin only", nil)
		return
	}

	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req struct {
		ElectionType  string `json:"election_type"`
		MaxVoters    int    `json:"max_voters"`
		WargameTokens int   `json:"wargame_tokens"`
		ValidityDays int    `json:"validity_days"`
		Quantity     int    `json:"quantity"`
		ProgramName  string `json:"program_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if req.ElectionType == "" {
		req.ElectionType = "pilkada"
	}
	if req.MaxVoters == 0 {
		req.MaxVoters = 10000
	}
	if req.ValidityDays == 0 {
		req.ValidityDays = 365
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	if req.Quantity > 50 {
		req.Quantity = 50
	}

	ctx := r.Context()
	var keys []map[string]interface{}

	for i := 0; i < req.Quantity; i++ {
		licenseKey := "WCH-" + strings.ToUpper(uuid.NewString()[:8]) + "-" + strings.ToUpper(req.ElectionType[:4])

		var id string
		err := DB.QueryRow(ctx, `
			INSERT INTO campaign_licenses (license_key, election_type, max_voters, wargame_tokens, validity_days, program_name)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (license_key) DO UPDATE SET license_key = EXCLUDED.license_key
			RETURNING id::text
		`, licenseKey, req.ElectionType, req.MaxVoters, req.WargameTokens, req.ValidityDays, req.ProgramName).Scan(&id)

		if err == nil {
			keys = append(keys, map[string]interface{}{
				"id":          id,
				"license_key": licenseKey,
				"days":        req.ValidityDays,
			})
		}
	}

	if len(keys) == 0 {
		response.Error(w, http.StatusInternalServerError, "Failed to generate licenses", nil)
		return
	}

	slog.Info("Campaign licenses generated", "count", len(keys), "election_type", req.ElectionType)
	response.JSON(w, http.StatusOK, "Licenses generated", map[string]interface{}{
		"keys": keys,
	})
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
