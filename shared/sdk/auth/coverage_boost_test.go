package auth

import (
	"context"
	"testing"
)

// Tests covering uncovered branches in quota.go and can_use.go
// All functions here are no-op / nil-guard paths (no real DB/cache in test env)

func TestCheckQuota_AllResources_NilCache(t *testing.T) {
	resources := []string{"transactions", "ai_text", "ai_vision", "ai_audio_stt", "ai_audio_tts", "image_gen"}
	for _, r := range resources {
		ok, limit := CheckQuota("tenant-1", r)
		if !ok {
			t.Errorf("CheckQuota(%q): expected true when cache nil", r)
		}
		if limit != -1 {
			t.Errorf("CheckQuota(%q): expected limit=-1 when cache nil, got %d", r, limit)
		}
	}
}

func TestIncrementUsage_AllResources_NilCache(t *testing.T) {
	// Should not panic for any resource when cache is nil
	resources := []string{"transactions", "ai_text", "ai_vision", "image_gen"}
	for _, r := range resources {
		IncrementUsage("tenant-1", r, 1)
	}
}

func TestCanUseAddon_NilDB_Returns_False(t *testing.T) {
	ok, err := CanUseAddon(context.Background(), "tenant-1", "wa_cloud_api")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if ok {
		t.Error("expected false when db.Pool is nil")
	}
}

func TestInvalidateAddonCache_NilClient_NoPanic(t *testing.T) {
	// Should not panic
	InvalidateAddonCache(context.Background(), "tenant-1", "wa_cloud_api")
	InvalidateFeatureDefCache(context.Background(), "chatbot")
}

func TestGetPlan_Inactive_NoDBNoCache(t *testing.T) {
	p := GetPlan("nonexistent")
	if p.Tier != "inactive" {
		t.Errorf("expected inactive, got %s", p.Tier)
	}
	if p.Features == nil {
		t.Error("expected non-nil Features map")
	}
}

func TestSetTenantPlan_NilCacheNoPanic(t *testing.T) {
	SetTenantPlan(context.Background(), "tenant-1", "pro")
}

func TestCanUseFeature_NoDBNoCache_Denied(t *testing.T) {
	// With no DB/cache, all features should be denied (inactive plan)
	allowed, reason := CanUseFeature(context.Background(), "tenant-1", "chatbot")
	if allowed {
		t.Error("expected denied when no DB/cache")
	}
	if reason == "" {
		t.Error("expected non-empty reason when denied")
	}
}

func TestIncrementQuota_NilCache(t *testing.T) {
	count, limit, err := IncrementQuota(context.Background(), "tenant-1", "ai_text", 1)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	// No cache → count stays 0
	if count != 0 {
		t.Errorf("expected count=0 with nil cache, got %d", count)
	}
	_ = limit // limit depends on GetPlanFeatures result
}

func TestConsumeWalletAddon_NilDB_NoPanic(t *testing.T) {
	// With nil DB, AddonPricePerUnit returns 0 → no-op
	ConsumeWalletAddon(context.Background(), "tenant-1", "pos_addon")
}
