package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"core_project/shared/sdk/config"
	openai "github.com/sashabaranov/go-openai"
)

// callLLMWith3TierFallback implements 3-tier fallback chain for LLM calls
func callLLMWith3TierFallback(ctx context.Context, req ChatRequest) (string, int, int, string, int, error) {
	systemPrompt := req.SystemMsg
	if systemPrompt == "" {
		systemPrompt = "Kamu adalah asisten cerdas untuk platform WCH Indonesia. Jawab dalam bahasa Indonesia yang jelas dan profesional."
	}

	useCase := req.UseCase
	if useCase == "" {
		useCase = "general"
	}

	primaryModel := selectModelByCapability(useCase, req.Provider, req.Model)

	// Build fallback chain: primary + resolved fallbacks
	chain := []*config.LLMModel{&primaryModel}
	for _, fbID := range []string{primaryModel.FallbackTier1, primaryModel.FallbackTier2, primaryModel.FallbackTier3} {
		if fbID == "" {
			continue
		}
		if m := resolveModel(fbID); m != nil {
			chain = append(chain, m)
		}
	}

	baseURL := primaryModel.BaseURL
	if req.BaseURL != "" {
		baseURL = req.BaseURL
	}

	for i, m := range chain {
		tier := i + 1
		// ponytail: only primary model uses request-level baseURL override; fallbacks use their own
		if i > 0 {
			baseURL = m.BaseURL
		}
		slog.Info("LLM call", "tier", tier, "model", m.ID, "use_case", useCase)
		text, tin, tout, err := callLLM(ctx, m.Provider, m.Model, baseURL, m.APIKey, systemPrompt, req.Message)
		if err == nil {
			return text, tin, tout, m.ID, tier, nil
		}
		slog.Warn("LLM tier failed", "tier", tier, "model", m.ID, "error", err)
	}

	return "", 0, 0, "", 0, fmt.Errorf("all LLM providers failed")
}

func resolveModel(modelID string) *config.LLMModel {
	for i := range cfg.AI.LLM.Models {
		if cfg.AI.LLM.Models[i].ID == modelID {
			return &cfg.AI.LLM.Models[i]
		}
	}
	return nil
}

func callLLM(ctx context.Context, provider, model, baseURL, apiKey, systemPrompt, message string) (string, int, int, error) {
	if apiKey == "" {
		return "", 0, 0, fmt.Errorf("no API key for provider: %s", provider)
	}

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: message},
	}

	clientCfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		clientCfg.BaseURL = baseURL
	}
	client := openai.NewClientWithConfig(clientCfg)

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
		time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
	}

	return "", 0, 0, lastErr
}

func selectModelByCapability(useCase, providerOverride, modelOverride string) config.LLMModel {
	if m, ok := tryExactModelMatch(providerOverride, modelOverride); ok {
		return m
	}
	if m, ok := findEnabledByCapability(useCase); ok {
		return m
	}
	if m, ok := findFirstEnabled(); ok {
		return m
	}
	return defaultModel()
}

func tryExactModelMatch(providerOverride, modelOverride string) (config.LLMModel, bool) {
	if modelOverride == "" {
		return config.LLMModel{}, false
	}
	prov := providerOverride
	if prov == "" {
		prov = "Anthropic"
	}
	targetID := prov + ":" + modelOverride
	for _, m := range cfg.AI.LLM.Models {
		if m.ID == targetID {
			return m, true
		}
	}
	return config.LLMModel{}, false
}

func findEnabledByCapability(useCase string) (config.LLMModel, bool) {
	for _, m := range cfg.AI.LLM.Models {
		if m.IsEnabled && containsCapability(m.Capability, useCase) {
			return m, true
		}
	}
	return config.LLMModel{}, false
}

func findFirstEnabled() (config.LLMModel, bool) {
	for _, m := range cfg.AI.LLM.Models {
		if m.IsEnabled {
			return m, true
		}
	}
	return config.LLMModel{}, false
}

func defaultModel() config.LLMModel {
	return config.LLMModel{
		ID:       "Anthropic:" + cfg.AI.MiniMaxModel,
		Provider: "Anthropic",
		Model:    cfg.AI.MiniMaxModel,
		BaseURL:  cfg.AI.MiniMaxBaseURL,
		APIKey:   cfg.AI.MiniMaxAPIKey,
		Tier:     1,
	}
}

func containsCapability(capabilities, target string) bool {
	for _, cap := range strings.Split(capabilities, ",") {
		cap = strings.TrimSpace(cap)
		if cap == target {
			return true
		}
	}
	return false
}

func parseModelID(modelID string) (string, string) {
	parts := strings.SplitN(modelID, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", modelID
}

func estimateCost(modelID string, tokensIn, tokensOut int) float64 {
	for _, m := range cfg.AI.LLM.Models {
		if m.ID == modelID {
			return float64(tokensIn)*m.CostPer1MIn/1_000_000 + float64(tokensOut)*m.CostPer1MOut/1_000_000
		}
	}

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
