package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/config"
)

const (
	healthEndpoint      = "/health"
	headerContentType   = "Content-Type"
	contentTypeJSON     = "application/json"
	contentTypeTextPlain = "text/plain; version=0.0.4"
)

func handleReferralLinkRedirect(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/r/")
	segments := strings.SplitN(path, "/", 2)
	code := segments[0]
	if code == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/register?referral_code="+code, http.StatusFound)
}

func handleAggregatedHealthz(getTarget func(string, string) string, cfg *config.Config) http.HandlerFunc {
	services := []struct {
		name string
		port string
	}{
		{"auth-service", "8001"},
		{"ai-gateway", "8002"},
		{"billing-service", "8003"},
		{"notification-service", "8005"},
		{"subscription-worker", "8006"},
		{"umkm-accounting", "8201"},
		{"wa-gateway", "8202"},
		{"wa-cloud-api", "8210"},
		{"campaign-api", "9002"},
		{"umkm-business", "9005"},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		results := make(map[string]string, len(services))
		allHealthy := true

		for _, svc := range services {
			healthURL := getTarget(svc.name, svc.port) + "/healthz"
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode >= 500 {
				results[svc.name] = "down"
				allHealthy = false
				continue
			}
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
			results[svc.name] = "up"
		}

		status := http.StatusOK
		if !allHealthy {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     map[bool]string{true: "healthy", false: "degraded"}[allHealthy],
			"env":        cfg.Env,
			"services":   results,
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// DEPRECATED: handleAggregatedMetrics removed — each service now exposes /metrics directly
// Prometheus scrapes individual service endpoints, not aggregated
func handleAggregatedMetrics(getTarget func(string, string) string, cfg *config.Config) http.HandlerFunc {
	type serviceMetrics struct {
		name     string
		port     string
		endpoint string
	}

	svcPorts := []serviceMetrics{
		{"wa-gateway", "8202", "/metrics"},
		{"umkm-chatbot", "8203", "/metrics"},
		{"auth-service", "8001", healthEndpoint},
		{"ai-gateway", "8002", healthEndpoint},
		{"billing-service", "8003", healthEndpoint},
		{"umkm-accounting", "8201", healthEndpoint},
		{"campaign-api", "9002", healthEndpoint},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var sb strings.Builder
		hostname, _ := os.Hostname()

		sb.WriteString("# HELP wch_platform_info WCH Platform info\n")
		sb.WriteString("# TYPE wch_platform_info gauge\n")
		sb.WriteString(fmt.Sprintf("wch_platform_info{env=%q,host=%q} 1\n", cfg.Env, hostname))

		upCount := 0
		for _, svc := range svcPorts {
			url := getTarget(svc.name, svc.port) + svc.endpoint
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode >= 500 {
				sb.WriteString(fmt.Sprintf("wch_service_up{service=%q} 0\n", svc.name))
				continue
			}
			if resp.Body != nil {
				defer resp.Body.Close()
				if body, readErr := io.ReadAll(resp.Body); readErr == nil {
					// Append raw metrics from downstream service
					sb.Write(body)
					sb.WriteString("\n")
				}
			}
			sb.WriteString(fmt.Sprintf("wch_service_up{service=%q} 1\n", svc.name))
			upCount++
		}

		sb.WriteString("# HELP wch_services_up_total Number of services up\n")
		sb.WriteString("# TYPE wch_services_up_total gauge\n")
		sb.WriteString(fmt.Sprintf("wch_services_up_total %d\n", upCount))

		w.Header().Set(headerContentType, contentTypeTextPlain)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sb.String()))
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
