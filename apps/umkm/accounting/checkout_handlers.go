package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

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
			PaymentMethod string `json:"payment_method"` // "cash" or "qris"
			Items         []struct {
				ProductID string `json:"product_id"`
				Quantity  int    `json:"quantity"`
				Price     int64  `json:"price"` // frontend sends integer price in IDR
			} `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
			return
		}

		if len(req.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Items cannot be empty"})
			return
		}

		ctx := r.Context()
		handleCheckoutPostRequest(w, r, ctx, tenantID, req.PaymentMethod, req.Items)
	} else {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}
type CheckoutItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"`
}

func handleCheckoutPostRequest(w http.ResponseWriter, r *http.Request, ctx context.Context, tenantID, paymentMethod string, items []struct{ProductID string "json:\"product_id\""; Quantity int "json:\"quantity\""; Price int64 "json:\"price\""}) {
	var xenditApiKey *string
	err := DB.QueryRow(ctx, "SELECT xendit_api_key FROM tenants WHERE id = $1", tenantID).Scan(&xenditApiKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to read tenant config"})
		return
	}

	if paymentMethod == "qris" && (xenditApiKey == nil || *xenditApiKey == "") {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Tenant belum setup integrasi Xendit."})
		return
	}

	var totalAmount int64
	for _, item := range items {
		totalAmount += item.Price * int64(item.Quantity)
	}
	realTotalAmount := totalAmount * 100 // convert IDR to cents

	if !validateCheckoutStock(w, ctx, tenantID, items) {
		return
	}

	dateStr := time.Now().Format("2006-01-02")
	description := "Penjualan via " + paymentMethod
	reference := "INV-" + time.Now().Format("060102150405")

	itemsJSON, _ := json.Marshal(map[string]interface{}{
		"items": items,
	})

	status := "paid"
	if paymentMethod == "qris" {
		status = "pending"
	}

	_, err = DB.Exec(ctx, "INSERT INTO pos_transactions (tenant_id, reference, total_amount, payment_method, status, items_json) VALUES ($1, $2, $3, $4, $5, $6)",
		tenantID, reference, realTotalAmount, paymentMethod, status, itemsJSON)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menyimpan transaksi POS"})
		return
	}

	if paymentMethod == "qris" {
		handleCheckoutXendit(w, ctx, *xenditApiKey, reference, realTotalAmount)
		return
	}

	handleCheckoutCash(w, ctx, tenantID, dateStr, description, reference, itemsJSON, realTotalAmount, items)
}

func validateCheckoutStock(w http.ResponseWriter, ctx context.Context, tenantID string, items []struct{ProductID string "json:\"product_id\""; Quantity int "json:\"quantity\""; Price int64 "json:\"price\""}) bool {
	for _, item := range items {
		var stock, price int64
		err := DB.QueryRow(ctx, "SELECT stock, price FROM products WHERE id = $1 AND tenant_id = $2", item.ProductID, tenantID).Scan(&stock, &price)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Produk tidak ditemukan"})
			return false
		}
		if stock < int64(item.Quantity) {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Stok tidak mencukupi untuk beberapa produk"})
			return false
		}
		if price != item.Price*100 { // Convert IDR to Cents for comparison
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Harga produk tidak sesuai master data"})
			return false
		}
	}
	return true
}

func handleCheckoutXendit(w http.ResponseWriter, ctx context.Context, xenditApiKey, reference string, realTotalAmount int64) {
	xClient := xendit.NewClient(xenditApiKey)

	externalID := reference
	createInvoiceReq := invoice.NewCreateInvoiceRequest(externalID, float64(realTotalAmount))
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
}

func handleCheckoutCash(w http.ResponseWriter, ctx context.Context, tenantID, dateStr, description, reference string, itemsJSON []byte, realTotalAmount int64, items []struct{ProductID string "json:\"product_id\""; Quantity int "json:\"quantity\""; Price int64 "json:\"price\""}) {
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
		entryID, "1000", realTotalAmount) // 1000 = Kas
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create line"})
		return
	}

	// Insert Credit Line
	_, err = tx.Exec(ctx,
		"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)",
		entryID, "4000", realTotalAmount) // 4000 = Pendapatan
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create line"})
		return
	}

	// Deduct stock
	for _, item := range items {
		_, err = tx.Exec(ctx, "UPDATE products SET stock = stock - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ProductID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update stock"})
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction commit failed"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Checkout berhasil dicatat",
		Data: map[string]interface{}{
			"reference": reference,
		},
	})
}
