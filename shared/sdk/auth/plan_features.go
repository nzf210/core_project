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

// defaultPlanFeatures returns hardcoded feature defaults for known tiers.
// Used when DB/Redis unavailable to prevent feature gate failures.
func defaultPlanFeatures(tier string) PlanFeaturesRow {
	switch tier {
	case "superadmin":
		return PlanFeaturesRow{Tier: "superadmin", PlanName: "Superadmin", MaxUsers: -1, MaxTransactions: -1, MaxAIText: -1, MaxAIVision: -1, MaxAIAudioMinutes: -1, MaxImageGen: -1, MaxProducts: -1, MaxCustomers: -1, MaxStorageMB: -1, HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true, HasAdvancedReport: true, HasCustomBranding: true, HasPrioritySupport: true}
	case "ultimate":
		return PlanFeaturesRow{Tier: "ultimate", PlanName: "Ultimate", MaxUsers: -1, MaxTransactions: -1, MaxAIText: -1, MaxAIVision: 500, MaxAIAudioMinutes: 60, MaxImageGen: 30, MaxProducts: -1, MaxCustomers: -1, MaxStorageMB: -1, HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true, HasAdvancedReport: true, HasCustomBranding: true, HasPrioritySupport: true}
	case "pro":
		return PlanFeaturesRow{Tier: "pro", PlanName: "Pro", MaxUsers: 10, MaxTransactions: 10000, MaxAIText: 5000, MaxAIVision: 50, MaxAIAudioMinutes: 0, MaxImageGen: 0, MaxProducts: 1000, MaxCustomers: 5000, MaxStorageMB: 10000, HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true, HasAdvancedReport: false, HasCustomBranding: false, HasPrioritySupport: true}
	case "lite":
		return PlanFeaturesRow{Tier: "lite", PlanName: "Lite", MaxUsers: 3, MaxTransactions: 1000, MaxAIText: 250, MaxAIVision: 0, MaxAIAudioMinutes: 0, MaxImageGen: 0, MaxProducts: 100, MaxCustomers: 500, MaxStorageMB: 1000, HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: false, HasAPIAccess: false, HasAdvancedReport: false, HasCustomBranding: false, HasPrioritySupport: false}
	default:
		return PlanFeaturesRow{Tier: tier, PlanName: tier}
	}
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
		// Return default features for known tiers when DB unavailable
		return defaultPlanFeatures(tier), nil
	}

	// Superadmin gets all features (before DB query)
	if tier == "superadmin" {
		p := PlanFeaturesRow{
			Tier:               "superadmin",
			PlanName:           "Superadmin",
			MaxUsers:           -1,
			MaxTransactions:    -1,
			MaxAIText:          -1,
			MaxAIVision:        -1,
			MaxAIAudioMinutes:  -1,
			MaxImageGen:        -1,
			MaxProducts:        -1,
			MaxCustomers:       -1,
			MaxStorageMB:       -1,
			APIRateLimitPerMin: 9999,
			DataRetentionMonths: 999,
			HasAccounting:      true,
			HasPOS:             true,
			HasChatbot:         true,
			HasAI:              true,
			HasInventory:       true,
			HasReports:         true,
			HasMultiUser:        true,
			HasAPIAccess:       true,
			HasAdvancedReport:  true,
			HasCustomBranding:  true,
			HasPrioritySupport: true,
		}
		// Cache superadmin features too
		if cache.Client != nil {
			if b, _ := json.Marshal(p); b != nil {
				cache.Client.Set(ctx, cacheKey, string(b), 1*time.Hour)
			}
		}
		return p, nil
	}

	// Query all feature_key rows for this plan and map to boolean flags
	var p PlanFeaturesRow
	p.Tier = tier
	rows, err := db.Pool.Query(ctx, "SELECT feature_key, is_enabled, feature_value FROM plan_features WHERE plan_id = $1", tier)
	if err != nil {
		return PlanFeaturesRow{Tier: tier, PlanName: tier}, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var enabled bool
		var value string
		if err := rows.Scan(&key, &enabled, &value); err != nil {
			continue
		}
		switch key {
		case "chatbot":
			p.HasChatbot = enabled
		case "pos":
			p.HasPOS = enabled
		case "ai_requests":
			p.HasAI = enabled
		case "accounting":
			p.HasAccounting = enabled
		case "reports":
			p.HasReports = enabled
		case "inventory":
			p.HasInventory = enabled
		case "api_access":
			p.HasAPIAccess = enabled
		case "multi_user":
			p.HasMultiUser = enabled
		case "custom_branding":
			p.HasCustomBranding = enabled
		case "priority_support":
			p.HasPrioritySupport = enabled
		}
	}

	p.PlanName = tier

	// 4. Save to Cache (TTL 1 hour)
	if cache.Client != nil {
		if b, err := json.Marshal(p); err == nil {
			cache.Client.Set(ctx, cacheKey, string(b), 1*time.Hour)
		}
	}

	return p, nil
}
