package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/db"
)

// PlanFeaturesRow represents a tenant's plan features loaded from DB.
// Source of truth: `plan_features` table (DB), seeded by migration 000040.
// Sentinel: -1 means unlimited for any max_* field. 0 means feature disabled.
type PlanFeaturesRow struct {
	Tier                string `json:"tier"`
	PlanName            string `json:"plan_name"`
	MaxUsers            int    `json:"max_users"`
	MaxTransactions     int    `json:"max_transactions"`
	MaxAIText           int    `json:"max_ai_text"`
	MaxAIVision         int    `json:"max_ai_vision"`
	MaxAIAudioMinutes   int    `json:"max_ai_audio_minutes"`
	MaxImageGen         int    `json:"max_image_gen"`
	MaxProducts         int    `json:"max_products"`
	MaxCustomers        int    `json:"max_customers"`
	MaxStorageMB        int    `json:"max_storage_mb"`
	APIRateLimitPerMin  int    `json:"api_rate_limit_per_min"`
	DataRetentionMonths int    `json:"data_retention_months"`
	HasAccounting       bool   `json:"has_accounting"`
	HasPOS              bool   `json:"has_pos"`
	HasChatbot          bool   `json:"has_chatbot"`
	HasAI               bool   `json:"has_ai"`
	HasInventory        bool   `json:"has_inventory"`
	HasReports          bool   `json:"has_reports"`
	HasMultiUser        bool   `json:"has_multi_user"`
	HasAPIAccess        bool   `json:"has_api_access"`
	HasAdvancedReport   bool   `json:"has_advanced_report"`
	HasCustomBranding   bool   `json:"has_custom_branding"`
	HasPrioritySupport  bool   `json:"has_priority_support"`
}

// IsUnlimited returns true if the given field's value is the unlimited sentinel (-1).
// Supported fields: max_users, max_transactions, max_ai_text, max_ai_vision,
// max_ai_audio_minutes, max_image_gen, max_products, max_customers, max_storage_mb.
// Returns false for unknown fields (YAGNI: don't add fields without explicit support).
func (p PlanFeaturesRow) IsUnlimited(field string) bool {
	switch field {
	case "max_users":
		return p.MaxUsers == -1
	case "max_transactions":
		return p.MaxTransactions == -1
	case "max_ai_text":
		return p.MaxAIText == -1
	case "max_ai_vision":
		return p.MaxAIVision == -1
	case "max_ai_audio_minutes":
		return p.MaxAIAudioMinutes == -1
	case "max_image_gen":
		return p.MaxImageGen == -1
	case "max_products":
		return p.MaxProducts == -1
	case "max_customers":
		return p.MaxCustomers == -1
	case "max_storage_mb":
		return p.MaxStorageMB == -1
	}
	return false
}

// GetPlanFeatures returns the plan features for a tenant.
// It resolves the tenant's current plan tier, then fetches the limits from DB/cache.
func GetPlanFeatures(ctx context.Context, tenantID string) (PlanFeaturesRow, error) {
	// 1. Get Tenant's Plan Tier (e.g. "lite", "pro")
	tier := GetTenantPlan(ctx, tenantID)
	if tier == "inactive" || tier == "" {
		return PlanFeaturesRow{Tier: "inactive", PlanName: "inactive"}, nil
	}

	// 2. Check Cache for Plan Features
	cacheKey := "plan_features:" + tier
	if cache.Client != nil {
		if val, err := cache.Client.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			var row PlanFeaturesRow
			if json.Unmarshal([]byte(val), &row) == nil {
				return row, nil
			}
		}
	}

	// 3. Fallback to DB
	if db.Pool == nil {
		return PlanFeaturesRow{Tier: tier, PlanName: tier}, fmt.Errorf("db not initialized")
	}

	query := `
		SELECT plan_id, max_users, max_transactions, max_ai_text,
		       max_ai_vision, max_ai_audio_minutes, max_image_gen,
		       max_products, max_customers, max_storage_mb,
		       api_rate_limit_per_min, data_retention_months
		FROM plan_features WHERE plan_id = $1`
	var p PlanFeaturesRow
	err := db.Pool.QueryRow(ctx, query, tier).Scan(
		&p.Tier, &p.MaxUsers, &p.MaxTransactions, &p.MaxAIText,
		&p.MaxAIVision, &p.MaxAIAudioMinutes, &p.MaxImageGen,
		&p.MaxProducts, &p.MaxCustomers, &p.MaxStorageMB,
		&p.APIRateLimitPerMin, &p.DataRetentionMonths,
	)
	if err != nil {
		// Log error but fallback safely
		return PlanFeaturesRow{Tier: tier, PlanName: tier}, err
	}
	p.PlanName = p.Tier

	// 4. Save to Cache (TTL 1 hour)
	if cache.Client != nil {
		if b, err := json.Marshal(p); err == nil {
			cache.Client.Set(ctx, cacheKey, string(b), 1*time.Hour)
		}
	}

	return p, nil
}
