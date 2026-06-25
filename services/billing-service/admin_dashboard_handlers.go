package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/response"
)

func handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()

	// Tenant counts
	var (
		totalTenants, activeTenants, frozenTenants int
	)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&totalTenants)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE is_frozen = false`).Scan(&activeTenants)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE is_frozen = true`).Scan(&frozenTenants)

	// Voucher stats (last 30 days)
	var (
		linksGenerated30d, linksRedeemed30d int
	)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_links WHERE created_at >= NOW() - INTERVAL '30 days'`).Scan(&linksGenerated30d)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_links WHERE redeemed_at >= NOW() - INTERVAL '30 days'`).Scan(&linksRedeemed30d)

	// Active programs
	var activePrograms int
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_programs WHERE is_active = true`).Scan(&activePrograms)

	// Revenue (Xendit fallback, last 30 days, in sen)
	var revenue30d int64
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM invoices
		WHERE status = 'paid' AND paid_at >= NOW() - INTERVAL '30 days'
	`).Scan(&revenue30d)

	// Recent frozen accounts (top 10)
	rows, _ := DB.Query(ctx, `
		SELECT t.id, t.name, t.plan, t.frozen_at, t.current_plan_expires_at
		FROM tenants t WHERE t.is_frozen = true
		ORDER BY t.frozen_at DESC NULLS LAST LIMIT 10
	`)
	recentFrozen := []map[string]interface{}{}
	if rows != nil {
		for rows.Next() {
			var id, name, plan string
			var frozenAt, expiresAt *time.Time
			if rows.Scan(&id, &name, &plan, &frozenAt, &expiresAt) == nil {
				entry := map[string]interface{}{
					"id":   id,
					"name": name,
					"plan": plan,
				}
				if frozenAt != nil {
					entry["frozen_at"] = frozenAt.Format(time.RFC3339)
				}
				if expiresAt != nil {
					entry["expired_at"] = expiresAt.Format(time.RFC3339)
				}
				recentFrozen = append(recentFrozen, entry)
			}
		}
		rows.Close()
	}

	// Active subscriptions by plan (using tenants table for source of truth of active plans)
	planRows, _ := DB.Query(ctx, `
		SELECT plan, COUNT(*) FROM tenants
		WHERE is_frozen = false GROUP BY plan
	`)
	subsByPlan := map[string]int{}
	if planRows != nil {
		for planRows.Next() {
			var pid string
			var cnt int
			if planRows.Scan(&pid, &cnt) == nil {
				subsByPlan[pid] = cnt
			}
		}
		planRows.Close()
	}

	response.JSON(w, http.StatusOK, "Dashboard data", map[string]interface{}{
		"tenants": map[string]interface{}{
			"total":  totalTenants,
			"active": activeTenants,
			"frozen": frozenTenants,
		},
		"vouchers_30d": map[string]interface{}{
			"links_generated": linksGenerated30d,
			"links_redeemed":  linksRedeemed30d,
			"active_programs": activePrograms,
		},
		"revenue_30d_sen": revenue30d,
		"recent_frozen":   recentFrozen,
		"subs_by_plan":    subsByPlan,
	})
}

// nilStr returns "" for any string (helper for null target_plan_id)
func nilStr() string { return "" }

// generateVoucherCode creates an auto-generated voucher code for a payment
// Format: {PLAN_PREFIX}-{TIMESTAMP}-{RANDOM}
func generateVoucherCode(planID, tenantID string) string {
	planPrefix := map[string]string{
		"lite":       "LITE",
		"pro":        "PRO",
		"enterprise": "ENT",
	}
	prefix := planPrefix[planID]
	if prefix == "" {
		prefix = "WCH"
	}

	// Use timestamp + tenant hash for uniqueness
	timestamp := time.Now().Unix() % 100000
	shortTenant := tenantID
	if len(shortTenant) > 4 {
		shortTenant = shortTenant[len(shortTenant)-4:]
	}

	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, strings.ToUpper(shortTenant))
}

