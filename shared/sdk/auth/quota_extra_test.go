package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTenantPlan_NilCacheNilDB(t *testing.T) {
	// Both cache.Client and db.Pool are nil in test env
	result := GetTenantPlan(context.Background(), "any-tenant")
	if result != "inactive" {
		t.Errorf("expected inactive, got %s", result)
	}
}

func TestSetTenantPlan_NilCache(t *testing.T) {
	// Should not panic when cache.Client is nil
	SetTenantPlan(context.Background(), "tenant-1", "pro")
}

func TestCheckQuota_NilCache(t *testing.T) {
	// cache.Client is nil — should return true, -1 (pass-through)
	ok, limit := CheckQuota("tenant-1", "transactions")
	if !ok {
		t.Error("expected true (allow) when cache is nil")
	}
	if limit != -1 {
		t.Errorf("expected -1 limit when cache is nil, got %d", limit)
	}
}

func TestCheckQuota_UnknownResourceDefault(t *testing.T) {
	ok, limit := CheckQuota("tenant-1", "unknown_resource")
	if !ok {
		t.Error("expected true for unknown resource")
	}
	if limit != -1 {
		t.Errorf("expected -1 for unknown resource, got %d", limit)
	}
}

func TestIncrementUsage_NilCache(t *testing.T) {
	// Should not panic when cache.Client is nil
	IncrementUsage("tenant-1", "transactions", 1)
}

func TestCheckWalletBalance_NilDB(t *testing.T) {
	// db.Pool nil → skip check, return true (test/mock mode)
	ok := CheckWalletBalance(context.Background(), "tenant-1", 1000)
	if !ok {
		t.Error("expected true (pass-through) when db.Pool is nil")
	}
}

func TestDeductWalletBalance_NilDB(t *testing.T) {
	// db.Pool nil → no-op, return nil
	err := DeductWalletBalance(context.Background(), "tenant-1", 1000, "test", "test deduction")
	if err != nil {
		t.Errorf("expected nil error when db.Pool is nil, got %v", err)
	}
}

func TestAddonPricePerUnit_NilDB(t *testing.T) {
	price := AddonPricePerUnit(context.Background(), "pos_addon")
	if price != 0 {
		t.Errorf("expected 0 when db.Pool is nil, got %d", price)
	}
}

func TestConsumeWalletAddon_NilDB(t *testing.T) {
	// Should not panic — price=0 from nil DB means no-op
	ConsumeWalletAddon(context.Background(), "tenant-1", "pos_addon")
}

func TestRequireFeature_NoTenantContext(t *testing.T) {
	mw := RequireFeature("chatbot")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No tenant ID in context
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called without tenant context")
	}
}

func TestRequireFeature_WithTenantContext(t *testing.T) {
	mw := RequireFeature("chatbot")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	// With no DB/cache, CanUseFeature returns false → 403
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 (no plan), got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called when feature denied")
	}
}

func TestQuotaMiddlewareFeature_NoTenantContext(t *testing.T) {
	mw := QuotaMiddlewareFeature("ai_text")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("next should not be called without tenant context")
	}
}

func TestQuotaMiddlewareFeature_WithTenantContext(t *testing.T) {
	mw := QuotaMiddlewareFeature("ai_text")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	// cache.Client is nil → IncrementQuota no-ops, no limit → passes through
	if !called {
		t.Error("next should be called when cache is nil (no quota enforcement)")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSyncHasFields(t *testing.T) {
	p := &PlanFeaturesRow{
		Features: map[string]bool{
			"pos":             true,
			"chatbot":         true,
			"ai_requests":     true,
			"accounting":      false,
			"reports":         true,
			"inventory":       false,
			"api_access":      true,
			"multi_user":      false,
			"custom_branding": true,
			"priority_support": false,
			"advanced_reports": true,
			"wa_cloud_api":    true,
		},
	}
	p.syncHasFields()

	if !p.HasPOS {
		t.Error("expected HasPOS true")
	}
	if !p.HasChatbot {
		t.Error("expected HasChatbot true")
	}
	if !p.HasAI {
		t.Error("expected HasAI true")
	}
	if p.HasAccounting {
		t.Error("expected HasAccounting false")
	}
	if !p.HasReports {
		t.Error("expected HasReports true")
	}
	if p.HasInventory {
		t.Error("expected HasInventory false")
	}
	if !p.HasAPIAccess {
		t.Error("expected HasAPIAccess true")
	}
	if !p.HasWACloudAPI {
		t.Error("expected HasWACloudAPI true")
	}
}
