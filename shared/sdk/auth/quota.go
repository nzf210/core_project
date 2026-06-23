package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/db"
	"core_project/shared/sdk/response"
)

func GetTenantPlan(ctx context.Context, tenantID string) string {
	if cache.Client != nil {
		val, err := cache.Client.Get(ctx, "tenant:plan:"+tenantID).Result()
		if err == nil && val != "" {
			return val
		}
	}
	// Fallback to DB if cache miss
	if db.Pool != nil {
		var tier string
		err := db.Pool.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&tier)
		if err == nil && tier != "" {
			// populate cache for next time
			if cache.Client != nil {
				cache.Client.Set(ctx, "tenant:plan:"+tenantID, tier, 30*24*time.Hour)
			}
			return tier
		}
	}
	return "inactive"
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

// CheckWalletBalance checks if the tenant has enough wallet credits (in rupiah).
func CheckWalletBalance(ctx context.Context, tenantID string, amountRupiah int64) bool {
	if db.Pool == nil {
		return true // skip if db not wired (test/mock mode)
	}
	var balance int64
	err := db.Pool.QueryRow(ctx, "SELECT balance_rupiah FROM wallet_credits WHERE tenant_id = $1", tenantID).Scan(&balance)
	if err != nil {
		return false // missing row or error means 0 balance
	}
	return balance >= amountRupiah
}

// DeductWalletBalance deducts credits and logs the transaction.
func DeductWalletBalance(ctx context.Context, tenantID string, amountRupiah int64, ref string, desc string) error {
	if db.Pool == nil {
		return nil
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	
	_, err = tx.Exec(ctx, "UPDATE wallet_credits SET balance_rupiah = balance_rupiah - $1, updated_at = NOW() WHERE tenant_id = $2 AND balance_rupiah >= $1", amountRupiah, tenantID)
	if err != nil { return err }
	_, err = tx.Exec(ctx, "INSERT INTO wallet_transactions (tenant_id, amount_rupiah, transaction_type, reference, description) VALUES ($1, $2, 'consume', $3, $4)", tenantID, -amountRupiah, ref, desc)
	if err != nil { return err }
	return tx.Commit(ctx)
}

// HasFeatureAccess checks if a tenant has access to a specific feature.
// DEPRECATED: Use CanUseFeature instead. This only checks bundled plan features,
// not addon purchases. Kept for backward compatibility during F052 transition.
// Returns (allowed, reason) where reason is a human-readable message if denied.
func HasFeatureAccess(tenantID string, feature string) (bool, string) {
	plan := GetPlan(tenantID)

	switch feature {
	case "pos":
		if !plan.HasPOS && plan.Tier != "superadmin" {
			return false, "Fitur POS memerlukan paket Lite atau lebih tinggi."
		}
	case "chatbot":
		if !plan.HasChatbot && plan.Tier != "superadmin" {
			return false, "Fitur Chatbot memerlukan paket Lite atau lebih tinggi."
		}
	case "ai":
		if !plan.HasAI && plan.Tier != "superadmin" {
			return false, "Fitur AI memerlukan paket Lite atau lebih tinggi."
		}
	case "inventory":
		if !plan.HasInventory && plan.Tier != "superadmin" {
			return false, "Fitur Inventory memerlukan paket Lite atau lebih tinggi."
		}
	case "reports":
		if !plan.HasReports && plan.Tier != "superadmin" {
			return false, "Fitur Laporan memerlukan paket Lite atau lebih tinggi."
		}
	case "multi_user":
		if !plan.HasMultiUser && plan.Tier != "superadmin" {
			return false, "Fitur Multi-User memerlukan paket Lite atau lebih tinggi."
		}
	case "api_access":
		if !plan.HasAPIAccess && plan.Tier != "superadmin" {
			return false, "API Access memerlukan paket Pro."
		}
	case "accounting":
		// All plans have accounting
	}

	return true, ""
}

// RequireFeature returns a middleware that blocks access to a feature
// if the tenant's plan does not include it. Delegates to CanUseFeature
// so both bundled plan features and addon purchases are honored.
// Usage: mux.Handle("/api/umkm/pos/", auth.RequireFeature("pos")(auth.Middleware(handler)))
func RequireFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := r.Context().Value(TenantIDKey).(string)
			if !ok || tenantID == "" {
				response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
				return
			}

			allowed, reason := CanUseFeature(r.Context(), tenantID, feature)
			if !allowed {
				w.Header().Set("X-Feature-Gate", "denied")
				w.Header().Set("X-Required-Feature", feature)
				w.Header().Set("X-Plan-Tier", GetTenantPlan(r.Context(), tenantID))
				response.Error(w, http.StatusForbidden, reason, nil)
				return
			}

			w.Header().Set("X-Feature-Gate", "allowed")
			w.Header().Set("X-Required-Feature", feature)
			w.Header().Set("X-Plan-Tier", GetTenantPlan(r.Context(), tenantID))
			next.ServeHTTP(w, r)
		})
	}
}
