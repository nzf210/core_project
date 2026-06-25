package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

func buildCacheKey(provider, model, system, message string) string {
	raw := provider + "|" + model + "|" + system + "|" + message
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("ai:cache:%x", hash[:16])
}

func checkCache(ctx context.Context, key string) (string, bool) {
	if Redis != nil {
		val, err := Redis.Get(ctx, key).Result()
		if err == nil && val != "" {
			return val, true
		}
	}
	return "", false
}

func storeCache(ctx context.Context, key, value string, ttlSeconds int) {
	if Redis != nil {
		Redis.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second)
	}
}
