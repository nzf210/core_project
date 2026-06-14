package auth

import (
	"net/http"
	"strconv"

	"core_project/shared/sdk/response"
)

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
				response.JSON(w, http.StatusPaymentRequired, "Quota exceeded for feature: "+feature+". Upgrade your plan.", map[string]interface{}{
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
