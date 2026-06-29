package auth

import (
	"context"
	"encoding/json"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/db"
)

// PlanFeaturesRow represents a tenant's plan features loaded from DB.
// Source of truth: `plan_features` table (DB), seeded by migration 000040.
// Sentinel: -1 means unlimited for any max_* field. 0 means feature disabled.
// Has* fields are derived from Features map for backward-compat JSON serialization.
type PlanFeaturesRow struct {
	Tier                string          `json:"tier"`
	PlanName            string          `json:"plan_name"`
	MaxUsers            int             `json:"max_users"`
	MaxTransactions     int             `json:"max_transactions"`
	MaxAIText           int             `json:"max_ai_text"`
	MaxAIVision         int             `json:"max_ai_vision"`
	MaxAIAudioMinutes   int             `json:"max_ai_audio_minutes"`
	MaxImageGen         int             `json:"max_image_gen"`
	MaxProducts         int             `json:"max_products"`
	MaxCustomers        int             `json:"max_customers"`
	MaxStorageMB        int             `json:"max_storage_mb"`
	APIRateLimitPerMin  int             `json:"api_rate_limit_per_min"`
	DataRetentionMonths int             `json:"data_retention_months"`
	Features            map[string]bool `json:"features"` // dynamic feature toggles — source of truth
	// ponytail: Has* kept for backward-compat JSON serialization (billing quota handler).
	HasAccounting      bool `json:"has_accounting"`
	HasPOS             bool `json:"has_pos"`
	HasChatbot         bool `json:"has_chatbot"`
	HasAI              bool `json:"has_ai"`
	HasInventory       bool `json:"has_inventory"`
	HasReports         bool `json:"has_reports"`
	HasMultiUser       bool `json:"has_multi_user"`
	HasAPIAccess       bool `json:"has_api_access"`
	HasAdvancedReport  bool `json:"has_advanced_report"`
	HasCustomBranding  bool `json:"has_custom_branding"`
	HasPrioritySupport bool `json:"has_priority_support"`
	HasWACloudAPI      bool `json:"has_wa_cloud_api"`
}

