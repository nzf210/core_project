package main

import (
	"net/http"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

func handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	ctx := r.Context()
	var planID, status string
	var currentPeriodEnd *time.Time
	var planTier string
	var activatedBy string

	err := DB.QueryRow(ctx, `
		SELECT ts.plan_id, ts.status, ts.current_period_end, ts.plan_tier, ts.activated_by
		FROM tenant_subscriptions ts
		WHERE ts.tenant_id = $1
	`, tenantID).Scan(&planID, &status, &currentPeriodEnd, &planTier, &activatedBy)

	if err != nil || planID == "" {
		response.JSON(w, http.StatusOK, "No active subscription", map[string]interface{}{
			"has_subscription": false,
		})
		return
	}

	var planName string
	var priceMonthly int64
	DB.QueryRow(ctx, "SELECT name, price_monthly FROM saas_plans WHERE id = $1", planID).Scan(&planName, &priceMonthly)

	rows, _ := DB.Query(ctx, `
		SELECT feature_key, feature_name, feature_value
		FROM plan_features WHERE plan_id = $1 AND is_enabled = true
	`, planID)
	var features []map[string]string
	if rows != nil {
		for rows.Next() {
			var key, name, val string
			if rows.Scan(&key, &name, &val) == nil {
				features = append(features, map[string]string{
					"key":   key,
					"name":  name,
					"value": val,
				})
			}
		}
		rows.Close()
	}

	var periodEndStr *string
	if currentPeriodEnd != nil {
		s := currentPeriodEnd.Format(time.RFC3339)
		periodEndStr = &s
	}

	response.JSON(w, http.StatusOK, "Subscription retrieved", map[string]interface{}{
		"has_subscription": true,
		"plan_id":          planID,
		"plan_name":        planName,
		"plan_tier":        planTier,
		"price_monthly":    priceMonthly,
		"status":           status,
		"period_end":       periodEndStr,
		"features":         features,
		"activated_by":     activatedBy,
	})
}
