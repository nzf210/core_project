package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
	"core_project/shared/sdk/response"
)

type CheckoutItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"`
}

type CheckoutRequest struct {
	PaymentMethod string         `json:"payment_method"`
	Items         []CheckoutItem `json:"items"`
	CustomerPhone string         `json:"customer_phone"`
}

type cashCheckoutParams struct {
	tenantID        string
	dateStr         string
	description     string
	reference       string
	itemsJSON       []byte
	realTotalAmount int64
	items           []CheckoutItem
	customerPhone   string
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant"})
		return
	}

	if r.Method == http.MethodPost {
		var req CheckoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: response.InvalidRequest})
			return
		}

		if len(req.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Items cannot be empty"})
			return
		}

		ctx := r.Context()
		handleCheckoutPostRequest(w, ctx, tenantID, req.PaymentMethod, req.Items, req.CustomerPhone)
	} else {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func handleCheckoutPostRequest(w http.ResponseWriter, ctx context.Context, tenantID, paymentMethod string, items []CheckoutItem, customerPhone string) {
	var xenditApiKey, staticQRIS *string
	err := DB.QueryRow(ctx, "SELECT xendit_api_key, static_qris_payload FROM tenants WHERE id = $1", tenantID).Scan(&xenditApiKey, &staticQRIS)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to read tenant config"})
		return
	}

	hasStaticQRIS := staticQRIS != nil && strings.TrimSpace(*staticQRIS) != ""
	hasXendit := xenditApiKey != nil && strings.TrimSpace(*xenditApiKey) != ""

	if paymentMethod == "qris" && !hasStaticQRIS && !hasXendit {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Tenant belum setup QRIS. Silakan masukkan QRIS Statik Toko atau API Key Xendit di menu Pengaturan."})
		return
	}

	var totalAmount int64
	for _, item := range items {
		totalAmount += item.Price * int64(item.Quantity)
	}
	realTotalAmount := totalAmount * 100

	if !validateCheckoutStock(w, ctx, tenantID, items) {
		return
	}

	dateStr := time.Now().Format("2006-01-02")
	description := "Penjualan via " + paymentMethod
	reference := "INV-" + time.Now().Format("060102150405")

	itemsJSON, _ := json.Marshal(map[string]any{
		"items":          items,
		"customer_phone": customerPhone,
	})

	status := "paid"
	if paymentMethod == "qris" {
		status = "pending"
	}

	_, err = DB.Exec(ctx, `INSERT INTO pos_transactions (tenant_id, reference, total_amount, payment_method, status, items_json, customer_phone)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, reference, realTotalAmount, paymentMethod, status, itemsJSON, customerPhone)
	if err != nil {
		_, err = DB.Exec(ctx, `INSERT INTO pos_transactions (tenant_id, reference, total_amount, payment_method, status, items_json)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, reference, realTotalAmount, paymentMethod, status, itemsJSON)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menyimpan transaksi POS"})
			return
		}
	}

	if paymentMethod == "qris" {
		// Priority 1: Merchant Static QRIS converted to Dynamic QRIS (0% fee, instant direct to merchant bank/e-wallet)
		if hasStaticQRIS {
			dynamicPayload := generateDynamicQRIS(*staticQRIS, float64(totalAmount))
			writeJSON(w, http.StatusOK, map[string]any{
				"success":        true,
				"message":        "Dynamic QRIS berhasil dibuat",
				"status":         "pending",
				"type":           "dynamic_qris",
				"reference":      reference,
				"qris_content":   dynamicPayload,
				"total_amount":   totalAmount,
				"customer_phone": customerPhone,
			})
			return
		}

		// Priority 2: Xendit Payment Gateway
		if hasXendit {
			handleCheckoutXendit(w, ctx, *xenditApiKey, reference, realTotalAmount)
			return
		}
	}

	handleCheckoutCash(w, ctx, cashCheckoutParams{
		tenantID:        tenantID,
		dateStr:         dateStr,
		description:     description,
		reference:       reference,
		itemsJSON:       itemsJSON,
		realTotalAmount: realTotalAmount,
		items:           items,
		customerPhone:   customerPhone,
	})
}

func validateCheckoutStock(w http.ResponseWriter, ctx context.Context, tenantID string, items []CheckoutItem) bool {
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
		if price != item.Price*100 {
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
		errMsg := "Gagal membuat invoice Xendit. Cek API Key Anda."
		if strings.Contains(strings.ToLower(err.Error()), "forbidden") || strings.Contains(strings.ToLower(err.Error()), "permission") {
			errMsg = "API Key Xendit tidak memiliki izin membuat invoice. Buka Xendit Dashboard > Settings > API Keys, pastikan izin 'Money-in: Invoices' diatur ke 'Write'."
		}
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errMsg})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"message":   "Menunggu pembayaran via Xendit",
		"status":    "pending",
		"qris_url":  resp.InvoiceUrl,
		"reference": reference,
	})
}

