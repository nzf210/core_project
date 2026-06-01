package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")
	
	if err := db.InitDB(cfg); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.CloseDB()

	if err := runMigrations(); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	repo := NewRepository()
	handlers := NewHandlers(repo, cfg.EncryptionKey)

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	// Protected routes
	apiMux := http.NewServeMux()
	
	// API Keys
	apiMux.HandleFunc("/api/v1/api-keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListAPIKeys(w, r)
		case http.MethodPost:
			handlers.CreateAPIKey(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/v1/api-keys/", handlers.DeleteAPIKey) // /api/v1/api-keys/{id}

	// Bots
	apiMux.HandleFunc("/api/v1/bots", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListBots(w, r)
		case http.MethodPost:
			handlers.CreateBot(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/v1/bots/", func(w http.ResponseWriter, r *http.Request) {
		// Handlers:
		// GET /api/v1/bots/{id}
		// PUT /api/v1/bots/{id}/status
		if r.Method == http.MethodPut {
			handlers.UpdateBotStatus(w, r)
		} else if r.Method == http.MethodGet {
			handlers.GetBot(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Orders
	apiMux.HandleFunc("/api/v1/orders", handlers.ListOrders) // GET /api/v1/orders?bot_id=...

	// Dashboard
	apiMux.HandleFunc("/api/v1/dashboard", handlers.GetDashboard)

	// Notifications
	apiMux.HandleFunc("/api/v1/notifications", handlers.ListNotifications) // GET
	apiMux.HandleFunc("/api/v1/notifications/", func(w http.ResponseWriter, r *http.Request) {
		// PUT /api/v1/notifications/{id}/read
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/read") {
			handlers.MarkNotificationRead(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Trade
	apiMux.HandleFunc("/api/v1/trade", handlers.ExecuteQuickTrade) // POST

	// Apply auth middleware to API mux
	mux.Handle("/api/v1/", auth.Middleware(apiMux))

	port := "8101"
	serverAddress := fmt.Sprintf(":%s", port)
	slog.Info(fmt.Sprintf("🚀 Starting Crypto API Service in %s mode on port %s...", cfg.Env, port))
	
	server := &http.Server{
		Addr:    serverAddress,
		Handler: loggingMiddleware(mux),
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		slog.Info("Incoming request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
		slog.Info("Request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
