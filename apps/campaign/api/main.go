package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/apps/campaign/api/handlers"
	"core_project/apps/campaign/api/repository"
	"core_project/shared/sdk/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")
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

	// Voters & Endorsements (F031)
	mux.HandleFunc("/voters", handlers.HandleVoters)
	mux.HandleFunc("/voters/stats", handlers.HandleVoterStats)
	mux.HandleFunc("/endorsements", handlers.HandleEndorsements)
	mux.HandleFunc("/endorsements/conflicts", handlers.HandleEndorsementConflicts)
	mux.HandleFunc("/endorsements/cross-level", handlers.HandleCrossLevelEndorsements)

	// Real Count C1 (F032)
	mux.HandleFunc("/real-count", handlers.HandleRealCount)

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

	server := &http.Server{
		Addr:    ":9002", // Campaign port
		Handler: loggingMiddleware(mux),
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
