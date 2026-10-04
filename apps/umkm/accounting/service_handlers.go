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

// ServiceOrder represents a repair or maintenance job order / SPK.
type ServiceOrder struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	OrderNo        string     `json:"order_no"`
	CustomerName   string     `json:"customer_name"`
	CustomerPhone  string     `json:"customer_phone"`
	UnitName       string     `json:"unit_name"`
	UnitIdentifier string     `json:"unit_identifier"`
	Complaint      string     `json:"complaint"`
	TechnicianName string     `json:"technician_name"`
	Status         string     `json:"status"` // received, diagnosing, working, testing, ready, completed, cancelled
	EstimatedCost  int64      `json:"estimated_cost"` // sen
	FinalCost      int64      `json:"final_cost"`     // sen
	IsPaid         bool       `json:"is_paid"`
	PaymentMethod  string     `json:"payment_method"`
	Notes          string     `json:"notes"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateServiceOrderReq struct {
	CustomerName   string `json:"customer_name"`
	CustomerPhone  string `json:"customer_phone"`
	UnitName       string `json:"unit_name"`
	UnitIdentifier string `json:"unit_identifier"`
	Complaint      string `json:"complaint"`
	TechnicianName string `json:"technician_name"`
	EstimatedCost  int64  `json:"estimated_cost"`
	FinalCost      int64  `json:"final_cost"`
	IsPaid         bool   `json:"is_paid"`
	PaymentMethod  string `json:"payment_method"`
	Notes          string `json:"notes"`
}

type UpdateServiceStatusReq struct {
	Status    string `json:"status"`
	FinalCost *int64 `json:"final_cost,omitempty"`
}

type PayServiceOrderReq struct {
	PaymentMethod string `json:"payment_method"`
}

func generateServiceOrderNo() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("SPK-%s-%04d", time.Now().Format("20060102"), n.Int64()+1000)
}

func isValidServiceStatus(status string) bool {
	switch status {
	case "received", "diagnosing", "working", "testing", "ready", "completed", "cancelled":
		return true
	default:
		return false
	}
}

// handleServiceOrders handles GET (list) and POST (create) on /api/umkm/service/orders
func handleServiceOrders(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listServiceOrders(w, r, tenantID)
	case http.MethodPost:
		createServiceOrder(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func listServiceOrders(w http.ResponseWriter, r *http.Request, tenantID string) {
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
		SELECT id, tenant_id, order_no, customer_name, customer_phone, unit_name,
		       unit_identifier, complaint, technician_name, status, estimated_cost,
		       final_cost, is_paid, payment_method, notes, completed_at, created_at, updated_at
		FROM service_orders
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		pattern := "%" + search + "%"
		query += fmt.Sprintf(" AND (order_no ILIKE $%d OR customer_name ILIKE $%d OR customer_phone ILIKE $%d OR unit_name ILIKE $%d OR unit_identifier ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, pattern)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(ctx, query, args...)
	if err != nil {
		slog.Error("listServiceOrders: query failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal memuat daftar order servis", nil)
		return
	}
	defer rows.Close()

	orders := make([]ServiceOrder, 0)
	for rows.Next() {
		var o ServiceOrder
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.OrderNo, &o.CustomerName, &o.CustomerPhone, &o.UnitName,
			&o.UnitIdentifier, &o.Complaint, &o.TechnicianName, &o.Status, &o.EstimatedCost,
			&o.FinalCost, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CompletedAt, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			slog.Error("listServiceOrders: row scan failed", "error", err)
			continue
		}
		orders = append(orders, o)
	}

	response.JSON(w, http.StatusOK, "Daftar order servis berhasil dimuat", orders)
}

func createServiceOrder(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req CreateServiceOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.UnitName = strings.TrimSpace(req.UnitName)
	req.Complaint = strings.TrimSpace(req.Complaint)

	if req.CustomerName == "" {
		response.Error(w, http.StatusBadRequest, "Nama pelanggan wajib diisi", nil)
		return
	}
	if req.CustomerPhone == "" {
		response.Error(w, http.StatusBadRequest, "Nomor WhatsApp pelanggan wajib diisi", nil)
		return
	}
	if req.UnitName == "" {
		response.Error(w, http.StatusBadRequest, "Nama / jenis unit wajib diisi", nil)
		return
	}
	if req.Complaint == "" {
		response.Error(w, http.StatusBadRequest, "Keluhan / deskripsi servis wajib diisi", nil)
		return
	}

	if req.PaymentMethod == "" {
		req.PaymentMethod = "unpaid"
	}
	if req.FinalCost == 0 && req.EstimatedCost > 0 {
		req.FinalCost = req.EstimatedCost
	}

	orderNo := generateServiceOrderNo()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var newOrder ServiceOrder
	err := DB.QueryRow(ctx, `
		INSERT INTO service_orders (
			tenant_id, order_no, customer_name, customer_phone, unit_name,
			unit_identifier, complaint, technician_name, status, estimated_cost,
			final_cost, is_paid, payment_method, notes
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, 'received', $9,
			$10, $11, $12, $13
		) RETURNING id, tenant_id, order_no, customer_name, customer_phone, unit_name,
		          unit_identifier, complaint, technician_name, status, estimated_cost,
		          final_cost, is_paid, payment_method, notes, completed_at, created_at, updated_at`,
		tenantID, orderNo, req.CustomerName, req.CustomerPhone, req.UnitName,
		req.UnitIdentifier, req.Complaint, req.TechnicianName, req.EstimatedCost,
		req.FinalCost, req.IsPaid, req.PaymentMethod, req.Notes,
	).Scan(
		&newOrder.ID, &newOrder.TenantID, &newOrder.OrderNo, &newOrder.CustomerName, &newOrder.CustomerPhone, &newOrder.UnitName,
		&newOrder.UnitIdentifier, &newOrder.Complaint, &newOrder.TechnicianName, &newOrder.Status, &newOrder.EstimatedCost,
		&newOrder.FinalCost, &newOrder.IsPaid, &newOrder.PaymentMethod, &newOrder.Notes, &newOrder.CompletedAt, &newOrder.CreatedAt, &newOrder.UpdatedAt,
	)

	if err != nil {
		slog.Error("createServiceOrder: insert failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal membuat order servis baru", nil)
		return
	}

	// If paid immediately, record kas journal
	if newOrder.IsPaid && newOrder.FinalCost > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Servis %s (%s)","price":%d,"quantity":1}]`, newOrder.UnitName, newOrder.OrderNo, newOrder.FinalCost/100)
		createPaymentJournal(ctx, tenantID, newOrder.OrderNo, float64(newOrder.FinalCost)/100, itemsDesc)
	}

	response.JSON(w, http.StatusCreated, "Order servis berhasil dibuat", newOrder)
}
