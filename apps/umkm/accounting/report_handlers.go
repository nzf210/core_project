package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"core_project/shared/sdk/response"
)

const (
	headerTenantRep    = "X-Tenant-ID"
	errMissingTenantRep = "Missing X-Tenant-ID"
	errDBRep           = response.DBError
	labelDateRep       = "02 Jan"
)


func handleIncomeStatement(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantRep)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if tenantID == "" || from == "" || to == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing parameters"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database connection error"})
		return
	}

	// Query Revenue and Expenses
	query := `
		SELECT c.type, c.name, SUM(l.credit - l.debit) as balance
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type IN ('revenue', 'expense') AND e.date >= $2 AND e.date <= $3
		GROUP BY c.type, c.name
	`
	rows, err := DB.Query(r.Context(), query, tenantID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBRep})
		return
	}
	defer rows.Close()

	var totalRevenue, totalExpense int64
	var details []map[string]any
	for rows.Next() {
		var typ, name string
		var balance int64
		if err := rows.Scan(&typ, &name, &balance); err == nil {
			// for expense, natural balance is debit, but query gives credit-debit
			if typ == "expense" {
				balance = -balance
				totalExpense += balance
			} else {
				totalRevenue += balance
			}
			details = append(details, map[string]any{"type": typ, "name": name, "balance": balance})
		}
	}

	netIncome := totalRevenue - totalExpense
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"net_income": netIncome,
			"revenue":    totalRevenue,
			"expense":    totalExpense,
			"details":    details,
		},
	})
}

func handleSalesChart(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantRep)
	period := r.URL.Query().Get("period")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRep})
		return
	}
	if period == "" {
		period = "week"
	}
	if period != "week" && period != "month" && period != "year" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid period: use week, month, or year"})
		return
	}
	labels, rowMap := querySalesChartData(r.Context(), tenantID, period)
	revenue, expense, profit := buildChartSeries(labels, rowMap)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"period":  period,
			"labels":  labels,
			"revenue": revenue,
			"expense": expense,
			"profit":  profit,
		},
	})
}

func querySalesChartData(ctx context.Context, tenantID, period string) ([]string, map[string][]int64) {
	query := chartQuery(period)
	labels := chartLabels(period)

	rows, err := DB.Query(ctx, query, tenantID)
	if err != nil {
		return labels, map[string][]int64{}
	}
	defer rows.Close()

	rowMap := map[string][]int64{}
	if period == "year" {
		for rows.Next() {
			var lbl string
			var rev, exp int64
			if rows.Scan(&lbl, &rev, &exp) == nil {
				rowMap[lbl] = []int64{rev, exp}
			}
		}
	} else {
		for rows.Next() {
			var day time.Time
			var rev, exp int64
			if rows.Scan(&day, &rev, &exp) == nil {
				rowMap[day.Format(labelDateRep)] = []int64{rev, exp}
			}
		}
	}
	return labels, rowMap
}

func chartQuery(period string) string {
	base := `
		SELECT
			DATE(e.created_at) AS day,
			COALESCE(SUM(CASE WHEN c.type = 'revenue' THEN l.credit - l.debit ELSE 0 END), 0) AS revenue,
			COALESCE(SUM(CASE WHEN c.type = 'expense' THEN l.debit - l.credit ELSE 0 END), 0) AS expense
		FROM journal_entries e
		JOIN journal_lines l ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND e.created_at >= `
	switch period {
	case "week":
		return base + `NOW() - INTERVAL '7 days' GROUP BY DATE(e.created_at) ORDER BY day`
	case "month":
		return base + `NOW() - INTERVAL '30 days' GROUP BY DATE(e.created_at) ORDER BY day`
	case "year":
		return `
			SELECT
				TO_CHAR(DATE_TRUNC('month', e.created_at), 'Mon YYYY') AS month_label,
				COALESCE(SUM(CASE WHEN c.type = 'revenue' THEN l.credit - l.debit ELSE 0 END), 0) AS revenue,
				COALESCE(SUM(CASE WHEN c.type = 'expense' THEN l.debit - l.credit ELSE 0 END), 0) AS expense
			FROM journal_entries e
			JOIN journal_lines l ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND e.created_at >= NOW() - INTERVAL '12 months'
			GROUP BY DATE_TRUNC('month', e.created_at)
			ORDER BY DATE_TRUNC('month', e.created_at)`
	default:
		return ""
	}
}

func chartLabels(period string) []string {
	var labels []string
	switch period {
	case "week":
		for i := 6; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, 0, -i).Format(labelDateRep))
		}
	case "month":
		for i := 29; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, 0, -i).Format(labelDateRep))
		}
	case "year":
		for i := 11; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, -i, 0).Format("Jan YYYY"))
		}
	}
	return labels
}

func buildChartSeries(labels []string, rowMap map[string][]int64) ([]int64, []int64, []int64) {
	var revenue, expense, profit []int64
	for _, lbl := range labels {
		rev, exp := int64(0), int64(0)
		if v, ok := rowMap[lbl]; ok {
			rev, exp = v[0], v[1]
		}
		revenue = append(revenue, rev)
		expense = append(expense, exp)
		p := rev - exp
		if p < 0 {
			p = 0
		}
		profit = append(profit, p)
	}
	return revenue, expense, profit
}

func handleTopProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	ctx := r.Context()
	query := `
		SELECT
			REGEXP_REPLACE(e.description, 'Penjualan\s*-\s*', '', 'i') AS product_name,
			SUM(l.credit - l.debit) AS revenue_rupiah,
			COUNT(*) AS transaction_count
		FROM journal_entries e
		JOIN journal_lines l ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1
		  AND c.type = 'revenue'
		  AND e.description ILIKE 'Penjualan%%'
		GROUP BY REGEXP_REPLACE(e.description, 'Penjualan\s*-\s*', '', 'i')
		ORDER BY revenue_rupiah DESC
		LIMIT $2`

	rows, err := DB.Query(ctx, query, tenantID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBRep})
		return
	}
	defer rows.Close()

	var products []map[string]any
	for rows.Next() {
		var name string
		var revenue, count int64
		if rows.Scan(&name, &revenue, &count) == nil && revenue > 0 {
			products = append(products, map[string]any{
				"name":               strings.TrimSpace(name),
				"revenue_rupiah":     revenue,
				"transaction_count":  count,
			})
		}
	}
	if products == nil {
		products = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: products})
}

func handleRecentTransactions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantRep)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRep})
		return
	}
	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	ctx := r.Context()
	query := `
		SELECT e.id, e.created_at, e.description,
			COALESCE(SUM(l.credit - l.debit), 0) AS amount_rupiah
		FROM journal_entries e
		JOIN journal_lines l ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'revenue'
		GROUP BY e.id
		ORDER BY e.created_at DESC
		LIMIT $2`

	rows, err := DB.Query(ctx, query, tenantID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBRep})
		return
	}
	defer rows.Close()

	type txResult struct {
		ID          string    `json:"id"`
		Date        time.Time `json:"date"`
		Description string    `json:"description"`
		AmountRupiah int64     `json:"amount_rupiah"`
	}
	var txs []txResult
	for rows.Next() {
		var tx txResult
		if rows.Scan(&tx.ID, &tx.Date, &tx.Description, &tx.AmountRupiah) == nil {
			txs = append(txs, tx)
		}
	}
	if txs == nil {
		txs = []txResult{}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: txs})
}