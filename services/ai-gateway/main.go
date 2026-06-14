package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
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

	// Metrics (Prometheus-style, using atomic for thread safety)
	aiRequestsTotal  atomic.Int64
	aiCacheHits      atomic.Int64
	aiTokensInTotal  atomic.Int64
	aiTokensOutTotal atomic.Int64
	aiCostUSDTotal   float64 // Protected by costMu
	aiCostMu         sync.Mutex

	// Per-model metrics (protected by mutex for map access)
	modelRequests   map[string]int64
	modelRequestsMu sync.RWMutex
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
	mux.HandleFunc("/metrics", handleMetrics)

	server := &http.Server{
		Addr:         ":8002",
		Handler:      loggingMiddleware(rateLimitMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
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
		tenantID := r.Header.Get("X-Tenant-ID")
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

		tenantID := r.Header.Get("X-Tenant-ID")
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

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "message cannot be empty"})
		return
	}

	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
		if req.TenantID == "" {
			req.TenantID = "00000000-0000-0000-0000-000000000000"
		}
	}

	// Determine cache key (use provider:model if specified)
	cacheKey := buildCacheKey(req.Provider, req.Model, req.SystemMsg, req.Message)

	if cfg.AI.CacheEnabled {
		if cached, hit := checkCache(r.Context(), cacheKey); hit {
			slog.Info("cache hit", "key", cacheKey[:16])
			aiCacheHits.Add(1)
			aiRequestsTotal.Add(1)
			writeJSON(w, http.StatusOK, ChatResponse{
				Success:  true,
				Provider: req.Provider,
				Model:    req.Model,
				Text:     cached,
				CacheHit: true,
			})
			return
		}
	}

	// Call LLM with 3-tier fallback
	text, tokensIn, tokensOut, modelID, tier, err := callLLMWith3TierFallback(r.Context(), req)
	if err != nil {
		slog.Error("LLM call failed after all fallbacks", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("All LLM providers failed: %v", err),
		})
		return
	}

	// Parse provider:model from modelID
	provider, model := parseModelID(modelID)

	if cfg.AI.CacheEnabled {
		ttl := cfg.AI.CacheTTL
		if req.CacheTTL > 0 {
			ttl = req.CacheTTL
		}
		storeCache(r.Context(), cacheKey, text, ttl)
	}

	costUSD := estimateCost(modelID, tokensIn, tokensOut)

	// Update metrics
	aiRequestsTotal.Add(1)
	aiTokensInTotal.Add(int64(tokensIn))
	aiTokensOutTotal.Add(int64(tokensOut))
	aiCostMu.Lock()
	aiCostUSDTotal += costUSD
	aiCostMu.Unlock()
	modelRequestsMu.Lock()
	modelRequests[modelID]++
	modelRequestsMu.Unlock()

	if DB != nil {
		_, dbErr := DB.Exec(r.Context(),
			"INSERT INTO ai_usage_logs (tenant_id, model, tokens_in, tokens_out, cost_usd) VALUES ($1, $2, $3, $4, $5)",
			req.TenantID, modelID, tokensIn, tokensOut, costUSD,
		)
		if dbErr != nil {
			slog.Error("Failed to log AI usage to DB", "error", dbErr)
		}
	}

	writeJSON(w, http.StatusOK, ChatResponse{
		Success:      true,
		Provider:     provider,
		Model:        model,
		Text:         text,
		CacheHit:     false,
		TokensInput:  tokensIn,
		TokensOutput: tokensOut,
		CostUSD:      costUSD,
		FallbackTier: tier,
	})
}

func handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request payload"})
		return
	}
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "message cannot be empty"})
		return
	}
	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
		if req.TenantID == "" {
			req.TenantID = "00000000-0000-0000-0000-000000000000"
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Streaming not supported"})
		return
	}

	systemPrompt := req.SystemMsg
	if systemPrompt == "" {
		systemPrompt = "Kamu adalah asisten cerdas untuk platform WCH. Jawab dalam bahasa Indonesia yang jelas dan profesional."
	}

	// Try Anthropic stream
	stream, err := aiClient.CreateChatCompletionStream(r.Context(), openai.ChatCompletionRequest{
		Model: cfg.AI.MiniMaxModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: req.Message},
		},
	})
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
		flusher.Flush()
		return
	}
	defer stream.Close()

	// Increment quota once per stream on first successful chunk.
	firstChunk := true
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
		if err != nil {
			fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
			flusher.Flush()
			return
		}

		content := chunk.Choices[0].Delta.Content
		if content != "" {
			if firstChunk {
				firstChunk = false
			}
			data, _ := json.Marshal(map[string]string{"text": content})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "AI Gateway is healthy",
		Data: map[string]any{
			"primary_model":    cfg.AI.MiniMaxModel,
			"gemini_fallback":  "enabled",
			"models_available": len(cfg.AI.LLM.Models),
		},
	})
}

