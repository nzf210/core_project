package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)


func handleIncomeStatement(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
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
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var totalRevenue, totalExpense int64
	var details []map[string]interface{}
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
			details = append(details, map[string]interface{}{"type": typ, "name": name, "balance": balance})
		}
	}

	netIncome := totalRevenue - totalExpense
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"net_income": netIncome,
			"revenue":    totalRevenue,
			"expense":    totalExpense,
			"details":    details,
		},
	})
}

func handleSalesChart(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	period := r.URL.Query().Get("period")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if period == "" {
		period = "week"
	}

	ctx := r.Context()
	var query string
	labels := []string{}

	switch period {
	case "week":
		query = `
			SELECT
				DATE(e.created_at) AS day,
				COALESCE(SUM(CASE WHEN c.type = 'revenue' THEN l.credit - l.debit ELSE 0 END), 0) AS revenue,
				COALESCE(SUM(CASE WHEN c.type = 'expense' THEN l.debit - l.credit ELSE 0 END), 0) AS expense
			FROM journal_entries e
			JOIN journal_lines l ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND e.created_at >= NOW() - INTERVAL '7 days'
			GROUP BY DATE(e.created_at)
			ORDER BY day`
		for i := 6; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, 0, -i).Format("02 Jan"))
		}
	case "month":
		query = `
			SELECT
				DATE(e.created_at) AS day,
				COALESCE(SUM(CASE WHEN c.type = 'revenue' THEN l.credit - l.debit ELSE 0 END), 0) AS revenue,
				COALESCE(SUM(CASE WHEN c.type = 'expense' THEN l.debit - l.credit ELSE 0 END), 0) AS expense
			FROM journal_entries e
			JOIN journal_lines l ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND e.created_at >= NOW() - INTERVAL '30 days'
			GROUP BY DATE(e.created_at)
			ORDER BY day`
		for i := 29; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, 0, -i).Format("02 Jan"))
		}
	case "year":
		query = `
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
		for i := 11; i >= 0; i-- {
			labels = append(labels, time.Now().AddDate(0, -i, 0).Format("Jan YYYY"))
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid period: use week, month, or year"})
		return
	}

	rows, err := DB.Query(ctx, query, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
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
				rowMap[day.Format("02 Jan")] = []int64{rev, exp}
			}
		}
	}

	revenue := []int64{}
	expense := []int64{}
	for _, lbl := range labels {
		if v, ok := rowMap[lbl]; ok {
			revenue = append(revenue, v[0])
			expense = append(expense, v[1])
		} else {
			revenue = append(revenue, 0)
			expense = append(expense, 0)
		}
	}

	profit := []int64{}
	for i := range revenue {
		p := revenue[i] - expense[i]
		if p < 0 {
			p = 0
		}
		profit = append(profit, p)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"period":  period,
			"labels":  labels,
			"revenue": revenue,
			"expense": expense,
			"profit":  profit,
		},
	})
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
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var name string
		var revenue, count int64
		if rows.Scan(&name, &revenue, &count) == nil && revenue > 0 {
			products = append(products, map[string]interface{}{
				"name":                strings.TrimSpace(name),
				"revenue_rupiah":      revenue,
				"transaction_count":   count,
			})
		}
	}
	if products == nil {
		products = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: products})
}

func handleRecentTransactions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
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
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
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