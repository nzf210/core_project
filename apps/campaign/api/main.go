package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/apps/campaign/api/handlers"
	"core_project/apps/campaign/api/repository"
	"core_project/shared/observability"
	"core_project/shared/sdk/config"
)

var (
	// Business metrics (Prometheus)
	campaignVolunteersActive = observability.NewGauge(
		"campaign_volunteers_active",
		"Active campaign volunteers count",
		[]string{},
	)
	campaignVotersOnboarded = observability.NewCounter(
		"campaign_voters_onboarded",
		"Total voters onboarded",
		[]string{},
	)
	campaignRealcountProgress = observability.NewGauge(
		"campaign_realcount_progress",
		"Real count progress percentage",
		[]string{},
	)
	campaignLogisticsStatus = observability.NewGauge(
		"campaign_logistics_status",
		"Logistics item count by status",
		[]string{"status"},
	)
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")
	handlers.InitConfig(cfg)
	if err := repository.InitDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer repository.CloseDB()

	if err := runMigrations(); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", handleHealth)

	// Public Routes
	mux.HandleFunc("/dashboard", handlers.HandlePublicDashboard)

	// Campaigns & Candidates
	mux.HandleFunc("/candidates", handlers.HandleCandidates)
	mux.HandleFunc("PUT /candidates/{id}/verify", handlers.HandleCandidateVerify)
	mux.HandleFunc("/campaigns", handlers.HandleCampaigns)

	// Volunteers
	mux.HandleFunc("/volunteers", handlers.HandleVolunteers)
	mux.HandleFunc("/volunteers/stats", handlers.HandleVolunteerStats)

	// Coordinator Hierarchy (F046)
	mux.HandleFunc("/coordinator/assign", handlers.HandleAssignCoordinator)
	mux.HandleFunc("/coordinator/list", handlers.HandleListCoordinators)
	mux.HandleFunc("/coordinator/hierarchy", handlers.HandleCoordinatorHierarchy)

	// Voters & Endorsements (F031)
	mux.HandleFunc("/voters", handlers.HandleVoters)
	mux.HandleFunc("/voters/stats", handlers.HandleVoterStats)
	mux.HandleFunc("/endorsements", handlers.HandleEndorsements)
	mux.HandleFunc("/endorsements/conflicts", handlers.HandleEndorsementConflicts)
	mux.HandleFunc("/endorsements/cross-level", handlers.HandleCrossLevelEndorsements)
	mux.HandleFunc("/endorsements/ocr", handlers.HandleKTPScan)

	// Real Count C1 (F032)
	mux.HandleFunc("/real-count", handlers.HandleRealCount)

	// Logistics Tracking (F033)
	mux.HandleFunc("/logistics", handlers.HandleLogistics)
	mux.HandleFunc("/logistics/distribute", handlers.HandleDistributeLogistics)
	mux.HandleFunc("/logistics/stalled", handlers.HandleStalledDistributions)

	// Cost-per-Vote / Finance (F034)
	mux.HandleFunc("/finance", handlers.HandleCampaignFinance)

	// Sentiment Issues (F036)
	mux.HandleFunc("/issues", handlers.HandleSentimentIssues)

	// Wargame Simulator (F037)
	mux.HandleFunc("/wargame/simulate", handlers.HandleSimulationWargame)

	// Fraud Reports / Peta Kerawanan (F038)
	mux.HandleFunc("/fraud-reports", handlers.HandleFraudReports)

	// Anomaly Detector / Pemilih Siluman (F039)
	mux.HandleFunc("POST /anomalies/detect", handlers.HandleAnomalyDetection)
	// F039 AC-3: Cron automation — auto-detect anomalies setiap hari jam 02:00
	mux.HandleFunc("POST /anomalies/auto-detect", handlers.HandleAutoAnomalyDetection)

	// WA Blast Micro-targeting (F040)
	mux.HandleFunc("POST /blast/target", handlers.HandleBlastTarget)

	// Gamification Leaderboard (F041)
	mux.HandleFunc("GET /leaderboard", handlers.HandleGamificationLeaderboard)

	// WA Bot FAQ RAG (F042)
	mux.HandleFunc("POST /bot/faq", handlers.HandleBotFAQ)

	// Sainte-Lague Simulator (F043)
	mux.HandleFunc("GET /wargame/sainte-lague", handlers.HandleSainteLague)

	// Payment & Licensing (F044)
	mux.HandleFunc("POST /billing/checkout", handlers.HandleBillingCheckout)
	mux.HandleFunc("POST /billing/webhook", handlers.HandleBillingWebhook)
	mux.HandleFunc("POST /licenses/redeem", handlers.HandleRedeemLicense)
	mux.HandleFunc("GET /licenses/active", handlers.HandleTenantActiveAddons)
	// In production, this should be under a different superadmin mux/port, mapped here for simplicity
	mux.HandleFunc("POST /superadmin/licenses/generate", handlers.HandleSuperadminGenerateLicense)
	mux.HandleFunc("GET /superadmin/licenses", handlers.HandleListLicenses)

	// F037 (Campaign): Affiliate Referral — extends global UMKM affiliate to Campaign
	mux.HandleFunc("POST /affiliate/redeem-referral", handlers.HandleCampaignAffiliateRedeemReferral)
	mux.HandleFunc("GET /affiliate/leaderboard", handlers.HandleCampaignAffiliateLeaderboard)

	// Regions
	mux.HandleFunc("/regions/provinces", handlers.HandleProvinces)

	// Surveys & Events
	mux.HandleFunc("/surveys", handlers.HandleSurveys)
	mux.HandleFunc("/events", handlers.HandleEvents)

	// New Features (Part 2)
	mux.HandleFunc("/users", handlers.HandleUsers)
	mux.HandleFunc("/tasks", handlers.HandleTasks)
	mux.HandleFunc("/notifications", handlers.HandleNotifications)
	mux.HandleFunc("/roles", handlers.HandleRoles)
	mux.HandleFunc("/audit-logs", handlers.HandleAuditLogs)
	mux.HandleFunc("/reports", handlers.HandleReports)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	server := &http.Server{
		Addr:           ":9002",
		Handler:        observability.Middleware("campaign-api")(loggingMiddleware(mux)),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	slog.Info("Campaign API Server listening", "port", 9002)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "latency_ms", time.Since(start).Milliseconds())
	})
}