package main

import (
	"context"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

// applyVoucherDiscount applies voucher discount to base price.
func applyVoucherDiscount(ctx context.Context, priceMonthly int64, voucherCode, planID string) (int64, bool) {
	if voucherCode == "" {
		return priceMonthly, false
	}

	voucherApplied := validateVoucherOnly(ctx, voucherCode, planID)
	if !voucherApplied {
		return priceMonthly, false
	}

	var discountType string
	var discountValue int
	DB.QueryRow(ctx, `
		SELECT voucher_type, discount_value FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id WHERE vc.code = $1
	`, voucherCode).Scan(&discountType, &discountValue)

	finalPrice := priceMonthly
	switch discountType {
	case "discount_percent":
		finalPrice = priceMonthly * int64(100-discountValue) / 100
	case "discount_fixed":
		finalPrice = maxInt64(0, priceMonthly-int64(discountValue))
	}

	return finalPrice, true
}

// applyReferralDiscount applies referral discount on top of voucher price.
func applyReferralDiscount(ctx context.Context, basePrice int64, tenantID string) (finalPrice int64, discountAmount int64, affiliateID *int) {
	finalPrice = basePrice
	DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&affiliateID)
	if affiliateID == nil {
		return finalPrice, 0, nil
	}

	var discountPct float64
	_ = DB.QueryRow(ctx, `SELECT COALESCE(discount_percent,0) FROM referral_config WHERE id=1`).Scan(&discountPct)
	if discountPct > 0 {
		discountAmount = finalPrice * int64(discountPct) / 100
		finalPrice = maxInt64(0, finalPrice-discountAmount)
	}

	return finalPrice, discountAmount, affiliateID
}

// SubscriptionRequest groups subscription request parameters.
type SubscriptionRequest struct {
	TenantID    string
	PlanID      string
	PlanName    string
	VoucherCode string
	FinalPrice  int64
}

// ReferralInfo groups referral-related parameters.
type ReferralInfo struct {
	AffiliateID    *int
	DiscountAmount int64
}

// processWalletSubscription handles wallet payment and immediate activation.
func processWalletSubscription(w http.ResponseWriter, ctx context.Context, req SubscriptionRequest, referral ReferralInfo) bool {
	tenantID := req.TenantID
	planID := req.PlanID
	planName := req.PlanName
	voucherCode := req.VoucherCode
	finalPrice := req.FinalPrice
	if !auth.CheckWalletBalance(ctx, tenantID, finalPrice) {
		var balance int64
		_ = DB.QueryRow(ctx, "SELECT COALESCE(balance_cents,0) FROM wallet_credits WHERE tenant_id=$1", tenantID).Scan(&balance)
		response.JSON(w, http.StatusPaymentRequired, "Saldo wallet tidak cukup", map[string]interface{}{
			"required_cents": finalPrice,
			"balance_cents":  balance,
			"topup_url":      walletEndpoint,
		})
		return true
	}

	ref := fmt.Sprintf("subscription:%s:%d", planID, time.Now().Unix())
	desc := fmt.Sprintf("Pembayaran langganan %s via Wallet", planName)
	if err := auth.DeductWalletBalance(ctx, tenantID, finalPrice, ref, desc); err != nil {
		slog.Error("Wallet deduct failed for subscription", "tenant_id", tenantID, "error", err)
		response.Error(w, http.StatusInternalServerError, "Gagal memproses pembayaran wallet", nil)
		return true
	}

	validityDays := 30
	if voucherCode != "" {
		_ = DB.QueryRow(ctx, "SELECT validity_days FROM voucher_codes WHERE code=$1", voucherCode).Scan(&validityDays)
	}
	activateSubscription(ctx, tenantID, planID, planName, validityDays, "wallet", nil, "")

	if referral.AffiliateID != nil && referral.DiscountAmount >= 0 {
		var commRate float64
		_ = DB.QueryRow(ctx, `SELECT COALESCE(commission_rate_percent,0) FROM referral_config WHERE id=1`).Scan(&commRate)
		if commRate > 0 {
			comm := float64(finalPrice) * commRate / 100
			_, _ = DB.Exec(ctx, `
				INSERT INTO affiliate_earnings (affiliate_id,tenant_id,invoice_id,amount_cents,commission_rate_percent,transaction_type,description)
				VALUES ($1,$2,$3,$4,$5,'subscription',$6)
			`, *referral.AffiliateID, tenantID, ref, int64(comm), commRate, desc)
		}
	}

	slog.Info("Subscription paid via wallet", "tenant_id", tenantID, "plan", planID, "amount", finalPrice)
	response.JSON(w, http.StatusOK, "Subscription activated via wallet", map[string]interface{}{
		"status":         "activated",
		"payment_method": "wallet",
		"plan_id":        planID,
		"amount_charged": finalPrice,
	})
	return true
}

