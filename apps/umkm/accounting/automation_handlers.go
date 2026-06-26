package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

func handleInternalAutomationsDue(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	rows, err := DB.Query(r.Context(), `
		SELECT a.id, a.tenant_id, a.type, a.name, a.cron_expression, a.config, COALESCE(a.target_wa, t.wa_number, '') as wa_number, t.name as tenant_name
		FROM tenant_automations a
		JOIN tenants t ON a.tenant_id = t.id
		WHERE a.enabled = true
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	now := time.Now()
	due := []map[string]any{}

	for rows.Next() {
		var id, tenantID, typ, name, cronExpr, waNumber, tenantName string
		var configJSON []byte
		if err := rows.Scan(&id, &tenantID, &typ, &name, &cronExpr, &configJSON, &waNumber, &tenantName); err == nil {
			if cronMatchesNow(cronExpr, now) {
				var cfg map[string]any
				json.Unmarshal(configJSON, &cfg)
				due = append(due, map[string]any{
					"automation_id": id,
					"tenant_id":     tenantID,
					"type":          typ,
					"name":          name,
					"wa_number":     waNumber,
					"tenant_name":   tenantName,
					"config":        cfg,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: due})
}

func handleInternalAutomationExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	var req struct {
		AutomationID string `json:"automation_id"`
		TenantID     string `json:"tenant_id"`
		Type         string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	ctx := r.Context()
	now := time.Now()
	today := now.Format(dateLayout)

	message := generateAutomationMessage(ctx, req.Type, req.TenantID, req.AutomationID, now, today)
	if message == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Unknown automation type: " + req.Type})
		return
	}

	DB.Exec(ctx, "UPDATE tenant_automations SET last_run_at = NOW() WHERE id = $1", req.AutomationID)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"message":       message,
			"automation_id": req.AutomationID,
			"tenant_id":     req.TenantID,
			"type":          req.Type,
		},
	})
}

func generateAutomationMessage(ctx context.Context, typ, tenantID, automationID string, now time.Time, today string) string {
	switch typ {
	case "daily_report":
		return generateDailyReport(ctx, tenantID, today)
	case "weekly_report":
		return generateWeeklyReport(ctx, tenantID, now, today)
	case "monthly_report":
		return generateMonthlyReport(ctx, tenantID, now, today)
	case "low_stock_alert":
		return generateLowStockAlert(ctx, tenantID, automationID, today)
	default:
		return ""
	}
}

func generateDailyReport(ctx context.Context, tenantID, today string) string {
	var totalRevenue int64
	var txCount int
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.credit - l.debit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date = $2
	`, tenantID, today).Scan(&totalRevenue)
	DB.QueryRow(ctx, "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date = $2", tenantID, today).Scan(&txCount)

	return fmt.Sprintf("📊 *LAPORAN HARIAN*\n📅 %s\n\n💰 Total Pendapatan: Rp %s\n🧾 Jumlah Transaksi: %d\n\n_Laporan otomatis dari SaaS UMKM WCH_",
		today, formatRupiah(totalRevenue), txCount)
}

func generateWeeklyReport(ctx context.Context, tenantID string, now time.Time, today string) string {
	weekAgo := now.AddDate(0, 0, -7).Format(dateLayout)
	var totalRevenue int64
	var txCount int
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.credit - l.debit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date >= $2 AND e.date <= $3
	`, tenantID, weekAgo, today).Scan(&totalRevenue)
	DB.QueryRow(ctx, "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date >= $2 AND date <= $3", tenantID, weekAgo, today).Scan(&txCount)

	return fmt.Sprintf("📊 *LAPORAN MINGGUAN*\n📅 %s s/d %s\n\n💰 Total Pendapatan: Rp %s\n🧾 Jumlah Transaksi: %d\n\n_Laporan otomatis dari SaaS UMKM WCH_",
		weekAgo, today, formatRupiah(totalRevenue), txCount)
}

func generateMonthlyReport(ctx context.Context, tenantID string, now time.Time, today string) string {
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format(dateLayout)
	var totalRevenue, totalExpense int64
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN c.type='revenue' THEN l.credit-l.debit ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN c.type='expense' THEN l.debit-l.credit ELSE 0 END), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type IN ('revenue', 'expense') AND e.date >= $2 AND e.date <= $3
	`, tenantID, firstOfMonth, today).Scan(&totalRevenue, &totalExpense)

	netIncome := totalRevenue - totalExpense
	return fmt.Sprintf("📊 *LAPORAN BULANAN*\n📅 %s s/d %s\n\n💰 Pendapatan: Rp %s\n💸 Beban: Rp %s\n📈 Laba Bersih: Rp %s\n\n_Laporan otomatis dari SaaS UMKM WCH_",
		firstOfMonth, today, formatRupiah(totalRevenue), formatRupiah(totalExpense), formatRupiah(netIncome))
}

func generateLowStockAlert(ctx context.Context, tenantID, automationID, today string) string {
	threshold := 5
	var configJSON []byte
	DB.QueryRow(ctx, "SELECT config FROM tenant_automations WHERE id = $1", automationID).Scan(&configJSON)
	if configJSON != nil {
		var cfg map[string]any
		json.Unmarshal(configJSON, &cfg)
		if t, ok := cfg["threshold"]; ok {
			if tf, ok := t.(float64); ok {
				threshold = int(tf)
			}
		}
	}

	rows, err := DB.Query(ctx, "SELECT name, COALESCE(stock_quantity, 0) FROM products WHERE tenant_id = $1 AND COALESCE(stock_quantity, 0) <= $2 ORDER BY stock_quantity ASC LIMIT 20", tenantID, threshold)
	if err != nil {
		return ""
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		var pName string
		var stock int
		if rows.Scan(&pName, &stock) == nil {
			items = append(items, fmt.Sprintf("• %s (stok: %d)", pName, stock))
		}
	}
	if len(items) > 0 {
		return fmt.Sprintf("⚠️ *ALERT STOK RENDAH*\n📅 %s\n\nProduk dengan stok ≤ %d:\n%s\n\n_Segera restok agar tidak kehabisan!_",
			today, threshold, strings.Join(items, "\n"))
	}
	return fmt.Sprintf("✅ *STOK AMAN*\n📅 %s\n\nSemua produk memiliki stok di atas %d. Tidak ada yang perlu restok saat ini.", today, threshold)
}
