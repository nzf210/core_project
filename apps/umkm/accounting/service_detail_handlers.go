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

// handleServiceOrderDetail dispatches /service/orders/{id}[/status|/pay]
func handleServiceOrderDetail(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, "/api/umkm")
	path := strings.TrimPrefix(rawPath, "/service/orders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		response.Error(w, http.StatusBadRequest, "Missing order ID", nil)
		return
	}

	orderID := parts[0]

	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			getServiceOrderDetail(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	action := parts[1]
	switch action {
	case "status":
		if r.Method == http.MethodPatch || r.Method == http.MethodPut {
			updateServiceOrderStatus(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	case "pay":
		if r.Method == http.MethodPost {
			payServiceOrder(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func getServiceOrderDetail(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o ServiceOrder
	err := DB.QueryRow(ctx, `
		SELECT id, tenant_id, order_no, customer_name, customer_phone, unit_name,
		       unit_identifier, complaint, technician_name, status, estimated_cost,
		       final_cost, is_paid, payment_method, notes, completed_at, created_at, updated_at
		FROM service_orders
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.UnitName,
		&o.UnitIdentifier, &o.Complaint, &o.TechnicianName, &o.Status, &o.EstimatedCost,
		&o.FinalCost, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CompletedAt, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		response.Error(w, http.StatusNotFound, "Order servis tidak ditemukan", nil)
		return
	}

	response.JSON(w, http.StatusOK, "Detail order servis berhasil dimuat", o)
}

func updateServiceOrderStatus(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req UpdateServiceStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	targetStatus := strings.ToLower(strings.TrimSpace(req.Status))
	if !isValidServiceStatus(targetStatus) {
		response.Error(w, http.StatusBadRequest, "Status order servis tidak valid", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o ServiceOrder
	var completedAtUpdate *time.Time
	if targetStatus == "completed" {
		now := time.Now()
		completedAtUpdate = &now
	}

	query := `
		UPDATE service_orders
		SET status = $1,
		    completed_at = COALESCE($2, completed_at),
		    final_cost = CASE WHEN $3 > 0 THEN $3 ELSE final_cost END,
		    updated_at = NOW()
		WHERE tenant_id = $4 AND id = $5
		RETURNING id, tenant_id, order_no, customer_name, customer_phone, unit_name,
		          unit_identifier, complaint, technician_name, status, estimated_cost,
		          final_cost, is_paid, payment_method, notes, completed_at, created_at, updated_at`

	var finalCostParam int64
	if req.FinalCost != nil {
		finalCostParam = *req.FinalCost
	}

	err := DB.QueryRow(ctx, query,
		targetStatus, completedAtUpdate, finalCostParam, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.UnitName,
		&o.UnitIdentifier, &o.Complaint, &o.TechnicianName, &o.Status, &o.EstimatedCost,
		&o.FinalCost, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CompletedAt, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("updateServiceOrderStatus: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal mengupdate status order servis", nil)
		return
	}

	// Send WhatsApp notification automatically when unit is ready for pick-up
	if targetStatus == "ready" {
		go sendServiceReadyWANotification(o)
	}

	response.JSON(w, http.StatusOK, "Status order servis berhasil diupdate", o)
}

func payServiceOrder(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req PayServiceOrderReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o ServiceOrder
	err := DB.QueryRow(ctx, `
		UPDATE service_orders
		SET is_paid = true,
		    payment_method = $1,
		    updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
		RETURNING id, tenant_id, order_no, customer_name, customer_phone, unit_name,
		          unit_identifier, complaint, technician_name, status, estimated_cost,
		          final_cost, is_paid, payment_method, notes, completed_at, created_at, updated_at`,
		req.PaymentMethod, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.UnitName,
		&o.UnitIdentifier, &o.Complaint, &o.TechnicianName, &o.Status, &o.EstimatedCost,
		&o.FinalCost, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CompletedAt, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("payServiceOrder: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal memproses pembayaran order servis", nil)
		return
	}

	if o.FinalCost > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Servis %s (%s)","price":%d,"quantity":1}]`, o.UnitName, o.OrderNo, o.FinalCost/100)
		createPaymentJournal(ctx, tenantID, o.OrderNo, float64(o.FinalCost)/100, itemsDesc)
	}

	response.JSON(w, http.StatusOK, "Pembayaran order servis berhasil dikonfirmasi", o)
}

func sendServiceReadyWANotification(o ServiceOrder) {
	if strings.TrimSpace(o.CustomerPhone) == "" {
		return
	}

	var storeName *string
	_ = DB.QueryRow(context.Background(), "SELECT name FROM tenants WHERE id = $1", o.TenantID).Scan(&storeName)
	toko := "Bengkel / Servis Kami"
	if storeName != nil && *storeName != "" {
		toko = *storeName
	}

	statusBayar := "Belum Lunas"
	if o.IsPaid {
		statusBayar = "LUNAS ✅"
	}

	unitInfo := o.UnitName
	if o.UnitIdentifier != "" {
		unitInfo += fmt.Sprintf(" (%s)", o.UnitIdentifier)
	}

	cost := o.FinalCost
	if cost == 0 {
		cost = o.EstimatedCost
	}

	msg := fmt.Sprintf("🔧 *STATUS SERVIS: SIAP DIAMBIL!*\n*%s*\n\n"+
		"Halo Kak *%s*,\nPengerjaan servis untuk unit Anda telah selesai dan siap diambil:\n\n"+
		"🔖 No. SPK: *%s*\n"+
		"🛵 Unit: *%s*\n"+
		"👨‍🔧 Teknisi: *%s*\n"+
		"💰 Total Biaya: *Rp %s* (%s)\n\n"+
		"Silakan tunjukkan nomor SPK ini saat pengambilan unit. Terima kasih telah mempercayakan perbaikan kepada %s! ✨",
		toko, o.CustomerName, o.OrderNo, unitInfo, o.TechnicianName, formatRupiah(cost), statusBayar, toko)

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
	req.Header.Set("X-Source", "umkm-service-ready")

	client := &http.Client{Timeout: 10 * time.Second}
	_, _ = client.Do(req)
}
