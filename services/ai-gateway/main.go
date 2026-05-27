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
	"time"

	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
)

// =====================================================
// Request / Response Structures
// =====================================================

type ChatRequest struct {
	Provider  string  `json:"provider"`
	Model     string  `json:"model,omitempty"`
	BaseURL   string  `json:"base_url,omitempty"`
	Message   string  `json:"message"`
	SystemMsg string  `json:"system_msg"`
	CacheTTL  int     `json:"cache_ttl"`
	TenantID  string  `json:"tenant_id"`
}

type ChatResponse struct {
	Success      bool    `json:"success"`
	Provider     string  `json:"provider"`
	Text         string  `json:"text"`
	CacheHit     bool    `json:"cache_hit"`
	TokensInput  int     `json:"tokens_input,omitempty"`
	TokensOutput int     `json:"tokens_output,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	cfg      *config.Config
	aiClient *openai.Client
)

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

	slog.Info("🤖 AI Gateway starting",
		"env", cfg.Env,
		"primary_model", cfg.AI.MiniMaxModel,
		"cache_enabled", cfg.AI.CacheEnabled,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat", handleChat)
	mux.HandleFunc("/v1/chat/stream", handleChatStream)
	mux.HandleFunc("/health", handleHealth)

	server := &http.Server{
		Addr:         ":8002",
		Handler:      loggingMiddleware(rateLimitMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	slog.Info("AI Gateway listening", "port", 8003)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start AI Gateway", "error", err)
	}
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
			req.TenantID = "00000000-0000-0000-0000-000000000000" // Default empty UUID for unknown
		}
	}

	if req.Provider == "" {
		req.Provider = "minimax"
	}

	cacheKey := buildCacheKey(req.Provider, req.SystemMsg, req.Message)

	if cfg.AI.CacheEnabled {
		if cached, hit := checkCache(r.Context(), cacheKey); hit {
			slog.Info("cache hit", "key", cacheKey[:16])
			writeJSON(w, http.StatusOK, ChatResponse{
				Success:  true,
				Provider: req.Provider + " (cached)",
				Text:     cached,
				CacheHit: true,
			})
			return
		}
	}

	// Route to LLM with Fallback logic
	text, tokensIn, tokensOut, actualProvider, err := callLLMWithFallback(r.Context(), req)
	if err != nil {
		slog.Error("LLM call failed", "provider", req.Provider, "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("LLM provider error: %v", err),
		})
		return
	}

	if cfg.AI.CacheEnabled {
		ttl := cfg.AI.CacheTTL
		if req.CacheTTL > 0 {
			ttl = req.CacheTTL
		}
		storeCache(r.Context(), cacheKey, text, ttl)
	}

	costUSD := estimateCost(actualProvider, tokensIn, tokensOut)

	// Log to DB
	if DB != nil {
		_, dbErr := DB.Exec(r.Context(), 
			"INSERT INTO ai_usage_logs (tenant_id, model, tokens_in, tokens_out, cost_usd) VALUES ($1, $2, $3, $4, $5)",
			req.TenantID, actualProvider, tokensIn, tokensOut, costUSD,
		)
		if dbErr != nil {
			slog.Error("Failed to log AI usage to DB", "error", dbErr)
		}
	}

	writeJSON(w, http.StatusOK, ChatResponse{
		Success:      true,
		Provider:     actualProvider,
		Text:         text,
		CacheHit:     false,
		TokensInput:  tokensIn,
		TokensOutput: tokensOut,
		CostUSD:      costUSD,
	})
}

func handleChatStream(w http.ResponseWriter, r *http.Request) {
	// Simple stream implementation (fallback logic is harder in streams, omitting fallback for brevity in stream)
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request payload"})
		return
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

	// Try MiniMax stream
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
		Data: map[string]string{
			"primary_model":    cfg.AI.MiniMaxModel,
			"gemini_fallback":  "enabled",
		},
	})
}

// callLLMWithFallback implements the required retry and fallback logic
func callLLMWithFallback(ctx context.Context, req ChatRequest) (string, int, int, string, error) {
	systemPrompt := req.SystemMsg
	if systemPrompt == "" {
		systemPrompt = "Kamu adalah asisten cerdas untuk platform WCH Indonesia. Jawab dalam bahasa Indonesia yang jelas dan profesional."
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: req.Message},
	}

	useGeminiFallback := func() (string, int, int, string, error) {
		slog.Info("Falling back to Gemini")
		apiKey := cfg.AI.GeminiApiKey
		if apiKey == "" {
			return fmt.Sprintf("[gemini mock] Anda mengirim: \"%s\". API key belum dikonfigurasi.", req.Message), 100, 50, "gemini-mock", nil
		}
		model := req.Model
		if model == "" {
			model = "gemini-1.5-flash"
		}
		fullMessage := systemPrompt + "\n\n" + req.Message
		text, tin, tout, err := callGeminiREST(ctx, apiKey, model, req.BaseURL, fullMessage)
		return text, tin, tout, model, err
	}

	provider := req.Provider
	if provider == "" {
		provider = "minimax"
	}

	if provider == "gemini" {
		return useGeminiFallback()
	}

	var apiKey, baseURL, model string
	if provider == "openai" {
		apiKey = cfg.AI.OpenAIApiKey
		baseURL = req.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model = req.Model
		if model == "" {
			model = "gpt-4o"
		}
	} else { // minimax or default
		apiKey = cfg.AI.MiniMaxAPIKey
		baseURL = req.BaseURL
		if baseURL == "" {
			baseURL = cfg.AI.MiniMaxBaseURL
		}
		model = req.Model
		if model == "" {
			model = cfg.AI.MiniMaxModel
		}
	}

	if apiKey == "" {
		slog.Warn("API key empty for provider, falling back to Gemini", "provider", provider)
		return useGeminiFallback()
	}

	clientCfg := openai.DefaultConfig(apiKey)
	clientCfg.BaseURL = baseURL
	client := openai.NewClientWithConfig(clientCfg)

	var lastErr error
	for i := 0; i < 3; i++ { // Initial try + 2 retries
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
		})
		
		if err == nil {
			return resp.Choices[0].Message.Content, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, provider, nil
		}

		slog.Warn("LLM request failed", "provider", provider, "attempt", i+1, "error", err)
		lastErr = err
		time.Sleep(500 * time.Millisecond) // Backoff
	}

	slog.Error("Provider exhausted all retries, executing fallback", "provider", provider, "last_error", lastErr)
	return useGeminiFallback()
}

func callGeminiREST(ctx context.Context, apiKey, model, baseURL, message string) (string, int, int, error) {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, model, apiKey)
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": message}}},
		},
	}
	jsonPayload, _ := json.Marshal(payload)

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", 0, 0, err
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("gemini returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, 
			geminiResp.UsageMetadata.PromptTokenCount, 
			geminiResp.UsageMetadata.CandidatesTokenCount, 
			nil
	}
	return "", 0, 0, fmt.Errorf("failed to parse gemini response")
}

func buildCacheKey(provider, system, message string) string {
	raw := provider + "|" + system + "|" + message
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

func estimateCost(provider string, tokensIn, tokensOut int) float64 {
	switch provider {
	case "minimax", "":
		return float64(tokensIn)*0.30/1_000_000 + float64(tokensOut)*1.20/1_000_000
	case "gemini-1.5-flash":
		// Gemini 1.5 Flash (assuming <= 128k context tier): $0.075/1M in, $0.30/1M out
		return float64(tokensIn)*0.075/1_000_000 + float64(tokensOut)*0.30/1_000_000
	default:
		return 0
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Suppress unused import warning for strconv
var _ = strconv.Itoa