// handleListModels returns available LLM models with their capabilities
// GET /v1/models
func handleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	models := make([]map[string]any, 0, len(cfg.AI.LLM.Models))
	for _, m := range cfg.AI.LLM.Models {
		models = append(models, map[string]any{
			"id":             m.ID,
			"provider":       m.Provider,
			"model":          m.Model,
			"base_url":       m.BaseURL,
			"capability":     m.Capability,
			"context_window": m.ContextWindow,
			"cost_per_1m_in":  m.CostPer1MIn,
			"cost_per_1m_out": m.CostPer1MOut,
			"priority":       m.Priority,
			"tier":           m.Tier,
			"fallback_1":     m.FallbackTier1,
			"fallback_2":     m.FallbackTier2,
			"fallback_3":     m.FallbackTier3,
			"is_enabled":    m.IsEnabled,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   models,
	})
}

// handleMetrics returns Prometheus-compatible metrics
// GET /metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	aiCostMu.Lock()
	costTotal := aiCostUSDTotal
	aiCostMu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP ai_gateway_requests_total Total number of chat requests\n")
	b.WriteString("# TYPE ai_gateway_requests_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_requests_total %d\n", aiRequestsTotal.Load()))

	b.WriteString("# HELP ai_gateway_cache_hits_total Total semantic cache hits\n")
	b.WriteString("# TYPE ai_gateway_cache_hits_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_cache_hits_total %d\n", aiCacheHits.Load()))

	b.WriteString("# HELP ai_gateway_tokens_total Total tokens processed\n")
	b.WriteString("# TYPE ai_gateway_tokens_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_tokens_in_total %d\n", aiTokensInTotal.Load()))
	b.WriteString(fmt.Sprintf("ai_gateway_tokens_out_total %d\n", aiTokensOutTotal.Load()))

	b.WriteString("# HELP ai_gateway_cost_usd_total Total estimated cost in USD\n")
	b.WriteString("# TYPE ai_gateway_cost_usd_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_cost_usd_total %.6f\n", costTotal))

	b.WriteString("# HELP ai_gateway_model_requests_total Requests per model\n")
	b.WriteString("# TYPE ai_gateway_model_requests_total counter\n")
	modelRequestsMu.RLock()
	for model, count := range modelRequests {
		b.WriteString(fmt.Sprintf("ai_gateway_model_requests_total{model=\"%s\"} %d\n", model, count))
	}
	modelRequestsMu.RUnlock()

	_, _ = w.Write([]byte(b.String()))
}

