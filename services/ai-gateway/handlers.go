package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"core_project/shared/sdk/response"
)

const (
	methodNotAllowed  = "Method not allowed"
	headerTenantID    = "X-Tenant-ID"
	systemTenantID    = "00000000-0000-0000-0000-000000000000"
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

// resolveTenantID extracts tenant ID from request body or header.
func resolveTenantID(r *http.Request, reqTenantID string) string {
	if reqTenantID != "" {
		return reqTenantID
	}
	if id := r.Header.Get(headerTenantID); id != "" {
		return id
	}
	return systemTenantID
}

// tryRespondCache tries to serve the response from semantic cache.
// Returns true if response was sent and handler should return.
func tryRespondCache(w http.ResponseWriter, r *http.Request, cacheKey string, req ChatRequest) bool {
	if !cfg.AI.CacheEnabled {
		return false
	}
	cached, hit := checkCache(r.Context(), cacheKey)
	if !hit {
		return false
	}
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
	return true
}

// storeCacheEntry stores the LLM response in cache if enabled.
func storeCacheEntry(ctx context.Context, cacheKey, text string, reqTTL int) {
	if !cfg.AI.CacheEnabled {
		return
	}
	ttl := cfg.AI.CacheTTL
	if reqTTL > 0 {
		ttl = reqTTL
	}
	storeCache(ctx, cacheKey, text, ttl)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: methodNotAllowed})
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

	req.TenantID = resolveTenantID(r, req.TenantID)
	cacheKey := buildCacheKey(req.Provider, req.Model, req.SystemMsg, req.Message)

	if tryRespondCache(w, r, cacheKey, req) {
		return
	}

	text, tokensIn, tokensOut, modelID, tier, err := callLLMWith3TierFallback(r.Context(), req)
	if err != nil {
		slog.Error("LLM call failed after all fallbacks", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("All LLM providers failed: %v", err),
		})
		return
	}

	provider, model := parseModelID(modelID)
	storeCacheEntry(r.Context(), cacheKey, text, req.CacheTTL)

	costUSD := estimateCost(modelID, tokensIn, tokensOut)

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
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: methodNotAllowed})
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
	req.TenantID = resolveTenantID(r, req.TenantID)

	w.Header().Set(headerContentType, "text/event-stream")
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
		Data: map[string]any{
			"primary_model":    cfg.AI.MiniMaxModel,
			"gemini_fallback":  "enabled",
			"models_available": len(cfg.AI.LLM.Models),
		},
	})
}

func handleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: methodNotAllowed})
		return
	}

	models := make([]map[string]any, 0, len(cfg.AI.LLM.Models))
	for _, m := range cfg.AI.LLM.Models {
		models = append(models, map[string]any{
			"id":              m.ID,
			"provider":        m.Provider,
			"model":           m.Model,
			"base_url":        m.BaseURL,
			"capability":      m.Capability,
			"context_window":  m.ContextWindow,
			"cost_per_1m_in":  m.CostPer1MIn,
			"cost_per_1m_out": m.CostPer1MOut,
			"priority":        m.Priority,
			"tier":            m.Tier,
			"fallback_1":      m.FallbackTier1,
			"fallback_2":      m.FallbackTier2,
			"fallback_3":      m.FallbackTier3,
			"is_enabled":      m.IsEnabled,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   models,
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerContentType, "text/plain; version=0.0.4")

	aiCostMu.Lock()
	costTotal := aiCostUSDTotal
	aiCostMu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP ai_gateway_requests_total Total number of chat requests\n")
	b.WriteString("# TYPE ai_gateway_requests_total counter\n")
	fmt.Fprintf(&b, "ai_gateway_requests_total %d\n", aiRequestsTotal.Load())

	b.WriteString("# HELP ai_gateway_cache_hits_total Total semantic cache hits\n")
	b.WriteString("# TYPE ai_gateway_cache_hits_total counter\n")
	fmt.Fprintf(&b, "ai_gateway_cache_hits_total %d\n", aiCacheHits.Load())

	b.WriteString("# HELP ai_gateway_tokens_total Total tokens processed\n")
	b.WriteString("# TYPE ai_gateway_tokens_total counter\n")
	fmt.Fprintf(&b, "ai_gateway_tokens_in_total %d\n", aiTokensInTotal.Load())
	fmt.Fprintf(&b, "ai_gateway_tokens_out_total %d\n", aiTokensOutTotal.Load())

	b.WriteString("# HELP ai_gateway_cost_usd_total Total estimated cost in USD\n")
	b.WriteString("# TYPE ai_gateway_cost_usd_total counter\n")
	fmt.Fprintf(&b, "ai_gateway_cost_usd_total %.6f\n", costTotal)

	b.WriteString("# HELP ai_gateway_model_requests_total Requests per model\n")
	b.WriteString("# TYPE ai_gateway_model_requests_total counter\n")
	modelRequestsMu.RLock()
	for model, count := range modelRequests {
		fmt.Fprintf(&b, "ai_gateway_model_requests_total{model=\"%s\"} %d\n", model, count)
	}
	modelRequestsMu.RUnlock()

	_, _ = w.Write([]byte(b.String()))
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: methodNotAllowed})
		return
	}

	var req struct {
		Input    string `json:"input"`
		Model    string `json:"model"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: response.InvalidRequest})
		return
	}
	if req.Input == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "input cannot be empty"})
		return
	}
	req.TenantID = resolveTenantID(r, req.TenantID)

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

	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"object": "embedding", "embedding": emb, "index": 0},
		},
		"model":     req.Model,
		"tenant_id": req.TenantID,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
