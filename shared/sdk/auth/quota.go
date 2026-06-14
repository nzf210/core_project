package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

// FeatureFlags represents per-feature access within a plan tier.
type FeatureFlags struct {
	HasAccounting  bool `json:"has_accounting"`
	HasPOS         bool `json:"has_pos"`
	HasChatbot     bool `json:"has_chatbot"`
	HasAI          bool `json:"has_ai"`
	HasInventory   bool `json:"has_inventory"`
	HasReports     bool `json:"has_reports"`
	HasMultiUser   bool `json:"has_multi_user"`
	HasAPIAccess   bool `json:"has_api_access"`
}

type PlanTier struct {
	Tier              string        `json:"tier"`
	MaxUsers          int           `json:"max_users"`
	MaxTransactions   int           `json:"max_transactions"`
	MaxAIRequests     int           `json:"max_ai_requests"`
	MaxBots           int           `json:"max_bots"`
	CanExport         bool          `json:"can_export"`
	HasAdvancedReport bool          `json:"has_advanced_report"`
	HasMultiUser      bool          `json:"has_multi_user"`
	Features          FeatureFlags  `json:"features"`
}

var Plans = map[string]PlanTier{
	"inactive": {Tier: "inactive", MaxUsers: 0, MaxTransactions: 0, MaxAIRequests: 0, MaxBots: 0, CanExport: false, HasAdvancedReport: false, HasMultiUser: false, Features: FeatureFlags{HasAccounting: false, HasPOS: false, HasChatbot: false, HasAI: false, HasInventory: false, HasReports: false, HasMultiUser: false, HasAPIAccess: false}},
	"lite":     {Tier: "lite", MaxUsers: 3, MaxTransactions: 1000, MaxAIRequests: 250, MaxBots: 0, CanExport: true, HasAdvancedReport: false, HasMultiUser: true, Features: FeatureFlags{HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: false}},
	"pro":      {Tier: "pro", MaxUsers: 10, MaxTransactions: 10000, MaxAIRequests: 5000, MaxBots: 3, CanExport: true, HasAdvancedReport: true, HasMultiUser: true, Features: FeatureFlags{HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true}},
	"enterprise": {Tier: "enterprise", MaxUsers: -1, MaxTransactions: -1, MaxAIRequests: -1, MaxBots: -1, CanExport: true, HasAdvancedReport: true, HasMultiUser: true, Features: FeatureFlags{HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true}},
	"ultimate": {Tier: "ultimate", MaxUsers: -1, MaxTransactions: -1, MaxAIRequests: -1, MaxBots: -1, CanExport: true, HasAdvancedReport: true, HasMultiUser: true, Features: FeatureFlags{HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true}},
	"superadmin": {Tier: "superadmin", MaxUsers: -1, MaxTransactions: -1, MaxAIRequests: -1, MaxBots: -1, CanExport: true, HasAdvancedReport: true, HasMultiUser: true, Features: FeatureFlags{HasAccounting: true, HasPOS: true, HasChatbot: true, HasAI: true, HasInventory: true, HasReports: true, HasMultiUser: true, HasAPIAccess: true}},
}

func GetTenantPlan(ctx context.Context, tenantID string) string {
	if cache.Client == nil {
		return "inactive"
	}
	val, err := cache.Client.Get(ctx, "tenant:plan:"+tenantID).Result()
	if err != nil || val == "" {
		return "inactive"
	}
	return val
}

func SetTenantPlan(ctx context.Context, tenantID, tier string) {
	if cache.Client != nil {
		cache.Client.Set(ctx, "tenant:plan:"+tenantID, tier, 30*24*time.Hour)
	}
}

func GetPlan(tenantID string) PlanFeaturesRow {
	row, err := GetPlanFeatures(context.Background(), tenantID)
	if err != nil {
		return PlanFeaturesRow{Tier: "inactive"}
	}
	return row
}

