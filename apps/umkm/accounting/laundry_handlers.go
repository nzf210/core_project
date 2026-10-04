package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"core_project/shared/sdk/response"
)

// LaundryOrder represents an order in the laundry tracking workflow.
type LaundryOrder struct {
	ID                    string     `json:"id"`
	TenantID              string     `json:"tenant_id"`
	OrderNo               string     `json:"order_no"`
	CustomerName          string     `json:"customer_name"`
	CustomerPhone         string     `json:"customer_phone"`
	ServiceType           string     `json:"service_type"` // kiloan, satuan, dry_clean
	WeightGrams           int        `json:"weight_grams"`
	ItemCount             int        `json:"item_count"`
	RackLocation          string     `json:"rack_location"`
	Status                string     `json:"status"` // received, washing, drying, ironing, ready, completed, cancelled
	TotalAmount           int64      `json:"total_amount"` // sen (1 IDR = 100 sen)
	IsPaid                bool       `json:"is_paid"`
	PaymentMethod         string     `json:"payment_method"`
	EstimatedCompletionAt *time.Time `json:"estimated_completion_at,omitempty"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	Notes                 string     `json:"notes"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type CreateLaundryOrderReq struct {
	CustomerName          string     `json:"customer_name"`
	CustomerPhone         string     `json:"customer_phone"`
	ServiceType           string     `json:"service_type"`
	WeightGrams           int        `json:"weight_grams"`
	ItemCount             int        `json:"item_count"`
	RackLocation          string     `json:"rack_location"`
	TotalAmount           int64      `json:"total_amount"`
	IsPaid                bool       `json:"is_paid"`
	PaymentMethod         string     `json:"payment_method"`
	EstimatedCompletionAt *time.Time `json:"estimated_completion_at,omitempty"`
	Notes                 string     `json:"notes"`
}

type UpdateLaundryStatusReq struct {
	Status string `json:"status"`
}

type PayLaundryOrderReq struct {
	PaymentMethod string `json:"payment_method"`
}

func generateLaundryOrderNo() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("LND-%s-%04d", time.Now().Format("20060102"), n.Int64()+1000)
}

func isValidLaundryStatus(status string) bool {
	switch status {
	case "received", "washing", "drying", "ironing", "ready", "completed", "cancelled":
		return true
	default:
		return false
	}
}

// handleLaundryOrders handles GET (list) and POST (create) on /api/umkm/laundry/orders
func handleLaundryOrders(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listLaundryOrders(w, r, tenantID)
	case http.MethodPost:
		createLaundryOrder(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func listLaundryOrders(w http.ResponseWriter, r *http.Request, tenantID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	query := `
		SELECT id, tenant_id, order_no, customer_name, customer_phone, service_type,
		       weight_grams, item_count, rack_location, status, total_amount, is_paid,
		       payment_method, estimated_completion_at, completed_at, notes, created_at, updated_at
		FROM laundry_orders
		WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query += fmt.Sprintf(" AND (order_no ILIKE $%d OR customer_name ILIKE $%d OR customer_phone ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, searchPattern)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(ctx, query, args...)
	if err != nil {
		slog.Error("listLaundryOrders: query failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal memuat daftar order laundry", nil)
		return
	}
	defer rows.Close()

	orders := make([]LaundryOrder, 0)
	for rows.Next() {
		var o LaundryOrder
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.ServiceType,
			&o.WeightGrams, &o.ItemCount, &o.RackLocation, &o.Status, &o.TotalAmount, &o.IsPaid,
			&o.PaymentMethod, &o.EstimatedCompletionAt, &o.CompletedAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			slog.Error("listLaundryOrders: row scan failed", "error", err)
			continue
		}
		orders = append(orders, o)
	}

	response.JSON(w, http.StatusOK, "Daftar order laundry berhasil dimuat", orders)
}

func createLaundryOrder(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req CreateLaundryOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	if req.CustomerName == "" {
		response.Error(w, http.StatusBadRequest, "Nama pelanggan wajib diisi", nil)
		return
	}
	if req.CustomerPhone == "" {
		response.Error(w, http.StatusBadRequest, "Nomor WhatsApp pelanggan wajib diisi", nil)
		return
	}

	if req.ServiceType == "" {
		req.ServiceType = "kiloan"
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "unpaid"
	}

	orderNo := generateLaundryOrderNo()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var newOrder LaundryOrder
	err := DB.QueryRow(ctx, `
		INSERT INTO laundry_orders (
			tenant_id, order_no, customer_name, customer_phone, service_type,
			weight_grams, item_count, rack_location, status, total_amount, is_paid,
			payment_method, estimated_completion_at, notes
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, 'received', $9, $10,
			$11, $12, $13
		) RETURNING id, tenant_id, order_no, customer_name, customer_phone, service_type,
		          weight_grams, item_count, rack_location, status, total_amount, is_paid,
		          payment_method, estimated_completion_at, completed_at, notes, created_at, updated_at`,
		tenantID, orderNo, req.CustomerName, req.CustomerPhone, req.ServiceType,
		req.WeightGrams, req.ItemCount, req.RackLocation, req.TotalAmount, req.IsPaid,
		req.PaymentMethod, req.EstimatedCompletionAt, req.Notes,
	).Scan(
		&newOrder.ID, &newOrder.TenantID, &newOrder.OrderNo, &newOrder.CustomerName, &newOrder.CustomerPhone, &newOrder.ServiceType,
		&newOrder.WeightGrams, &newOrder.ItemCount, &newOrder.RackLocation, &newOrder.Status, &newOrder.TotalAmount, &newOrder.IsPaid,
		&newOrder.PaymentMethod, &newOrder.EstimatedCompletionAt, &newOrder.CompletedAt, &newOrder.Notes, &newOrder.CreatedAt, &newOrder.UpdatedAt,
	)

	if err != nil {
		slog.Error("createLaundryOrder: insert failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal membuat order laundry baru", nil)
		return
	}

	// Jika langsung dibayar lunas saat order dibuat, catat pembukuan kas
	if newOrder.IsPaid && newOrder.TotalAmount > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Laundry %s","price":%d,"quantity":1}]`, newOrder.ServiceType, newOrder.TotalAmount/100)
		createPaymentJournal(ctx, tenantID, newOrder.OrderNo, float64(newOrder.TotalAmount)/100, itemsDesc)
	}

	response.JSON(w, http.StatusCreated, "Order laundry berhasil dibuat", newOrder)
}
