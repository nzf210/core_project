package config

import (
	"log"
	"os"
	"strconv"

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

	// N8N Webhook Configuration
	N8N struct {
		WebhookSecret string
	}

	// RabbitMQ Message Queue
	RabbitMQ struct {
		URL string // AMQP connection string (amqp://user:pass@host:port/)
	}
}

// Global Config instance
var GlobalConfig *Config

// LoadConfig loads variables from .env and system environments
func LoadConfig(envPath string) *Config {
	if envPath != "" {
		if err := godotenv.Load(envPath); err != nil {
			// envPath not found in cwd — walk up to find .env (monorepo: services/*, apps/*)
			log.Printf("Warning: No .env at %s, searching parent dirs.", envPath)
			_ = godotenv.Load("../.env")
			_ = godotenv.Load("../../.env")
			_ = godotenv.Load("../../../.env")
		}
	} else {
		// Try reading .env from current directory or parent directory
		_ = godotenv.Load(".env")
		_ = godotenv.Load("../.env")
		_ = godotenv.Load("../../.env")
		_ = godotenv.Load("../../../.env")
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

	// RabbitMQ
	cfg.RabbitMQ.URL = getEnv("RABBITMQ_URL", "amqp://wch_admin:rabbitmq_pass@127.0.0.1:5672/")

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

	// N8N Webhook
	cfg.N8N.WebhookSecret = getEnv("N8N_WEBHOOK_SECRET", "")

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
func loadLLMModels(cfg *Config) LLMConfig {
	var models []LLMModel
	byCapability := make(map[string][]LLMModel)
	byProvider := make(map[string][]LLMModel)

	providers := []struct {
		cfg      providerConfig
		provider string
	}{
		{
			cfg: providerConfig{
				apiKey:       cfg.AI.MiniMaxAPIKey,
				prefix:       "MINIMAX",
				defaultModel: cfg.AI.MiniMaxModel,
				defaultCaps:  "general,product,faq",
				defaultTier:  1,
				priority:     0,
			},
			provider: "minimax",
		},
		{
			cfg: providerConfig{
				apiKey:       cfg.AI.OpenAIApiKey,
				prefix:       "OPENAI",
				defaultModel: "gpt-4o",
				defaultCaps:  "general,coding",
				defaultTier:  2,
				priority:     10,
			},
			provider: "openai",
		},
		{
			cfg: providerConfig{
				apiKey:       cfg.AI.GeminiApiKey,
				prefix:       "GEMINI",
				defaultModel: "gemini-1.5-flash",
				defaultCaps:  "general,vision",
				defaultTier:  3,
				priority:     100,
			},
			provider: "gemini",
		},
		{
			cfg: providerConfig{
				apiKey:       getEnv("LLM_API_KEY", ""),
				prefix:       "LLM",
				defaultModel: getEnv("LLM_MODEL", "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free"),
				defaultCaps:  "general,coding",
				defaultTier:  1,
				priority:     5,
			},
			provider: "custom",
		},
	}

	for _, p := range providers {
		providerModels := loadProviderModels(p.cfg)
		models = append(models, providerModels...)
		indexModelsByProvider(providerModels, p.provider, byProvider)
		indexModelsByCapability(providerModels, byCapability)

		// Special handling for custom provider sonnet alias
		if p.provider == "custom" && len(providerModels) > 0 {
			defaultModel := getEnv("LLM_MODEL", "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free")
			for _, m := range providerModels {
				if m.Model == defaultModel {
					sonnetAlias := m
					sonnetAlias.ID = "Anthropic:sonnet"
					models = append(models, sonnetAlias)
					break
				}
			}
		}
	}

	return LLMConfig{
		Models:       models,
		ByCapability: byCapability,
		ByProvider:   byProvider,
	}
}