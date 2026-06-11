package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────

type SubscribeReq struct {
	PlanID      string `json:"plan_id"`
	VoucherCode string `json:"voucher_code,omitempty"`
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

// ─────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────

var (
	xenditClient *xendit.APIClient
	tenantCache  = make(map[string]cachedTenant)
)

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

	xKey := os.Getenv("XENDIT_API_KEY")
	if xKey == "" {
		xKey = "xnd_development_mock_key_1234567890"
	}
	xenditClient = xendit.NewClient(xKey)

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
	mux.Handle("/admin/voucher-programs", auth.Middleware(http.HandlerFunc(handleAdminVoucherProgramsCollection)))
	mux.Handle("/admin/voucher-analytics", auth.Middleware(http.HandlerFunc(handleAdminVoucherAnalytics)))

	// Voucher link routes (public redeem + superadmin generate)
	mux.HandleFunc("/voucher/redeem-link", handleRedeemVoucherLink) // public, via signed token
	mux.Handle("/admin/voucher-links/generate", auth.Middleware(http.HandlerFunc(handleAdminGenerateVoucherLinks)))
	mux.Handle("/admin/voucher-links", auth.Middleware(http.HandlerFunc(handleAdminListVoucherLinks)))

	// Superadmin dashboard (single endpoint, aggregated)
	mux.Handle("/admin/dashboard", auth.Middleware(http.HandlerFunc(handleAdminDashboard)))

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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing Tenant ID", nil)
		return
	}

	var req SubscribeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body", nil)
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
	var voucherCodeID *string
	if req.VoucherCode != "" {
		voucherApplied, voucherCodeID = applyVoucher(ctx, req.VoucherCode, req.PlanID, tenantID, priceMonthly)
		if voucherApplied {
			slog.Info("Voucher applied", "tenant_id", tenantID, "code", req.VoucherCode)
		}
	}

	// Calculate final price
	finalPrice := priceMonthly
	if voucherApplied {
		// Recalculate with voucher discount
		var discountType string
		var discountValue int
		DB.QueryRow(ctx, `
			SELECT voucher_type, discount_value FROM voucher_programs vp
			JOIN voucher_codes vc ON vc.program_id = vp.id WHERE vc.code = $1
		`, req.VoucherCode).Scan(&discountType, &discountValue)

		if discountType == "discount_percent" {
			finalPrice = priceMonthly * int64(100-discountValue) / 100
		} else if discountType == "discount_fixed" {
			finalPrice = maxInt64(0, priceMonthly-int64(discountValue))
		}
		// free_months: finalPrice stays same, we'll handle differently
	}

	// Generate Xendit invoice
	invoiceID := uuid.NewString()
	externalID := fmt.Sprintf("INV-%s|%s", invoiceID, tenantID)

	createReq := invoice.NewCreateInvoiceRequest(externalID, float64(finalPrice)/100) // Xendit uses float but we convert sen→rupiah
	desc := fmt.Sprintf("Langganan %s — WCH Platform", planName)
	createReq.Description = &desc
	createReq.PaymentMethods = []string{"BANK_TRANSFER", "EWALLET", "QRIS"}

	resp, _, err := xenditClient.InvoiceApi.CreateInvoice(r.Context()).
		CreateInvoiceRequest(*createReq).Execute()
	if err != nil {
		slog.Error("Failed to create xendit invoice", "error", err)

		// In development mode, mock the invoice creation instead of failing
		if os.Getenv("ENV") == "development" || os.Getenv("APP_ENV") == "development" || os.Getenv("ENV") == "" {
			slog.Warn("⚠️ Development mode: Mocking Xendit invoice creation")
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
			response.JSON(w, http.StatusOK, "Subscription invoice created (MOCK)", map[string]interface{}{
				"invoice_id":   externalID,
				"payment_url":  mockInvoiceUrl,
				"amount":       finalPrice,
			})
			return
		}

		// Fallback: activate free tier or voucher subscription
		if finalPrice == 0 || req.VoucherCode != "" {
			activateSubscription(ctx, tenantID, req.PlanID, planName, 30, "voucher", voucherCodeID)
			response.JSON(w, http.StatusOK, "Subscription activated (free)", map[string]interface{}{
				"plan_id":       req.PlanID,
				"plan_name":     planName,
				"voucher_applied": voucherApplied,
			})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to create invoice", nil)
		return
	}

	// Persist invoice
	_, err = DB.Exec(ctx, `
		INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, payment_url = EXCLUDED.payment_url
	`, externalID, tenantID, req.PlanID, finalPrice, resp.InvoiceUrl, req.VoucherCode)
	if err != nil {
		slog.Warn("Failed to save invoice", "error", err)
	}

	response.JSON(w, http.StatusOK, "Invoice created", map[string]interface{}{
		"invoice_id":       *resp.Id,
		"payment_url":     resp.InvoiceUrl,
		"plan_id":         req.PlanID,
		"plan_name":       planName,
		"original_price":  priceMonthly,
		"final_price":     finalPrice,
		"voucher_applied": voucherApplied,
	})
}