// syncHasFields populates backward-compat Has* fields from the Features map.
func (p *PlanFeaturesRow) syncHasFields() {
	p.HasPOS = p.Features["pos"]
	p.HasChatbot = p.Features["chatbot"]
	p.HasAI = p.Features["ai_requests"]
	p.HasAccounting = p.Features["accounting"]
	p.HasReports = p.Features["reports"]
	p.HasInventory = p.Features["inventory"]
	p.HasAPIAccess = p.Features["api_access"]
	p.HasMultiUser = p.Features["multi_user"]
	p.HasCustomBranding = p.Features["custom_branding"]
	p.HasPrioritySupport = p.Features["priority_support"]
	p.HasAdvancedReport = p.Features["advanced_reports"]
	p.HasWACloudAPI = p.Features["wa_cloud_api"]
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

// defaultPlanFeatures returns hardcoded feature defaults for known tiers.
// Used when DB/Redis unavailable to prevent feature gate failures.
func defaultPlanFeatures(tier string) PlanFeaturesRow {
	p := PlanFeaturesRow{
		Tier:     tier,
		Features: make(map[string]bool),
	}
	switch tier {
	case "superadmin":
		p.PlanName = "Superadmin"
		p.MaxUsers = -1
		p.MaxTransactions = -1
		p.MaxAIText = -1
		p.MaxAIVision = -1
		p.MaxAIAudioMinutes = -1
		p.MaxImageGen = -1
		p.MaxProducts = -1
		p.MaxCustomers = -1
		p.MaxStorageMB = -1
		p.APIRateLimitPerMin = 9999
		p.DataRetentionMonths = 999
		for _, k := range []string{"accounting", "pos", "chatbot", "ai_requests", "inventory", "reports", "multi_user", "api_access", "advanced_reports", "custom_branding", "priority_support"} {
			p.Features[k] = true
		}
	case "ultimate":
		p.PlanName = "Ultimate"
		p.MaxUsers = -1
		p.MaxTransactions = -1
		p.MaxAIText = -1
		p.MaxAIVision = 500
		p.MaxAIAudioMinutes = 60
		p.MaxImageGen = 30
		p.MaxProducts = -1
		p.MaxCustomers = -1
		p.MaxStorageMB = -1
		for _, k := range []string{"accounting", "pos", "chatbot", "ai_requests", "inventory", "reports", "multi_user", "api_access", "advanced_reports", "custom_branding", "priority_support"} {
			p.Features[k] = true
		}
	case "pro":
		p.PlanName = "Pro"
		p.MaxUsers = 10
		p.MaxTransactions = 10000
		p.MaxAIText = 5000
		p.MaxAIVision = 50
		p.MaxAIAudioMinutes = 0
		p.MaxImageGen = 0
		p.MaxProducts = 1000
		p.MaxCustomers = 5000
		p.MaxStorageMB = 10000
		for _, k := range []string{"accounting", "pos", "chatbot", "ai_requests", "inventory", "reports", "multi_user", "api_access", "priority_support"} {
			p.Features[k] = true
		}
		p.Features["advanced_reports"] = false
		p.Features["custom_branding"] = false
	case "lite":
		p.PlanName = "Lite"
		p.MaxUsers = 3
		p.MaxTransactions = 1000
		p.MaxAIText = 250
		p.MaxAIVision = 0
		p.MaxAIAudioMinutes = 0
		p.MaxImageGen = 0
		p.MaxProducts = 100
		p.MaxCustomers = 500
		p.MaxStorageMB = 1000
		for _, k := range []string{"accounting", "pos", "chatbot", "ai_requests", "inventory", "reports"} {
			p.Features[k] = true
		}
		p.Features["multi_user"] = false
		p.Features["api_access"] = false
		p.Features["advanced_reports"] = false
		p.Features["custom_branding"] = false
		p.Features["priority_support"] = false
	default:
		p.PlanName = tier
	}
	p.syncHasFields()
	return p
}

// GetPlanFeatures returns the plan features for a tenant.
// It resolves the tenant's current plan tier, then fetches the limits from DB/cache.
func GetPlanFeatures(ctx context.Context, tenantID string) (PlanFeaturesRow, error) {
	// 1. Get Tenant's Plan Tier (e.g. "lite", "pro")
	tier := GetTenantPlan(ctx, tenantID)
	if tier == "inactive" || tier == "" {
		return PlanFeaturesRow{Tier: "inactive", PlanName: "inactive", Features: make(map[string]bool)}, nil
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
		return defaultPlanFeatures(tier), nil
	}

	// Superadmin gets all features (before DB query)
	if tier == "superadmin" {
		p := defaultPlanFeatures("superadmin")
		// Cache superadmin features too
		if cache.Client != nil {
			if b, _ := json.Marshal(p); b != nil {
				cache.Client.Set(ctx, cacheKey, string(b), 1*time.Hour)
			}
		}
		return p, nil
	}

	// Query all feature_key rows for this plan — fully dynamic, no switch-case
	var p PlanFeaturesRow
	p.Tier = tier
	p.Features = make(map[string]bool)

	rows, err := db.Pool.Query(ctx,
		"SELECT feature_key, is_enabled, feature_value FROM plan_features WHERE plan_id = $1", tier)
	if err != nil {
		return PlanFeaturesRow{Tier: tier, PlanName: tier, Features: make(map[string]bool)}, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var enabled bool
		var value string
		if err := rows.Scan(&key, &enabled, &value); err != nil {
			continue
		}
		p.Features[key] = enabled
	}

	p.PlanName = tier
	p.syncHasFields()

	// 4. Save to Cache (TTL 1 hour)
	if cache.Client != nil {
		if b, err := json.Marshal(p); err == nil {
			cache.Client.Set(ctx, cacheKey, string(b), 1*time.Hour)
		}
	}

	return p, nil
}