// handleEmbeddings generates vector embeddings for RAG indexing.
// POST /v1/embeddings  { "input": "...", "model": "text-embedding-ada-002" }
func handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Input    string `json:"input"`
		Model    string `json:"model"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}
	if req.Input == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "input cannot be empty"})
		return
	}
	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
		if req.TenantID == "" {
			req.TenantID = "00000000-0000-0000-0000-000000000000"
		}
	}

	// Use OpenAI embeddings by default (ada-002, 1536 dimensions)
	// Falls back to Anthropic's embed endpoint if configured
	var emb []float64
	var err error

	apiKey := cfg.AI.OpenAIApiKey
	if apiKey != "" {
		emb, err = callOpenAIEmbeddings(r.Context(), apiKey, req.Input, req.Model)
	} else if cfg.AI.MiniMaxAPIKey != "" {
		emb, err = callMiniMaxEmbeddings(r.Context(), cfg.AI.MiniMaxAPIKey, req.Input)
	} else {
		writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Message: "No embedding provider configured"})
		return
	}

	if err != nil {
		slog.Error("Embedding generation failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Embedding generation failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{"object": "embedding", "embedding": emb, "index": 0},
		},
		"model":     req.Model,
		"tenant_id": req.TenantID,
	})
}

func callOpenAIEmbeddings(ctx context.Context, apiKey, input, model string) ([]float64, error) {
	if model == "" {
		model = "text-embedding-ada-002"
	}
	payload := map[string]string{"input": input, "model": model}
	body, _ := json.Marshal(payload)
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewBuffer(body))
	reqHTTP.Header.Set("Authorization", "Bearer "+apiKey)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return result.Data[0].Embedding, nil
}

func callMiniMaxEmbeddings(ctx context.Context, apiKey, input string) ([]float64, error) {
	// Anthropic embo1 model returns 1536-dim vectors
	payload := map[string]interface{}{
		"model": "embo1",
		"text":  input,
	}
	body, _ := json.Marshal(payload)
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.Anthropic.io/v1/embeddings", bytes.NewBuffer(body))
	reqHTTP.Header.Set("Authorization", "Bearer "+apiKey)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from Anthropic")
	}
	return result.Data[0].Embedding, nil
}

// callLLMWith3TierFallback implements 3-tier fallback chain for LLM calls
// Returns: text, tokensIn, tokensOut, modelID, tier (1-3), error
func callLLMWith3TierFallback(ctx context.Context, req ChatRequest) (string, int, int, string, int, error) {
	systemPrompt := req.SystemMsg
	if systemPrompt == "" {
		systemPrompt = "Kamu adalah asisten cerdas untuk platform WCH Indonesia. Jawab dalam bahasa Indonesia yang jelas dan profesional."
	}

	useCase := req.UseCase
	if useCase == "" {
		useCase = "general"
	}

	// Select primary model based on use_case
	primaryModel := selectModelByCapability(useCase, req.Provider, req.Model)

	// If custom base_url provided, use it
	baseURL := primaryModel.BaseURL
	if req.BaseURL != "" {
		baseURL = req.BaseURL
	}

	// Tier 1: Try primary model
	slog.Info("LLM call tier 1", "model", primaryModel.ID, "use_case", useCase)
	text, tin, tout, err := callLLM(ctx, primaryModel.Provider, primaryModel.Model, baseURL, primaryModel.APIKey, systemPrompt, req.Message)
	if err == nil {
		return text, tin, tout, primaryModel.ID, 1, nil
	}
	slog.Warn("Tier 1 failed, trying fallback 1", "model", primaryModel.ID, "error", err)

	// Tier 2: Try first fallback
	if primaryModel.FallbackTier1 != "" {
		fb1 := resolveModel(primaryModel.FallbackTier1)
		if fb1 != nil {
			slog.Info("LLM call tier 2", "model", fb1.ID)
			text, tin, tout, err := callLLM(ctx, fb1.Provider, fb1.Model, fb1.BaseURL, fb1.APIKey, systemPrompt, req.Message)
			if err == nil {
				return text, tin, tout, fb1.ID, 2, nil
			}
			slog.Warn("Tier 2 failed, trying fallback 2", "model", fb1.ID, "error", err)
		}
	}

	// Tier 3: Try second fallback
	if primaryModel.FallbackTier2 != "" {
		fb2 := resolveModel(primaryModel.FallbackTier2)
		if fb2 != nil {
			slog.Info("LLM call tier 3", "model", fb2.ID)
			text, tin, tout, err := callLLM(ctx, fb2.Provider, fb2.Model, fb2.BaseURL, fb2.APIKey, systemPrompt, req.Message)
			if err == nil {
				return text, tin, tout, fb2.ID, 3, nil
			}
			slog.Warn("Tier 3 failed, trying fallback 3", "model", fb2.ID, "error", err)
		}
	}

	// Tier 4: Last resort - try third fallback or mock response
	if primaryModel.FallbackTier3 != "" {
		fb3 := resolveModel(primaryModel.FallbackTier3)
		if fb3 != nil {
			slog.Info("LLM call tier 4 (last resort)", "model", fb3.ID)
			text, tin, tout, err := callLLM(ctx, fb3.Provider, fb3.Model, fb3.BaseURL, fb3.APIKey, systemPrompt, req.Message)
			if err == nil {
				return text, tin, tout, fb3.ID, 4, nil
			}
		}
	}

	return "", 0, 0, "", 0, fmt.Errorf("all LLM providers failed")
}

// resolveModel finds a model by its "provider:model" ID
func resolveModel(modelID string) *config.LLMModel {
	for i := range cfg.AI.LLM.Models {
		if cfg.AI.LLM.Models[i].ID == modelID {
			return &cfg.AI.LLM.Models[i]
		}
	}
	return nil
}

// callLLM makes an actual LLM API call with retries
func callLLM(ctx context.Context, provider, model, baseURL, apiKey, systemPrompt, message string) (string, int, int, error) {
	if apiKey == "" {
		return "", 0, 0, fmt.Errorf("no API key for provider: %s", provider)
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: message},
	}

	// Create client with custom base URL
	clientCfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		clientCfg.BaseURL = baseURL
	}
	client := openai.NewClientWithConfig(clientCfg)

	// Retry logic
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
		})

		if err == nil {
			return resp.Choices[0].Message.Content, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, nil
		}

		slog.Warn("LLM request failed", "provider", provider, "model", model, "attempt", i+1, "error", err)
		lastErr = err
		time.Sleep(time.Duration(500*(i+1)) * time.Millisecond) // Exponential backoff
	}

	return "", 0, 0, lastErr
}

// selectModelByCapability selects the best model based on use_case and provider override
func selectModelByCapability(useCase, providerOverride, modelOverride string) config.LLMModel {
	// If model explicitly specified, find and return it
	if modelOverride != "" {
		prov := providerOverride
		if prov == "" {
			prov = "Anthropic"
		}
		targetID := prov + ":" + modelOverride
		for _, m := range cfg.AI.LLM.Models {
			if m.ID == targetID {
				return m
			}
		}
	}

	// Try to find model matching the use_case capability
	for _, m := range cfg.AI.LLM.Models {
		if !m.IsEnabled {
			continue
		}
		if containsCapability(m.Capability, useCase) {
			return m
		}
	}

	// Fallback to first available model
	if len(cfg.AI.LLM.Models) > 0 {
		for _, m := range cfg.AI.LLM.Models {
			if m.IsEnabled {
				return m
			}
		}
	}

	// Last resort: return default Anthropic
	return config.LLMModel{
		ID:       "Anthropic:" + cfg.AI.MiniMaxModel,
		Provider: "Anthropic",
		Model:    cfg.AI.MiniMaxModel,
		BaseURL:  cfg.AI.MiniMaxBaseURL,
		APIKey:   cfg.AI.MiniMaxAPIKey,
		Tier:     1,
	}
}

// containsCapability checks if capabilities string contains the target use_case
func containsCapability(capabilities, target string) bool {
	for _, cap := range strings.Split(capabilities, ",") {
		cap = strings.TrimSpace(cap)
		if cap == target {
			return true
		}
	}
	return false
}

// parseModelID splits "provider:model" into separate parts
func parseModelID(modelID string) (string, string) {
	parts := strings.SplitN(modelID, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", modelID
}

func buildCacheKey(provider, model, system, message string) string {
	raw := provider + "|" + model + "|" + system + "|" + message
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("ai:cache:%x", hash[:16])
}

func checkCache(ctx context.Context, key string) (string, bool) {
	if Redis != nil {
		val, err := Redis.Get(ctx, key).Result()
		if err == nil && val != "" {
			return val, true
		}
	}
	return "", false
}

func storeCache(ctx context.Context, key, value string, ttlSeconds int) {
	if Redis != nil {
		Redis.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second)
	}
}

func estimateCost(modelID string, tokensIn, tokensOut int) float64 {
	// Find model in registry for accurate cost
	for _, m := range cfg.AI.LLM.Models {
		if m.ID == modelID {
			return float64(tokensIn)*m.CostPer1MIn/1_000_000 + float64(tokensOut)*m.CostPer1MOut/1_000_000
		}
	}

	// Default estimates
	switch {
	case strings.Contains(modelID, "Anthropic"):
		return float64(tokensIn)*0.30/1_000_000 + float64(tokensOut)*1.20/1_000_000
	case strings.Contains(modelID, "gemini"):
		return float64(tokensIn)*0.075/1_000_000 + float64(tokensOut)*0.30/1_000_000
	case strings.Contains(modelID, "openai"):
		return float64(tokensIn)*5.00/1_000_000 + float64(tokensOut)*15.00/1_000_000
	default:
		return 0
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}