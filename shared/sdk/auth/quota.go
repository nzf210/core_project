package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

type PlanTier struct {
	Tier              string `json:"tier"`
	MaxUsers          int    `json:"max_users"`
	MaxTransactions   int    `json:"max_transactions"`
	MaxAIRequests     int    `json:"max_ai_requests"`
	MaxBots           int    `json:"max_bots"`
	CanExport         bool   `json:"can_export"`
	HasAdvancedReport bool   `json:"has_advanced_report"`
	HasMultiUser      bool   `json:"has_multi_user"`
}

var Plans = map[string]PlanTier{
	"free":  {Tier: "free", MaxUsers: 1, MaxTransactions: 100, MaxAIRequests: 5, MaxBots: 0, CanExport: false, HasAdvancedReport: false, HasMultiUser: false},
	"lite":  {Tier: "lite", MaxUsers: 3, MaxTransactions: 1000, MaxAIRequests: 250, MaxBots: 0, CanExport: true, HasAdvancedReport: false, HasMultiUser: true},
	"pro":   {Tier: "pro", MaxUsers: 10, MaxTransactions: 10000, MaxAIRequests: 5000, MaxBots: 3, CanExport: true, HasAdvancedReport: true, HasMultiUser: true},
	"enterprise": {Tier: "enterprise", MaxUsers: -1, MaxTransactions: -1, MaxAIRequests: -1, MaxBots: -1, CanExport: true, HasAdvancedReport: true, HasMultiUser: true},
}

func GetTenantPlan(ctx context.Context, tenantID string) string {
	if cache.Client == nil {
		return "free"
	}
	val, err := cache.Client.Get(ctx, "tenant:plan:"+tenantID).Result()
	if err != nil || val == "" {
		return "free"
	}
	return val
}

func SetTenantPlan(ctx context.Context, tenantID, tier string) {
	if cache.Client != nil {
		cache.Client.Set(ctx, "tenant:plan:"+tenantID, tier, 30*24*time.Hour)
	}
}

func GetPlan(tenantID string) PlanTier {
	plan := GetTenantPlan(context.Background(), tenantID)
	if p, ok := Plans[plan]; ok {
		return p
	}
	return Plans["free"]
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
	case "ai_requests":
		limit = plan.MaxAIRequests
	case "bots":
		limit = plan.MaxBots
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
				"users":         plan.MaxUsers,
				"transactions":  plan.MaxTransactions,
				"ai_requests":   plan.MaxAIRequests,
				"export":        plan.CanExport,
			},
		}

		w.Header().Set("X-Plan-Tier", plan.Tier)
		quotaJSON, _ := json.Marshal(quotaInfo)
		w.Header().Set("X-Plan-Quota", string(quotaJSON))

		next.ServeHTTP(w, r)
	})
}
