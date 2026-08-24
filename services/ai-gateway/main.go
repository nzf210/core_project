package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"core_project/shared/observability"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
	"core_project/shared/sdk/response"
)

// =====================================================
// Request / Response Structures
// =====================================================

type ChatRequest struct {
	Provider  string `json:"provider"`            // Override provider (optional)
	Model     string `json:"model,omitempty"`    // Override specific model (optional)
	BaseURL   string `json:"base_url,omitempty"` // Override base URL (optional)
	Message   string `json:"message"`
	SystemMsg string `json:"system_msg"`
	CacheTTL  int    `json:"cache_ttl"`
	TenantID  string `json:"tenant_id"`
	// UseCase enables automatic model routing based on task type
	// Valid values: "product" (product data retrieval), "faq" (FAQ answering),
	// "general" (default, any other task)
	UseCase string `json:"use_case,omitempty"`
}

type ChatResponse struct {
	Success      bool    `json:"success"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Text         string  `json:"text"`
	CacheHit     bool    `json:"cache_hit"`
	TokensInput  int     `json:"tokens_input,omitempty"`
	TokensOutput int     `json:"tokens_output,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
	FallbackTier int     `json:"fallback_tier,omitempty"` // 1=primary, 2=secondary, 3=tertiary
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	cfg      *config.Config
	aiClient *openai.Client

	// Legacy atomic metrics (kept for backward compat)
	aiRequestsTotal  atomic.Int64
	aiCacheHits      atomic.Int64
	aiTokensInTotal  atomic.Int64
	aiTokensOutTotal atomic.Int64
	aiCostUSDTotal   float64 // Protected by costMu
	aiCostMu         sync.Mutex

	// Per-model metrics (protected by mutex for map access)
	modelRequests   map[string]int64
	modelRequestsMu sync.RWMutex

	// Business metrics (Prometheus)
	aiRequestsTotalCounter = observability.NewCounter(
		"ai_requests_total",
		"Total AI requests by provider, model, modality, and status",
		[]string{"provider", "model", "modality", "status"},
	)
	aiRequestDurationHistogram = observability.NewHistogram(
		"ai_request_duration_seconds",
		"AI request latency in seconds by provider and model",
		[]string{"provider", "model"},
		[]float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	)
)

func init() {
	modelRequests = make(map[string]int64)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg = config.LoadConfig(".env")

	if err := initDB(cfg); err != nil {
		slog.Error("Failed to initialize database", "error", err)
	} else {
		defer DB.Close()
	}

	if err := initRedis(cfg); err != nil {
		slog.Error("Failed to initialize redis", "error", err)
	} else {
		defer Redis.Close()
	}

	aiCfg := openai.DefaultConfig(cfg.AI.MiniMaxAPIKey)
	aiCfg.BaseURL = cfg.AI.MiniMaxBaseURL
	aiClient = openai.NewClientWithConfig(aiCfg)

	slog.Info("AI Gateway starting",
		"env", cfg.Env,
		"primary_model", cfg.AI.MiniMaxModel,
		"cache_enabled", cfg.AI.CacheEnabled,
		"models_loaded", len(cfg.AI.LLM.Models),
	)

	mux := http.NewServeMux()
	// Quota-enforced text routes: API Gateway forwards X-Tenant-ID header (set from
	// JWT claims). tenantContextMiddleware lifts that into r.Context() so the
	// shared auth.QuotaMiddlewareFeature (which reads TenantIDKey) can enforce.
	quota := auth.QuotaMiddlewareFeature("ai_text")
	mux.Handle("/v1/chat", tenantContextMiddleware(quota(http.HandlerFunc(handleChat))))
	mux.Handle("/v1/chat/stream", tenantContextMiddleware(quota(http.HandlerFunc(handleChatStream))))
	mux.Handle("/v1/embeddings", tenantContextMiddleware(quota(http.HandlerFunc(handleEmbeddings))))
	
	// Quota-wrapped multimodal endpoints (MOCK)
	mux.Handle("/v1/vision", tenantContextMiddleware(auth.QuotaMiddlewareFeature("ai_vision")(http.HandlerFunc(HandleVision))))
	mux.Handle("/v1/audio/transcribe", tenantContextMiddleware(auth.QuotaMiddlewareFeature("ai_audio_stt")(http.HandlerFunc(HandleTranscribe))))
	mux.Handle("/v1/audio/speak", tenantContextMiddleware(auth.QuotaMiddlewareFeature("ai_audio_tts")(http.HandlerFunc(HandleSpeak))))
	mux.Handle("/v1/image/generate", tenantContextMiddleware(auth.QuotaMiddlewareFeature("image_gen")(http.HandlerFunc(HandleGenerateImage))))

	mux.HandleFunc("/v1/models", handleListModels)
	mux.HandleFunc("/health", handleHealth)
	// Legacy text metrics kept for backward compat
	// mux.HandleFunc("/metrics", handleMetrics)
	// Prometheus metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	server := &http.Server{
		Addr:           ":8002",
		Handler:        observability.Middleware("ai-gateway")(loggingMiddleware(rateLimitMiddleware(mux))),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	slog.Info("AI Gateway listening", "port", 8002)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start AI Gateway", "error", err)
	}
}

// tenantContextMiddleware bridges the X-Tenant-ID header (set by API Gateway
// from JWT claims) into the request context under auth.TenantIDKey. Required
// because auth.QuotaMiddlewareFeature reads tenant from context, not header.
func tenantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get(response.XTenantID)
		if tenantID != "" {
			r = r.WithContext(context.WithValue(r.Context(), auth.TenantIDKey, tenantID))
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request processed", "method", r.Method, "path", r.URL.Path, "latency_ms", time.Since(start).Milliseconds())
	})
}

// Sliding window rate limiter via Redis (100 req/min)
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Redis == nil {
			next.ServeHTTP(w, r)
			return
		}

		tenantID := r.Header.Get(response.XTenantID)
		if tenantID == "" {
			tenantID = "anonymous"
		}

		key := fmt.Sprintf("rate_limit:tenant:%s", tenantID)
		now := time.Now().UnixMilli()
		windowStart := now - 60000 // 1 minute ago

		ctx := r.Context()

		pipe := Redis.TxPipeline()
		pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		countCmd := pipe.ZCard(ctx, key)
		pipe.Expire(ctx, key, 2*time.Minute)

		_, err := pipe.Exec(ctx)
		if err != nil {
			slog.Error("Rate limiter redis error", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if countCmd.Val() > 100 {
			writeJSON(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "Rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

