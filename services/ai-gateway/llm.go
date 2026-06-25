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

	// Tier 4: Last resort
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

	for _, m := range cfg.AI.LLM.Models {
		if !m.IsEnabled {
			continue
		}
		if containsCapability(m.Capability, useCase) {
			return m
		}
	}

	if len(cfg.AI.LLM.Models) > 0 {
		for _, m := range cfg.AI.LLM.Models {
			if m.IsEnabled {
				return m
			}
		}
	}

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
