package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/auth"
)

func TestNewProxy_ModifyResponse(t *testing.T) {
	proxy := newProxy("http://localhost:8001")
	if proxy == nil {
		t.Fatal("expected non-nil proxy")
	}
	// Test ModifyResponse strips CORS headers
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Access-Control-Allow-Origin", "*")
	resp.Header.Set("Access-Control-Allow-Methods", "GET")
	if err := proxy.ModifyResponse(resp); err != nil {
		t.Errorf("ModifyResponse error: %v", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected CORS header stripped")
	}
}

func TestN8nProxy_ModifyResponse(t *testing.T) {
	proxy := n8nProxy("http://localhost:5678")
	if proxy == nil {
		t.Fatal("expected non-nil proxy")
	}
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Access-Control-Allow-Origin", "*")
	resp.Header.Set("X-Frame-Options", "DENY")
	resp.Header.Set("Content-Security-Policy", "default-src 'self'")
	if err := proxy.ModifyResponse(resp); err != nil {
		t.Errorf("ModifyResponse error: %v", err)
	}
	if resp.Header.Get("X-Frame-Options") != "" {
		t.Error("expected X-Frame-Options stripped")
	}
	if resp.Header.Get("Content-Security-Policy") != "" {
		t.Error("expected CSP header stripped")
	}
}

func TestNewTenantProxy_ModifyResponse(t *testing.T) {
	proxy := newTenantProxy("http://localhost:8201")
	if proxy == nil {
		t.Fatal("expected non-nil proxy")
	}
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Access-Control-Allow-Credentials", "true")
	if err := proxy.ModifyResponse(resp); err != nil {
		t.Errorf("ModifyResponse error: %v", err)
	}
	if resp.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Error("expected CORS credentials header stripped")
	}
}

func TestNewTenantProxy_Director_WithContext(t *testing.T) {
	proxy := newTenantProxy("http://localhost:8201")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ctx := context.WithValue(req.Context(), auth.TenantIDKey, "tenant-abc")
	ctx = context.WithValue(ctx, auth.UserIDKey, "user-xyz")
	ctx = context.WithValue(ctx, auth.RoleKey, "owner")
	req = req.WithContext(ctx)

	proxy.Director(req)

	if req.Header.Get("X-Tenant-ID") != "tenant-abc" {
		t.Errorf("expected X-Tenant-ID=tenant-abc, got %s", req.Header.Get("X-Tenant-ID"))
	}
	if req.Header.Get("X-User-ID") != "user-xyz" {
		t.Errorf("expected X-User-ID=user-xyz, got %s", req.Header.Get("X-User-ID"))
	}
	if req.Header.Get("X-User-Role") != "owner" {
		t.Errorf("expected X-User-Role=owner, got %s", req.Header.Get("X-User-Role"))
	}
}

func TestNewTenantProxy_ErrorHandler(t *testing.T) {
	proxy := newTenantProxy("http://127.0.0.1:1") // unreachable
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	proxy.ErrorHandler(w, req, http.ErrHandlerTimeout)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}
