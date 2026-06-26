package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"
)

// chatConfigCache mirrors the subset of fields chatbot needs at runtime.
// Defined inline to avoid coupling chatbot -> accounting types directly.
type chatConfigCache struct {
	IsActive            bool     `json:"is_active"`
	Language            string   `json:"language"`
	Tone                string   `json:"tone"`
	SystemPrompt        string   `json:"system_prompt"`
	WelcomeMessage      string   `json:"welcome_message"`
	FallbackMessage     string   `json:"fallback_message"`
	OutsideHoursMessage string   `json:"outside_hours_message"`
	BusinessHoursStart  string   `json:"business_hours_start"`
	BusinessHoursEnd    string   `json:"business_hours_end"`
	BusinessDays        []int    `json:"business_days"`
	EscalationEnabled   bool     `json:"escalation_enabled"`
	EscalationKeywords  []string `json:"escalation_keywords"`
}

// loadChatbotConfig fetches the per-tenant chatbot config from accounting,
// cached in Redis for 5 minutes. Returns nil if not reachable (graceful
// degradation: chatbot still works with hardcoded defaults).
func loadChatbotConfig(ctx context.Context, tenantID string) *chatConfigCache {
	if tenantID == "" {
		return nil
	}
	cacheKey := "chatbot:config:" + tenantID
	// Try Redis cache first
	if redisClient != nil {
		if val, err := redisClient.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			var cfg chatConfigCache
			if err := json.Unmarshal([]byte(val), &cfg); err == nil {
				return &cfg
			}
		}
	}
	// Fall back to HTTP fetch
	url := AccountingURL + "/chatbot/config"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("Failed to load chatbot config (will use defaults)", "tenant", tenantID, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var apiResp struct {
		Success bool             `json:"success"`
		Data    *chatConfigCache `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || !apiResp.Success || apiResp.Data == nil {
		return nil
	}
	// Store in Redis cache
	if redisClient != nil {
		if b, err := json.Marshal(apiResp.Data); err == nil {
			redisClient.Set(ctx, cacheKey, b, chatbotConfigCacheTTL)
		}
	}
	return apiResp.Data
}

// isWithinBusinessHours checks the current time against the tenant's business
// hours configuration. Returns (within, outsideMessage). If config is nil or
// fields are empty, treats as always-open (returns true, "").
func isWithinBusinessHours(cfg *chatConfigCache) (bool, string) {
	if cfg == nil || !cfg.IsActive {
		// is_active=false means chatbot is off — return outside-hours message
		msg := ""
		if cfg != nil {
			msg = cfg.OutsideHoursMessage
		}
		return false, msg
	}
	if cfg.BusinessHoursStart == "" || cfg.BusinessHoursEnd == "" {
		return true, ""
	}
	now := time.Now()
	// Business days: 0=Sunday, default allow Mon-Sat (1-6)
	if len(cfg.BusinessDays) > 0 {
		wd := int(now.Weekday())
		allowed := false
		for _, d := range cfg.BusinessDays {
			if d == wd {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, cfg.OutsideHoursMessage
		}
	}
	// Compare HH:MM
	nowHM := now.Format("15:04")
	if nowHM < cfg.BusinessHoursStart || nowHM > cfg.BusinessHoursEnd {
		return false, cfg.OutsideHoursMessage
	}
	return true, ""
}