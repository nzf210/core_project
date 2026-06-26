package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status, _ := payload["status"].(string)
	externalID, _ := payload["external_id"].(string)

	// ── Per-tenant webhook token verification ──
	// Extract tenantID from external_id format:
	//   Invoice: "INV-{uuid}|{tenantID}"
	//   Topup:   "{uuid}-wallet-topup-{tenantID}"
	var tenantID string
	if strings.Contains(externalID, keyWalletTopup) {
		parts := strings.Split(externalID, keyWalletTopup)
		if len(parts) == 2 {
			tenantID = parts[1]
		}
	} else {
		parts := strings.Split(externalID, "|")
		if len(parts) >= 2 {
			tenantID = parts[1]
		}
	}

	if tenantID == "" {
		slog.Warn("Malformed external_id in webhook", "external_id", externalID)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify per-tenant webhook token
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken != "" {
		dbToken, err := getTenantXenditWebhookToken(r.Context(), tenantID)
		if err == nil && dbToken != "" {
			// Per-tenant token takes precedence
			if callbackToken != dbToken {
				slog.Warn("Unauthorized webhook: token mismatch", "tenant_id", tenantID,
					"expected", dbToken, "got", callbackToken)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else if envToken := os.Getenv("XENDIT_WEBHOOK_TOKEN"); envToken != "" {
			// Fallback to global env token (backward compat)
			if callbackToken != envToken {
				slog.Warn("Unauthorized webhook attempt", "token", callbackToken)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	paidAmountFloat, _ := payload["paid_amount"].(float64)
	paidAmountCents := int64(paidAmountFloat)

	// --- HANDLE WALLET TOPUP INVOICE ---
	if strings.Contains(externalID, keyWalletTopup) {
		topupParts := strings.Split(externalID, keyWalletTopup)
		if len(topupParts) == 2 {
			wTenantID := topupParts[1]
			if status == "EXPIRED" {
				w.WriteHeader(http.StatusOK)
				return
			}
			if status != "PAID" && status != "SETTLED" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Get topup amount from xendit payload
			amtFloat, _ := payload["paid_amount"].(float64)
			amountCents := int64(amtFloat)

			// Deduplication check: have we processed this topup externalID?
			var existing int
			DB.QueryRow(r.Context(), "SELECT count(id) FROM wallet_transactions WHERE reference = $1", externalID).Scan(&existing)
			if existing > 0 {
				w.WriteHeader(http.StatusOK) // Already processed
				return
			}

			// Process topup transaction
			tx, err := DB.Begin(r.Context())
			if err != nil {
				http.Error(w, errDB, http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(r.Context())

			// Upsert wallet_credits
			_, err = tx.Exec(r.Context(), `
				INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at) 
				VALUES ($1, $2, NOW()) 
				ON CONFLICT (tenant_id) 
				DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
			`, wTenantID, amountCents)
			if err != nil {
				slog.Error("Topup: Failed to update wallet_credits", "tenant", wTenantID, "err", err)
				return
			}

			// Insert transaction log
			_, err = tx.Exec(r.Context(), `
				INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description) 
				VALUES ($1, $2, 'topup', $3, 'Top-up via Xendit invoice')
			`, wTenantID, amountCents, externalID)
			if err != nil {
				return
			}

			tx.Commit(r.Context())
			slog.Info("Wallet topup successful", "tenant_id", wTenantID, "amount_cents", amountCents)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	ctx := r.Context()

	// IDEMPOTENCY CHECK + ROW LOCK: SELECT FOR UPDATE prevents concurrent webhooks
	// from both passing the PAID check and double-granting subscription.
	var currentStatus string
	var planID, voucherCode string
	var invoiceAmount int64
	err := DB.QueryRow(ctx, `
		SELECT status, COALESCE(plan_id,''), COALESCE(voucher_code,''), amount
		FROM invoices WHERE id = $1
		FOR UPDATE
	`, externalID).Scan(&currentStatus, &planID, &voucherCode, &invoiceAmount)
	if err != nil {
		slog.Warn("Invoice not found", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}
	if currentStatus == "paid" {
		slog.Info("Invoice already paid, ignoring duplicate webhook", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if status == "PAID" || status == "SETTLED" {
		if paidAmountCents < invoiceAmount {
			slog.Warn("PAYMENT AMOUNT MISMATCH", "external_id", externalID, "expected", invoiceAmount, "paid", paidAmountCents)
			// Still process but log as anomaly, or block activation. Let's block for safety.
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	if planID == "" {
		planID = "lite"
	}

	if status == "EXPIRED" {
		// Refund voucher so it can be reused
		if voucherCode != "" {
			_, err := DB.Exec(ctx, `
				UPDATE voucher_codes 
				SET is_redeemed = false, used_by = NULL, used_at = NULL 
				WHERE code = $1
			`, voucherCode)
			if err == nil {
				slog.Info("Voucher refunded due to expired invoice", "code", voucherCode, "tenant_id", tenantID)
			}
		}
		_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'expired' WHERE id = $1", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if status != "PAID" && status != "SETTLED" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get plan name
	var planName string
	DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", planID).Scan(&planName)

	// Update invoice
	_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", externalID)

	// HANDLE OVERPAYMENT: If paid > invoice, put difference in wallet
	if paidAmountCents > invoiceAmount {
		excess := paidAmountCents - invoiceAmount
		slog.Info("Overpayment detected, crediting wallet", "tenant_id", tenantID, "excess_cents", excess)
		_, errW := DB.Exec(ctx, `
			INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at) 
			VALUES ($1, $2, NOW()) 
			ON CONFLICT (tenant_id) 
			DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
		`, tenantID, excess)
		if errW == nil {
			_, _ = DB.Exec(ctx, `
				INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description) 
				VALUES ($1, $2, 'topup', $3, 'Excess payment from invoice')
			`, tenantID, excess, externalID)
		}
	}

	// Apply Voucher (Mark as REDEEMED permanently on actual payment)
	if voucherCode != "" {
		ok, _ := applyVoucher(ctx, voucherCode, planID, tenantID)
		if !ok {
			slog.Warn("Failed to apply voucher on payment webhook", "code", voucherCode, "tenant_id", tenantID)
		}
	}

	// ─────────────────────────────────────────────
	// F054: AFFILIATE COMMISSION LOGIC (lifetime)
	// ─────────────────────────────────────────────
	var referredByID *int
	DB.QueryRow(ctx, querySelectAffiliateID, tenantID).Scan(&referredByID)
	if referredByID != nil {
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

		if isActive && commissionPct > 0 && invoiceAmount >= minPurchaseCents {
			commission := invoiceAmount * int64(commissionPct) / 100
			if maxCommissionCents > 0 && commission > maxCommissionCents {
				commission = maxCommissionCents
			}
			if commission > 0 {
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

					// Mark first_purchase_at in affiliate_referrals on first commission
					_, _ = DB.Exec(ctx, `
						UPDATE affiliate_referrals
						SET first_purchase_at = NOW()
						WHERE affiliate_id = $1 AND tenant_id = $2 AND first_purchase_at IS NULL
					`, *referredByID, tenantID)

					slog.Info("Affiliate commission granted", "affiliate_id", *referredByID, "tenant_id", tenantID, "amount_cents", commission, "rate", commissionPct)
				}
			}
		}
	}

	// Activate subscription
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

	// Auto-generate voucher code for this payment (for tenant's records and future reference)
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

	ticketID := activateSubscription(ctx, tenantID, planID, planName, periodDays, activatedBy, voucherCodeID, autoVoucherCode)

	slog.Info("Payment processed", "tenant_id", tenantID, "plan", planID, "ticket_id", ticketID, "voucher_code", autoVoucherCode)

	w.WriteHeader(http.StatusOK)
}

// ─────────────────────────────────────────────
// Core: Activate Subscription + Generate Ticket
// ─────────────────────────────────────────────
