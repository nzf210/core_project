package auth

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"core_project/shared/sdk/response"
)

// IsTenantFrozen checks the tenants.is_frozen flag (denormalized from
// tenant_subscriptions.status, maintained by freeze worker).
// Returned via a passed-in pool because the auth package shouldn't depend on DB
// directly — caller wires it up via SetSubscriptionPool.
type SubscriptionStatus struct {
	IsFrozen      bool   `json:"isFrozen"`
	PlanID        string `json:"planId"`
	CurrentPeriodEnd string `json:"currentPeriodEnd,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

var subscriptionPool *pgxpool.Pool

// SetSubscriptionPool wires the DB pool used by subscription checks.
// Call once at app startup: auth.SetSubscriptionPool(DB)
func SetSubscriptionPool(p *pgxpool.Pool) {
	subscriptionPool = p
}

// CheckSubscriptionStatus returns frozen state for a tenant.
// Falls back to "not frozen" if DB not wired.
func CheckSubscriptionStatus(tenantID string) SubscriptionStatus {
	if subscriptionPool == nil || tenantID == "" {
		return SubscriptionStatus{IsFrozen: false}
	}
	var (
		isFrozen bool
		planID string
		expiresAt *string
	)
	row := subscriptionPool.QueryRow(context.Background(), `
		SELECT COALESCE(is_frozen, false),
		       COALESCE(plan, 'lite'),
		       COALESCE(current_plan_expires_at::text, '')
		FROM tenants WHERE id = $1
	`, tenantID)
	if err := row.Scan(&isFrozen, &planID, &expiresAt); err != nil {
		return SubscriptionStatus{IsFrozen: false, PlanID: "lite"}
	}
	st := SubscriptionStatus{IsFrozen: isFrozen, PlanID: planID}
	if expiresAt != nil {
		st.CurrentPeriodEnd = *expiresAt
	}
	return st
}

// RequireActiveSubscription is a middleware that blocks write methods
// (POST/PATCH/PUT/DELETE) when the tenant is frozen. GET requests still pass
// (so user can browse history, view banner, etc — read-only mode).
// Apply AFTER auth.Middleware, e.g.:
//   handler := auth.RequireActiveSubscription(auth.Middleware(mux))
func RequireActiveSubscription(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		tenantID, _ := r.Context().Value(TenantIDKey).(string)
		if tenantID == "" {
			next.ServeHTTP(w, r)
			return
		}
		st := CheckSubscriptionStatus(tenantID)
		if st.IsFrozen {
			w.Header().Set("X-Subscription-Status", "frozen")
			w.Header().Set("X-Subscription-Plan", st.PlanID)
			response.Error(w, http.StatusForbidden, "Akun Anda dalam masa freeze. Redeem voucher untuk mengaktifkan kembali.", nil)
			return
		}
		w.Header().Set("X-Subscription-Status", "active")
		w.Header().Set("X-Subscription-Plan", st.PlanID)
		next.ServeHTTP(w, r)
	})
}
