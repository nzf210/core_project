package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

type BusinessType struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Icon                  string   `json:"icon"`
	DefaultModules        []string `json:"defaultModules"`
	DefaultDashboardWidgets []string `json:"defaultDashboardWidgets"`
}

type OnboardingRequest struct {
	BusinessType string `json:"businessType"`
	BusinessName string `json:"businessName"`
	BusinessAddress string `json:"businessAddress"`
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
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.LoadConfig(".env")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	DB = pool
	defer DB.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/umkm/business-types", handleGetBusinessTypes)
	mux.HandleFunc("/api/umkm/onboarding", handleOnboarding)
	mux.HandleFunc("/api/umkm/dashboard", handleGetDashboard)
	mux.HandleFunc("/api/umkm/plan", handleGetPlan)
	mux.HandleFunc("/api/umkm/upgrade", handleUpgradePlan)

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

	_, err := DB.Exec(r.Context(),
		`UPDATE tenants SET business_type=$1, business_name=$2, business_address=$3, onboarding_completed=true 
		 WHERE id=$4`,
		req.BusinessType, req.BusinessName, req.BusinessAddress, tenantID)
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

	if _, ok := auth.Plans[req.PlanID]; !ok {
		response.Error(w, http.StatusBadRequest, "Invalid plan", nil)
		return
	}

	auth.SetTenantPlan(r.Context(), tenantID, req.PlanID)

	response.JSON(w, http.StatusOK, "Plan upgraded", map[string]interface{}{
		"tenantId": tenantID,
		"plan":     req.PlanID,
	})
}
