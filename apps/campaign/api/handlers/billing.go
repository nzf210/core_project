package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"core_project/apps/campaign/api/repository"
)

// HandleBillingCheckout generates a mock Xendit invoice URL
func HandleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	type Request struct {
		CampaignID string `json:"campaign_id"`
		OrderType  string `json:"order_type"` // 'wargame_token', 'intelligence_pack'
		Quantity   int    `json:"quantity"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
		return
	}

	var priceCents int64
	if req.OrderType == "wargame_token" {
		priceCents = 100_000 * int64(req.Quantity) // Rp 100k
	} else if req.OrderType == "intelligence_pack" {
		priceCents = 5_000_000 * int64(req.Quantity) // Rp 5jt
	} else {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid order type"})
		return
	}

	// Mock Xendit integration
	mockInvoiceID := "inv_" + req.CampaignID[:8]
	mockInvoiceURL := "https://checkout.xendit.co/web/" + mockInvoiceID

	ctx := context.Background()
	query := `
		INSERT INTO campaign_billing_orders (tenant_id, campaign_id, order_type, amount_cents, quantity, invoice_url, xendit_invoice_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := repository.DB.Exec(ctx, query, tenantID, req.CampaignID, req.OrderType, priceCents, req.Quantity, mockInvoiceURL, mockInvoiceID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create order"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"invoice_url": mockInvoiceURL,
			"invoice_id":  mockInvoiceID,
		},
	})
}

// HandleBillingWebhook processes Xendit callback
func HandleBillingWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	// Normally we validate X-CALLBACK-TOKEN here
	
	type XenditPayload struct {
		ID     string `json:"id"`
		Status string `json:"status"` // "PAID", "EXPIRED"
	}

	var payload XenditPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
		return
	}

	ctx := context.Background()
	
	// Check order
	var orderID, tenantID, campaignID, orderType string
	var quantity int
	err := repository.DB.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, campaign_id::text, order_type, quantity
		FROM campaign_billing_orders
		WHERE xendit_invoice_id = $1 AND status = 'PENDING'
	`, payload.ID).Scan(&orderID, &tenantID, &campaignID, &orderType, &quantity)

	if err != nil {
		// Already processed or not found, just return 200 to acknowledge
		w.WriteHeader(http.StatusOK)
		return
	}

	if payload.Status == "PAID" {
		tx, _ := repository.DB.Begin(ctx)
		defer tx.Rollback(ctx)

		tx.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'PAID', paid_at = NOW() WHERE id = $1", orderID)

		if orderType == "wargame_token" {
			tokensToAdd := 10 * quantity
			tx.Exec(ctx, "UPDATE campaigns SET wargame_tokens = wargame_tokens + $1 WHERE id = $2", tokensToAdd, campaignID)
		} else if orderType == "intelligence_pack" {
			tx.Exec(ctx, `
				UPDATE campaigns 
				SET active_addons = (
					SELECT jsonb_agg(DISTINCT e) FROM (
						SELECT jsonb_array_elements_text(COALESCE(active_addons, '[]'::jsonb)) as e
						UNION SELECT 'fraud_map' UNION SELECT 'anomaly_detect'
					) sub
				)
				WHERE id = $1
			`, campaignID)
		}

		// F037: Campaign Affiliate Commission — share to referrer
		var referredByID *int
		repository.DB.QueryRow(ctx, "SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID).Scan(&referredByID)
		if referredByID != nil {
			// Read dynamic config from referral_config (shared with billing-service)
			var commissionPct float64
			err := repository.DB.QueryRow(ctx, `
				SELECT COALESCE(commission_percent, 10) FROM referral_config WHERE id = 1
			`).Scan(&commissionPct)
			if err != nil {
				commissionPct = 10
			}

			var fullAmount int64
			repository.DB.QueryRow(ctx, "SELECT amount_cents FROM campaign_billing_orders WHERE id = $1", orderID).Scan(&fullAmount)
			commission := fullAmount * int64(commissionPct) / 100
			if commission > 0 {
				tx.Exec(ctx, `
					UPDATE affiliates 
					SET cash_balance_cents = cash_balance_cents + $1,
						total_earnings_cents = total_earnings_cents + $1,
						updated_at = NOW()
					WHERE id = $2
				`, commission, *referredByID)
				tx.Exec(ctx, `
					INSERT INTO affiliate_earnings (affiliate_id, tenant_id, invoice_id, amount_cents, commission_rate_percent)
					VALUES ($1, $2, $3, $4, $5)
				`, *referredByID, tenantID, payload.ID, commission, int(commissionPct))
				slog.Info("Campaign: Affiliate commission granted", "affiliate_id", *referredByID, "tenant_id", tenantID, "amount_cents", commission, "rate", commissionPct)
			}
		}

		tx.Commit(ctx)
	} else if payload.Status == "EXPIRED" {
		repository.DB.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'EXPIRED' WHERE id = $1", orderID)
	}

	w.WriteHeader(http.StatusOK)
}
