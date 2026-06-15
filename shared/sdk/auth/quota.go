package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

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
	case "ai_vision":
		limit = plan.MaxAIVision
	case "ai_audio_stt":
		limit = plan.MaxAIAudioMinutes
	case "ai_audio_tts":
		limit = plan.MaxAIAudioMinutes
	case "image_gen":
		limit = plan.MaxImageGen
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
