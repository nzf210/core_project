package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/response"
)

func getTenantVouchers(ctx context.Context, tenantID string) []map[string]interface{} {
	vrows, _ := DB.Query(ctx, `
		SELECT id, plan_id, validity_days, remaining_days, redeemed_at, is_system_generated, source_voucher_code
		FROM voucher_subscriptions WHERE tenant_id = $1 AND remaining_days > 0
		ORDER BY remaining_days DESC
	`, tenantID)
	vouchers := []map[string]interface{}{}
	if vrows != nil {
		for vrows.Next() {
			var vid, pid, srcCode string
			var vd, rd int
			var redeemedAt time.Time
			var isSysGen bool
			if vrows.Scan(&vid, &pid, &vd, &rd, &redeemedAt, &isSysGen, &srcCode) == nil {
				vouchers = append(vouchers, map[string]interface{}{
					"id":                  vid,
					"plan_id":             pid,
					"validity_days":       vd,
					"remaining_days":      rd,
					"redeemed_at":         redeemedAt.Format(time.RFC3339),
					"is_system_generated": isSysGen,
					"source_voucher_code": srcCode,
				})
			}
		}
		vrows.Close()
	}
	return vouchers
}

func handleAdminTenantItem(w http.ResponseWriter, r *http.Request) {
	if !isSuperadmin(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/admin/tenants/")
	parts := strings.Split(path, "/")
	tenantID := parts[0]
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, "Missing tenant ID", nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		// Tenant info
		var tenant map[string]interface{}
		var tID, name, plan string
		var isFrozen bool
		var createdAt time.Time
		var frozenAt, expiresAt *time.Time
		var planPriority int
		err := DB.QueryRow(ctx, `
			SELECT id, name, plan, is_frozen, created_at, frozen_at, current_plan_expires_at, plan_priority
			FROM tenants WHERE id = $1
		`, tenantID).Scan(&tID, &name, &plan, &isFrozen, &createdAt, &frozenAt, &expiresAt, &planPriority)
		if err != nil {
			response.Error(w, http.StatusNotFound, "Tenant not found", nil)
			return
		}

		tenant = map[string]interface{}{
			"id":                 tID,
			"name":               name,
			"plan":               plan,
			"is_frozen":          isFrozen,
			"created_at":         createdAt.Format(time.RFC3339),
			"frozen_at":          formatTime(frozenAt),
			"expires_at":         formatTime(expiresAt),
			"plan_priority":      planPriority,
			"xendit_merchant_id": "",
		}
		// Fetch xendit_merchant_id separately (may be NULL)
		var merchantID *string
		DB.QueryRow(ctx, `SELECT xendit_merchant_id FROM tenants WHERE id = $1`, tenantID).Scan(&merchantID)
		if merchantID != nil {
			tenant["xendit_merchant_id"] = *merchantID
		}

		vouchers := getTenantVouchers(ctx, tenantID)

		// Subscription info
		var subStatus string
		DB.QueryRow(ctx, `SELECT status FROM tenant_subscriptions WHERE tenant_id = $1`, tenantID).Scan(&subStatus)
		tenant["subscription_status"] = subStatus
		tenant["active_vouchers"] = vouchers

		response.JSON(w, http.StatusOK, "Tenant retrieved", tenant)

	case http.MethodPatch:
		var req struct {
			Action string `json:"action"` // "activate", "freeze", "delete"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}

		switch req.Action {
		case "activate":
			_, _ = DB.Exec(ctx, `UPDATE tenants SET is_frozen = false, frozen_at = NULL WHERE id = $1`, tenantID)
			response.JSON(w, http.StatusOK, "Tenant activated", nil)
		case "freeze":
			_, _ = DB.Exec(ctx, `UPDATE tenants SET is_frozen = true, frozen_at = NOW() WHERE id = $1`, tenantID)
			response.JSON(w, http.StatusOK, "Tenant frozen", nil)
		case "delete":
			// Delete tenant + cascade (users, subscriptions, etc)
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "Failed to delete tenant", err)
				return
			}
			slog.Warn("Tenant deleted by superadmin", "tenant_id", tenantID)
			response.JSON(w, http.StatusOK, "Tenant deleted", nil)
		default:
			response.Error(w, http.StatusBadRequest, "action must be: activate, freeze, or delete", nil)
		}

	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

// ─────────────────────────────────────────────
// Superadmin: Cleanup Expired Pending Subscriptions (F015)
// POST /admin/cleanup/pending — manual trigger
// GET  /admin/cleanup/pending — list what would be cleaned
// Runs automatically via subscription-worker cron
// ─────────────────────────────────────────────

func handleAdminCleanupPending(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	ctx := r.Context()

	// List pending tenants past timeout
	rows, err := DB.Query(ctx, `
		SELECT t.id, t.name, t.email, t.plan, ts.pending_expires_at, t.created_at
		FROM tenants t
		JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE ts.status = 'pending'
		  AND ts.pending_expires_at < NOW()
		ORDER BY ts.pending_expires_at ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query pending tenants", err)
		return
	}
	defer rows.Close()

	type pendingTenant struct {
		ID               string     `json:"id"`
		Name             string     `json:"name"`
		Email            string     `json:"email"`
		Plan             string     `json:"plan"`
		PendingExpiredAt *time.Time `json:"pending_expires_at"`
		CreatedAt        time.Time  `json:"created_at"`
	}

	pending := []pendingTenant{}
	for rows.Next() {
		var pt pendingTenant
		if rows.Scan(&pt.ID, &pt.Name, &pt.Email, &pt.Plan, &pt.PendingExpiredAt, &pt.CreatedAt) == nil {
			pending = append(pending, pt)
		}
	}

	if r.Method == http.MethodGet {
		response.JSON(w, http.StatusOK, "Pending tenants (would be cleaned)", map[string]interface{}{
			"count":   len(pending),
			"tenants": pending,
		})
		return
	}

	if r.Method == http.MethodPost {
		// Execute cleanup: delete expired pending tenants + users
		deleted := 0
		for _, pt := range pending {
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, pt.ID)
			if err == nil {
				_, _ = DB.Exec(ctx, `
					INSERT INTO pending_tenant_cleanup_log (tenant_id, user_id, email, phone, reason)
					SELECT $1, id, email, phone_number, 'pending_timeout'
					FROM users WHERE tenant_id = $1
				`, pt.ID)
				deleted++
				slog.Info("Expired pending tenant cleaned up", "tenant_id", pt.ID, "name", pt.Name)
			} else {
				slog.Error("Failed to cleanup pending tenant", "tenant_id", pt.ID, "error", err)
			}
		}

		// Also delete any tenant_subscriptions rows that are stuck in pending
		res, _ := DB.Exec(ctx, `
			DELETE FROM tenant_subscriptions
			WHERE status = 'pending' AND pending_expires_at < NOW()
		`)
		_, _ = DB.Exec(ctx, `DELETE FROM tenant_subscriptions WHERE status = 'pending' AND pending_expires_at IS NULL AND updated_at < NOW() - INTERVAL '48 hours'`)

		response.JSON(w, http.StatusOK, "Cleanup completed", map[string]interface{}{
			"deleted_tenants":      deleted,
			"cleaned_pending_subs": res.RowsAffected(),
		})
	}
}

// ─────────────────────────────────────────────
// Start pending-cleanup background goroutine (F015)
// Runs every 15 minutes while service is up
// ─────────────────────────────────────────────

func startPendingCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				cleanupPendingTenants(context.Background())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	slog.Info("Pending cleanup worker started (every 15 min)")
}
