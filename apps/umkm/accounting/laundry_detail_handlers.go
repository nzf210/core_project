package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/response"
)

func capitalizeWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// handleLaundryOrderDetail dispatches /laundry/orders/{id}[/status|/pay]
func handleLaundryOrderDetail(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, "/api/umkm")
	path := strings.TrimPrefix(rawPath, "/laundry/orders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		response.Error(w, http.StatusBadRequest, "Missing order ID", nil)
		return
	}

	orderID := parts[0]

	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			getLaundryOrderDetail(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	action := parts[1]
	switch action {
	case "status":
		if r.Method == http.MethodPatch || r.Method == http.MethodPut {
			updateLaundryOrderStatus(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	case "pay":
		if r.Method == http.MethodPost {
			payLaundryOrder(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func getLaundryOrderDetail(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o LaundryOrder
	err := DB.QueryRow(ctx, `
		SELECT id, tenant_id, order_no, customer_name, customer_phone, service_type,
		       weight_grams, item_count, rack_location, status, total_amount, is_paid,
		       payment_method, estimated_completion_at, completed_at, notes, created_at, updated_at
		FROM laundry_orders
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.ServiceType,
		&o.WeightGrams, &o.ItemCount, &o.RackLocation, &o.Status, &o.TotalAmount, &o.IsPaid,
		&o.PaymentMethod, &o.EstimatedCompletionAt, &o.CompletedAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		response.Error(w, http.StatusNotFound, "Order laundry tidak ditemukan", nil)
		return
	}

	response.JSON(w, http.StatusOK, "Detail order laundry berhasil dimuat", o)
}

func updateLaundryOrderStatus(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req UpdateLaundryStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	targetStatus := strings.ToLower(strings.TrimSpace(req.Status))
	if !isValidLaundryStatus(targetStatus) {
		response.Error(w, http.StatusBadRequest, "Status laundry tidak valid", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o LaundryOrder
	var completedAtUpdate *time.Time
	if targetStatus == "completed" {
		now := time.Now()
		completedAtUpdate = &now
	}

	err := DB.QueryRow(ctx, `
		UPDATE laundry_orders
		SET status = $1,
		    completed_at = COALESCE($2, completed_at),
		    updated_at = NOW()
		WHERE tenant_id = $3 AND id = $4
		RETURNING id, tenant_id, order_no, customer_name, customer_phone, service_type,
		          weight_grams, item_count, rack_location, status, total_amount, is_paid,
		          payment_method, estimated_completion_at, completed_at, notes, created_at, updated_at`,
		targetStatus, completedAtUpdate, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.ServiceType,
		&o.WeightGrams, &o.ItemCount, &o.RackLocation, &o.Status, &o.TotalAmount, &o.IsPaid,
		&o.PaymentMethod, &o.EstimatedCompletionAt, &o.CompletedAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("updateLaundryOrderStatus: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal mengupdate status order laundry", nil)
		return
	}

	// Kirim notifikasi WhatsApp otomatis saat cucian siap diambil
	if targetStatus == "ready" {
		go sendLaundryReadyWANotification(o)
	}

	response.JSON(w, http.StatusOK, "Status order laundry berhasil diupdate", o)
}

func payLaundryOrder(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req PayLaundryOrderReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o LaundryOrder
	err := DB.QueryRow(ctx, `
		UPDATE laundry_orders
		SET is_paid = true,
		    payment_method = $1,
		    updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
		RETURNING id, tenant_id, order_no, customer_name, customer_phone, service_type,
		          weight_grams, item_count, rack_location, status, total_amount, is_paid,
		          payment_method, estimated_completion_at, completed_at, notes, created_at, updated_at`,
		req.PaymentMethod, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.ServiceType,
		&o.WeightGrams, &o.ItemCount, &o.RackLocation, &o.Status, &o.TotalAmount, &o.IsPaid,
		&o.PaymentMethod, &o.EstimatedCompletionAt, &o.CompletedAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("payLaundryOrder: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal memproses pembayaran order laundry", nil)
		return
	}

	if o.TotalAmount > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Laundry %s","price":%d,"quantity":1}]`, o.ServiceType, o.TotalAmount/100)
		createPaymentJournal(ctx, tenantID, o.OrderNo, float64(o.TotalAmount)/100, itemsDesc)
	}

	response.JSON(w, http.StatusOK, "Pembayaran order laundry berhasil dikonfirmasi", o)
}

func sendLaundryReadyWANotification(o LaundryOrder) {
	if strings.TrimSpace(o.CustomerPhone) == "" {
		return
	}

	var storeName *string
	_ = DB.QueryRow(context.Background(), "SELECT name FROM tenants WHERE id = $1", o.TenantID).Scan(&storeName)
	toko := "Laundry Kami"
	if storeName != nil && *storeName != "" {
		toko = *storeName
	}

	statusBayar := "Belum Lunas"
	if o.IsPaid {
		statusBayar = "LUNAS ✅"
	}

	msg := fmt.Sprintf("🧺 *STATUS CUCIAN: SIAP DIAMBIL!*\n*%s*\n\n"+
		"Halo Kak *%s*,\nCucian Anda dengan rincian berikut sudah selesai dan siap diambil:\n\n"+
		"🔖 No. Nota: *%s*\n"+
		"👕 Layanan: %s\n"+
		"📦 Rak/Keranjang: *%s*\n"+
		"💰 Total: *Rp %s* (%s)\n\n"+
		"Silakan tunjukkan nomor nota ini kepada kasir saat pengambilan. Terima kasih telah mempercayakan cucian Anda kepada %s! ✨",
		toko, o.CustomerName, o.OrderNo, capitalizeWord(o.ServiceType), o.RackLocation, formatRupiah(o.TotalAmount), statusBayar, toko)

	target := strings.TrimSpace(o.CustomerPhone)
	target = strings.TrimPrefix(target, "+")
	if strings.HasPrefix(target, "0") {
		target = "62" + target[1:]
	}
	if !strings.Contains(target, "@") {
		target = target + "@s.whatsapp.net"
	}

	data := url.Values{}
	data.Set("tenant_id", o.TenantID)
	data.Set("target", target)
	data.Set("message", msg)

	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = "http://wa-gateway:8202"
	}
	endpoint := strings.TrimRight(waURL, "/") + "/api/wa/send"

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Message-Type", "transactional")
	req.Header.Set("X-Source", "umkm-laundry-ready")

	client := &http.Client{Timeout: 10 * time.Second}
	_, _ = client.Do(req)
}
