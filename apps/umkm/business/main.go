package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool
var isTest bool

// validTierIDs is the allowlist of plan tier IDs accepted by the plan-upgrade
// handler. Mirrors the tiers defined in shared/sdk/auth/quota.go's plan_features
// table (the source of truth at runtime via GetTenantPlan/GetPlan).
var validTierIDs = map[string]bool{
	"inactive":   true,
	"lite":       true,
	"pro":        true,
	"enterprise": true,
	"ultimate":   true,
	"superadmin": true,
}

func initDB(cfg *config.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	DB = pool
	slog.Info("Connected to PostgreSQL")
	return nil
}

type BusinessType struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Icon                  string   `json:"icon"`
	DefaultModules        []string `json:"defaultModules"`
	DefaultDashboardWidgets []string `json:"defaultDashboardWidgets"`
}

type OnboardingRequest struct {
	BusinessType   string `json:"businessType"`
	BusinessName   string `json:"businessName"`
	BusinessAddress string `json:"businessAddress"`
	Plan           string `json:"plan"`
}

type DashboardWidget struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Title    string      `json:"title"`
	Module   string      `json:"module"`
	Config   interface{} `json:"config"`
	Position int         `json:"position"`
	Size     string      `json:"size"`
}

var widgetTemplates = map[string][]DashboardWidget{
	"umum": {
		{ID: "income_summary", Type: "metric", Title: "Pendapatan Bulan Ini", Module: "reports", Config: map[string]string{"metric": "income_current_month"}, Position: 0, Size: "medium"},
		{ID: "expense_summary", Type: "metric", Title: "Pengeluaran Bulan Ini", Module: "reports", Config: map[string]string{"metric": "expense_current_month"}, Position: 1, Size: "medium"},
		{ID: "profit_summary", Type: "metric", Title: "Laba Bersih", Module: "reports", Config: map[string]string{"metric": "profit_current_month"}, Position: 2, Size: "medium"},
		{ID: "recent_transactions", Type: "table", Title: "Transaksi Terbaru", Module: "transactions", Config: map[string]int{"limit": 10}, Position: 3, Size: "large"},
		{ID: "quick_actions", Type: "actions", Title: "Aksi Cepat", Module: "pos", Config: map[string][]string{"actions": {"new_transaction", "add_customer", "view_reports"}}, Position: 4, Size: "small"},
	},
	"warung": {
		{ID: "daily_sales", Type: "chart", Title: "Penjualan Hari Ini", Module: "transactions", Config: map[string]string{"chart": "bar", "period": "today"}, Position: 0, Size: "large"},
		{ID: "best_selling_items", Type: "list", Title: "Item Terlaris", Module: "inventory", Config: map[string]int{"limit": 5}, Position: 1, Size: "medium"},
		{ID: "stock_alert", Type: "alert", Title: "Stok Menipis", Module: "inventory", Config: map[string]int{"threshold": 5}, Position: 2, Size: "medium"},
		{ID: "income_summary", Type: "metric", Title: "Pendapatan Hari Ini", Module: "reports", Config: map[string]string{"metric": "income_today"}, Position: 3, Size: "medium"},
		{ID: "profit_margin", Type: "metric", Title: "Margin Keuntungan", Module: "reports", Config: map[string]string{"metric": "profit_margin"}, Position: 4, Size: "medium"},
		{ID: "expense_summary", Type: "metric", Title: "Pengeluaran", Module: "reports", Config: map[string]string{"metric": "expense_today"}, Position: 5, Size: "medium"},
	},
	"laundry": {
		{ID: "active_orders", Type: "list", Title: "Pesanan Aktif", Module: "order_tracking", Config: map[string]string{"status": "active"}, Position: 0, Size: "large"},
		{ID: "daily_revenue", Type: "metric", Title: "Pendapatan Hari Ini", Module: "reports", Config: map[string]string{"metric": "income_today"}, Position: 1, Size: "medium"},
		{ID: "package_breakdown", Type: "chart", Title: "Pesanan Per Paket", Module: "order_tracking", Config: map[string]string{"chart": "pie", "groupBy": "package"}, Position: 2, Size: "medium"},
		{ID: "customer_summary", Type: "metric", Title: "Total Pelanggan", Module: "customers", Config: map[string]string{"metric": "total_customers"}, Position: 3, Size: "medium"},
		{ID: "order_status_timeline", Type: "timeline", Title: "Status Pesanan", Module: "order_tracking", Config: map[string]string{"view": "timeline"}, Position: 4, Size: "medium"},
	},
	"industri_kreatif": {
		{ID: "active_projects", Type: "kanban", Title: "Proyek Aktif", Module: "project_tracking", Config: map[string]string{"view": "kanban"}, Position: 0, Size: "large"},
		{ID: "project_margin", Type: "metric", Title: "Margin Proyek", Module: "reports", Config: map[string]string{"metric": "avg_project_margin"}, Position: 1, Size: "medium"},
		{ID: "monthly_revenue", Type: "metric", Title: "Pendapatan Bulanan", Module: "reports", Config: map[string]string{"metric": "income_current_month"}, Position: 2, Size: "medium"},
		{ID: "material_spend", Type: "chart", Title: "Pengeluaran Bahan", Module: "material_costing", Config: map[string]string{"chart": "bar", "period": "month"}, Position: 3, Size: "medium"},
		{ID: "invoice_status", Type: "list", Title: "Status Invoice", Module: "invoice_generator", Config: map[string]string{"status": "pending"}, Position: 4, Size: "medium"},
	},
	"toko_online": {
		{ID: "order_volume", Type: "chart", Title: "Volume Pesanan", Module: "transactions", Config: map[string]string{"chart": "bar", "period": "7days"}, Position: 0, Size: "large"},
		{ID: "channel_breakdown", Type: "chart", Title: "Per Kanal Penjualan", Module: "pos", Config: map[string]string{"chart": "pie", "groupBy": "channel"}, Position: 1, Size: "medium"},
		{ID: "revenue_trend", Type: "chart", Title: "Tren Pendapatan", Module: "reports", Config: map[string]string{"chart": "line", "period": "30days"}, Position: 2, Size: "medium"},
		{ID: "top_products", Type: "list", Title: "Produk Teratas", Module: "inventory", Config: map[string]int{"limit": 5}, Position: 3, Size: "medium"},
		{ID: "pending_shipments", Type: "list", Title: "Pengiriman Pending", Module: "shipment_tracking", Config: map[string]string{"status": "pending"}, Position: 4, Size: "medium"},
	},
	"restoran": {
		{ID: "daily_revenue", Type: "metric", Title: "Pendapatan Hari Ini", Module: "reports", Config: map[string]string{"metric": "income_today"}, Position: 0, Size: "medium"},
		{ID: "popular_items", Type: "list", Title: "Menu Terpopuler", Module: "menu_management", Config: map[string]int{"limit": 5}, Position: 1, Size: "medium"},
		{ID: "cost_ratio", Type: "metric", Title: "Rasio Biaya", Module: "reports", Config: map[string]string{"metric": "cost_ratio"}, Position: 2, Size: "medium"},
		{ID: "peak_hours", Type: "chart", Title: "Jam Sibuk", Module: "transactions", Config: map[string]string{"chart": "bar", "groupBy": "hour"}, Position: 3, Size: "medium"},
		{ID: "table_turnover", Type: "metric", Title: "Table Turnover", Module: "table_management", Config: map[string]string{"metric": "avg_turnover"}, Position: 4, Size: "medium"},
	},
	"jasa": {
		{ID: "appointments_today", Type: "list", Title: "Janji Hari Ini", Module: "appointment_scheduling", Config: map[string]string{"date": "today"}, Position: 0, Size: "large"},
		{ID: "service_revenue", Type: "metric", Title: "Pendapatan Layanan", Module: "reports", Config: map[string]string{"metric": "income_today"}, Position: 1, Size: "medium"},
		{ID: "top_services", Type: "list", Title: "Layanan Terpopuler", Module: "service_catalog", Config: map[string]int{"limit": 5}, Position: 2, Size: "medium"},
		{ID: "customer_retention", Type: "metric", Title: "Retensi Pelanggan", Module: "customers", Config: map[string]string{"metric": "retention_rate"}, Position: 3, Size: "medium"},
		{ID: "staff_utilization", Type: "chart", Title: "Utilisasi Staff", Module: "staff_performance", Config: map[string]string{"chart": "bar"}, Position: 4, Size: "medium"},
	},
	"hotel": {
		{ID: "room_occupancy", Type: "metric", Title: "Tingkat Okupansi", Module: "room_management", Config: map[string]string{"metric": "occupancy_today"}, Position: 0, Size: "medium"},
		{ID: "daily_revenue", Type: "metric", Title: "Pendapatan Hari Ini", Module: "reports", Config: map[string]string{"metric": "income_today"}, Position: 1, Size: "medium"},
		{ID: "booking_status", Type: "list", Title: "Status Booking", Module: "booking", Config: map[string]int{"limit": 5}, Position: 2, Size: "large"},
		{ID: "income_summary", Type: "chart", Title: "Tren Pendapatan", Module: "reports", Config: map[string]string{"chart": "line"}, Position: 3, Size: "medium"},
		{ID: "guest_demographics", Type: "chart", Title: "Demografi Tamu", Module: "customers", Config: map[string]string{"chart": "pie"}, Position: 4, Size: "medium"},
	},
}

