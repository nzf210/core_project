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

// RestaurantOrderItem represents a single line item in a restaurant order.
type RestaurantOrderItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"` // sen
	Notes    string `json:"notes,omitempty"`
}

// RestaurantOrder represents an order in a restaurant or F&B kitchen queue.
type RestaurantOrder struct {
	ID            string                `json:"id"`
	TenantID      string                `json:"tenant_id"`
	OrderNo       string                `json:"order_no"`
	TableNumber   string                `json:"table_number"`
	CustomerName  string                `json:"customer_name"`
	CustomerPhone string                `json:"customer_phone"`
	Status        string                `json:"status"` // pending, cooking, ready_to_serve, served, completed, cancelled
	Items         []RestaurantOrderItem `json:"items"`
	TotalAmount   int64                 `json:"total_amount"` // sen (1 IDR = 100 sen)
	IsPaid        bool                  `json:"is_paid"`
	PaymentMethod string                `json:"payment_method"`
	Notes         string                `json:"notes"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type CreateRestaurantOrderReq struct {
	TableNumber   string                `json:"table_number"`
	CustomerName  string                `json:"customer_name"`
	CustomerPhone string                `json:"customer_phone"`
	Items         []RestaurantOrderItem `json:"items"`
	TotalAmount   int64                 `json:"total_amount"`
	IsPaid        bool                  `json:"is_paid"`
	PaymentMethod string                `json:"payment_method"`
	Notes         string                `json:"notes"`
}

type UpdateRestaurantStatusReq struct {
	Status string `json:"status"`
}

type PayRestaurantOrderReq struct {
	PaymentMethod string `json:"payment_method"`
}

func generateRestaurantOrderNo() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("KOT-%s-%04d", time.Now().Format("20060102"), n.Int64()+1000)
}

func isValidRestaurantStatus(status string) bool {
	switch status {
	case "pending", "cooking", "ready_to_serve", "served", "completed", "cancelled":
		return true
	default:
		return false
	}
}

// handleRestaurantOrders handles GET (list) and POST (create) on /api/umkm/restaurant/orders
func handleRestaurantOrders(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listRestaurantOrders(w, r, tenantID)
	case http.MethodPost:
		createRestaurantOrder(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func listRestaurantOrders(w http.ResponseWriter, r *http.Request, tenantID string) {
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
		SELECT id, tenant_id, order_no, table_number, customer_name, customer_phone,
		       status, items, total_amount, is_paid, payment_method, notes, created_at, updated_at
		FROM restaurant_orders
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
		query += fmt.Sprintf(" AND (order_no ILIKE $%d OR customer_name ILIKE $%d OR table_number ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, pattern)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(ctx, query, args...)
	if err != nil {
		slog.Error("listRestaurantOrders: query failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal memuat daftar pesanan restoran", nil)
		return
	}
	defer rows.Close()

	orders := make([]RestaurantOrder, 0)
	for rows.Next() {
		var o RestaurantOrder
		var itemsRaw []byte
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.OrderNo, &o.TableNumber, &o.CustomerName, &o.CustomerPhone,
			&o.Status, &itemsRaw, &o.TotalAmount, &o.IsPaid, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			slog.Error("listRestaurantOrders: row scan failed", "error", err)
			continue
		}
		if len(itemsRaw) > 0 {
			_ = json.Unmarshal(itemsRaw, &o.Items)
		}
		if o.Items == nil {
			o.Items = []RestaurantOrderItem{}
		}
		orders = append(orders, o)
	}

	response.JSON(w, http.StatusOK, "Daftar pesanan restoran berhasil dimuat", orders)
}

func createRestaurantOrder(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req CreateRestaurantOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	req.CustomerName = strings.TrimSpace(req.CustomerName)
	if req.CustomerName == "" {
		req.CustomerName = "Tamu"
	}
	req.TableNumber = strings.TrimSpace(req.TableNumber)
	if req.TableNumber == "" {
		req.TableNumber = "Bawa Pulang"
	}

	if req.PaymentMethod == "" {
		req.PaymentMethod = "unpaid"
	}

	// Calculate total amount if items exist and total amount is zero
	if req.TotalAmount == 0 && len(req.Items) > 0 {
		for _, it := range req.Items {
			req.TotalAmount += it.Price * int64(it.Quantity)
		}
	}

	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		itemsJSON = []byte("[]")
	}

	orderNo := generateRestaurantOrderNo()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var newOrder RestaurantOrder
	var itemsRaw []byte
	err = DB.QueryRow(ctx, `
		INSERT INTO restaurant_orders (
			tenant_id, order_no, table_number, customer_name, customer_phone,
			status, items, total_amount, is_paid, payment_method, notes
		) VALUES (
			$1, $2, $3, $4, $5,
			'pending', $6, $7, $8, $9, $10
		) RETURNING id, tenant_id, order_no, table_number, customer_name, customer_phone,
		          status, items, total_amount, is_paid, payment_method, notes, created_at, updated_at`,
		tenantID, orderNo, req.TableNumber, req.CustomerName, req.CustomerPhone,
		itemsJSON, req.TotalAmount, req.IsPaid, req.PaymentMethod, req.Notes,
	).Scan(
		&newOrder.ID, &newOrder.TenantID, &newOrder.OrderNo, &newOrder.TableNumber, &newOrder.CustomerName, &newOrder.CustomerPhone,
		&newOrder.Status, &itemsRaw, &newOrder.TotalAmount, &newOrder.IsPaid, &newOrder.PaymentMethod, &newOrder.Notes, &newOrder.CreatedAt, &newOrder.UpdatedAt,
	)

	if err != nil {
		slog.Error("createRestaurantOrder: insert failed", "error", err, "tenant_id", tenantID)
		response.Error(w, http.StatusInternalServerError, "Gagal membuat pesanan restoran baru", nil)
		return
	}

	if len(itemsRaw) > 0 {
		_ = json.Unmarshal(itemsRaw, &newOrder.Items)
	}
	if newOrder.Items == nil {
		newOrder.Items = []RestaurantOrderItem{}
	}

	// If paid immediately at order time, record kas journal
	if newOrder.IsPaid && newOrder.TotalAmount > 0 {
		itemsDesc := fmt.Sprintf(`[{"name":"Pesanan Meja %s","price":%d,"quantity":1}]`, newOrder.TableNumber, newOrder.TotalAmount/100)
		createPaymentJournal(ctx, tenantID, newOrder.OrderNo, float64(newOrder.TotalAmount)/100, itemsDesc)
	}

	response.JSON(w, http.StatusCreated, "Pesanan restoran berhasil dibuat", newOrder)
}
