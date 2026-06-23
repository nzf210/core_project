package auth

import (
	"context"
	"net/http"
	"strconv"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/db"
	"core_project/shared/sdk/response"
)

// AddonPricePerUnit returns the price in rupiah for an addon feature.
// Returns 0 if not an addon or not found in available_features.
func AddonPricePerUnit(ctx context.Context, addonKey string) int64 {
	if db.Pool == nil {
		return 0
	}
	var price int64
	_ = db.Pool.QueryRow(ctx,
		"SELECT addon_price_rupiah FROM available_features WHERE feature_key = $1 AND is_addon = true",
		addonKey).Scan(&price)
	return price
}

// ConsumeWalletAddon deducts the addon price from tenant wallet if applicable.
// safe: no-op if price=0 or insufficient balance.
func ConsumeWalletAddon(ctx context.Context, tenantID, addonKey string) {
	price := AddonPricePerUnit(ctx, addonKey)
	if price <= 0 {
		return
	}
	if !CheckWalletBalance(ctx, tenantID, price) {
		return
	}
	_ = DeductWalletBalance(ctx, tenantID, price, "addon:"+addonKey, "Consumption: "+addonKey)
	if cache.Client != nil {
		InvalidateAddonCache(ctx, tenantID, addonKey)
	}
}


// QuotaMiddlewareFeature returns an HTTP middleware that enforces a per-feature quota.
// Reads tenant ID from request context (must be set by auth middleware upstream).
// On quota exceeded, returns 402 Payment Required with headers X-Quota-Feature/Used/Limit.
func QuotaMiddlewareFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := r.Context().Value(TenantIDKey).(string)
			if !ok || tenantID == "" {
				response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
				return
			}
			
			// Check AND Increment BEFORE routing
			count, limit, err := IncrementQuota(r.Context(), tenantID, feature, 1)
			
			if err != nil || (limit != -1 && count > int64(limit)) {
				w.Header().Set("X-Quota-Feature", feature)
				w.Header().Set("X-Quota-Used", strconv.FormatInt(count, 10))
				w.Header().Set("X-Quota-Limit", strconv.FormatInt(int64(limit), 10))
				response.JSON(w, http.StatusPaymentRequired, "Quota exceeded for feature: "+feature+". Upgrade your plan.", map[string]any{
					"feature": feature,
					"used":    count,
					"limit":   limit,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
