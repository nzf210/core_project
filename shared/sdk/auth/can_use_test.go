package auth

import (
	"context"
	"testing"
)

func TestBoolToReason_True(t *testing.T) {
	result := boolToReason(true, "chatbot", "lite")
	if result != "" {
		t.Errorf("expected empty string when ok=true, got %q", result)
	}
}

func TestBoolToReason_False(t *testing.T) {
	result := boolToReason(false, "chatbot", "lite")
	if result == "" {
		t.Error("expected non-empty reason when ok=false")
	}
	expected := "Fitur chatbot tidak tersedia di paket lite."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestIsEnabledViaPlan_DirectFeature(t *testing.T) {
	pf := PlanFeaturesRow{
		Tier:     "pro",
		Features: map[string]bool{"chatbot": true},
	}
	if !isEnabledViaPlan(pf, "chatbot") {
		t.Error("expected true when feature is directly enabled in plan")
	}
}

func TestIsEnabledViaPlan_NotEnabled(t *testing.T) {
	pf := PlanFeaturesRow{
		Tier:     "lite",
		Features: map[string]bool{"chatbot": false},
	}
	// db.Pool is nil in test env — GetFeatureDef returns nil
	if isEnabledViaPlan(pf, "chatbot") {
		t.Error("expected false when feature is disabled and no DB")
	}
}

func TestIsEnabledViaPlan_MissingKey(t *testing.T) {
	pf := PlanFeaturesRow{
		Tier:     "lite",
		Features: map[string]bool{},
	}
	if isEnabledViaPlan(pf, "some_feature") {
		t.Error("expected false for missing feature key with no DB")
	}
}

func TestInvalidateFeatureDefCache_NilClient(t *testing.T) {
	// Should not panic when cache.Client is nil
	InvalidateFeatureDefCache(context.Background(), "chatbot")
}

func TestInvalidateAddonCache_NilClient(t *testing.T) {
	// Should not panic when cache.Client is nil
	InvalidateAddonCache(context.Background(), "tenant-1", "pos")
}

func TestGetFeatureDef_NilDB(t *testing.T) {
	// db.Pool is nil in test env — should return nil, nil
	fd, err := GetFeatureDef(context.Background(), "chatbot")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if fd != nil {
		t.Errorf("expected nil FeatureDef when db.Pool is nil, got %+v", fd)
	}
}

func TestCanUseAddon_NilDB(t *testing.T) {
	// db.Pool is nil in test env — should return false, nil
	ok, err := CanUseAddon(context.Background(), "tenant-1", "pos_addon")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if ok {
		t.Error("expected false when db.Pool is nil")
	}
}

func TestCheckFeatureInPlanMap_DirectMatch(t *testing.T) {
	pf := PlanFeaturesRow{
		Features: map[string]bool{"pos": true},
	}
	found, enabled := checkFeatureInPlanMap(pf, "pos")
	if !found {
		t.Error("expected found=true for direct feature match")
	}
	if !enabled {
		t.Error("expected enabled=true")
	}
}

func TestCheckFeatureInPlanMap_AIAlias(t *testing.T) {
	pf := PlanFeaturesRow{
		Features: map[string]bool{"ai_text": true},
	}
	found, enabled := checkFeatureInPlanMap(pf, "ai")
	if !found {
		t.Error("expected found=true via ai_text alias")
	}
	if !enabled {
		t.Error("expected enabled=true via ai_text alias")
	}
}

func TestCheckFeatureInPlanMap_NotFound(t *testing.T) {
	pf := PlanFeaturesRow{
		Features: map[string]bool{},
	}
	found, _ := checkFeatureInPlanMap(pf, "unknown_feature")
	if found {
		t.Error("expected found=false for missing feature")
	}
}

func TestCheckDefaultEnabled_NilFeat(t *testing.T) {
	// feat=nil, no DB — returns false
	enabled := checkDefaultEnabled(context.Background(), "tenant-1", "chatbot", nil)
	if enabled {
		t.Error("expected false when feat is nil")
	}
}

func TestCheckDefaultEnabled_WithFeat(t *testing.T) {
	feat := &FeatureDef{DefaultEnabled: []string{"pro", "ultimate"}}
	// GetTenantPlan returns "inactive" (no cache/db) — not in DefaultEnabled
	enabled := checkDefaultEnabled(context.Background(), "tenant-1", "chatbot", feat)
	if enabled {
		t.Error("expected false when tenant plan not in DefaultEnabled")
	}
}
