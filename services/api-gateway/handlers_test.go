package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"core_project/shared/sdk/config"
)

func TestHandleReferralLinkRedirect_WithCode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/r/ABC123", nil)
	w := httptest.NewRecorder()
	handleReferralLinkRedirect(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "ABC123") {
		t.Errorf("expected referral code in redirect, got %s", loc)
	}
}

func TestHandleReferralLinkRedirect_EmptyCode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/r/", nil)
	w := httptest.NewRecorder()
	handleReferralLinkRedirect(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/" {
		t.Errorf("expected redirect to /, got %s", w.Header().Get("Location"))
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if ip := getClientIP(req); ip != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "5.6.7.8")
	if ip := getClientIP(req); ip != "5.6.7.8" {
		t.Errorf("expected 5.6.7.8, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.10.11.12:1234"
	if ip := getClientIP(req); ip != "9.10.11.12:1234" {
		t.Errorf("expected 9.10.11.12:1234, got %s", ip)
	}
}

func TestHandleAggregatedHealthz_AllDown(t *testing.T) {
	cfg := &config.Config{Env: "test"}
	getTarget := func(name, port string) string {
		return "http://127.0.0.1:1" // unreachable port
	}
	handler := handleAggregatedHealthz(getTarget, cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when all services down, got %d", w.Code)
	}
}

func TestHandleAggregatedMetrics_AllDown(t *testing.T) {
	cfg := &config.Config{Env: "test"}
	getTarget := func(name, port string) string {
		return "http://127.0.0.1:1"
	}
	handler := handleAggregatedMetrics(getTarget, cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "wch_platform_info") {
		t.Error("expected wch_platform_info metric in output")
	}
	if !strings.Contains(body, "wch_services_up_total") {
		t.Error("expected wch_services_up_total metric in output")
	}
}

func TestHandleAggregatedMetrics_AllUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{Env: "test"}
	getTarget := func(name, port string) string { return srv.URL }
	handler := handleAggregatedMetrics(getTarget, cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleAggregatedHealthz_AllUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{Env: "test"}
	getTarget := func(name, port string) string {
		return srv.URL
	}
	handler := handleAggregatedHealthz(getTarget, cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when all services up, got %d", w.Code)
	}
}
