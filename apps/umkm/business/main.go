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
	mux.HandleFunc("/health", handleHealth)
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

