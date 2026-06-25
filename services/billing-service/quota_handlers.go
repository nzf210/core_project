package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

func handleAdminQuotaUsage(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID := strings.TrimPrefix(r.URL.Path, "/admin/quota/")
	if tenantID == "" || strings.Contains(tenantID, "/") {
		response.Error(w, http.StatusBadRequest, "tenant_id required", nil)
		return
	}

	ctx := r.Context()

	// Get plan features (limits) for tenant
	plan, _ := auth.GetPlanFeatures(ctx, tenantID)

	// Get current period counters (YYYYMM, e.g. "202606")
	period := time.Now().UTC().Format("200601")

	rows, err := DB.Query(ctx,
		`SELECT feature_key, count, reset_at
		 FROM quota_counters
		 WHERE tenant_id = $1 AND period_yyyymm = $2
		 ORDER BY feature_key ASC`,
		tenantID, period)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query counters", err)
		return
	}
	defer rows.Close()

	type counter struct {
		Feature string `json:"feature"`
		Used    int64  `json:"used"`
		ResetAt string `json:"reset_at"`
	}
	counters := []counter{}
	for rows.Next() {
		var c counter
		var resetAt time.Time
		if err := rows.Scan(&c.Feature, &c.Used, &resetAt); err != nil {
			continue
		}
		c.ResetAt = resetAt.Format(time.RFC3339)
		counters = append(counters, c)
	}

	response.JSON(w, http.StatusOK, "Quota usage retrieved", map[string]interface{}{
		"tenant_id": tenantID,
		"tier":      plan.Tier,
		"plan_name": plan.PlanName,
		"period":    period,
		"limits":    plan, // full struct: max_users, max_ai_text, etc.
		"usage":     counters,
	})
}

// ─────────────────────────────────────────────
// F053: Addon Purchase Flow
// GET /addon-marketplace — all available addons with price + has_addon for this tenant
// POST /addons/purchase — buy an addon (wallet deducted)
// GET /addons — list this tenant's active addons
// ─────────────────────────────────────────────

// GET /addon-marketplace
func handleAddonMarketplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	// Get all addon features
	rows, err := DB.Query(ctx,
		`SELECT af.feature_key, af.feature_name, af.description, af.category,
		        af.addon_price_cents, af.addon_unit, af.is_addon,
		        ta.status, ta.expires_at, ta.purchase_price_cents
		 FROM available_features af
		 LEFT JOIN tenant_addons ta ON ta.addon_key = af.feature_key
		        AND ta.tenant_id = $1
		 WHERE af.is_addon = true
		 ORDER BY af.category, af.feature_key`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query marketplace", err)
		return
	}
	defer rows.Close()

	type marketplaceItem struct {
		Key                string  `json:"addon_key"`
		Name               string  `json:"feature_name"`
		Description        string  `json:"description"`
		Category           string  `json:"category"`
		PriceCents         int64   `json:"price_cents"`
		Unit               string  `json:"addon_unit"`
		HasAddon           bool    `json:"has_addon"`
		AddonStatus        *string `json:"addon_status,omitempty"`
		ExpiresAt          *string `json:"expires_at,omitempty"`
		PurchasePriceCents *int64  `json:"purchase_price_cents,omitempty"`
	}

	var items []marketplaceItem
	for rows.Next() {
		var m marketplaceItem
		var expiresAt *time.Time
		var addonStatus *string
		var purchasePrice *int64
		if err := rows.Scan(&m.Key, &m.Name, &m.Description, &m.Category,
			&m.PriceCents, &m.Unit, &m.HasAddon,
			&addonStatus, &expiresAt, &purchasePrice); err != nil {
			continue
		}
		if addonStatus != nil && *addonStatus != "" {
			m.HasAddon = true
			m.AddonStatus = addonStatus
			m.PurchasePriceCents = purchasePrice
			if expiresAt != nil {
				ea := expiresAt.Format(time.RFC3339)
				m.ExpiresAt = &ea
			}
		}
		items = append(items, m)
	}

	response.JSON(w, http.StatusOK, "Addon marketplace", map[string]any{"addons": items})
}

func getAddonPrice(ctx context.Context, addonKey string) (int64, string, error) {
	var price int64
	var unit string
	err := DB.QueryRow(ctx,
		`SELECT addon_price_cents, addon_unit FROM available_features
		 WHERE feature_key = $1 AND is_addon = true`,
		addonKey).Scan(&price, &unit)
	return price, unit, err
}

