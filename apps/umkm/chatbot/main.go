package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"core_project/shared/observability"
	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func atomicAddInt64(addr *int64, val int64) {
	atomic.AddInt64(addr, val)
}

const waGatewayDefault = "http://wa-gateway:8202"

var AIGatewayURL = "http://localhost:8002/v1/chat"
var AccountingURL = "http://localhost:8201"
var WAGatewayURL = waGatewayDefault
var redisClient *redis.Client

// waSendURL returns the full URL for posting a WhatsApp message to wa-gateway.
func waSendURL() string {
	base := WAGatewayURL
	if base == "" {
		base = waGatewayDefault
	}
	return base + "/api/wa/send"
}

// Metrics counters
var (
	chatbotMessagesProcessed int64
	chatbotLLMCalls          int64
	chatbotErrors            int64
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	// Set service URLs based on environment
	if cfg.Env == "production" || cfg.DB.Host == "postgres" {
		AccountingURL = "http://umkm-accounting:8201"
		AIGatewayURL = "http://ai-gateway:8002/v1/chat"
	}

	// Use config for WA Gateway URL (env var WA_GATEWAY_URL or default)
	if cfg.WhatsApp.GatewayURL != "" {
		WAGatewayURL = cfg.WhatsApp.GatewayURL
	}

	if err := initDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	if err := runMigrations(DB); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
	} else {
		slog.Info("Connected to Redis for Queue")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/webhook/wa", handleWAWebhook)
	mux.Handle("/metrics", observability.PrometheusHandler())

	port := cfg.Port
	if port == "" {
		port = "8203"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: observability.Middleware("umkm-chatbot")(loggingMiddleware(mux)),
	}

	startWorkerPool(100)

	slog.Info("UMKM Chatbot listening", "port", port)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "latency_ms", time.Since(start).Milliseconds())
	})
}