// tests/integration/quota_test.go
//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"core_project/shared/sdk/auth"
)

// TestQuotaCounter_RoundTrip verifies that IncrementQuota + CheckQuotaCounter
// work together: counter increments, then check sees the new value.
//
// Requires Redis and DB to be running. Skip if not available.
func TestQuotaCounter_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	// This test requires Redis + DB. If env not set up, skip.
	if !integrationEnvReady() {
		t.Skip("integration env not ready (Redis/DB required)")
	}

	ctx := context.Background()
	tenantID := "test-tenant-" + time.Now().Format("150405")
	feature := "test_feature"

	// Clean up before
	defer cleanupQuotaCounter(ctx, tenantID, feature)

	// Increment
	count1, limit1, err := auth.IncrementQuota(ctx, tenantID, feature, 1)
	if err != nil {
		t.Fatalf("IncrementQuota: %v", err)
	}
	if count1 != 1 {
		t.Errorf("expected count=1 after first increment, got %d", count1)
	}
	t.Logf("first increment: count=%d, limit=%d", count1, limit1)

	// Increment again
	count2, _, err := auth.IncrementQuota(ctx, tenantID, feature, 5)
	if err != nil {
		t.Fatalf("IncrementQuota: %v", err)
	}
	if count2 != 6 {
		t.Errorf("expected count=6 after second increment, got %d", count2)
	}

	// Check
	ok, used, limit := auth.CheckQuotaCounter(ctx, tenantID, feature)
	if !ok {
		t.Error("expected ok=true (unlimited when no plan features)")
	}
	if used != 6 {
		t.Errorf("expected used=6, got %d", used)
	}
	t.Logf("check: ok=%v, used=%d, limit=%d", ok, used, limit)
}

// Stub helpers — implement or skip if env not ready
func integrationEnvReady() bool {
	// Check Redis client + DB are initialized
	return false // disabled until env test infrastructure exists
}

func cleanupQuotaCounter(ctx context.Context, tenantID, feature string) {
	// DELETE FROM quota_counters WHERE tenant_id = $1 AND feature_key = $2
	// Implement when DB available; stub for now
}
