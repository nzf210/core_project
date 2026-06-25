package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"core_project/shared/sdk/webhook"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)


func handleCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant"})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			PaymentMethod string  `json:"payment_method"` // "cash" or "qris"
			TotalAmount   float64 `json:"total_amount"`
			Items         []struct {
				ID       string  `json:"id"`
				Name     string  `json:"name"`
				Quantity int     `json:"quantity"`
				Price    float64 `json:"price"`
			} `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
			return
		}

		ctx := r.Context()

		var realTotalAmount float64
		for _, item := range req.Items {
			var dbPrice float64
			err := DB.QueryRow(ctx, "SELECT price FROM products WHERE id = $1 AND tenant_id = $2", item.ID, tenantID).Scan(&dbPrice)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: fmt.Sprintf("Produk tidak valid: %s", item.Name)})
				return
			}
			realTotalAmount += dbPrice * float64(item.Quantity)
		}

		// Security Check: Compare calculated total with requested total
		if req.TotalAmount != realTotalAmount {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Harga tidak valid. Kemungkinan terdeteksi manipulasi."})
			return
		}

		if realTotalAmount <= 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid amount"})
			return
		}

		// Determine accounts
		var debitAccCode, creditAccCode string
		creditAccCode = "400" // Pendapatan Usaha

		var xenditApiKey *string
		var qrisEnabled *bool

		if req.PaymentMethod == "qris" {
			// Validate QRIS settings
			err := DB.QueryRow(ctx, "SELECT xendit_api_key, qris_enabled FROM tenants WHERE id = $1", tenantID).Scan(&xenditApiKey, &qrisEnabled)
			if err != nil || qrisEnabled == nil || !*qrisEnabled {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Pembayaran QRIS belum diaktifkan oleh toko ini"})
				return
			}
			if xenditApiKey == nil || *xenditApiKey == "" {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Toko belum mengatur Xendit API Key untuk menerima pembayaran QRIS."})
				return
			}
			debitAccCode = "101" // Bank / QRIS
		} else {
			debitAccCode = "100" // Kas (Cash)
		}

		var debitAccID, creditAccID string
		err := DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, debitAccCode).Scan(&debitAccID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Debit account not found. Try relogin to re-seed accounts."})
			return
		}

		err = DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, creditAccCode).Scan(&creditAccID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Credit account not found"})
			return
		}

		// Insert Transaction
		dateStr := time.Now().Format("2006-01-02")
		description := "Penjualan via " + req.PaymentMethod
		reference := "INV-" + time.Now().Format("060102150405")

		itemsJSON, _ := json.Marshal(map[string]interface{}{
			"items": req.Items,
		})

		status := "paid"
		if req.PaymentMethod == "qris" {
			status = "pending"
		}

		_, err = DB.Exec(ctx, "INSERT INTO pos_transactions (tenant_id, reference, total_amount, payment_method, status, items_json) VALUES ($1, $2, $3, $4, $5, $6)",
			tenantID, reference, realTotalAmount, req.PaymentMethod, status, itemsJSON)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menyimpan transaksi POS"})
			return
		}

		// Xendit QRIS/Invoice implementation
		if req.PaymentMethod == "qris" {
			xClient := xendit.NewClient(*xenditApiKey)

			externalID := reference
			createInvoiceReq := invoice.NewCreateInvoiceRequest(externalID, realTotalAmount)
			desc := "Pembayaran Toko UMKM: " + reference
			createInvoiceReq.Description = &desc

			resp, _, err := xClient.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*createInvoiceReq).Execute()
			if err != nil {
				slog.Error("Failed to create store invoice", "error", err)
				writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal membuat invoice Xendit. Cek API Key Anda."})
				return
			}

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":   true,
				"message":   "Menunggu pembayaran via Xendit",
				"status":    "pending",
				"qris_url":  resp.InvoiceUrl,
				"reference": reference,
			})
			return
		}

		// If CASH (Paid immediately), proceed to journal and stock deduction
		tx, err := DB.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer tx.Rollback(ctx)

		var entryID string
		err = tx.QueryRow(ctx,
			"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			tenantID, dateStr, description, reference, itemsJSON).Scan(&entryID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create entry"})
			return
		}

		// Insert Debit Line
		_, err = tx.Exec(ctx,
			"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)",
			entryID, debitAccID, realTotalAmount)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to insert debit line"})
			return
		}

		// Insert Credit Line
		_, err = tx.Exec(ctx,
			"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)",
			entryID, creditAccID, realTotalAmount)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to insert credit line"})
			return
		}

		err = tx.Commit(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to commit transaction"})
			return
		}

		for _, item := range req.Items {
			DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
		}

		webhook.DispatchEvent("pos_checkout_completed", tenantID, map[string]interface{}{
			"reference":      reference,
			"total_amount":   realTotalAmount,
			"payment_method": req.PaymentMethod,
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"message":   "Transaksi berhasil dicatat",
			"status":    "paid",
			"qris_url":  "",
			"reference": reference,
		})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}