// processFreeSubscription handles free subscription activation.
func processFreeSubscription(w http.ResponseWriter, ctx context.Context, req SubscriptionRequest) bool {
	slog.Info("FREE TRANSACTION DETECTED: Bypassing Xendit", "tenant_id", req.TenantID)

	var codeValidityDays int
	if req.VoucherCode != "" {
		DB.QueryRow(ctx, "SELECT validity_days FROM voucher_codes WHERE code = $1", req.VoucherCode).Scan(&codeValidityDays)
		applyVoucher(ctx, req.VoucherCode, req.PlanID, req.TenantID)
	}
	if codeValidityDays == 0 {
		codeValidityDays = 30
	}

	activateSubscription(ctx, req.TenantID, req.PlanID, req.PlanName, codeValidityDays, "voucher_direct", nil, "")

	extID := fmt.Sprintf("FREE-%s|%s", uuid.NewString()[:8], req.TenantID)
	_, _ = DB.Exec(ctx, "INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code, paid_at) VALUES ($1, $2, $3, 0, 'paid', 'free_bypass', $4, NOW())", extID, req.TenantID, req.PlanID, req.VoucherCode)

	response.JSON(w, http.StatusOK, "Success: Free subscription activated", map[string]string{"status": "activated", "payment_url": ""})
	return true
}

// validatePlan validates plan existence and returns plan details.
func validatePlan(ctx context.Context, planID string) (planName string, priceMonthly int64, err error) {
	err = DB.QueryRow(ctx, `
		SELECT name, price_monthly FROM saas_plans WHERE id = $1 AND is_active = true
	`, planID).Scan(&planName, &priceMonthly)
	return planName, priceMonthly, err
}

// InvoiceRequest groups invoice creation parameters.
type InvoiceRequest struct {
	ExternalID  string
	TenantID    string
	PlanID      string
	PlanName    string
	VoucherCode string
	FinalPrice  int64
}

// createXenditInvoice creates a Xendit invoice and persists it to DB.
func createXenditInvoice(ctx context.Context, _ *http.Request, req InvoiceRequest) (string, error) {
	createReq := invoice.NewCreateInvoiceRequest(req.ExternalID, float64(req.FinalPrice)/100)
	desc := fmt.Sprintf("Langganan %s — WCH Platform", req.PlanName)
	createReq.Description = &desc
	createReq.PaymentMethods = []string{"BANK_TRANSFER", "EWALLET", "QRIS"}

	xClient, errXc := getTenantXenditClient(ctx, req.TenantID)
	if errXc != nil {
		slog.Error("Failed to get xendit client for tenant", "tenant_id", req.TenantID, "error", errXc)
		return "", errXc
	}

	resp, _, invoiceErr := xClient.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*createReq).Execute()
	var paymentURL string

	if invoiceErr != nil {
		slog.Error("Failed to create xendit invoice", "error", invoiceErr)
		if Cfg.Env == "development" {
			paymentURL = handleDevMockInvoice(ctx, req.ExternalID, req.TenantID, req.PlanID, req.FinalPrice, req.VoucherCode)
		} else {
			return "", invoiceErr
		}
	} else {
		paymentURL = resp.InvoiceUrl
	}

	_, err := DB.Exec(ctx, `
		INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, payment_url = EXCLUDED.payment_url
	`, req.ExternalID, req.TenantID, req.PlanID, req.FinalPrice, paymentURL, req.VoucherCode)
	if err != nil {
		slog.Warn("Failed to save invoice", "error", err)
	}

	return paymentURL, nil
}

// handleDevMockInvoice creates a mock invoice for development mode.
func handleDevMockInvoice(ctx context.Context, externalID, tenantID, planID string, finalPrice int64, voucherCode string) string {
	slog.Warn("DEV mode: Mocking Xendit invoice creation")
	mockInvoiceUrl := fmt.Sprintf("https://checkout.xendit.co/web/%s", externalID)
	_, err := DB.Exec(ctx, `
		INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url, voucher_code)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, payment_url = EXCLUDED.payment_url
	`, externalID, tenantID, planID, finalPrice, mockInvoiceUrl, voucherCode)
	if err != nil {
		slog.Warn("Failed to save mock invoice", "error", err)
	}
	return mockInvoiceUrl
}

