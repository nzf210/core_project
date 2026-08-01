package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/db"
)

const featureDefCacheKey = "feature_def:"
const featureDefCacheTTL = 10 * 60 // 10 minutes

// FeatureDef describes a feature from the available_features registry.
type FeatureDef struct {
	Key            string
	Name           string
	Description   string
	Category       string
	IsAddon       bool
	DefaultEnabled []string
	PriceRupiah    int64
	Unit          string
}

// GetFeatureDef returns feature metadata from available_features (cached).
func GetFeatureDef(ctx context.Context, featureKey string) (*FeatureDef, error) {
	if db.Pool == nil {
		return nil, nil
	}
	cacheKey := featureDefCacheKey + featureKey

	if cache.Client != nil {
		if val, err := cache.Client.Get(ctx, cacheKey).Result(); err == nil && val != "" {
			var fd FeatureDef
			if json.Unmarshal([]byte(val), &fd) == nil {
				return &fd, nil
			}
		}
	}

	var fd FeatureDef
	var defaultEnabled []string
	err := db.Pool.QueryRow(ctx,
		`SELECT feature_key, feature_name, description, category, is_addon,
		        default_enabled, addon_price_rupiah, addon_unit
		 FROM available_features WHERE feature_key = $1`, featureKey).Scan(
		&fd.Key, &fd.Name, &fd.Description, &fd.Category,
		&fd.IsAddon, &defaultEnabled, &fd.PriceRupiah, &fd.Unit,
	)
	if err != nil {
		return nil, nil
	}
	fd.DefaultEnabled = defaultEnabled

	if cache.Client != nil {
		if b, _ := json.Marshal(fd); b != nil {
			cache.Client.Set(ctx, cacheKey, string(b), featureDefCacheTTL)
		}
	}
	return &fd, nil
}

// InvalidateFeatureDefCache removes the cached feature definition.
func InvalidateFeatureDefCache(ctx context.Context, featureKey string) {
	if cache.Client != nil {
		cache.Client.Del(ctx, featureDefCacheKey+featureKey)
	}
}

// checkFeatureInPlanMap checks if feature exists in plan_features map (handles aliases).
func checkFeatureInPlanMap(pf PlanFeaturesRow, featureKey string) (bool, bool) {
	keys := []string{featureKey}
	if featureKey == "ai" {
		keys = append(keys, "ai_requests", "ai_text")
	}
	for _, k := range keys {
		if v, exists := pf.Features[k]; exists {
			return v, true
		}
	}
	return false, false
}

// checkDefaultEnabled checks if feature is enabled by default for tenant's plan.
func checkDefaultEnabled(ctx context.Context, tenantID, featureKey string, feat *FeatureDef) bool {
	tenantPlan := GetTenantPlan(ctx, tenantID)
	if feat != nil {
		for _, t := range feat.DefaultEnabled {
			if t == tenantPlan {
				return true
			}
		}
	}
	return false
}

// checkAliasDefaultEnabled checks alias feature definitions for default_enabled.
func checkAliasDefaultEnabled(ctx context.Context, tenantID, featureKey string) bool {
	if featureKey != "ai" {
		return false
	}
	tenantPlan := GetTenantPlan(ctx, tenantID)
	for _, alias := range []string{"ai_requests", "ai_text"} {
		if aliasFeat, _ := GetFeatureDef(ctx, alias); aliasFeat != nil {
			for _, t := range aliasFeat.DefaultEnabled {
				if t == tenantPlan {
					return true
				}
			}
		}
	}
	return false
}

