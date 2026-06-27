package main

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

const (
	corsOrigin      = "Access-Control-Allow-Origin"
	corsMethods     = "Access-Control-Allow-Methods"
	corsHeaders     = "Access-Control-Allow-Headers"
	corsCredentials = "Access-Control-Allow-Credentials"
)

func newProxy(targetHost string) *httputil.ReverseProxy {
	url, _ := url.Parse(targetHost)
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Del(corsOrigin)
		r.Header.Del(corsMethods)
		r.Header.Del(corsHeaders)
		r.Header.Del(corsCredentials)
		return nil
	}
	return proxy
}

func n8nProxy(targetHost string) *httputil.ReverseProxy {
	targetURL, _ := url.Parse(targetHost)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:n8n_secure_admin_password_123")))
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Del(corsOrigin)
		r.Header.Del(corsMethods)
		r.Header.Del(corsHeaders)
		r.Header.Del(corsCredentials)
		r.Header.Del("X-Frame-Options")
		r.Header.Del("Content-Security-Policy")
		return nil
	}
	return proxy
}

func newTenantProxy(targetHost string) *httputil.ReverseProxy {
	targetURL, _ := url.Parse(targetHost)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.Header.Del("X-Tenant-ID")
		req.Header.Del("X-User-ID")
		req.Header.Del("X-User-Role")

		if tenantID, ok := req.Context().Value(auth.TenantIDKey).(string); ok {
			req.Header.Set("X-Tenant-ID", tenantID)
		}
		if userID, ok := req.Context().Value(auth.UserIDKey).(string); ok {
			req.Header.Set("X-User-ID", userID)
		}
		if role, ok := req.Context().Value(auth.RoleKey).(string); ok {
			req.Header.Set("X-User-Role", role)
		}
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Del(corsOrigin)
		r.Header.Del(corsMethods)
		r.Header.Del(corsHeaders)
		r.Header.Del(corsCredentials)
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Proxy error", "target", targetHost, "error", err)
		response.Error(w, http.StatusBadGateway, "Service unavailable", err)
	}

	return proxy
}
