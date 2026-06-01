import re

with open("apps/umkm/accounting/main.go", "r") as f:
    content = f.read()

# Replace handleCheckout logic
checkout_regex = re.compile(r'(// Insert Transaction\n\s*dateStr := time\.Now\(\)\.Format\("2006-01-02"\)\n\s*description := "Penjualan via " \+ req\.PaymentMethod\n\s*reference := "INV-" \+ time\.Now\(\)\.Format\("060102150405"\))(.*?)(writeJSON\(w, http\.StatusOK, map\[string\]interface\{\}\{\n\s*"success": true,\n\s*"message": "Transaksi berhasil dicatat",\n\s*"qris_url": qrisURL,\n\s*"reference": reference,\n\s*\}\)\n\s*return)', re.DOTALL)

replacement = """// Insert Transaction
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

		// Mock QRIS implementation
		qrisURL := ""
		if req.PaymentMethod == "qris" {
			if qrisData != nil && *qrisData != "" {
				dynQris := generateDynamicQRIS(*qrisData, realTotalAmount)
				qrisURL = "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=" + dynQris
			} else {
				qData := "QRIS_UMKM_" + reference + "_AMT_" + fmt.Sprintf("%.0f", realTotalAmount)
				qrisURL = "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=" + qData
			}
			
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Menunggu pembayaran QRIS",
				"status": "pending",
				"qris_url": qrisURL,
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

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Transaksi berhasil dicatat",
			"status": "paid",
			"qris_url": "",
			"reference": reference,
		})
		return"""

new_content = checkout_regex.sub(replacement, content)

# Add handler for webhook and status
additional_handlers = """
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
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status": status,
	})
}

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
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
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
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
	var waNumber, fonnteToken *string
	DB.QueryRow(ctx, "SELECT wa_number, fonnte_token FROM tenants WHERE id = $1", tenantID).Scan(&waNumber, &fonnteToken)
	if waNumber != nil && *waNumber != "" && fonnteToken != nil && *fonnteToken != "" {
		go func(phone, token, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA* ✅\\n\\nRef: %s\\nNominal: Rp %.0f\\nMetode: QRIS\\n\\nTerima kasih, dana telah masuk ke rekening Anda dan sistem telah mencatat transaksi ini.", ref, amount)
			
			// Fonnte API call
			data := map[string]string{"target": phone, "message": msg}
			jsonData, _ := json.Marshal(data)
			req, _ := http.NewRequest("POST", "https://api.fonnte.com/send", strings.NewReader(string(jsonData)))
			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(*waNumber, *fonnteToken, req.Reference, totalAmount)
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Payment processed successfully"})
}

// Add these handlers to mux at the end
"""

# Insert before mux setup
handler_setup_idx = new_content.find("mux := http.NewServeMux()")
new_content = new_content[:handler_setup_idx] + additional_handlers + new_content[handler_setup_idx:]

# Add to mux
mux_setup = """	mux.HandleFunc("/api/umkm/checkout", handleCheckout)
	mux.HandleFunc("/api/umkm/transactions/status", handleTransactionStatus)
	mux.HandleFunc("/api/umkm/webhook/payment", handlePaymentWebhook)"""

new_content = new_content.replace('mux.HandleFunc("/api/umkm/checkout", handleCheckout)', mux_setup)

with open("apps/umkm/accounting/main.go", "w") as f:
    f.write(new_content)