var moduleLabels = map[string]string{
	"transactions":        "Transaksi",
	"customers":           "Pelanggan",
	"reports":             "Laporan",
	"pos":                 "Kasir / POS",
	"inventory":           "Stok Barang",
	"supplier":            "Supplier",
	"best_seller":         "Analisis Terlaris",
	"order_tracking":      "Lacak Pesanan",
	"package_pricing":     "Paket Harga",
	"project_tracking":    "Lacak Proyek",
	"material_costing":    "Biaya Bahan",
	"invoice_generator":   "Pembuatan Invoice",
	"shipment_tracking":   "Lacak Pengiriman",
	"marketplace_sync":    "Sinkron Marketplace",
	"menu_management":     "Manajemen Menu",
	"ingredient_costing":  "Biaya Bahan Baku",
	"table_management":    "Manajemen Meja",
	"appointment_scheduling": "Janji Temu",
	"service_catalog":     "Katalog Layanan",
	"staff_performance":   "Performa Staff",
	"quick_actions":       "Aksi Cepat",
	"room_management":     "Manajemen Kamar",
	"booking":             "Booking / Reservasi",
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	if err := initDB(cfg); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("/business-types", handleGetBusinessTypes)
	mux.HandleFunc("/onboarding", handleOnboarding)
	mux.HandleFunc("/dashboard", handleGetDashboard)
	mux.HandleFunc("/plan", handleGetPlan)
	mux.HandleFunc("/upgrade", handleUpgradePlan)
	mux.HandleFunc("/stores", handleStoresCollection)
	mux.HandleFunc("/stores/", handleStoresItem)

	handler := auth.QuotaMiddleware(auth.Middleware(mux))
	port := "9001"

	slog.Info("UMKM Business Service starting", "port", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func handleGetBusinessTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := DB.Query(r.Context(),
		"SELECT id, name, description, icon, default_modules, default_dashboard_widgets FROM business_types ORDER BY id")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch business types", err)
		return
	}
	defer rows.Close()

	var types []BusinessType
	for rows.Next() {
		var t BusinessType
		var modulesJSON, widgetsJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Icon, &modulesJSON, &widgetsJSON); err != nil {
			continue
		}
		json.Unmarshal(modulesJSON, &t.DefaultModules)
		json.Unmarshal(widgetsJSON, &t.DefaultDashboardWidgets)
		types = append(types, t)
	}

	response.JSON(w, http.StatusOK, "Business types retrieved", types)
}

func handleOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := r.Context().Value(auth.TenantIDKey).(string)

	var req OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if req.BusinessType == "" {
		response.Error(w, http.StatusBadRequest, "businessType is required", nil)
		return
	}

	var btExists bool
	DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM business_types WHERE id=$1)", req.BusinessType).Scan(&btExists)
	if !btExists {
		response.Error(w, http.StatusBadRequest, "Invalid business type", nil)
		return
	}

	plan := req.Plan
	if plan == "" { plan = "free" }
	_, err := DB.Exec(r.Context(),
		`UPDATE tenants SET business_type=$1, business_name=$2, business_address=$3, plan=$4, onboarding_completed=true
		 WHERE id=$5`,
		req.BusinessType, req.BusinessName, req.BusinessAddress, plan, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update tenant", err)
		return
	}

	var bt BusinessType
	DB.QueryRow(r.Context(),
		"SELECT id, name, description, icon, default_modules, default_dashboard_widgets FROM business_types WHERE id=$1",
		req.BusinessType).Scan(&bt.ID, &bt.Name, &bt.Description, &bt.Icon, nil, nil)

	response.JSON(w, http.StatusOK, "Onboarding completed", map[string]interface{}{
		"tenantId":         tenantID,
		"businessType":     bt.ID,
		"businessName":     req.BusinessName,
		"initialDashboard": getDashboardForType(req.BusinessType),
		"modules":          getModuleListForType(req.BusinessType),
	})
}

func handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(auth.TenantIDKey).(string)

	var businessType string
	DB.QueryRow(r.Context(), "SELECT COALESCE(business_type, 'umum') FROM tenants WHERE id=$1", tenantID).Scan(&businessType)

	plan := auth.GetPlan(tenantID)

	result := map[string]interface{}{
		"businessType": businessType,
		"widgets":      getDashboardForType(businessType),
		"plan":         plan,
	}

	response.JSON(w, http.StatusOK, "Dashboard data", result)
}

func getDashboardForType(bt string) []DashboardWidget {
	if widgets, ok := widgetTemplates[bt]; ok {
		return widgets
	}
	return widgetTemplates["umum"]
}

func getModuleListForType(bt string) []map[string]interface{} {
	var modules []map[string]interface{}

	allModules := []string{"transactions", "customers", "reports", "pos"}
	var defaults []string

	if DB == nil {
		if isTest {
			return []map[string]interface{}{
				{"key": "transactions", "label": "Transaksi", "enabled": true},
				{"key": "customers", "label": "Pelanggan", "enabled": true},
				{"key": "reports", "label": "Laporan", "enabled": true},
				{"key": "pos", "label": "POS", "enabled": true},
			}
		}
		return modules
	}

	rows, _ := DB.Query(context.Background(), "SELECT default_modules FROM business_types WHERE id=$1", bt)
	if rows.Next() {
		var defaultModulesJSON []byte
		rows.Scan(&defaultModulesJSON)
		json.Unmarshal(defaultModulesJSON, &defaults)
	}
	rows.Close()

	if len(defaults) > 0 {
		allModules = defaults
	}

	for _, m := range allModules {
		label, ok := moduleLabels[m]
		if !ok {
			label = m
		}
		modules = append(modules, map[string]interface{}{
			"key":     m,
			"label":   label,
			"enabled": true,
		})
	}
	return modules
}

func handleGetPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value(auth.TenantIDKey).(string)
	plan := auth.GetPlan(tenantID)
	response.JSON(w, http.StatusOK, "Plan info", plan)
}

func handleUpgradePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := r.Context().Value(auth.TenantIDKey).(string)

	var req struct {
		PlanID string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if !validTierIDs[req.PlanID] {
		response.Error(w, http.StatusBadRequest, "Invalid plan", nil)
		return
	}

	auth.SetTenantPlan(r.Context(), tenantID, req.PlanID)

	response.JSON(w, http.StatusOK, "Plan upgraded", map[string]interface{}{
		"tenantId": tenantID,
		"plan":     req.PlanID,
	})
}

// ─────────────────────────────────────────────
// Multi-Store CRUD
// 1 owner bisa punya banyak toko (restoran + cafe, dll).
// Quota max_stores dibaca dari plan_features.feature_key='max_stores'
// sesuai plan_id tenant aktif. Superadmin bisa ubah via billing-service.
// ─────────────────────────────────────────────