// ─────────────────────────────────────────────
// Protected: Get Current Subscription
// ─────────────────────────────────────────────

func handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing Tenant ID", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing Tenant ID", nil)
		return
	}

	var req VoucherRedeemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	ctx := r.Context()

	// Lookup voucher
	var programID, programName, voucherType string
	var discountValue, durationMonths int
	var targetPlanID *string
	var expiresAt *time.Time
	var maxUses, usesCount int

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.name, vp.voucher_type, vp.discount_value, vp.duration_months,
		       vp.target_plan_id, vp.expires_at, vp.max_uses, vp.uses_count
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		LIMIT 1
	`, req.Code).Scan(&programID, &programName, &voucherType, &discountValue, &durationMonths,
		&targetPlanID, &expiresAt, &maxUses, &usesCount)

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
	effectiveMonths := durationMonths

	if voucherType == "free_months" {
		// No charge, just activate
		amountToCharge = 0
	} else if voucherType == "discount_percent" {
		amountToCharge = priceMonthly * int64(100-discountValue) / 100
	} else if voucherType == "discount_fixed" {
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

	// Activate subscription
	ticketID := activateSubscription(ctx, tenantID, planID, planName, effectiveMonths, "voucher", nil)

	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", map[string]interface{}{
		"program_name":    programName,
		"voucher_type":    voucherType,
		"discount_value":  discountValue,
		"target_plan":     planName,
		"duration_months": effectiveMonths,
		"amount_charged":  amountToCharge,
		"ticket_id":       ticketID,
	})
}

// ─────────────────────────────────────────────
// Protected: List Tickets
// ─────────────────────────────────────────────

func handleListTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing Tenant ID", nil)
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

	callbackToken := r.Header.Get("x-callback-token")
	if expectedToken := os.Getenv("XENDIT_WEBHOOK_TOKEN"); expectedToken != "" && callbackToken != expectedToken {
		slog.Warn("Unauthorized webhook attempt", "token", callbackToken)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status, _ := payload["status"].(string)
	externalID, _ := payload["external_id"].(string)

	if status != "PAID" && status != "SETTLED" {
		w.WriteHeader(http.StatusOK)
		return
	}

	parts := strings.Split(externalID, "|")
	if len(parts) < 2 {
		slog.Warn("Malformed external_id", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}
	tenantID := parts[1]

	ctx := r.Context()

	// Get plan from invoice
	var planID, voucherCode string
	var amount int64
	DB.QueryRow(ctx, "SELECT plan_id, amount, COALESCE(voucher_code,'') FROM invoices WHERE id = $1", externalID).Scan(&planID, &amount, &voucherCode)
	if planID == "" {
		planID = "lite"
	}

	// Get plan name
	var planName string
	DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", planID).Scan(&planName)

	// Update invoice
	_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", externalID)

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

	ticketID := activateSubscription(ctx, tenantID, planID, planName, periodDays, activatedBy, voucherCodeID)

	slog.Info("Payment processed", "tenant_id", tenantID, "plan", planID, "ticket_id", ticketID, "voucher_code", autoVoucherCode)

	w.WriteHeader(http.StatusOK)
}

// ─────────────────────────────────────────────
// Core: Activate Subscription + Generate Ticket
// ─────────────────────────────────────────────

func activateSubscription(ctx context.Context, tenantID, planID, planName string, periodDays int, activatedBy string, voucherCodeID *string) string {
	now := time.Now()
	periodEnd := now.AddDate(0, 0, periodDays)

	ticketNumber := generateTicketNumber()

	// 1. Upsert subscription
	_, err := DB.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, current_period_end, period_days, activated_by, voucher_code_id, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $5, $6, $7, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_tier = EXCLUDED.plan_tier,
			status = 'active',
			current_period_end = EXCLUDED.current_period_end,
			period_days = EXCLUDED.period_days,
			activated_by = EXCLUDED.activated_by,
			voucher_code_id = COALESCE(EXCLUDED.voucher_code_id, tenant_subscriptions.voucher_code_id),
			updated_at = NOW()
	`, tenantID, planID, planID, periodEnd, periodDays, activatedBy, voucherCodeID)
	if err != nil {
		slog.Error("Failed to upsert subscription", "error", err)
	}

	// 2. Update tenant plan
	_, _ = DB.Exec(ctx, "UPDATE tenants SET plan = $1 WHERE id = $2", planID, tenantID)

	// Sync Redis cache so quota gates read the correct plan tier.
	// Without this, auth.GetTenantPlan always falls back to "free" even for active subscriptions.
	auth.SetTenantPlan(ctx, tenantID, planID)

	// 3. Create subscription ticket
	var ticketID string
	err = DB.QueryRow(ctx, `
		INSERT INTO subscription_tickets (tenant_id, plan_id, plan_name, ticket_number, expires_at, activated_by, notify_wa, notify_telegram, notify_email)
		VALUES ($1, $2, $3, $4, $5, $6, true, true, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_name = EXCLUDED.plan_name,
			ticket_number = EXCLUDED.ticket_number,
			status = 'active',
			expires_at = EXCLUDED.expires_at,
			activated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, tenantID, planID, planName, ticketNumber, periodEnd, activatedBy).Scan(&ticketID)
	if err != nil {
		slog.Error("Failed to create ticket", "error", err)
		return ""
	}

	// 4. Update subscription with ticket ID
	_, _ = DB.Exec(ctx, "UPDATE tenant_subscriptions SET ticket_id = $1 WHERE tenant_id = $2", ticketID, tenantID)

	// 5. Send ticket notifications (async)
	go sendTicketNotifications(tenantID, TicketPayload{
		TicketNumber: ticketNumber,
		PlanName:     planName,
		PlanID:       planID,
		ActivatedAt:  now.Format("02 Jan 2006, 15:04 WIB"),
		ExpiresAt:    periodEnd.Format("02 Jan 2006, 15:04 WIB"),
		AmountPaid:   0,
		PaymentMethod: activatedBy,
	})

	slog.Info("Subscription activated + ticket created", "tenant_id", tenantID, "plan", planID, "ticket", ticketNumber)
	return ticketID
}

// ─────────────────────────────────────────────
// Voucher Application
// ─────────────────────────────────────────────

func applyVoucher(ctx context.Context, code, planID, tenantID string, priceMonthly int64) (bool, *string) {
	// Check voucher validity
	var programID string
	var voucherType string
	var discountValue, durationMonths int
	var targetPlanID *string

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.voucher_type, vp.discount_value, vp.duration_months, vp.target_plan_id
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
	`, code).Scan(&programID, &voucherType, &discountValue, &durationMonths, &targetPlanID)

	if err != nil {
		return false, nil
	}

	// Check if plan matches
	if targetPlanID != nil && *targetPlanID != "" && *targetPlanID != planID {
		return false, nil
	}

	// Mark as redeemed
	result, _ := DB.Exec(ctx, `
		UPDATE voucher_codes SET is_redeemed = true, used_by = $1, used_at = NOW()
		WHERE code = $2 AND is_redeemed = false
	`, tenantID, code)

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return false, nil
	}

	// Increment usage
	_, _ = DB.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

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

	resp, err := http.PostForm(waURL, data)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	planID := strings.TrimPrefix(r.URL.Path, "/admin/plans/")
	if planID == "" {
		response.Error(w, http.StatusBadRequest, "Missing plan ID", nil)
		return
	}

	var req UpdatePlanReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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

	// Active subscriptions by plan
	planRows, _ := DB.Query(ctx, `
		SELECT plan_id, COUNT(*) FROM tenant_subscriptions
		WHERE status = 'active' GROUP BY plan_id
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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
		response.Error(w, http.StatusBadRequest, "Invalid request body", err)
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
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
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

