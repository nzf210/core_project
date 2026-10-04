package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/response"
)

// handleRestaurantOrderDetail dispatches /restaurant/orders/{id}[/status|/pay]
func handleRestaurantOrderDetail(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, "/api/umkm")
	path := strings.TrimPrefix(rawPath, "/restaurant/orders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		response.Error(w, http.StatusBadRequest, "Missing order ID", nil)
		return
	}

	orderID := parts[0]

	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			getRestaurantOrderDetail(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	action := parts[1]
	switch action {
	case "status":
		if r.Method == http.MethodPatch || r.Method == http.MethodPut {
			updateRestaurantOrderStatus(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	case "pay":
		if r.Method == http.MethodPost {
			payRestaurantOrder(w, r, tenantID, orderID)
			return
		}
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func getRestaurantOrderDetail(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o RestaurantOrder
	var itemsRaw []byte
	err := DB.QueryRow(ctx, `
		SELECT id, tenant_id, order_no, table_number, customer_name, customer_phone,
		       status, items, total_amount, is_paid, payment_method, notes, created_at, updated_at
		FROM restaurant_orders
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.TableNumber, &o.CustomerName, &o.CustomerPhone,
		&o.Status, &itemsRaw, &o.TotalAmount, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		response.Error(w, http.StatusNotFound, "Pesanan restoran tidak ditemukan", nil)
		return
	}

	if len(itemsRaw) > 0 {
		_ = json.Unmarshal(itemsRaw, &o.Items)
	}
	if o.Items == nil {
		o.Items = []RestaurantOrderItem{}
	}

	response.JSON(w, http.StatusOK, "Detail pesanan restoran berhasil dimuat", o)
}

func updateRestaurantOrderStatus(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req UpdateRestaurantStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	targetStatus := strings.ToLower(strings.TrimSpace(req.Status))
	if !isValidRestaurantStatus(targetStatus) {
		response.Error(w, http.StatusBadRequest, "Status pesanan restoran tidak valid", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o RestaurantOrder
	var itemsRaw []byte
	err := DB.QueryRow(ctx, `
		UPDATE restaurant_orders
		SET status = $1,
		    updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
		RETURNING id, tenant_id, order_no, table_number, customer_name, customer_phone,
		          status, items, total_amount, is_paid, payment_method, notes, created_at, updated_at`,
		targetStatus, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.TableNumber, &o.CustomerName, &o.CustomerPhone,
		&o.Status, &itemsRaw, &o.TotalAmount, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("updateRestaurantOrderStatus: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal mengupdate status pesanan restoran", nil)
		return
	}

	if len(itemsRaw) > 0 {
		_ = json.Unmarshal(itemsRaw, &o.Items)
	}
	if o.Items == nil {
		o.Items = []RestaurantOrderItem{}
	}

	response.JSON(w, http.StatusOK, "Status pesanan restoran berhasil diupdate", o)
}

func payRestaurantOrder(w http.ResponseWriter, r *http.Request, tenantID, orderID string) {
	var req PayRestaurantOrderReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o RestaurantOrder
	var itemsRaw []byte
	err := DB.QueryRow(ctx, `
		UPDATE restaurant_orders
		SET is_paid = true,
		    payment_method = $1,
		    updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
		RETURNING id, tenant_id, order_no, table_number, customer_name, customer_phone,
		          status, items, total_amount, is_paid, payment_method, notes, created_at, updated_at`,
		req.PaymentMethod, tenantID, orderID,
	).Scan(
		&o.ID, &o.TenantID, &o.OrderNo, &o.TableNumber, &o.CustomerName, &o.CustomerPhone,
		&o.Status, &itemsRaw, &o.TotalAmount, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err != nil {
		slog.Error("payRestaurantOrder: update failed", "error", err, "order_id", orderID)
		response.Error(w, http.StatusInternalServerError, "Gagal memproses pembayaran pesanan restoran", nil)
		return
	}

	if len(itemsRaw) > 0 {
		_ = json.Unmarshal(itemsRaw, &o.Items)
	}
	if o.Items == nil {
		o.Items = []RestaurantOrderItem{}
	}

	if o.TotalAmount > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Restoran Meja %s","price":%d,"quantity":1}]`, o.TableNumber, o.TotalAmount/100)
		createPaymentJournal(ctx, tenantID, o.OrderNo, float64(o.TotalAmount)/100, itemsDesc)
	}

	response.JSON(w, http.StatusOK, "Pembayaran pesanan restoran berhasil dikonfirmasi", o)
}
