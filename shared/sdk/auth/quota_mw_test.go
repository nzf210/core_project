package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuotaMiddlewareFeature_AllowsWhenUnderLimit(t *testing.T) {
	var called bool
	handler := QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "test-tenant")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if !called {
		t.Error("expected handler to be called when under limit")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestQuotaMiddlewareFeature_RejectsWhenTenantMissing(t *testing.T) {
	handler := QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should NOT be called when tenant context missing")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
