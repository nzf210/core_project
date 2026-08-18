package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// extractTenantIDFromExternalID parses tenant ID from Xendit external_id.
// Format: "INV-{uuid}|{tenantID}" or "{uuid}-wallet-topup-{tenantID}"
func extractTenantIDFromExternalID(externalID string) (string, error) {
	if strings.Contains(externalID, keyWalletTopup) {
		parts := strings.Split(externalID, keyWalletTopup)
		if len(parts) == 2 {
			return parts[1], nil
		}
	} else {
		parts := strings.Split(externalID, "|")
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("malformed external_id: %s", externalID)
}

// verifyWebhookToken checks per-tenant webhook token with env fallback.
func verifyWebhookToken(ctx context.Context, callbackToken, tenantID string) error {
	if callbackToken == "" {
		return nil // No token provided — skip verification (legacy behavior)
	}

	// Priority 1: Per-tenant token from DB
	dbToken, err := getTenantXenditWebhookToken(ctx, tenantID)
	if err == nil && dbToken != "" {
		if callbackToken != dbToken {
			slog.Warn("Unauthorized webhook: token mismatch", "tenant_id", tenantID)
			return fmt.Errorf("unauthorized")
		}
		return nil
	}

	// Priority 2: Global env token (backward compat)
	// Note: This is kept as os.Getenv for runtime override capability
	// Xendit webhook tokens may need emergency rotation without config rebuild
	envToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")
	if envToken != "" {
		if callbackToken != envToken {
			slog.Warn("Unauthorized webhook attempt", "token", callbackToken)
			return fmt.Errorf("unauthorized")
		}
		return nil
	}

	// Token provided but no token configured anywhere — reject to avoid silent bypass.
	slog.Warn("Unauthorized webhook: token provided but none configured", "tenant_id", tenantID)
	return fmt.Errorf("unauthorized")
}

// handleWalletTopupWebhook processes wallet topup invoice payment.
// Returns true if this was a topup (handled), false otherwise.
func handleWalletTopupWebhook(ctx context.Context, externalID, status string, paidAmountFloat float64) (bool, error) {
	if !strings.Contains(externalID, keyWalletTopup) {
		return false, nil
	}

	topupParts := strings.Split(externalID, keyWalletTopup)
	if len(topupParts) != 2 {
		return true, fmt.Errorf("invalid topup external_id format")
	}

	tenantID := topupParts[1]

	// Ignore non-successful statuses
	if status == "EXPIRED" || (status != "PAID" && status != "SETTLED") {
		return true, nil
	}

	amountCents := int64(paidAmountFloat)

	// Deduplication check
	var existing int
	DB.QueryRow(ctx, "SELECT count(id) FROM wallet_transactions WHERE reference = $1", externalID).Scan(&existing)
	if existing > 0 {
		return true, nil // Already processed
	}

	// Process topup transaction
	tx, err := DB.Begin(ctx)
	if err != nil {
		return true, fmt.Errorf("DB transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert wallet_credits
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
	`, tenantID, amountCents)
	if err != nil {
		slog.Error("Topup: Failed to update wallet_credits", "tenant", tenantID, "err", err)
		return true, err
	}

	// Insert transaction log
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description)
		VALUES ($1, $2, 'topup', $3, 'Top-up via Xendit invoice')
	`, tenantID, amountCents, externalID)
	if err != nil {
		return true, err
	}

	if err := tx.Commit(ctx); err != nil {
		return true, err
	}

	slog.Info("Wallet topup successful", "tenant_id", tenantID, "amount_cents", amountCents)
	return true, nil
}

// processWebhookAffiliateCommission calculates and records affiliate commission for webhook events.
func processWebhookAffiliateCommission(ctx context.Context, tenantID, externalID string, invoiceAmount int64) {
	var referredByID *int
	DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&referredByID)
	if referredByID == nil {
		return
	}

	var commissionPct float64
	var minPurchaseCents, maxCommissionCents int64
	var isActive bool
	errCfg := DB.QueryRow(ctx, `
		SELECT COALESCE(commission_percent,10), COALESCE(min_purchase_cents,0),
		       COALESCE(max_commission_cents,0), COALESCE(is_active,true)
		FROM referral_config WHERE id = 1
	`).Scan(&commissionPct, &minPurchaseCents, &maxCommissionCents, &isActive)
	if errCfg != nil {
		commissionPct = 10
		minPurchaseCents = 0
		maxCommissionCents = 0
		isActive = true
	}

	if !isActive || commissionPct <= 0 || invoiceAmount < minPurchaseCents {
		return
	}

	commission := invoiceAmount * int64(commissionPct) / 100
	if maxCommissionCents > 0 && commission > maxCommissionCents {
		commission = maxCommissionCents
	}
	if commission <= 0 {
		return
	}

	_, errAff := DB.Exec(ctx, `
		UPDATE affiliates
		SET cash_balance_cents = cash_balance_cents + $1,
			total_earnings_cents = total_earnings_cents + $1,
			updated_at = NOW()
		WHERE id = $2
	`, commission, *referredByID)

	if errAff == nil {
		_, _ = DB.Exec(ctx, `
			INSERT INTO affiliate_earnings (affiliate_id, tenant_id, invoice_id, amount_cents, commission_rate_percent, transaction_type, description)
			VALUES ($1, $2, $3, $4, $5, 'subscription_renewal', 'Subscription renewal')
		`, *referredByID, tenantID, externalID, commission, int(commissionPct))

		// Mark first_purchase_at
		_, _ = DB.Exec(ctx, `
			UPDATE affiliate_referrals
			SET first_purchase_at = NOW()
			WHERE affiliate_id = $1 AND tenant_id = $2 AND first_purchase_at IS NULL
		`, *referredByID, tenantID)

		slog.Info("Affiliate commission granted", "affiliate_id", *referredByID, "tenant_id", tenantID, "amount_cents", commission, "rate", commissionPct)
	}
}

// finalizeSubscriptionActivation determines subscription period and activates.
func finalizeSubscriptionActivation(ctx context.Context, tenantID, planID, planName, voucherCode string) string {
	periodDays := 30
	activatedBy := "payment"

	if voucherCode != "" {
		activatedBy = "voucher"
		var duration int
		DB.QueryRow(ctx, `
			SELECT duration_months FROM voucher_programs vp
			JOIN voucher_codes vc ON vc.program_id = vp.id WHERE vc.code = $1
		`, voucherCode).Scan(&duration)
		if duration > 0 {
			periodDays = duration * 30
		}
	}

	// Auto-generate voucher code for this payment
	autoVoucherCode := generateVoucherCode(planID, tenantID)
	var voucherCodeID *string
	if autoVoucherCode != "" {
		var vid string
		DB.QueryRow(ctx, `
			INSERT INTO voucher_codes (program_id, code, is_redeemed, used_by, used_at, created_at)
			SELECT id, $1, true, $2, NOW(), NOW()
			FROM voucher_programs WHERE target_plan_id = $3 AND is_active = true
			LIMIT 1
			RETURNING id
		`, autoVoucherCode, tenantID, planID).Scan(&vid)
		if vid != "" {
			voucherCodeID = &vid
		}
	}

	ticketID := activateSubscription(ctx, tenantID, planID, planName, periodDays, activatedBy, voucherActivationOpts{VoucherCodeID: voucherCodeID, SystemVoucherCode: autoVoucherCode})

	slog.Info("Payment processed", "tenant_id", tenantID, "plan", planID, "ticket_id", ticketID, "voucher_code", autoVoucherCode)

	return ticketID
}
