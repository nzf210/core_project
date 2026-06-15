package auth

import (
	"context"
	"fmt"
	"time"

	"core_project/shared/sdk/cache"
)

const counterKeyPrefix = "quota_counter:"

func currentPeriod() string {
	return time.Now().UTC().Format("200601")
}

func currentPeriodEnd() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

// IncrementQuota increments the counter for a tenant+feature in current period.
// Uses Redis fast path (atomic INCRBY). DB write is best-effort.
// Returns (currentCount, limit, error). If no cache wired, returns (0, -1, nil).
func IncrementQuota(ctx context.Context, tenantID, feature string, delta int) (int64, int, error) {
	period := currentPeriod()
	redisKey := fmt.Sprintf("%s%s:%s:%s", counterKeyPrefix, tenantID, period, feature)

	var count int64
	if cache.Client != nil {
		newCount, err := cache.Client.IncrBy(ctx, redisKey, int64(delta)).Result()
		if err == nil {
			cache.Client.ExpireAt(ctx, redisKey, currentPeriodEnd().Add(48*time.Hour))
			count = newCount
		}
	}

	// DB persist (best-effort stub - real impl in later task)
	_ = persistQuotaCounter(ctx, tenantID, period, feature, count)

	plan, _ := GetPlanFeatures(ctx, tenantID)
	limit := getFeatureLimit(plan, feature)
	return count, limit, nil
}

// CheckQuotaCounter returns (ok, used, limit).
// When no cache/DB, returns (true, 0, -1) = allowed (no enforcement).
func CheckQuotaCounter(ctx context.Context, tenantID, feature string) (bool, int64, int) {
	// No cache wired = no enforcement path available; allow through.
	if cache.Client == nil {
		return true, 0, -1
	}
	period := currentPeriod()
	redisKey := fmt.Sprintf("%s%s:%s:%s", counterKeyPrefix, tenantID, period, feature)

	var used int64
	if val, err := cache.Client.Get(ctx, redisKey).Int64(); err == nil {
		used = val
	}
	plan, _ := GetPlanFeatures(ctx, tenantID)
	limit := getFeatureLimit(plan, feature)
	if limit == -1 {
		return true, used, limit
	}
	return used < int64(limit), used, limit
}

// getFeatureLimit returns the limit for a given feature on a plan.
// -1 means unlimited. 0 means feature disabled for this tier.
func getFeatureLimit(p PlanFeaturesRow, feature string) int {
	switch feature {
	case "ai_text":
		return p.MaxAIText
	case "ai_vision":
		return p.MaxAIVision
	case "ai_audio_stt":
		return p.MaxAIAudioMinutes
	case "ai_audio_tts":
		return p.MaxAIAudioMinutes
	case "image_gen":
		return p.MaxImageGen
	case "chatbot_messages":
		return p.MaxAIText // Using MaxAIText from DB instead of hardcoded
	}
	return 0
}

// persistQuotaCounter writes to DB (stub - real impl in later task using upsert).
func persistQuotaCounter(ctx context.Context, tenantID, period, feature string, count int64) error {
	// Real impl: INSERT INTO quota_counters (...) ON CONFLICT (tenant_id, period_yyyymm, feature_key) DO UPDATE SET count = $4, updated_at = NOW()
	_ = ctx
	_ = tenantID
	_ = period
	_ = feature
	_ = count
	return nil
}
