package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

const (
	minWithdrawCents = 10000000 // Rp 100.000
)

func handleAffiliateLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	rows, err := DB.Query(context.Background(), `
		SELECT 
			u.full_name, 
			COUNT(DISTINCT ae.tenant_id) as total_closing, 
			SUM(ae.amount_cents) as total_revenue
		FROM affiliate_earnings ae
		JOIN affiliates a ON ae.affiliate_id = a.id
		JOIN users u ON a.user_id = u.id
		GROUP BY a.id, u.full_name
		ORDER BY total_revenue DESC
		LIMIT 10
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query leaderboard", err)
		return
	}
	defer rows.Close()

	type leader struct {
		Name         string `json:"name"`
		TotalClosing int    `json:"total_closing"`
		TotalRevenue int64  `json:"total_revenue_cents"`
	}
	var leaders []leader
	for rows.Next() {
		var l leader
		var rawName string
		if err := rows.Scan(&rawName, &l.TotalClosing, &l.TotalRevenue); err != nil {
			slog.Warn("Failed to scan leaderboard row", "error", err)
			continue
		}
		// Masking name (e.g. "Budi Santoso" -> "Budi S.")
		parts := strings.Split(rawName, " ")
		if len(parts) > 1 && len(parts[1]) > 0 {
			l.Name = parts[0] + " " + string(parts[1][0]) + "."
		} else {
			l.Name = parts[0]
		}
		leaders = append(leaders, l)
	}

	response.JSON(w, http.StatusOK, "Leaderboard retrieved", leaders)
}

func handleAffiliateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get(response.XUserID)

	var affID int
	var refCode string
	var balance, earnings int64
	err := DB.QueryRow(context.Background(), "SELECT id, referral_code, cash_balance_cents, total_earnings_cents FROM affiliates WHERE user_id = $1", userID).Scan(&affID, &refCode, &balance, &earnings)
	if err != nil {
		response.JSON(w, http.StatusOK, errNotAffiliate, map[string]interface{}{"is_affiliate": false})
		return
	}

	// Fetch recent earnings (last 50)
	rows, err2 := DB.Query(context.Background(),
		`SELECT ae.id, ae.tenant_id, ae.invoice_id, ae.amount_cents, ae.commission_rate_percent, ae.transaction_type, ae.created_at,
		        t.name as tenant_name
		 FROM affiliate_earnings ae
		 LEFT JOIN tenants t ON t.id = ae.tenant_id
		 WHERE ae.affiliate_id = $1
		 ORDER BY ae.created_at DESC
		 LIMIT 50`, affID)
	if err2 != nil {
		response.JSON(w, http.StatusOK, "Profile retrieved", map[string]interface{}{
			"is_affiliate":         true,
			"affiliate_id":         affID,
			"referral_code":        refCode,
			"cash_balance_cents":   balance,
			"total_earnings_cents": earnings,
			"earnings":             []interface{}{},
		})
		return
	}
	defer rows.Close()

	type earning struct {
		ID              string    `json:"id"`
		TenantID        string    `json:"tenant_id"`
		TenantName      string    `json:"tenant_name"`
		InvoiceID       string    `json:"invoice_id"`
		AmountCents     int64     `json:"amount_cents"`
		CommissionRate  int       `json:"commission_rate_percent"`
		TransactionType string    `json:"transaction_type"`
		CreatedAt       time.Time `json:"created_at"`
	}
	var earningsList []earning
	for rows.Next() {
		var e earning
		var eid int
		if err2 := rows.Scan(&eid, &e.TenantID, &e.InvoiceID, &e.AmountCents, &e.CommissionRate, &e.TransactionType, &e.CreatedAt, &e.TenantName); err2 != nil {
			slog.Warn("Failed to scan earning row", "error", err2)
			continue
		}
		e.ID = strconv.Itoa(eid)
		earningsList = append(earningsList, e)
	}

	response.JSON(w, http.StatusOK, "Affiliate profile retrieved", map[string]interface{}{
		"is_affiliate":         true,
		"affiliate_id":         affID,
		"referral_code":        refCode,
		"referral_link":        "https://wch.id/r/" + refCode,
		"cash_balance_cents":   balance,
		"total_earnings_cents": earnings,
		"earnings":             earningsList,
	})
}

func handleAffiliateRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get(response.XUserID)

	var existing int
	if err := DB.QueryRow(context.Background(), queryAffiliateUserID, userID).Scan(&existing); err == nil && existing > 0 {
		response.Error(w, http.StatusBadRequest, "Already registered", nil)
		return
	}

	refCode := "AGEN-" + strings.ToUpper(uuid.NewString()[:6])
	_, err := DB.Exec(context.Background(), "INSERT INTO affiliates (user_id, referral_code) VALUES ($1, $2)", userID, refCode)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Registration failed", err)
		return
	}

	response.JSON(w, http.StatusOK, "Registered as affiliate", map[string]string{"referral_code": refCode})
}

func handleAffiliateWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get(response.XUserID)

	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountCents < minWithdrawCents {
		response.Error(w, http.StatusBadRequest, "Minimum withdraw Rp 100.000", nil)
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, errDB, err)
		return
	}
	defer tx.Rollback(ctx)

	var affID int
	err = tx.QueryRow(ctx, "UPDATE affiliates SET cash_balance_cents = cash_balance_cents - $1 WHERE user_id = $2 AND cash_balance_cents >= $1 RETURNING id", req.AmountCents, userID).Scan(&affID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Insufficient balance", err)
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO affiliate_withdrawals (affiliate_id, amount_cents, status) VALUES ($1, $2, 'pending')", affID, req.AmountCents)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to record withdrawal", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}
	response.JSON(w, http.StatusOK, "Withdrawal requested", nil)
}

func handleAffiliateRedeemReferral(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req struct {
		ReferralCode string `json:"referral_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ReferralCode == "" {
		response.Error(w, http.StatusBadRequest, "Referral code required", nil)
		return
	}

	var affID int
	err := DB.QueryRow(r.Context(), "SELECT id FROM affiliates WHERE referral_code = $1", req.ReferralCode).Scan(&affID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid referral code", nil)
		return
	}

	tx, err := DB.Begin(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, errDB, err)
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), "UPDATE tenants SET referred_by_affiliate_id = $1 WHERE id = $2 AND referred_by_affiliate_id IS NULL", affID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to apply referral", err)
		return
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO affiliate_referrals (affiliate_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		affID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to record referral", err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}
	response.JSON(w, http.StatusOK, "Referral applied successfully", nil)
}

