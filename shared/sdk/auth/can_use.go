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

// CanUseFeature is the primary feature gate.
// Returns (allowed, reason).
func CanUseFeature(ctx context.Context, tenantID, featureKey string) (bool, string) {
	// 1. Superadmin always allowed
	if tier := GetTenantPlan(ctx, tenantID); tier == "superadmin" {
		return true, ""
	}

	feat, _ := GetFeatureDef(ctx, featureKey)

	// 2. If this is an addon-only feature, delegate to CanUseAddon
	if feat != nil && feat.IsAddon {
		ok, _ := CanUseAddon(ctx, tenantID, featureKey)
		return ok, boolToReason(ok, feat.Name, GetTenantPlan(ctx, tenantID))
	}

	// 3. Bundled feature: check plan_features via PlanFeaturesRow
	pf, _ := GetPlanFeatures(ctx, tenantID)
	if isEnabledViaPlan(pf, featureKey) {
		return true, ""
	}

	// 4. Not enabled via plan — check if it's an addon the tenant purchased
	if feat != nil && feat.IsAddon {
		ok, _ := CanUseAddon(ctx, tenantID, featureKey)
		if ok {
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

func isEnabledViaPlan(pf PlanFeaturesRow, featureKey string) bool {
	switch featureKey {
	case "pos":
		return pf.HasPOS
	case "chatbot":
		return pf.HasChatbot
	case "ai", "ai_text":
		return pf.HasAI
	case "inventory":
		return pf.HasInventory
	case "reports":
		return pf.HasReports
	case "multi_user":
		return pf.HasMultiUser
	case "api_access":
		return pf.HasAPIAccess
	case "advanced_reports":
		return pf.HasAdvancedReport
	case "custom_branding":
		return pf.HasCustomBranding
	case "priority_support":
		return pf.HasPrioritySupport
	case "accounting":
		return pf.HasAccounting
	case "wa_cloud_api":
		return false // only via addon purchase
	}
	return false
}
