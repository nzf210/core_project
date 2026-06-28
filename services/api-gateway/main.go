package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

const (
	svcAuth     = "auth-service"
	svcCampaign = "campaign-api"
	svcBilling  = "billing-service"
	pathWebhook = "/webhooks"
	pathSA      = "/api/superadmin"
	pathAdmin   = "/admin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	if err := cache.InitRedis(cfg); err != nil {
		slog.Warn("Redis not available, rate limiting disabled", "error", err)
	}
	if err := db.InitDB(cfg); err != nil {
		slog.Warn("DB not available, falling back to cache-only mode", "error", err)
	}

	mux := http.NewServeMux()

	getTarget := func(service string, port string) string {
		if cfg.Env == "production" {
			return "http://" + service + ":" + port
		}
		return "http://localhost:" + port
	}

	// Public routes (Auth & Public Campaign API) — rate limited
	mux.Handle("/auth/", rateLimitMiddleware(rateLimitPublic)(http.StripPrefix("/auth", newProxy(getTarget(svcAuth, "8001")))))
	mux.Handle("/api/public/campaign/", rateLimitMiddleware(rateLimitPublic)(http.StripPrefix("/api/public/campaign", newProxy(getTarget(svcCampaign, "9002")))))

	// Uploads — public static files
	mux.Handle("/uploads/", http.StripPrefix("/uploads", newProxy(getTarget("auth-service", "8001")+"/static")))

	// Webhooks — public, lenient rate limit, signature-verified by downstream service
	// Path convention: /webhooks/<service>/<event>
	// - Xendit: POST /webhooks/xendit/{invoice.paid,subscription.activated,subscription.expired}
	// - wa-gateway: POST /webhooks/wa/{message.status,device.status} (whatsmeow internal)
	// - wa-cloud-api: POST /webhooks/wa-cloud (Meta Cloud API callbacks)
	// - n8n:    POST /webhooks/n8n/{workflow_name}  (custom workflow trigger)
	// F054: Campaign webhook — must be BEFORE the catch-all /webhooks/xendit/ route
	mux.Handle("/webhooks/xendit/campaign/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks/xendit/campaign", newProxy(getTarget(svcCampaign, "9002")+"/billing/webhook"))))
	mux.Handle("/webhooks/xendit/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix(pathWebhook, newProxy(getTarget(svcBilling, "8003")))))
	mux.Handle("/webhooks/wa/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix(pathWebhook, newProxy(getTarget("wa-gateway", "8202")))))
	mux.Handle("/webhooks/wa-cloud/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks/wa-cloud", newProxy(getTarget("wa-cloud-api", "8210")+"/webhook"))))
	mux.Handle("/webhooks/n8n/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix(pathWebhook, newProxy(getTarget("n8n", "5678")))))

	// Superadmin routes — protected with auth + role check
	// F044: Campaign Licenses routes — must come BEFORE catch-all /api/superadmin/ to avoid 404
	mux.Handle(pathSA+"/licenses", auth.Middleware(http.StripPrefix(pathSA, newTenantProxy(getTarget(svcBilling, "8003")+pathAdmin))))
	mux.Handle(pathSA+"/licenses/", auth.Middleware(http.StripPrefix(pathSA, newTenantProxy(getTarget(svcBilling, "8003")+pathAdmin))))
	mux.Handle(pathSA+"/billing/", auth.Middleware(http.StripPrefix(pathSA+"/billing", newTenantProxy(getTarget(svcBilling, "8003")+pathAdmin))))
	// Login endpoint — NO auth middleware (otherwise login itself requires auth!)
	mux.Handle(pathSA+"/login", http.StripPrefix(pathSA, newTenantProxy(getTarget(svcAuth, "8001")+"/superadmin")))
	// F064: Platform WA provider — must come BEFORE /wa/ catch-all (Go 1.22 exact match wins)
	mux.Handle(pathSA+"/wa/platform-provider", auth.Middleware(http.StripPrefix(pathSA, newTenantProxy(getTarget(svcAuth, "8001")+"/superadmin"))))
	// F063: WA Center — superadmin manages platform-level WhatsApp for REG/OTP/VERIF
	mux.Handle(pathSA+"/wa/", auth.Middleware(http.StripPrefix(pathSA, newProxy(getTarget("wa-gateway", "8202")))))
	// Catch-all superadmin routes — must be LAST
	mux.Handle(pathSA+"/", auth.Middleware(http.StripPrefix(pathSA, newTenantProxy(getTarget(svcAuth, "8001")+"/superadmin"))))
	mux.Handle(pathSA+"/n8n/", auth.Middleware(http.StripPrefix(pathSA+"/n8n", n8nProxy(getTarget("n8n", "5678")))))

	// Profile routes — user can edit own profile
	mux.Handle("/api/profile", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget(svcAuth, "8001"))))))
	mux.Handle("/api/profile/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget(svcAuth, "8001"))))))
	// /me route — lightweight GET endpoint for frontend router guard to re-sync
	// onboarding_completed, plan, role, is_frozen on every page reload.
	// Fixes the onboarding redirect loop when localStorage flags are missing.
	mux.Handle("/api/me", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget(svcAuth, "8001"))))))
	mux.Handle("/api/ai/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(auth.RequireFeature("ai")(http.StripPrefix("/api/ai", newTenantProxy(getTarget("ai-gateway", "8002"))))))))
	mux.Handle("/api/umkm/business/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/umkm/business", newTenantProxy(getTarget("umkm-business", "9005"))))))
	mux.Handle("/api/umkm/automation/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/umkm/automation", newTenantProxy(getTarget("umkm-automation", "8203"))))))
	mux.Handle("/api/umkm/chat", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/umkm", newTenantProxy(getTarget("umkm-chatbot", "8203")))))))
	// F053: Addon marketplace & purchase — proxied to billing-service (handlers at root level)
	mux.Handle("/api/umkm/addon-marketplace", auth.Middleware(tenantRateLimitMiddleware(
		http.StripPrefix("/api/umkm/addon-marketplace", newTenantProxy(getTarget(svcBilling, "8003")+"/addon-marketplace")),
	)))
	mux.Handle("/api/umkm/addons/purchase", auth.Middleware(tenantRateLimitMiddleware(
		http.StripPrefix("/api/umkm/addons/purchase", newTenantProxy(getTarget(svcBilling, "8003")+"/addons/purchase")),
	)))
	mux.Handle("/api/umkm/addons", auth.Middleware(tenantRateLimitMiddleware(
		http.StripPrefix("/api/umkm/addons", newTenantProxy(getTarget(svcBilling, "8003")+"/addons")),
	)))

	mux.Handle("/api/umkm/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/umkm", newTenantProxy(getTarget("umkm-accounting", "8201")))))))
	mux.Handle("/api/campaign/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/campaign", newTenantProxy(getTarget(svcCampaign, "9002")))))))
	mux.Handle("/api/billing/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/billing", newTenantProxy(getTarget(svcBilling, "8003"))))))
	mux.Handle("/plans", http.StripPrefix("", newProxy(getTarget(svcBilling, "8003"))))
	mux.Handle("/subscribe", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget(svcBilling, "8003"))))))
	mux.Handle("/subscription", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget(svcBilling, "8003"))))))
	mux.Handle("/voucher/redeem", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget(svcBilling, "8003"))))))
	// F036: Affiliate — all /affiliate/* routes proxy to billing-service (some are public, some require auth)
	mux.Handle("/api/public/affiliate-leaderboard", rateLimitMiddleware(rateLimitPublic)(newProxy(getTarget(svcBilling, "8003"))))
	mux.Handle("/affiliate/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/affiliate", newTenantProxy(getTarget(svcBilling, "8003"))))))
	// F054: Referral link redirect — /r/{code} → frontend register with code pre-filled
	mux.HandleFunc("/r/{code}", handleReferralLinkRedirect)
	mux.Handle("/api/wa/", auth.Middleware(tenantRateLimitMiddleware(auth.RequireFeature("chatbot")(newTenantProxy(getTarget("wa-gateway", "8202"))))))
	// F048: Validate Meta Cloud API credential directly against wa-cloud-api
	mux.Handle("/api/wa/validate", auth.Middleware(tenantRateLimitMiddleware(
		http.StripPrefix("/api/wa/validate", newTenantProxy(getTarget("wa-cloud-api", "8210")+"/validate")),
	)))
	mux.Handle("/api/notifications/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/notifications", newTenantProxy(getTarget("notification-service", "8005"))))))

	// Aggregated health check — panggil semua service & return ringkas
	mux.HandleFunc("/healthz", handleAggregatedHealthz(getTarget, cfg))

	// Prometheus metrics endpoint — aggregate dari semua service
	mux.HandleFunc("/metrics", handleAggregatedMetrics(getTarget, cfg))

	server := &http.Server{
		Addr:         ":8000",
		Handler:      corsMiddleware(ipRateLimitMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("API Gateway listening", "port", 8000)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start API Gateway", "error", err)
	}
}