func handleCheckoutCash(w http.ResponseWriter, ctx context.Context, params cashCheckoutParams) {
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
		return
	}
	defer tx.Rollback(ctx)

	var entryID string
	err = tx.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		params.tenantID, params.dateStr, params.description, params.reference, params.itemsJSON).Scan(&entryID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create entry"})
		return
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)",
		entryID, "1000", params.realTotalAmount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create line"})
		return
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)",
		entryID, "4000", params.realTotalAmount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create line"})
		return
	}

	for _, item := range params.items {
		_, err = tx.Exec(ctx, "UPDATE products SET stock = stock - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ProductID, params.tenantID)
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

	if params.customerPhone != "" {
		sendCustomerReceiptWANotification(params.tenantID, params.customerPhone, params.reference, float64(params.realTotalAmount)/100, "Tunai (Cash)", params.items)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Checkout berhasil dicatat",
		Data: map[string]any{
			"reference": params.reference,
		},
	})
}

func handleCheckoutConfirm(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reference == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid reference"})
		return
	}

	ctx := r.Context()
	var currentStatus, paymentMethod, itemsJSONStr, custPhoneDB string
	var totalAmount float64
	err := DB.QueryRow(ctx, `SELECT status, payment_method, total_amount, items_json::text, COALESCE(customer_phone, '')
		FROM pos_transactions WHERE reference = $1 AND tenant_id = $2 FOR UPDATE`, req.Reference, tenantID).
		Scan(&currentStatus, &paymentMethod, &totalAmount, &itemsJSONStr, &custPhoneDB)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
		return
	}

	if currentStatus == "paid" {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Transaksi sudah lunas"})
		return
	}

	_, err = DB.Exec(ctx, "UPDATE pos_transactions SET status = 'paid', updated_at = NOW() WHERE reference = $1 AND tenant_id = $2", req.Reference, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal memperbarui status transaksi"})
		return
	}

	createPaymentJournal(ctx, tenantID, req.Reference, totalAmount, itemsJSONStr)
	deductStockFromItems(ctx, tenantID, itemsJSONStr)

	customerPhone := custPhoneDB
	if customerPhone == "" {
		var parsed struct {
			CustomerPhone string `json:"customer_phone"`
		}
		_ = json.Unmarshal([]byte(itemsJSONStr), &parsed)
		customerPhone = parsed.CustomerPhone
	}

	if customerPhone != "" {
		var parsedItems struct {
			Items []CheckoutItem `json:"items"`
		}
		_ = json.Unmarshal([]byte(itemsJSONStr), &parsedItems)
		sendCustomerReceiptWANotification(tenantID, customerPhone, req.Reference, totalAmount/100, "QRIS (Dinamis)", parsedItems.Items)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pembayaran QRIS berhasil dikonfirmasi",
		Data: map[string]any{
			"reference": req.Reference,
			"status":    "paid",
		},
	})
}

func sendCustomerReceiptWANotification(tenantID, phone, reference string, totalAmount float64, paymentMethod string, items []CheckoutItem) {
	if phone == "" {
		return
	}
	var storeName *string
	_ = DB.QueryRow(context.Background(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&storeName)
	toko := "Toko UMKM"
	if storeName != nil && *storeName != "" {
		toko = *storeName
	}

	go func(targetPhone, ref, sName string, amount float64, method string, itms []CheckoutItem) {
		dateStr := time.Now().Format("02 Jan 2006 15:04")
		itemLines := ""
		for idx, it := range itms {
			var prodName string
			_ = DB.QueryRow(context.Background(), "SELECT name FROM products WHERE id = $1", it.ProductID).Scan(&prodName)
			if prodName == "" {
				prodName = fmt.Sprintf("Produk #%d", idx+1)
			}
			itemLines += fmt.Sprintf("• %s x%d = Rp %s\n", prodName, it.Quantity, formatRupiah(it.Price*int64(it.Quantity)))
		}

		msg := fmt.Sprintf("🧾 *STRUK PEMBAYARAN RESMI*\n*%s*\n\n"+
			"📅 Waktu: %s\n"+
			"🔖 No. Ref: %s\n"+
			"💳 Metode: %s\n"+
			"--------------------------------\n"+
			"%s"+
			"--------------------------------\n"+
			"💰 *Total Bayar: Rp %s*\n"+
			"Status: *LUNAS (PAID)* ✅\n\n"+
			"Terima kasih telah berbelanja di %s! Simpan pesan ini sebagai bukti pembayaran digital.",
			sName, dateStr, ref, method, itemLines, formatRupiah(int64(amount)), sName)

		target := strings.TrimSpace(targetPhone)
		if strings.HasPrefix(target, "+") {
			target = target[1:]
		}
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

		req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Message-Type", "invoice")
		req.Header.Set("X-Source", "umkm-pos-receipt")
		client := &http.Client{Timeout: 10 * time.Second}
		_, err := client.Do(req)
		if err != nil {
			slog.Warn("Failed to dispatch customer receipt WA", "error", err, "phone", target)
		}
	}(phone, reference, toko, totalAmount, paymentMethod, items)
}