// CanUseFeature is the primary feature gate.
// Returns (allowed, reason).
func CanUseFeature(ctx context.Context, tenantID, featureKey string) (bool, string) {
	if tier := GetTenantPlan(ctx, tenantID); tier == "superadmin" {
		return true, ""
	}

	feat, _ := GetFeatureDef(ctx, featureKey)

	if feat != nil && feat.IsAddon {
		ok, _ := CanUseAddon(ctx, tenantID, featureKey)
		return ok, boolToReason(ok, feat.Name, GetTenantPlan(ctx, tenantID))
	}

	pf, _ := GetPlanFeatures(ctx, tenantID)
	if enabled, exists := checkFeatureInPlanMap(pf, featureKey); exists {
		return enabled, ""
	}

	if checkDefaultEnabled(ctx, tenantID, featureKey, feat) {
		return true, ""
	}

	if checkAliasDefaultEnabled(ctx, tenantID, featureKey) {
		return true, ""
	}

	if feat != nil && feat.IsAddon {
		if ok, _ := CanUseAddon(ctx, tenantID, featureKey); ok {
			return true, ""
		}
	}

	name := featureKey
	if feat != nil {
		name = feat.Name
	}
	return false, fmt.Sprintf("Fitur %s tidak tersedia di paket %s.", name, GetTenantPlan(ctx, tenantID))
}

// tierPriority returns numeric priority for tier comparison.
// Higher = more capable. Used for min_tier enforcement.
func tierPriority(tier string) int {
	switch tier {
	case "superadmin":
		return 100
	case "ultimate":
		return 4
	case "pro":
		return 3
	case "lite":
		return 2
	default: // inactive, unknown
		return 0
	}
}

// CanUseAddon checks if a tenant has an active addon.
// If the addon has a min_tier requirement in plan_features, the tenant's
// plan tier must meet or exceed it before checking tenant_addons.
func CanUseAddon(ctx context.Context, tenantID, addonKey string) (bool, error) {
	if db.Pool == nil {
		return false, nil
	}

	// 1. Check min_tier requirement
	tenantTier := GetTenantPlan(ctx, tenantID)
	if tierPriority(tenantTier) == 0 {
		return false, nil // inactive or unknown
	}

	var minTier *string
	err := db.Pool.QueryRow(ctx,
		`SELECT min_tier FROM plan_features
		 WHERE plan_id = $1 AND feature_key = $2 AND min_tier IS NOT NULL`,
		tenantTier, addonKey).Scan(&minTier)
	if err == nil && minTier != nil {
		if tierPriority(tenantTier) < tierPriority(*minTier) {
			return false, nil // tenant tier too low to even purchase this addon
		}
	}

	// 2. Check active tenant_addons purchase
	cacheKey := "addon_check:" + tenantID + ":" + addonKey
	if cache.Client != nil {
		if val, err := cache.Client.Get(ctx, cacheKey).Result(); err == nil && val == "1" {
			return true, nil
		}
	}

	var exists bool
	err = db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM tenant_addons
			WHERE tenant_id = $1 AND addon_key = $2
			  AND status = 'active'
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`, tenantID, addonKey).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists && cache.Client != nil {
		cache.Client.Set(ctx, cacheKey, "1", 60)
	}
	return exists, nil
}

// InvalidateAddonCache removes the cached addon check result.
func InvalidateAddonCache(ctx context.Context, tenantID, addonKey string) {
	if cache.Client != nil {
		cache.Client.Del(ctx, "addon_check:"+tenantID+":"+addonKey)
	}
}

func boolToReason(ok bool, name, tier string) string {
	if ok {
		return ""
	}
	return fmt.Sprintf("Fitur %s tidak tersedia di paket %s.", name, tier)
}

// isEnabledViaPlan checks the dynamic Features map and falls back to default_enabled.
// Not called from within this package anymore, but kept exported in case
// external consumers reference it directly.
func isEnabledViaPlan(pf PlanFeaturesRow, featureKey string) bool {
	if pf.Features[featureKey] {
		return true
	}
	feat, _ := GetFeatureDef(context.Background(), featureKey)
	if feat != nil {
		for _, t := range feat.DefaultEnabled {
			if t == pf.Tier {
				return true
			}
		}
	}
	return false
}

