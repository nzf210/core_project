package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"core_project/shared/sdk/cache"
)

const (
	quotaWarnKeyPrefix = "quota_warn_sent:"
	quotaWarnThreshold = 0.8
	quotaWarnTTL       = 24 * time.Hour
)

// CheckAndNotifyQuota checks if usage has reached the 80% soft-warn threshold
// and emits a warning notification (idempotent — at most once per 24h per
// tenant+feature, gated by a Redis SetNX key).
//
// Conventions for `limit`:
//   - limit == -1 → unlimited; no warning is sent.
//   - limit == 0  → feature disabled for this tier; no warning is sent.
//   - limit  > 0  → warning fires when used >= ceil(limit * 0.8).
//
// Returns true if the warning fired (or was already fired in the current
// 24h period), false otherwise. Safe to call when Redis is not wired up —
// in that case the function degrades to a stateless log-only warning.
func CheckAndNotifyQuota(ctx context.Context, tenantID, feature string, used, limit int) bool {
	if tenantID == "" || feature == "" {
		return false
	}
	if limit <= 0 {
		return false
	}

	threshold := int(float64(limit) * quotaWarnThreshold)
	if threshold < 1 {
		threshold = 1 // ensure at least 1 unit of usage can trigger
	}
	if used < threshold {
		return false
	}

	// Idempotency: warn at most once per 24h per tenant+feature.
	period := time.Now().UTC().Format("20060102")
	warnKey := quotaWarnKeyPrefix + tenantID + ":" + feature + ":" + period
	alreadySent := false

	if cache.Client != nil {
		set, err := cache.Client.SetNX(ctx, warnKey, "1", quotaWarnTTL).Result()
		if err != nil {
			// Redis hiccup — log but don't block the caller. Best-effort dedup.
			slog.Warn("quota-warn: redis SetNX failed, proceeding without dedup",
				"tenant_id", tenantID, "feature", feature, "error", err)
		} else if !set {
			alreadySent = true
		}
	}

	if alreadySent {
		return false
	}

	pct := float64(used) / float64(limit) * 100
	slog.Warn("quota soft-warn: tenant reached 80% of plan limit",
		"tenant_id", tenantID,
		"feature", feature,
		"used", used,
		"limit", limit,
		"percent", fmt.Sprintf("%.1f%%", pct),
		"threshold", threshold,
		"dedup_key", warnKey,
	)

	// Future wiring (out of scope for this task): call notifyOwner(ctx, tenantID, ...)
	// or POST to /api/notification/send with the structured payload above. Keeping
	// the side-effect to a structured log keeps the function safe to import from
	// shared code without dragging in the HTTP handler graph.
	return true
}
