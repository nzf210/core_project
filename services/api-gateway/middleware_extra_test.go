package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"core_project/shared/sdk/auth"
)

func TestBuildAllowedOrigins_Empty(t *testing.T) {
	os.Unsetenv("ALLOWED_ORIGINS")
	origins := buildAllowedOrigins()
	if len(origins) != 0 {
		t.Errorf("expected empty map, got %v", origins)
	}
}

func TestBuildAllowedOrigins_WithValues(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://app.wch.id, https://admin.wch.id")
	defer os.Unsetenv("ALLOWED_ORIGINS")
	origins := buildAllowedOrigins()
	if len(origins) != 2 {
		t.Errorf("expected 2 origins, got %d", len(origins))
	}
	if _, ok := origins["https://app.wch.id"]; !ok {
		t.Error("expected https://app.wch.id in allowed origins")
	}
}

func TestCORSMiddleware_OptionsWithAllowedOrigin(t *testing.T) {
	old := allowedOrigins
	allowedOrigins = map[string]struct{}{"https://app.wch.id": {}}
	defer func() { allowedOrigins = old }()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.wch.id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS with allowed origin, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.wch.id" {
		t.Errorf("expected origin header set, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_OptionsRejectedOrigin(t *testing.T) {
	old := allowedOrigins
	allowedOrigins = map[string]struct{}{"https://app.wch.id": {}}
	defer func() { allowedOrigins = old }()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := corsMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for OPTIONS with rejected origin, got %d", w.Code)
	}
}

func TestQuotaMiddleware_NoTenantID(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	handler := quotaMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("next should not be called without tenant ID")
	}
}

func TestQuotaMiddleware_WithTenantID(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := quotaMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), auth.TenantIDKey, "tenant-123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// With no cache/DB, quota check passes through
	if !called {
		t.Error("next should be called with tenant ID (no quota enforcement without cache)")
	}
}

func TestTenantRateLimitMiddleware_NilCache(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := tenantRateLimitMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// cache.Client is nil → pass through
	if !called {
		t.Error("next should be called when cache is nil")
	}
}

func TestIPRateLimitMiddleware_NilCache(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := ipRateLimitMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next should be called when cache is nil")
	}
}

func TestRateLimitMiddleware_NilCache(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimitMiddleware(100)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next should be called when cache is nil")
	}
}
