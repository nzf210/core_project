package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"core_project/shared/observability"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
)

const (
	walletEndpoint         = "/wallet"
	querySelectAffiliateID = "SELECT referred_by_affiliate_id FROM tenants WHERE id = $1"
	queryAffiliateUserID   = "SELECT id FROM affiliates WHERE user_id = $1"
	keyPlanFeatures        = "plan_features:"
	errNotAffiliate        = "Not an affiliate"
	keyWalletTopup         = "-wallet-topup-"
	timeFormatWIB          = "02 Jan 2006, 15:04 WIB"
	errDB                  = "DB error"
)

var (
	// Business metrics
	billingSubscriptionsActive = observability.NewGauge("billing_subscriptions_active", "Active subscriptions count", nil)
	billingPaymentsTotal       = observability.NewCounter("billing_payments_total", "Total payments", []string{"method", "status"})
	billingTenantsActive       = observability.NewGauge("billing_tenants_active", "Active tenants count", nil)
	billingRevenueCents        = observability.NewCounter("billing_revenue_cents", "Revenue in cents", nil)
	billingSubscriptionsNew    = observability.NewCounter("billing_subscriptions_new", "New subscriptions created", nil)
)

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

	// Metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	// Health check
	mux.HandleFunc("GET /health", handleHealth)

	// Public routes
	mux.HandleFunc("/plans", handleListPlans)
	mux.HandleFunc("/vouchers/validate", handleValidateVoucher)
	// F060: Landing page dynamic content
	mux.HandleFunc("/landing-config", handleGetLandingConfig)
	mux.HandleFunc("/landing-configs", handleGetAllLandingConfigs)

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
	// F060: Landing page content management
	mux.Handle("/admin/landing-configs", auth.Middleware(http.HandlerFunc(handleAdminListLandingConfigs)))
	mux.Handle("/admin/landing-configs/", auth.Middleware(http.HandlerFunc(handleAdminUpdateLandingConfig)))

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
		Handler: observability.Middleware("billing-service")(mux),
	}

	slog.Info("Billing Service listening", "port", 8003)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start Billing Service", "error", err)
	}
}
