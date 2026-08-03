package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// authRateLimit wraps a handler with IP-based sliding window rate limiting.
// limit = max requests per minute from a single IP.
func authRateLimit(limit int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Redis == nil {
			next(w, r)
			return
		}

		ip := clientIP(r)
		key := fmt.Sprintf("auth_rl:%s:%s", r.URL.Path, ip)
		now := time.Now().UnixMilli()
		windowStart := now - 60000 // 1-minute sliding window

		ctx := r.Context()
		pipe := Redis.TxPipeline()
		pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		countCmd := pipe.ZCard(ctx, key)
		pipe.Expire(ctx, key, 2*time.Minute)

		if _, err := pipe.Exec(ctx); err != nil {
			slog.Error("auth rate limiter redis error", "error", err)
			next(w, r)
			return
		}

		if countCmd.Val() > limit {
			writeJSON(w, http.StatusTooManyRequests, Response{
				Success: false,
				Message: "Terlalu banyak permintaan. Coba lagi dalam 1 menit.",
			})
			return
		}

		next(w, r)
	}
}

// clientIP extracts the real client IP, respecting X-Forwarded-For from trusted proxies.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) IP — the original client
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