func checkAddonExists(ctx context.Context, tenantID, addonKey string) error {
	var existingStatus string
	return DB.QueryRow(ctx,
		`SELECT status FROM tenant_addons
		 WHERE tenant_id = $1 AND addon_key = $2
		   AND status = 'active'
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		tenantID, addonKey).Scan(&existingStatus)
}

func calculateAddonFinalPrice(ctx context.Context, tenantID string, basePrice int64) (int64, *int) {
	var refAid *int
	_ = DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&refAid)
	if refAid == nil {
		return basePrice, nil
	}
	var dpct float64
	_ = DB.QueryRow(ctx, `SELECT COALESCE(discount_percent,0) FROM referral_config WHERE id=1`).Scan(&dpct)
	if dpct > 0 {
		disc := basePrice * int64(dpct) / 100
		return maxInt64(0, basePrice-disc), refAid
	}
	return basePrice, refAid
}

func processAffiliateCommission(ctx context.Context, tenantID, addonKey string, finalPrice int64, refAid *int) {
	if refAid == nil || finalPrice <= 0 {
		return
	}
	var cpct float64
	_ = DB.QueryRow(ctx, `SELECT COALESCE(commission_percent,0) FROM referral_config WHERE id=1`).Scan(&cpct)
	if cpct <= 0 {
		return
	}
	comm := finalPrice * int64(cpct) / 100
	if comm > 0 {
		txID := "addon:" + addonKey + ":" + uuid.NewString()[:8]
		_, _ = DB.Exec(ctx,
			`INSERT INTO affiliate_earnings (affiliate_id,tenant_id,invoice_id,amount_cents,commission_rate_percent,transaction_type,description)
			 VALUES ($1,$2,$3,$4,$5,'addon_purchase',$6)`,
			*refAid, tenantID, txID, comm, int(cpct), "Addon: "+addonKey)
		_, _ = DB.Exec(ctx, `UPDATE affiliates SET cash_balance_cents = cash_balance_cents + $1 WHERE id = $2`, comm, *refAid)
	}
}

// POST /addons/purchase
func handlePurchaseAddon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	var req struct {
		AddonKey string `json:"addon_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AddonKey == "" {
		response.Error(w, http.StatusBadRequest, "addon_key required", nil)
		return
	}

	// 1. Verify addon exists and is active
	price, _, err := getAddonPrice(ctx, req.AddonKey)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Addon not found", nil)
		return
	}

	// 2. Check if already has active addon
	if err := checkAddonExists(ctx, tenantID, req.AddonKey); err == nil {
		response.Error(w, http.StatusConflict, "Addon already active", nil)
		return
	}

	// 3. Deduct wallet (after referral discount)
	addonFinalPrice, refAid := calculateAddonFinalPrice(ctx, tenantID, price)

	if addonFinalPrice > 0 {
		if !auth.CheckWalletBalance(ctx, tenantID, addonFinalPrice) {
			response.JSON(w, http.StatusPaymentRequired, "Insufficient wallet balance. Please top up.", map[string]any{
				"wallet_url": walletEndpoint,
			})
			return
		}
		if err := auth.DeductWalletBalance(ctx, tenantID, addonFinalPrice,
			"addon_purchase:"+req.AddonKey,
			"Pembelian addon: "+req.AddonKey); err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to deduct wallet", err)
			return
		}
	}

	// 4. Calculate expiry (1 month from now)
	expiresAt := time.Now().AddDate(0, 1, 0)

	// 5. Upsert tenant_addons
	_, err = DB.Exec(ctx,
		`INSERT INTO tenant_addons (tenant_id, addon_key, status, purchased_at, expires_at, purchase_price_cents)
		 VALUES ($1, $2, 'active', NOW(), $3, $4)
		 ON CONFLICT (tenant_id, addon_key)
		 DO UPDATE SET status='active', purchased_at=NOW(), expires_at=$3, purchase_price_cents=$4`,
		tenantID, req.AddonKey, expiresAt, addonFinalPrice)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to activate addon", err)
		return
	}

	// 7. F054: Affiliate commission for addon purchases — from actual paid amount (addonFinalPrice)
	processAffiliateCommission(ctx, tenantID, req.AddonKey, addonFinalPrice, refAid)

	// 6. Invalidate addon cache
	auth.InvalidateAddonCache(ctx, tenantID, req.AddonKey)

	response.JSON(w, http.StatusOK, "Addon purchased", map[string]any{
		"addon_key":  req.AddonKey,
		"expires_at": expiresAt.Format(time.RFC3339),
		"price":      price,
	})
}

// GET /addons — list tenant's active (and recent) addons
func handleMyAddons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingXTenantID, nil)
		return
	}
	ctx := r.Context()

	rows, err := DB.Query(ctx,
		`SELECT ta.addon_key, af.feature_name, ta.status,
		        ta.purchased_at, ta.expires_at, ta.auto_renew,
		        ta.purchase_price_cents, af.addon_unit
		 FROM tenant_addons ta
		 JOIN available_features af ON af.feature_key = ta.addon_key
		 WHERE ta.tenant_id = $1
		 ORDER BY ta.created_at DESC`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query addons", err)
		return
	}
	defer rows.Close()

	type addonItem struct {
		Key                string  `json:"addon_key"`
		Name               string  `json:"feature_name"`
		Status             string  `json:"status"`
		PurchasedAt        string  `json:"purchased_at"`
		ExpiresAt          *string `json:"expires_at,omitempty"`
		AutoRenew          bool    `json:"auto_renew"`
		PurchasePriceCents int64   `json:"purchase_price_cents"`
		Unit               string  `json:"addon_unit"`
	}
	var items []addonItem
	for rows.Next() {
		var a addonItem
		var expiresAt *time.Time
		if err := rows.Scan(&a.Key, &a.Name, &a.Status,
			&a.PurchasedAt, &expiresAt, &a.AutoRenew,
			&a.PurchasePriceCents, &a.Unit); err != nil {
			continue
		}
		if expiresAt != nil {
			ea := expiresAt.Format(time.RFC3339)
			a.ExpiresAt = &ea
		}
		items = append(items, a)
	}

	response.JSON(w, http.StatusOK, "My addons", map[string]any{"addons": items})
}
