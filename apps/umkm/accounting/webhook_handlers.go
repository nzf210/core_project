package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"core_project/shared/sdk/webhook"
)

const (
	errMethodNotAllowed    = "Method not allowed"
	errTransactionNotFound = "Transaction not found"
)


func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowed})
		return
	}

	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	ctx := r.Context()

	var tenantID, currentStatus, paymentMethod, itemsJSONStr string
	var totalAmount float64
	err := DB.QueryRow(ctx, "SELECT tenant_id, status, payment_method, total_amount, items_json::text FROM pos_transactions WHERE reference = $1 FOR UPDATE", req.Reference).Scan(&tenantID, &currentStatus, &paymentMethod, &totalAmount, &itemsJSONStr)

	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: errTransactionNotFound})
		return
	}

	if currentStatus == "paid" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Already paid"})
		return
	}

	// Update to paid
	_, err = DB.Exec(ctx, "UPDATE pos_transactions SET status = 'paid', updated_at = NOW() WHERE reference = $1", req.Reference)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update status"})
		return
	}

	// Create journal entries
	var debitAccID, creditAccID string
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '101'", tenantID).Scan(&debitAccID)
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '400'", tenantID).Scan(&creditAccID)

	dateStr := time.Now().Format("2006-01-02")
	var entryID string
	err = DB.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, dateStr, "Pembayaran Webhook: "+req.Reference, req.Reference, itemsJSONStr).Scan(&entryID)

	if err == nil {
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)", entryID, debitAccID, totalAmount)
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)", entryID, creditAccID, totalAmount)
	}

	// Deduct Stock
	var parsedItems struct {
		Items []struct {
			ID       string `json:"id"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	json.Unmarshal([]byte(itemsJSONStr), &parsedItems)

	for _, item := range parsedItems.Items {
		DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
	}

	// Send WA Notification
	var waNumber *string
	DB.QueryRow(ctx, "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&waNumber)
	if waNumber != nil && *waNumber != "" {
		go func(tenantID, phone, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA* ✅\n\nRef: %s\nNominal: Rp %.0f\nMetode: QRIS\n\nTerima kasih, dana telah masuk ke rekening Anda dan sistem telah mencatat transaksi ini.", ref, amount)

			// Format phone to JID
			target := phone
			if strings.HasPrefix(target, "0") {
				target = "62" + target[1:]
			}
			if !strings.Contains(target, "@") {
				target = target + "@s.whatsapp.net"
			}

			// Internal WA Gateway call
			data := url.Values{}
			data.Set("tenant_id", tenantID)
			data.Set("target", target)
			data.Set("message", msg)

			req, _ := http.NewRequest("POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Message-Type", "subscription")
			req.Header.Set("X-Source", "umkm-accounting")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(tenantID, *waNumber, req.Reference, totalAmount)
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Payment processed successfully"})
}

func handleStorePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowed})
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	status, _ := payload["status"].(string)
	reference, _ := payload["external_id"].(string)

	if status != "PAID" && status != "SETTLED" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Ignored"})
		return
	}

	ctx := r.Context()

	var tenantID, currentStatus, paymentMethod, itemsJSONStr string
	var totalAmount float64
	err := DB.QueryRow(ctx, "SELECT tenant_id, status, payment_method, total_amount, items_json::text FROM pos_transactions WHERE reference = $1 FOR UPDATE", reference).Scan(&tenantID, &currentStatus, &paymentMethod, &totalAmount, &itemsJSONStr)

	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: errTransactionNotFound})
		return
	}

	// Verify Xendit Webhook Token
	callbackToken := r.Header.Get("x-callback-token")
	var xenditWebhookToken *string
	DB.QueryRow(ctx, "SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID).Scan(&xenditWebhookToken)
	if xenditWebhookToken != nil && *xenditWebhookToken != "" && callbackToken != *xenditWebhookToken {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Invalid webhook token"})
		return
	}

	if currentStatus == "paid" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Already paid"})
		return
	}

	// Update to paid
	_, err = DB.Exec(ctx, "UPDATE pos_transactions SET status = 'paid', updated_at = NOW() WHERE reference = $1", reference)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update status"})
		return
	}

	webhook.DispatchEvent("pos_checkout_completed", tenantID, map[string]interface{}{
		"reference":      reference,
		"total_amount":   totalAmount,
		"payment_method": paymentMethod,
	})

	createPaymentJournal(ctx, tenantID, reference, totalAmount, itemsJSONStr)
	deductStockFromItems(ctx, tenantID, itemsJSONStr)
	sendPaymentWANotification(tenantID, reference, totalAmount)

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Store payment webhook processed"})
}

func createPaymentJournal(ctx context.Context, tenantID, reference string, totalAmount float64, itemsJSONStr string) {
	var debitAccID, creditAccID string
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '101'", tenantID).Scan(&debitAccID)
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '400'", tenantID).Scan(&creditAccID)

	var entryID string
	err := DB.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, time.Now().Format("2006-01-02"), "Pembayaran Xendit: "+reference, reference, itemsJSONStr).Scan(&entryID)
	if err != nil {
		return
	}
	DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)", entryID, debitAccID, totalAmount)
	DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)", entryID, creditAccID, totalAmount)
}

func deductStockFromItems(ctx context.Context, tenantID, itemsJSONStr string) {
	var parsedItems struct {
		Items []struct {
			ID       string `json:"id"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(itemsJSONStr), &parsedItems); parsedItems.Items == nil {
		return
	}
	for _, item := range parsedItems.Items {
		DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
	}
}

