package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// LLMModel defines a single LLM model with 3-tier fallback chain
type LLMModel struct {
	ID            string  // Unique ID: "minimax:m2.7-general", "gemini:flash-product"
	Provider      string  // "minimax", "openai", "gemini", "anthropic", "custom"
	Model         string  // Model ID (e.g., "MiniMax-M2.7", "gpt-4o")
	BaseURL       string  // Custom API base URL (e.g., https://api.minimax.io/v1)
	APIKey        string  // API key for this provider
	Capability    string  // "product", "faq", "general", "vision", "coding" (comma-separated)
	CostPer1MIn   float64 // Cost per 1M input tokens in USD
	CostPer1MOut  float64 // Cost per 1M output tokens in USD
	ContextWindow int     // Max context window tokens
	Priority      int     // Lower = higher priority within same capability

	// 3-Tier Fallback Chain (max 3 levels)
	FallbackTier1 string // "provider:model" for first fallback
	FallbackTier2 string // "provider:model" for second fallback
	FallbackTier3 string // "provider:model" for third fallback

	// Metadata
	IsEnabled bool   // Enable/disable this model
	Tier      int    // 1=Primary, 2=Secondary, 3=Tertiary fallback
}

// LLMConfig manages all LLM models with fallback chains
type LLMConfig struct {
	Models       []LLMModel           // All registered models
	ByCapability map[string][]LLMModel // Capability → sorted models
	ByProvider   map[string][]LLMModel // Provider → models
}

// TenantAIConfig per-tenant AI config override (loaded from DB or cache)
type TenantAIConfig struct {
	TenantID       string
	DefaultModel   string // Override default model
	UseCaseRouting map[string]string // use_case → model override
}

// Config represents the complete application configuration
type Config struct {
	Port          string
	Env           string
	JWTSecret     string
	EncryptionKey string

	// Database configurations
	DB struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
		SSLMode  string
	}

	// Cache configurations
	Redis struct {
		Host     string
		Port     int
		Password string
	}

	// AI APIs configurations
	AI struct {
		OpenAIApiKey   string
		GeminiApiKey   string
		MiniMaxAPIKey  string // MiniMax M2.7 (primary LLM for all products)
		MiniMaxBaseURL string // Default: https://api.minimax.io/v1
		MiniMaxModel   string // Default: MiniMax-M2.7
		CacheEnabled   bool   // Semantic caching via Redis
		CacheTTL       int    // Cache TTL in seconds
		CostAlertUSD   float64 // Daily cost alert threshold in USD

		// Flexible LLM Configuration with 3-tier fallback support
		LLM LLMConfig // Dynamic model registry with fallback chains
	}

	// WhatsApp Gateway
	//
	// Hybrid architecture:
	// - Cloud API (Meta official) untuk pesan transaksional: OTP, invoice, subscription
	// - whatsmeow (unofficial) untuk chatbot AI, broadcast, voucher
	// Lihat docs/WHATSAPP_GATEWAY_PLAN.md untuk arsitektur lengkap.
	WhatsApp struct {
		GatewayURL   string // Override base URL wa-gateway (default: http://wa-gateway:8202)
		CloudAPIURL  string // Meta Graph API base URL (default: https://graph.facebook.com)
		CloudAPIToken string // Default server access token (optional fallback)
		CloudAPIPort string // Cloud API service port (default: 8210)
	}

	// Telegram Bot untuk auth (register & login via Telegram)
	Telegram struct {
		BotToken string // Telegram Bot Token dari @BotFather
	}
}

// Global Config instance
var GlobalConfig *Config

