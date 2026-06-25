package main

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

var (
	rateLimitWindow = 60 * time.Second
	rateLimitPublic = 30
	rateLimitAuth   = 10
	rateLimitPerIP  = 200
	mu              sync.Mutex
)

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
		limits := map[string]int{"lite": 300, "pro": 1000, "enterprise": 999999, "ultimate": 999999}
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