// recordReferralDiscount records referral discount applied to invoice.
func recordReferralDiscount(ctx context.Context, externalID string, referredByAffiliateID *int, referralDiscountAmount int64) {
	if referredByAffiliateID != nil && referralDiscountAmount > 0 {
		_, _ = DB.Exec(ctx, `
			INSERT INTO invoice_referrals (invoice_id, affiliate_id, discount_amount)
			VALUES ($1, $2, $3)
			ON CONFLICT (invoice_id) DO NOTHING
		`, externalID, *referredByAffiliateID, referralDiscountAmount)
	}
}

// createPendingSubscription creates a pending subscription record.
func createPendingSubscription(ctx context.Context, tenantID, planID string) {
	pendingHours := int64(24)
	if v := os.Getenv("SUBSCRIPTION_PENDING_TIMEOUT_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			pendingHours = int64(h)
		}
	}
	_, _ = DB.Exec(ctx, `
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, plan_tier, status, period_days, remaining_days, activated_by, pending_expires_at, updated_at)
		VALUES ($1, $2, $2, 'pending', 0, 0, 'pending', NOW() + ($3 || ' hours')::interval, NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			plan_tier = EXCLUDED.plan_tier,
			status = 'pending',
			period_days = 0,
			remaining_days = 0,
			pending_expires_at = EXCLUDED.pending_expires_at,
			updated_at = NOW()
	`, tenantID, planID, pendingHours)
}

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req SubscribeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	ctx := r.Context()
	planName, priceMonthly, err := validatePlan(ctx, req.PlanID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid plan", nil)
		return
	}

	priceAfterVoucher, voucherApplied := applyVoucherDiscount(ctx, priceMonthly, req.VoucherCode, req.PlanID)
	if voucherApplied {
		slog.Info("Voucher applied", "tenant_id", tenantID, "code", req.VoucherCode)
	}

	finalPrice, referralDiscountAmount, referredByAffiliateID := applyReferralDiscount(ctx, priceAfterVoucher, tenantID)

	if req.PayViaWallet && finalPrice > 0 {
		subReq := SubscriptionRequest{
			TenantID:    tenantID,
			PlanID:      req.PlanID,
			PlanName:    planName,
			VoucherCode: req.VoucherCode,
			FinalPrice:  finalPrice,
		}
		referral := ReferralInfo{
			AffiliateID:    referredByAffiliateID,
			DiscountAmount: referralDiscountAmount,
		}
		if processWalletSubscription(w, ctx, subReq, referral) {
			return
		}
	}

	if finalPrice == 0 {
		subReq := SubscriptionRequest{
			TenantID:    tenantID,
			PlanID:      req.PlanID,
			PlanName:    planName,
			VoucherCode: req.VoucherCode,
			FinalPrice:  0,
		}
		if processFreeSubscription(w, ctx, subReq) {
			return
		}
	}

	invoiceID := uuid.NewString()
	externalID := fmt.Sprintf("INV-%s|%s", invoiceID, tenantID)

	invoiceReq := InvoiceRequest{
		ExternalID:  externalID,
		TenantID:    tenantID,
		PlanID:      req.PlanID,
		PlanName:    planName,
		VoucherCode: req.VoucherCode,
		FinalPrice:  finalPrice,
	}
	paymentURL, err := createXenditInvoice(ctx, r, invoiceReq)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create invoice", nil)
		return
	}

	recordReferralDiscount(ctx, externalID, referredByAffiliateID, referralDiscountAmount)
	createPendingSubscription(ctx, tenantID, req.PlanID)

	slog.Info("Subscription pending created", "tenant_id", tenantID, "plan", req.PlanID, "invoice", externalID)

	response.JSON(w, http.StatusOK, "Invoice created", map[string]interface{}{
		"invoice_id":      externalID,
		"payment_url":     paymentURL,
		"plan_id":         req.PlanID,
		"plan_name":       planName,
		"original_price":  priceMonthly,
		"final_price":     finalPrice,
		"voucher_applied": voucherApplied,
		"status":          "pending",
	})
}