// LoadConfig loads variables from .env and system environments
func LoadConfig(envPath string) *Config {
	if envPath != "" {
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: No custom .env file found at %s. Relying on system environment variables.", envPath)
		}
	} else {
		// Try reading .env from current directory or parent directory
		_ = godotenv.Load(".env")
		_ = godotenv.Load("../../../.env")
		_ = godotenv.Load("../../.env")
	}

	cfg := &Config{}

	// Core
	cfg.Port = getEnv("PORT", "8080")
	cfg.Env = getEnv("APP_ENV", getEnv("ENV", "development"))
	cfg.JWTSecret = getEnv("JWT_SECRET", "super_jwt_secret_key_minimum_32_characters")
	cfg.EncryptionKey = getEnv("ENCRYPTION_KEY", "aes_256_encryption_key_must_be_32")

	// DB
	cfg.DB.Host = getEnv("DB_HOST", "127.0.0.1")
	cfg.DB.Port = getEnvAsInt("DB_PORT", 5432)
	cfg.DB.User = getEnv("DB_USER", "wch_admin")
	cfg.DB.Password = getEnv("DB_PASSWORD", "secure_postgres_password_123")
	cfg.DB.Name = getEnv("DB_NAME", "wch_platform")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")

	// Redis
	cfg.Redis.Host = getEnv("REDIS_HOST", "127.0.0.1")
	cfg.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "secure_redis_password_123")

	// AI
	cfg.AI.OpenAIApiKey = getEnv("OPENAI_API_KEY", "")
	cfg.AI.GeminiApiKey = getEnv("GEMINI_API_KEY", "")
	cfg.AI.MiniMaxAPIKey = getEnv("MINIMAX_API_KEY", "")
	cfg.AI.MiniMaxBaseURL = getEnv("MINIMAX_BASE_URL", "https://api.minimax.io/v1")
	cfg.AI.MiniMaxModel = getEnv("MINIMAX_MODEL", "MiniMax-M2.7")
	cfg.AI.CacheEnabled = getEnv("AI_CACHE_ENABLED", "true") == "true"
	cfg.AI.CacheTTL = getEnvAsInt("AI_CACHE_TTL_SECONDS", 600)
	cfg.AI.CostAlertUSD = 10.00 // Default $10/day alert

	// Initialize flexible LLM model registry with 3-tier fallback from environment
	cfg.AI.LLM = loadLLMModels(cfg)

	// WA Gateway (internal — hybrid: Cloud API + whatsmeow)
	cfg.WhatsApp.GatewayURL = getEnv("WA_GATEWAY_URL", "http://wa-gateway:8202")
	cfg.WhatsApp.CloudAPIURL = getEnv("WA_CLOUD_API_URL", "https://graph.facebook.com")
	cfg.WhatsApp.CloudAPIToken = getEnv("WA_CLOUD_API_TOKEN", "")
	cfg.WhatsApp.CloudAPIPort = getEnv("WA_CLOUD_API_PORT", "8210")

	// Telegram Bot for auth
	cfg.Telegram.BotToken = getEnv("TELEGRAM_BOT_TOKEN", "")

	GlobalConfig = cfg
	return cfg
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// loadLLMModels loads LLM models with 3-tier fallback chains from environment variables.
//
// Environment format per provider:
//   MINIMAX_API_KEY=xxx
//   MINIMAX_BASE_URL=https://api.minimax.io/v1
//   MINIMAX_MODELS=MiniMax-M2.7;MiniMax-M2.7-Fast
//   MINIMAX_CAPABILITIES=general,product,faq;general
//   MINIMAX_CONTEXT_WINDOW=1000000;200000
//   MINIMAX_COST_PER_1M_IN=0.30;0.10
//   MINIMAX_COST_PER_1M_OUT=1.20;0.40
//   MINIMAX_FALLBACK_1=gemini:gemini-1.5-flash
//   MINIMAX_FALLBACK_2=openai:gpt-4o-mini
//   MINIMAX_FALLBACK_3=
//
// Fallback format: "provider:model" (e.g., "gemini:gemini-1.5-flash")
func loadLLMModels(cfg *Config) LLMConfig {
	var models []LLMModel
	byCapability := make(map[string][]LLMModel)
	byProvider := make(map[string][]LLMModel)

	// MiniMax (primary)
	if cfg.AI.MiniMaxAPIKey != "" {
		modelList := splitEnv(getEnv("MINIMAX_MODELS", cfg.AI.MiniMaxModel))
		caps := splitEnv(getEnv("MINIMAX_CAPABILITIES", "general,product,faq"))
		contexts := splitEnv(getEnv("MINIMAX_CONTEXT_WINDOW", "1000000"))
		costsIn := splitEnv(getEnv("MINIMAX_COST_PER_1M_IN", "0.30"))
		costsOut := splitEnv(getEnv("MINIMAX_COST_PER_1M_OUT", "1.20"))
		fb1 := splitEnv(getEnv("MINIMAX_FALLBACK_1", "gemini:gemini-1.5-flash"))
		fb2 := splitEnv(getEnv("MINIMAX_FALLBACK_2", ""))
		fb3 := splitEnv(getEnv("MINIMAX_FALLBACK_3", ""))

		for i, model := range modelList {
			m := LLMModel{
				ID:            "minimax:" + model,
				Provider:      "minimax",
				Model:         model,
				BaseURL:       getEnv("MINIMAX_BASE_URL", cfg.AI.MiniMaxBaseURL),
				APIKey:        cfg.AI.MiniMaxAPIKey,
				Capability:    getOrElse(caps, i, "general"),
				CostPer1MIn:   getOrElseFloat(costsIn, i, 0.30),
				CostPer1MOut:  getOrElseFloat(costsOut, i, 1.20),
				ContextWindow: getOrElseInt(contexts, i, 1000000),
				Priority:      i,
				FallbackTier1: getOrElse(fb1, i, ""),
				FallbackTier2: getOrElse(fb2, i, ""),
				FallbackTier3: getOrElse(fb3, i, ""),
				IsEnabled:     true,
				Tier:          1,
			}
			models = append(models, m)
			byProvider["minimax"] = append(byProvider["minimax"], m)
			for _, cap := range strings.Split(m.Capability, ",") {
				cap = strings.TrimSpace(cap)
				if cap != "" {
					byCapability[cap] = append(byCapability[cap], m)
				}
			}
		}
	}

	// OpenAI (optional)
	if cfg.AI.OpenAIApiKey != "" {
		modelList := splitEnv(getEnv("OPENAI_MODELS", "gpt-4o"))
		caps := splitEnv(getEnv("OPENAI_CAPABILITIES", "general,coding"))
		contexts := splitEnv(getEnv("OPENAI_CONTEXT_WINDOW", "128000"))
		costsIn := splitEnv(getEnv("OPENAI_COST_PER_1M_IN", "5.00"))
		costsOut := splitEnv(getEnv("OPENAI_COST_PER_1M_OUT", "15.00"))
		fb1 := splitEnv(getEnv("OPENAI_FALLBACK_1", "gemini:gemini-1.5-flash"))
		fb2 := splitEnv(getEnv("OPENAI_FALLBACK_2", ""))
		fb3 := splitEnv(getEnv("OPENAI_FALLBACK_3", ""))

		for i, model := range modelList {
			m := LLMModel{
				ID:            "openai:" + model,
				Provider:      "openai",
				Model:         model,
				BaseURL:       getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
				APIKey:        cfg.AI.OpenAIApiKey,
				Capability:    getOrElse(caps, i, "general"),
				CostPer1MIn:   getOrElseFloat(costsIn, i, 5.00),
				CostPer1MOut:  getOrElseFloat(costsOut, i, 15.00),
				ContextWindow: getOrElseInt(contexts, i, 128000),
				Priority:      i + 10,
				FallbackTier1: getOrElse(fb1, i, ""),
				FallbackTier2: getOrElse(fb2, i, ""),
				FallbackTier3: getOrElse(fb3, i, ""),
				IsEnabled:     true,
				Tier:          2,
			}
			models = append(models, m)
			byProvider["openai"] = append(byProvider["openai"], m)
			for _, cap := range strings.Split(m.Capability, ",") {
				cap = strings.TrimSpace(cap)
				if cap != "" {
					byCapability[cap] = append(byCapability[cap], m)
				}
			}
		}
	}

	// Gemini (fallback)
	if cfg.AI.GeminiApiKey != "" {
		modelList := splitEnv(getEnv("GEMINI_MODELS", "gemini-1.5-flash"))
		caps := splitEnv(getEnv("GEMINI_CAPABILITIES", "general,vision"))
		contexts := splitEnv(getEnv("GEMINI_CONTEXT_WINDOW", "1000000"))
		costsIn := splitEnv(getEnv("GEMINI_COST_PER_1M_IN", "0.075"))
		costsOut := splitEnv(getEnv("GEMINI_COST_PER_1M_OUT", "0.30"))
		fb1 := splitEnv(getEnv("GEMINI_FALLBACK_1", ""))
		fb2 := splitEnv(getEnv("GEMINI_FALLBACK_2", ""))
		fb3 := splitEnv(getEnv("GEMINI_FALLBACK_3", ""))

		for i, model := range modelList {
			m := LLMModel{
				ID:            "gemini:" + model,
				Provider:      "gemini",
				Model:         model,
				BaseURL:       getEnv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
				APIKey:        cfg.AI.GeminiApiKey,
				Capability:    getOrElse(caps, i, "general"),
				CostPer1MIn:   getOrElseFloat(costsIn, i, 0.075),
				CostPer1MOut:  getOrElseFloat(costsOut, i, 0.30),
				ContextWindow: getOrElseInt(contexts, i, 1000000),
				Priority:      i + 100,
				FallbackTier1: getOrElse(fb1, i, ""),
				FallbackTier2: getOrElse(fb2, i, ""),
				FallbackTier3: getOrElse(fb3, i, ""),
				IsEnabled:     true,
				Tier:          3,
			}
			models = append(models, m)
			byProvider["gemini"] = append(byProvider["gemini"], m)
			for _, cap := range strings.Split(m.Capability, ",") {
				cap = strings.TrimSpace(cap)
				if cap != "" {
					byCapability[cap] = append(byCapability[cap], m)
				}
			}
		}
	}

	// Custom / OpenRouter (optional)
	if getEnv("LLM_API_KEY", "") != "" {
		modelList := splitEnv(getEnv("LLM_MODELS", getEnv("LLM_MODEL", "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free")))
		caps := splitEnv(getEnv("LLM_CAPABILITIES", "general,coding"))
		contexts := splitEnv(getEnv("LLM_CONTEXT_WINDOW", "131072"))
		costsIn := splitEnv(getEnv("LLM_COST_PER_1M_IN", "0.0"))
		costsOut := splitEnv(getEnv("LLM_COST_PER_1M_OUT", "0.0"))
		fb1 := splitEnv(getEnv("LLM_FALLBACK_1", "gemini:gemini-1.5-flash"))
		fb2 := splitEnv(getEnv("LLM_FALLBACK_2", ""))
		fb3 := splitEnv(getEnv("LLM_FALLBACK_3", ""))

		for i, model := range modelList {
			m := LLMModel{
				ID:            "custom:" + model,
				Provider:      "custom",
				Model:         model,
				BaseURL:       getEnv("LLM_BASE_URL", "https://openrouter.ai/api/v1"),
				APIKey:        getEnv("LLM_API_KEY", ""),
				Capability:    getOrElse(caps, i, "general"),
				CostPer1MIn:   getOrElseFloat(costsIn, i, 0.0),
				CostPer1MOut:  getOrElseFloat(costsOut, i, 0.0),
				ContextWindow: getOrElseInt(contexts, i, 131072),
				Priority:      i + 5,
				FallbackTier1: getOrElse(fb1, i, ""),
				FallbackTier2: getOrElse(fb2, i, ""),
				FallbackTier3: getOrElse(fb3, i, ""),
				IsEnabled:     true,
				Tier:          1,
			}
			models = append(models, m)
			byProvider["custom"] = append(byProvider["custom"], m)
			for _, cap := range strings.Split(m.Capability, ",") {
				cap = strings.TrimSpace(cap)
				if cap != "" {
					byCapability[cap] = append(byCapability[cap], m)
				}
			}

			// Register "Anthropic:sonnet" alias to route "sonnet" requests to OpenRouter
			if model == getEnv("LLM_MODEL", "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free") {
				sonnetAlias := m
				sonnetAlias.ID = "Anthropic:sonnet"
				models = append(models, sonnetAlias)
			}
		}
	}


	return LLMConfig{
		Models:       models,
		ByCapability: byCapability,
		ByProvider:   byProvider,
	}
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