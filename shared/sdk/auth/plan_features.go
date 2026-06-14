package auth

import "context"

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
// STUB: always returns inactive (locked) for now. Real implementation (DB read + Redis cache)
// will be added in a later task. The stub is fail-safe: unknown tenant = no access.
func GetPlanFeatures(ctx context.Context, tenantID string) (PlanFeaturesRow, error) {
	_ = ctx
	_ = tenantID
	return PlanFeaturesRow{Tier: "inactive", PlanName: "inactive"}, nil
}