func CheckQuota(tenantID string, resource string) (bool, int) {
	if cache.Client == nil {
		return true, -1
	}
	ctx := context.Background()
	plan := GetPlan(tenantID)

	key := "quota:" + tenantID + ":" + resource
	used, _ := cache.Client.Get(ctx, key).Int()
	var limit int

	switch resource {
	case "transactions":
		limit = plan.MaxTransactions
	case "ai_text":
		limit = plan.MaxAIText
	// TODO(F025 Phase 2): add cases for ai_vision, ai_audio_stt, ai_audio_tts, image_gen, ocr_scans, chatbot_messages
	// when quota middleware (Task 2.3) routes by modality. Current CheckQuota is only called for
	// "transactions" resource; AI modality cases deferred to quota_counter.go in Phase 2.
	default:
		return true, -1
	}

	if limit == -1 {
		return true, limit
	}

	return used < limit, used
}

func IncrementUsage(tenantID string, resource string, delta int) {
	if cache.Client == nil {
		return
	}
	ctx := context.Background()
	key := "quota:" + tenantID + ":" + resource
	cache.Client.IncrBy(ctx, key, int64(delta))
	cache.Client.Expire(ctx, key, 30*24*time.Hour)
}

func QuotaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := r.Context().Value(TenantIDKey).(string)
		if !ok || tenantID == "" {
			response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
			return
		}

		plan := GetPlan(tenantID)
		quotaInfo := map[string]interface{}{
			"tier": plan.Tier,
			"limits": map[string]interface{}{
				"users":        plan.MaxUsers,
				"transactions": plan.MaxTransactions,
			},
		}

		w.Header().Set("X-Plan-Tier", plan.Tier)
		quotaJSON, _ := json.Marshal(quotaInfo)
		w.Header().Set("X-Plan-Quota", string(quotaJSON))

		next.ServeHTTP(w, r)
	})
}

// HasFeatureAccess checks if a tenant has access to a specific feature.
// Returns (allowed, reason) where reason is a human-readable message if denied.
func HasFeatureAccess(tenantID string, feature string) (bool, string) {
	plan := GetPlan(tenantID)

	switch feature {
	case "pos":
		if !plan.HasPOS {
			return false, "Fitur POS memerlukan paket Lite atau lebih tinggi."
		}
	case "chatbot":
		if !plan.HasChatbot {
			return false, "Fitur Chatbot memerlukan paket Lite atau lebih tinggi."
		}
	case "ai":
		if !plan.HasAI {
			return false, "Fitur AI memerlukan paket Lite atau lebih tinggi."
		}
	case "inventory":
		if !plan.HasInventory {
			return false, "Fitur Inventory memerlukan paket Lite atau lebih tinggi."
		}
	case "reports":
		if !plan.HasReports {
			return false, "Fitur Laporan memerlukan paket Lite atau lebih tinggi."
		}
	case "multi_user":
		if !plan.HasMultiUser {
			return false, "Fitur Multi-User memerlukan paket Lite atau lebih tinggi."
		}
	case "api_access":
		if !plan.HasAPIAccess {
			return false, "API Access memerlukan paket Pro."
		}
	case "accounting":
		// All plans have accounting
	}

	return true, ""
}

// RequireFeature returns a middleware that blocks access to a feature
// if the tenant's plan does not include it. Pass the feature name
// (e.g. "pos", "chatbot", "ai", "inventory").
// Usage: mux.Handle("/api/umkm/pos/", auth.RequireFeature("pos")(auth.Middleware(handler)))
func RequireFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := r.Context().Value(TenantIDKey).(string)
			if !ok || tenantID == "" {
				response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
				return
			}

			allowed, reason := HasFeatureAccess(tenantID, feature)
			if !allowed {
				w.Header().Set("X-Feature-Gate", "denied")
				w.Header().Set("X-Required-Feature", feature)
				w.Header().Set("X-Plan-Tier", GetPlan(tenantID).Tier)
				response.Error(w, http.StatusForbidden, reason, nil)
				return
			}

			w.Header().Set("X-Feature-Gate", "allowed")
			w.Header().Set("X-Required-Feature", feature)
			w.Header().Set("X-Plan-Tier", GetPlan(tenantID).Tier)
			next.ServeHTTP(w, r)
		})
	}
}
