package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

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
	}

	// WhatsApp Gateway
	WhatsApp struct {
		FonnteToken string
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

	// WA
	cfg.WhatsApp.FonnteToken = getEnv("FONNTE_TOKEN", "")

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
