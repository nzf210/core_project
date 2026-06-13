package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

var (
	rateLimitWindow  = 60 * time.Second
	rateLimitPublic  = 30
	rateLimitAuth    = 10
	rateLimitPerIP   = 200
	mu               sync.Mutex
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	if err := cache.InitRedis(cfg); err != nil {
		slog.Warn("Redis not available, rate limiting disabled", "error", err)
	}

	mux := http.NewServeMux()

	getTarget := func(service string, port string) string {
		if cfg.Env == "production" {
			return "http://" + service + ":" + port
		}
		return "http://localhost:" + port
	}

	// Public routes (Auth & Public Campaign API) — rate limited
	mux.Handle("/auth/", rateLimitMiddleware(rateLimitPublic)(http.StripPrefix("/auth", newProxy(getTarget("auth-service", "8001")))))
	mux.Handle("/api/public/campaign/", rateLimitMiddleware(rateLimitPublic)(http.StripPrefix("/api/public/campaign", newProxy(getTarget("campaign-api", "9002")))))

	// Uploads — public static files
	mux.Handle("/uploads/", http.StripPrefix("/uploads", newProxy(getTarget("auth-service", "8001")+"/static")))

	// Webhooks — public, lenient rate limit, signature-verified by downstream service
	// Path convention: /webhooks/<service>/<event>
	// - Xendit: POST /webhooks/xendit/{invoice.paid,subscription.activated,subscription.expired}
	// - wa-gateway: POST /webhooks/wa/{message.status,device.status} (whatsmeow internal)
	// - wa-cloud-api: POST /webhooks/wa-cloud (Meta Cloud API callbacks)
	// - n8n:    POST /webhooks/n8n/{workflow_name}  (custom workflow trigger)
	mux.Handle("/webhooks/xendit/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks", newProxy(getTarget("billing-service", "8003")))))
	mux.Handle("/webhooks/wa/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks", newProxy(getTarget("wa-gateway", "8202")))))
	mux.Handle("/webhooks/wa-cloud/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks/wa-cloud", newProxy(getTarget("wa-cloud-api", "8210")+"/webhook"))))
	mux.Handle("/webhooks/n8n/", rateLimitMiddleware(rateLimitPublic*5)(http.StripPrefix("/webhooks", newProxy(getTarget("n8n", "5678")))))

	// Superadmin routes — protected with auth + role check
	mux.Handle("/api/superadmin/", auth.Middleware(http.StripPrefix("/api/superadmin", newTenantProxy(getTarget("auth-service", "8001")+"/superadmin"))))
	mux.Handle("/api/superadmin/billing/", auth.Middleware(http.StripPrefix("/api/superadmin/billing", newTenantProxy(getTarget("billing-service", "8003")+"/admin"))))
	mux.Handle("/api/superadmin/n8n/", auth.Middleware(http.StripPrefix("/api/superadmin/n8n", n8nProxy(getTarget("n8n", "5678")))))

	// Profile routes — user can edit own profile
	mux.Handle("/api/profile", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget("auth-service", "8001"))))))
	mux.Handle("/api/profile/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget("auth-service", "8001"))))))
	// /me route — lightweight GET endpoint for frontend router guard to re-sync
	// onboarding_completed, plan, role, is_frozen on every page reload.
	// Fixes the onboarding redirect loop when localStorage flags are missing.
	mux.Handle("/api/me", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api", newTenantProxy(getTarget("auth-service", "8001"))))))
	mux.Handle("/api/ai/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(auth.RequireFeature("ai")(http.StripPrefix("/api/ai", newTenantProxy(getTarget("ai-gateway", "8002"))))))))
	mux.Handle("/api/umkm/business/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/umkm/business", newTenantProxy(getTarget("umkm-business", "9005"))))))
	mux.Handle("/api/umkm/automation/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/umkm/automation", newTenantProxy(getTarget("umkm-automation", "8203"))))))
	mux.Handle("/api/umkm/chat", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/umkm", newTenantProxy(getTarget("umkm-chatbot", "8203")))))))
	mux.Handle("/api/umkm/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/umkm", newTenantProxy(getTarget("umkm-accounting", "8201")))))))
	mux.Handle("/api/campaign/", auth.Middleware(tenantRateLimitMiddleware(quotaMiddleware(http.StripPrefix("/api/campaign", newTenantProxy(getTarget("campaign-api", "9002")))))))
	mux.Handle("/api/billing/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/billing", newTenantProxy(getTarget("billing-service", "8003"))))))
	mux.Handle("/plans", http.StripPrefix("", newProxy(getTarget("billing-service", "8003"))))
	mux.Handle("/subscribe", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget("billing-service", "8003"))))))
	mux.Handle("/subscription", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget("billing-service", "8003"))))))
	mux.Handle("/voucher/redeem", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("", newTenantProxy(getTarget("billing-service", "8003"))))))
	mux.Handle("/api/wa/", auth.Middleware(tenantRateLimitMiddleware(auth.RequireFeature("chatbot")(newTenantProxy(getTarget("wa-gateway", "8202"))))))
	mux.Handle("/api/notifications/", auth.Middleware(tenantRateLimitMiddleware(http.StripPrefix("/api/notifications", newTenantProxy(getTarget("notification-service", "8005"))))))

	// Aggregated health check — panggil semua service & return ringkas
	mux.HandleFunc("/healthz", handleAggregatedHealthz(getTarget, cfg))

	// Prometheus metrics endpoint — aggregate dari semua service
	mux.HandleFunc("/metrics", handleAggregatedMetrics(getTarget, cfg))

	server := &http.Server{
		Addr:         ":8000",
		Handler:      corsMiddleware(ipRateLimitMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("API Gateway listening", "port", 8000)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start API Gateway", "error", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Tenant-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions || r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func newProxy(targetHost string) *httputil.ReverseProxy {
	url, _ := url.Parse(targetHost)
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Del("Access-Control-Allow-Origin")
		r.Header.Del("Access-Control-Allow-Methods")
		r.Header.Del("Access-Control-Allow-Headers")
		r.Header.Del("Access-Control-Allow-Credentials")
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
		r.Header.Del("Access-Control-Allow-Origin")
		r.Header.Del("Access-Control-Allow-Methods")
		r.Header.Del("Access-Control-Allow-Headers")
		r.Header.Del("Access-Control-Allow-Credentials")
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
		r.Header.Del("Access-Control-Allow-Origin")
		r.Header.Del("Access-Control-Allow-Methods")
		r.Header.Del("Access-Control-Allow-Headers")
		r.Header.Del("Access-Control-Allow-Credentials")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Proxy error", "target", targetHost, "error", err)
		response.Error(w, http.StatusBadGateway, "Service unavailable", err)
	}

	return proxy
}

func quotaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
		if !ok || tenantID == "" {
			response.Error(w, http.StatusUnauthorized, "Tenant ID missing in context", nil)
			return
		}

		plan := auth.GetPlan(tenantID)

		if plan.Tier == "lite" && r.Method != http.MethodGet && r.Method != http.MethodOptions {
			ok, used := auth.CheckQuota(tenantID, "transactions")
			if !ok {
				response.Error(w, http.StatusPaymentRequired,
					"Lite tier quota exceeded. Upgrade to Pro plan to continue.", nil)
				slog.Warn("Quota exceeded", "tenant", tenantID, "resource", "transactions", "used", used)
				return
			}
			go auth.IncrementUsage(tenantID, "transactions", 1)
		}

		next.ServeHTTP(w, r)
	})
}

// handleAggregatedHealthz melakukan ping cepat ke semua downstream service
// dan mengembalikan status agregat. Berguna untuk monitoring & orchestration
// (Kubernetes liveness probe, uptime monitoring, dsb).
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   map[bool]string{true: "healthy", false: "degraded"}[allHealthy],
			"env":      cfg.Env,
			"services": results,
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// handleAggregatedMetrics aggregates metrics from all downstream services
func handleAggregatedMetrics(getTarget func(string, string) string, cfg *config.Config) http.HandlerFunc {
	type serviceMetrics struct {
		name    string
		port    string
		endpoint string
	}

	svcPorts := []serviceMetrics{
		{"wa-gateway", "8202", "/metrics"},
		{"umkm-chatbot", "8203", "/metrics"},
		{"auth-service", "8001", "/health"},
		{"ai-gateway", "8002", "/health"},
		{"billing-service", "8003", "/health"},
		{"umkm-accounting", "8201", "/health"},
		{"campaign-api", "9002", "/health"},
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

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
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

func ipRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cache.Client == nil {
			next.ServeHTTP(w, r)
			return
		}

		ip := getClientIP(r)
		key := "rate_limit:ip:" + ip

		count, err := cache.Client.Incr(r.Context(), key).Result()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			cache.Client.Expire(r.Context(), key, rateLimitWindow)
		}

		if count > int64(rateLimitPerIP) {
			response.Error(w, http.StatusTooManyRequests, "Rate limit exceeded. Try again later.", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func rateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cache.Client == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			key := "rate_limit:public:" + ip

			count, err := cache.Client.Incr(r.Context(), key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				cache.Client.Expire(r.Context(), key, rateLimitWindow)
			}

			if count > int64(limit) {
				response.Error(w, http.StatusTooManyRequests, "Too many requests. Please wait.", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func tenantRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cache.Client == nil {
			next.ServeHTTP(w, r)
			return
		}

		tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		plan := auth.GetPlan(tenantID)
		limits := map[string]int{"free": 60, "lite": 300, "pro": 1000, "enterprise": 999999, "ultimate": 999999}
		limit := limits[plan.Tier]
		if limit == 0 {
			limit = 60
		}

		key := "rate_limit:tenant:" + tenantID
		count, err := cache.Client.Incr(r.Context(), key).Result()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			cache.Client.Expire(r.Context(), key, rateLimitWindow)
		}

		if count > int64(limit) {
			response.Error(w, http.StatusTooManyRequests, "Tenant rate limit exceeded, upgrade for higher limits.", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
