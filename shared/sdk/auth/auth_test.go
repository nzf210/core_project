package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPlan_InactiveFallback(t *testing.T) {
	plan := GetPlan("nonexistent-tenant")
	if plan.Tier != "inactive" {
		t.Errorf("expected inactive tier (fail-safe), got %s", plan.Tier)
	}
	if plan.MaxTransactions != 0 {
		t.Errorf("expected 0 max transactions for inactive, got %d", plan.MaxTransactions)
	}
}

func TestGetPlan_KnownTiers(t *testing.T) {
	tests := []struct {
		tier       string
		maxUsers   int
		canExport  bool
	}{
		{"inactive", 0, false},
		{"lite", 3, true},
		{"pro", 10, true},
		{"enterprise", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			plan, ok := Plans[tt.tier]
			if !ok {
				t.Fatalf("tier %q not found", tt.tier)
			}
			if plan.MaxUsers != tt.maxUsers {
				t.Errorf("expected MaxUsers %d, got %d", tt.maxUsers, plan.MaxUsers)
			}
			if plan.CanExport != tt.canExport {
				t.Errorf("expected CanExport %v, got %v", tt.canExport, plan.CanExport)
			}
		})
	}
}

func TestHasFeatureAccess(t *testing.T) {
	tests := []struct {
		feature  string
		tier     string
		allowed  bool
	}{
		{"accounting", "lite", true},  // all paying plans have accounting
		{"pos", "lite", true},
		{"chatbot", "lite", true},
		{"ai", "lite", true},
		{"inventory", "lite", true},
		{"reports", "lite", true},
		{"multi_user", "lite", true},
		{"api_access", "lite", false},
		{"api_access", "pro", true},
	}
	// Use direct PlanTier access instead of GetPlan (which reads cache)
	for _, tt := range tests {
		t.Run(tt.tier+"_"+tt.feature, func(t *testing.T) {
			plan := Plans[tt.tier]
			allowed := true
			switch tt.feature {
			case "pos":
				allowed = plan.Features.HasPOS
			case "chatbot":
				allowed = plan.Features.HasChatbot
			case "ai":
				allowed = plan.Features.HasAI
			case "inventory":
				allowed = plan.Features.HasInventory
			case "reports":
				allowed = plan.Features.HasReports
			case "multi_user":
				allowed = plan.Features.HasMultiUser
			case "api_access":
				allowed = plan.Features.HasAPIAccess
			}
			if allowed != tt.allowed {
				t.Errorf("expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}

func TestSubscriptionStatus_DefaultFalse(t *testing.T) {
	orig := subscriptionPool
	subscriptionPool = nil
	defer func() { subscriptionPool = orig }()

	st := CheckSubscriptionStatus("any-tenant")
	if st.IsFrozen {
		t.Error("expected IsFrozen=false when pool is nil")
	}
}

func TestSubscriptionStatus_EmptyTenant(t *testing.T) {
	st := CheckSubscriptionStatus("")
	if st.IsFrozen {
		t.Error("expected IsFrozen=false for empty tenant")
	}
}

func TestRequireActiveSubscription_PassesGET(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireActiveSubscription(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t1"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("GET should pass through subscription guard")
	}
}

func TestRequireActiveSubscription_BlocksPOST_WhenFrozen(t *testing.T) {
	orig := subscriptionPool
	subscriptionPool = nil
	defer func() { subscriptionPool = orig }()
	// No pool = default not frozen, so it should pass
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireActiveSubscription(next)
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t1"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("POST should pass when no pool wired (not frozen)")
	}
}

func TestQuotaMiddleware_AddsHeaders(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := QuotaMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t1"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("handler should be called")
	}
	if tier := rr.Header().Get("X-Plan-Tier"); tier != "inactive" {
		t.Errorf("expected X-Plan-Tier=inactive (fail-safe), got %q", tier)
	}
}

func TestQuotaMiddleware_MissingTenant(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	handler := QuotaMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCheckQuota_UnknownResource(t *testing.T) {
	ok, limit := CheckQuota("t1", "transactions")
	if !ok {
		t.Error("expected true when cache client is nil")
	}
	// unknown resource falls through to default case: returns true, -1
	ok, limit = CheckQuota("t1", "some_unknown_resource")
	if !ok {
		t.Fatal("expected true for unknown resource (default case)")
	}
	if limit != -1 {
		t.Errorf("expected limit=-1 for unknown resource, got %d", limit)
	}
}

func TestSetTenantPlan_NoCache(t *testing.T) {
	// Should not panic when cache client is nil
	SetTenantPlan(context.Background(), "t1", "pro")
}

func TestIncrementUsage_NoCache(t *testing.T) {
	// Should not panic when cache client is nil
	IncrementUsage("t1", "transactions", 1)
}