type Store struct {
	ID            string `json:"id"`
	OwnerUserID   string `json:"ownerUserId"`
	TenantID      string `json:"tenantId"`
	Name          string `json:"name"`
	BusinessType  string `json:"businessType"`
	BusinessName  string `json:"businessName"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	LogoURL       string `json:"logoUrl"`
	IsActive      bool   `json:"isActive"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type StoreRequest struct {
	Name         string `json:"name"`
	BusinessType string `json:"businessType"`
	BusinessName string `json:"businessName"`
	Address      string `json:"address"`
	Phone        string `json:"phone"`
	LogoURL      string `json:"logoUrl"`
}

type StoreUpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	BusinessType *string `json:"businessType,omitempty"`
	Address      *string `json:"address,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	LogoURL      *string `json:"logoUrl,omitempty"`
	IsActive     *bool   `json:"isActive,omitempty"`
}

// getMaxStoresForTenant membaca quota dari plan_features DB.
// -1 = unlimited (untuk tier Business high-end), default ke 1 kalau tidak di-set.
func getMaxStoresForTenant(ctx context.Context, tenantID string) int {
	if DB == nil {
		if isTest {
			return 1 // default for tests
		}
		return 1
	}
	var planTier string
	if err := DB.QueryRow(ctx, "SELECT COALESCE(plan, 'lite') FROM tenants WHERE id=$1", tenantID).Scan(&planTier); err != nil {
		return 1
	}

	var featureValue string
	err := DB.QueryRow(ctx, `
		SELECT feature_value FROM plan_features
		WHERE plan_id = $1 AND feature_key = 'max_stores' AND is_enabled = true
	`, planTier).Scan(&featureValue)
	if err != nil {
		return 1
	}

	if featureValue == "unlimited" {
		return -1
	}

	var n int
	if _, err := fmt.Sscanf(featureValue, "%d", &n); err != nil {
		return 1
	}
	return n
}

func handleStoresCollection(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)

	switch r.Method {
	case http.MethodGet:
		listStores(w, r, tenantID, userID)
	case http.MethodPost:
		createStore(w, r, tenantID, userID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func handleStoresItem(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(auth.TenantIDKey).(string)
	userID, _ := r.Context().Value(auth.UserIDKey).(string)
	storeID := strings.TrimPrefix(r.URL.Path, "/api/umkm/stores/")
	if storeID == "" {
		response.Error(w, http.StatusBadRequest, "Store ID required", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getStore(w, r, storeID, tenantID, userID)
	case http.MethodPatch:
		updateStore(w, r, storeID, tenantID, userID)
	case http.MethodDelete:
		deleteStore(w, r, storeID, tenantID, userID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func listStores(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	rows, err := DB.Query(r.Context(), `
		SELECT id, owner_user_id, tenant_id, name, business_type, COALESCE(business_name, ''),
		       COALESCE(address, ''), COALESCE(phone, ''), COALESCE(logo_url, ''),
		       is_active, created_at, updated_at
		FROM stores
		WHERE owner_user_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC
	`, userID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list stores", err)
		return
	}
	defer rows.Close()

	stores := []Store{}
	for rows.Next() {
		var s Store
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.OwnerUserID, &s.TenantID, &s.Name, &s.BusinessType, &s.BusinessName,
			&s.Address, &s.Phone, &s.LogoURL, &s.IsActive, &createdAt, &updatedAt); err != nil {
			continue
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		s.UpdatedAt = updatedAt.Format(time.RFC3339)
		stores = append(stores, s)
	}

	maxStores := getMaxStoresForTenant(r.Context(), tenantID)
	response.JSON(w, http.StatusOK, "Stores retrieved", map[string]interface{}{
		"stores":     stores,
		"count":      len(stores),
		"max_stores": maxStores,
		"can_add":    maxStores == -1 || len(stores) < maxStores,
	})
}

func createStore(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	var req StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}
	if req.Name == "" || req.BusinessType == "" {
		response.Error(w, http.StatusBadRequest, "name and businessType are required", nil)
		return
	}

	var btExists bool
	if err := DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM business_types WHERE id=$1)", req.BusinessType).Scan(&btExists); err != nil || !btExists {
		response.Error(w, http.StatusBadRequest, "Invalid business type", nil)
		return
	}

	maxStores := getMaxStoresForTenant(r.Context(), tenantID)
	if maxStores != -1 {
		var currentCount int
		DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM stores WHERE owner_user_id=$1 AND tenant_id=$2", userID, tenantID).Scan(&currentCount)
		if currentCount >= maxStores {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Store quota exceeded for your plan. Upgrade to Business to add more stores.",
				"data": map[string]interface{}{
					"current_count": currentCount,
					"max_stores":    maxStores,
				},
			})
			return
		}
	}

	var newID string
	err := DB.QueryRow(r.Context(), `
		INSERT INTO stores (owner_user_id, tenant_id, name, business_type, business_name, address, phone, logo_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		RETURNING id
	`, userID, tenantID, req.Name, req.BusinessType, req.BusinessName, req.Address, req.Phone, req.LogoURL).Scan(&newID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create store", err)
		return
	}

	response.JSON(w, http.StatusCreated, "Store created", map[string]interface{}{
		"id":           newID,
		"ownerUserId":  userID,
		"tenantId":     tenantID,
		"name":         req.Name,
		"businessType": req.BusinessType,
	})
}

func getStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, userID string) {
	var s Store
	var createdAt, updatedAt time.Time
	err := DB.QueryRow(r.Context(), `
		SELECT id, owner_user_id, tenant_id, name, business_type, COALESCE(business_name, ''),
		       COALESCE(address, ''), COALESCE(phone, ''), COALESCE(logo_url, ''),
		       is_active, created_at, updated_at
		FROM stores
		WHERE id = $1 AND owner_user_id = $2 AND tenant_id = $3
	`, storeID, userID, tenantID).Scan(&s.ID, &s.OwnerUserID, &s.TenantID, &s.Name, &s.BusinessType, &s.BusinessName,
		&s.Address, &s.Phone, &s.LogoURL, &s.IsActive, &createdAt, &s.UpdatedAt)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Store not found", nil)
		return
	}
	s.CreatedAt = createdAt.Format(time.RFC3339)
	s.UpdatedAt = updatedAt.Format(time.RFC3339)
	response.JSON(w, http.StatusOK, "Store retrieved", s)
}

func updateStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, userID string) {
	var req StoreUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := []string{}
	args := []any{}
	idx := 1
	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.BusinessType != nil {
		var btExists bool
		DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM business_types WHERE id=$1)", *req.BusinessType).Scan(&btExists)
		if !btExists {
			response.Error(w, http.StatusBadRequest, "Invalid business type", nil)
			return
		}
		updates = append(updates, fmt.Sprintf("business_type = $%d", idx))
		args = append(args, *req.BusinessType)
		idx++
	}
	if req.Address != nil {
		updates = append(updates, fmt.Sprintf("address = $%d", idx))
		args = append(args, *req.Address)
		idx++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", idx))
		args = append(args, *req.Phone)
		idx++
	}
	if req.LogoURL != nil {
		updates = append(updates, fmt.Sprintf("logo_url = $%d", idx))
		args = append(args, *req.LogoURL)
		idx++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *req.IsActive)
		idx++
	}

	if len(updates) == 0 {
		response.Error(w, http.StatusBadRequest, "No fields to update", nil)
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, storeID, userID, tenantID)

	query := fmt.Sprintf(`
		UPDATE stores SET %s
		WHERE id = $%d AND owner_user_id = $%d AND tenant_id = $%d
	`, strings.Join(updates, ", "), idx, idx+1, idx+2)

	res, err := DB.Exec(r.Context(), query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update store", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Store not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, "Store updated", nil)
}

func deleteStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, userID string) {
	res, err := DB.Exec(r.Context(), `
		DELETE FROM stores WHERE id = $1 AND owner_user_id = $2 AND tenant_id = $3
	`, storeID, userID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete store", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Store not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, "Store deleted", nil)
}
