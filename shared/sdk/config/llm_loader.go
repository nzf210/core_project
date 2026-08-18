package config

import (
	"strconv"
	"strings"
)

// providerConfig holds environment configuration for a single LLM provider
type providerConfig struct {
	apiKey       string
	prefix       string
	defaultModel string
	defaultCaps  string
	defaultTier  int
	priority     int
}

// loadProviderModels loads models for a single provider from environment
func loadProviderModels(cfg providerConfig) []LLMModel {
	if cfg.apiKey == "" {
		return nil
	}

	modelList := splitEnv(getEnv(cfg.prefix+"_MODELS", cfg.defaultModel))
	caps := splitEnv(getEnv(cfg.prefix+"_CAPABILITIES", cfg.defaultCaps))
	contexts := splitEnv(getEnv(cfg.prefix+"_CONTEXT_WINDOW", "1000000"))
	costsIn := splitEnv(getEnv(cfg.prefix+"_COST_PER_1M_IN", "0.30"))
	costsOut := splitEnv(getEnv(cfg.prefix+"_COST_PER_1M_OUT", "1.20"))
	fb1 := splitEnv(getEnv(cfg.prefix+"_FALLBACK_1", ""))
	fb2 := splitEnv(getEnv(cfg.prefix+"_FALLBACK_2", ""))
	fb3 := splitEnv(getEnv(cfg.prefix+"_FALLBACK_3", ""))
	baseURL := getEnv(cfg.prefix+"_BASE_URL", "")

	var models []LLMModel
	provider := strings.ToLower(cfg.prefix)

	for i, model := range modelList {
		m := LLMModel{
			ID:            provider + ":" + model,
			Provider:      provider,
			Model:         model,
			BaseURL:       baseURL,
			APIKey:        cfg.apiKey,
			Capability:    getOrElse(caps, i, "general"),
			CostPer1MIn:   getOrElseFloat(costsIn, i, 0.30),
			CostPer1MOut:  getOrElseFloat(costsOut, i, 1.20),
			ContextWindow: getOrElseInt(contexts, i, 1000000),
			Priority:      cfg.priority + i,
			FallbackTier1: getOrElse(fb1, i, ""),
			FallbackTier2: getOrElse(fb2, i, ""),
			FallbackTier3: getOrElse(fb3, i, ""),
			IsEnabled:     true,
			Tier:          cfg.defaultTier,
		}
		models = append(models, m)
	}

	return models
}

// indexModelsByCapability adds models to capability map
func indexModelsByCapability(models []LLMModel, byCapability map[string][]LLMModel) {
	for _, m := range models {
		for _, cap := range strings.Split(m.Capability, ",") {
			cap = strings.TrimSpace(cap)
			if cap != "" {
				byCapability[cap] = append(byCapability[cap], m)
			}
		}
	}
}

// indexModelsByProvider adds models to provider map
func indexModelsByProvider(models []LLMModel, provider string, byProvider map[string][]LLMModel) {
	byProvider[provider] = append(byProvider[provider], models...)
}

func splitEnv(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getOrElse(slice []string, i int, defaultVal string) string {
	if i < len(slice) {
		return slice[i]
	}
	return defaultVal
}

func getOrElseFloat(slice []string, i int, defaultVal float64) float64 {
	if i < len(slice) && slice[i] != "" {
		if f, err := strconv.ParseFloat(slice[i], 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getOrElseInt(slice []string, i int, defaultVal int) int {
	if i < len(slice) && slice[i] != "" {
		if v, err := strconv.Atoi(slice[i]); err == nil {
			return v
		}
	}
	return defaultVal
}