func handleAffiliateReferrals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get(response.XUserID)
	ctx := r.Context()

	var affID int
	err := DB.QueryRow(ctx, queryAffiliateUserID, userID).Scan(&affID)
	if err != nil {
		response.JSON(w, http.StatusOK, errNotAffiliate, map[string]interface{}{"referrals": []interface{}{}})
		return
	}

	rows, err2 := DB.Query(ctx,
		`SELECT ar.id, ar.tenant_id, ar.referred_at, ar.first_purchase_at, t.name as tenant_name
		 FROM affiliate_referrals ar
		 LEFT JOIN tenants t ON t.id = ar.tenant_id
		 WHERE ar.affiliate_id = $1
		 ORDER BY ar.referred_at DESC`, affID)
	if err2 != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query referrals", err2)
		return
	}
	defer rows.Close()

	type referral struct {
		ID            string     `json:"id"`
		TenantID      string     `json:"tenant_id"`
		TenantName    string     `json:"tenant_name"`
		ReferredAt    time.Time  `json:"referred_at"`
		FirstPurchase *time.Time `json:"first_purchase_at"`
	}
	var referrals []referral
	for rows.Next() {
		var r referral
		var tid string
		if err := rows.Scan(&r.ID, &tid, &r.ReferredAt, &r.FirstPurchase, &r.TenantName); err != nil {
			slog.Warn("Failed to scan referral row", "error", err)
			continue
		}
		r.TenantID = tid
		referrals = append(referrals, r)
	}
	response.JSON(w, http.StatusOK, "Referrals retrieved", referrals)
}

func handleAffiliateEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	userID := r.Header.Get(response.XUserID)
	ctx := r.Context()

	var affID int
	err := DB.QueryRow(ctx, queryAffiliateUserID, userID).Scan(&affID)
	if err != nil {
		response.JSON(w, http.StatusOK, errNotAffiliate, map[string]interface{}{"earnings": []interface{}{}})
		return
	}

	rows, err2 := DB.Query(ctx,
		`SELECT ae.id, ae.tenant_id, ae.invoice_id, ae.amount_cents, ae.commission_rate_percent,
		        ae.transaction_type, ae.description, ae.created_at, t.name as tenant_name
		 FROM affiliate_earnings ae
		 LEFT JOIN tenants t ON t.id = ae.tenant_id
		 WHERE ae.affiliate_id = $1
		 ORDER BY ae.created_at DESC`, affID)
	if err2 != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query earnings", err2)
		return
	}
	defer rows.Close()

	type earning struct {
		ID              string    `json:"id"`
		TenantID        string    `json:"tenant_id"`
		TenantName      string    `json:"tenant_name"`
		InvoiceID       string    `json:"invoice_id"`
		AmountCents     int64     `json:"amount_cents"`
		CommissionRate  int       `json:"commission_rate_percent"`
		TransactionType string    `json:"transaction_type"`
		Description     string    `json:"description"`
		CreatedAt       time.Time `json:"created_at"`
	}
	var earningsList []earning
	for rows.Next() {
		var e earning
		var eid int
		if err := rows.Scan(&eid, &e.TenantID, &e.InvoiceID, &e.AmountCents, &e.CommissionRate, &e.TransactionType, &e.Description, &e.CreatedAt, &e.TenantName); err != nil {
			slog.Warn("Failed to scan earnings row", "error", err)
			continue
		}
		e.ID = strconv.Itoa(eid)
		earningsList = append(earningsList, e)
	}
	response.JSON(w, http.StatusOK, "Earnings retrieved", earningsList)
}

func handleAdminReferralConfig(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		var discountPct, commissionPct, minPurchase, maxCommission float64
		var isActive bool
		var linkBase string
		err := DB.QueryRow(ctx, `
			SELECT COALESCE(discount_percent,10), COALESCE(commission_percent,10),
			       COALESCE(min_purchase_rupiah,0), COALESCE(max_commission_rupiah,0),
			       COALESCE(is_active,true), COALESCE(referral_link_base,'wch.id/r')
			FROM referral_config WHERE id = 1
		`).Scan(&discountPct, &commissionPct, &minPurchase, &maxCommission, &isActive, &linkBase)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load config", err)
			return
		}
		response.JSON(w, http.StatusOK, "Referral config loaded", map[string]interface{}{
			"discount_percent":       discountPct,
			"commission_percent":     commissionPct,
			"min_purchase_rupiah":    minPurchase,
			"max_commission_rupiah":  maxCommission,
			"is_active":              isActive,
			"referral_link_base":     linkBase,
		})

	case http.MethodPut, http.MethodPost:
		var req struct {
			DiscountPercent      float64 `json:"discount_percent"`
			CommissionPercent    float64 `json:"commission_percent"`
			MinPurchaseRupiah    int64   `json:"min_purchase_rupiah"`
			MaxCommissionRupiah  int64   `json:"max_commission_rupiah"`
			IsActive             bool    `json:"is_active"`
			ReferralLinkBase     string  `json:"referral_link_base"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "Invalid body", err)
			return
		}
		if req.DiscountPercent < 0 || req.DiscountPercent > 100 || req.CommissionPercent < 0 || req.CommissionPercent > 100 {
			response.Error(w, http.StatusBadRequest, "Percentage must be 0-100", nil)
			return
		}

		_, err := DB.Exec(ctx, `
			INSERT INTO referral_config (id, discount_percent, commission_percent, min_purchase_rupiah, max_commission_rupiah, is_active, referral_link_base, updated_at)
			VALUES (1, $1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id)
			DO UPDATE SET discount_percent = EXCLUDED.discount_percent,
			              commission_percent = EXCLUDED.commission_percent,
			              min_purchase_rupiah = EXCLUDED.min_purchase_rupiah,
			              max_commission_rupiah = EXCLUDED.max_commission_rupiah,
			              is_active = EXCLUDED.is_active,
			              referral_link_base = EXCLUDED.referral_link_base,
			              updated_at = NOW()
		`, req.DiscountPercent, req.CommissionPercent, req.MinPurchaseRupiah, req.MaxCommissionRupiah, req.IsActive, req.ReferralLinkBase)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update config", err)
			return
		}
		slog.Info("Referral config updated", "discount", req.DiscountPercent, "commission", req.CommissionPercent)
		response.JSON(w, http.StatusOK, "Referral config updated", map[string]interface{}{
			"discount_percent":       req.DiscountPercent,
			"commission_percent":     req.CommissionPercent,
			"min_purchase_rupiah":    req.MinPurchaseRupiah,
			"max_commission_rupiah":  req.MaxCommissionRupiah,
			"is_active":              req.IsActive,
			"referral_link_base":   req.ReferralLinkBase,
		})

	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}
