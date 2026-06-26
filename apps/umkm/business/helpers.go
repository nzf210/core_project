package main

import (
	"context"

	"core_project/shared/sdk/auth"
)

func getDashboardForType(bt string) []DashboardWidget {
	if widgets, ok := widgetTemplates[bt]; ok {
		return widgets
	}
	return widgetTemplates["umum"]
}

func getModuleListForType(bt string) []map[string]interface{} {
	modules := []map[string]interface{}{}

	baseModules := []string{"transactions", "customers", "reports", "pos"}
	if bt == "warung" || bt == "restoran" || bt == "toko_online" {
		baseModules = append(baseModules, "inventory", "supplier", "best_seller")
	}
	if bt == "clinic" {
		baseModules = append(baseModules, "patient_records", "appointment_scheduling", "pharmacy")
	}

	for _, mod := range baseModules {
		if label, ok := moduleLabels[mod]; ok {
			modules = append(modules, map[string]interface{}{
				"id":    mod,
				"label": label,
			})
		}
	}
	return modules
}

func getMaxStoresForTenant(ctx context.Context, tenantID string) int {
	plan := auth.GetTenantPlan(ctx, tenantID)
	switch plan {
	case "lite":
		return 1
	case "pro":
		return 3
	case "enterprise", "ultimate":
		return 10
	default:
		return 0
	}
}