func sendPaymentWANotification(tenantID, reference string, totalAmount float64) {
	var waNumber *string
	// ponytail: blocking DB call ok here; goroutine below is the async part
	_ = DB.QueryRow(context.Background(), "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&waNumber)
	if waNumber == nil || *waNumber == "" {
		return
	}
	go func(phone, ref string, amount float64) {
		msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA VIA XENDIT* ✅\n\nRef: %s\nNominal: Rp %.0f\nMetode: Xendit (QRIS/VA)\n\nTerima kasih, dana otomatis tercatat dalam sistem POS Anda.", ref, amount)

		target := phone
		if strings.HasPrefix(target, "0") {
			target = "62" + target[1:]
		}
		if !strings.Contains(target, "@") {
			target = target + "@s.whatsapp.net"
		}

		data := url.Values{}
		data.Set("tenant_id", tenantID)
		data.Set("target", target)
		data.Set("message", msg)

		req, _ := http.NewRequest("POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Message-Type", "subscription")
		req.Header.Set("X-Source", "umkm-accounting")
		client := &http.Client{Timeout: 10 * time.Second}
		client.Do(req)
	}(*waNumber, reference, totalAmount)
}

func handleTransactionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	reference := r.URL.Query().Get("reference")
	if reference == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing reference"})
		return
	}

	var status string
	err := DB.QueryRow(r.Context(), "SELECT status FROM pos_transactions WHERE reference = $1 AND tenant_id = $2", reference, tenantID).Scan(&status)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: errTransactionNotFound})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  status,
	})
}

func handleForwarders(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, "SELECT id, phone_number FROM tenant_forwarders WHERE tenant_id = $1", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var list []map[string]string
		for rows.Next() {
			var id, phone string
			if err := rows.Scan(&id, &phone); err == nil {
				list = append(list, map[string]string{"id": id, "phone_number": phone})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: list})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid input"})
			return
		}
		var newID string
		err := DB.QueryRow(ctx, "INSERT INTO tenant_forwarders (tenant_id, phone_number) VALUES ($1, $2) RETURNING id", tenantID, req.PhoneNumber).Scan(&newID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert error"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": newID}})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		DB.Exec(ctx, "DELETE FROM tenant_forwarders WHERE id = $1 AND tenant_id = $2", id, tenantID)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Deleted"})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowed})
}