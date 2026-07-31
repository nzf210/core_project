package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"core_project/shared/sdk/response"
)

const (
	errDBErrorBal  = response.DBError
	dateFormatBal  = "2006-01-02"
)

func handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
	date := r.URL.Query().Get("date")

	if tenantID == "" || date == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing parameters"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true})
		return
	}

	query := `
		SELECT c.type, c.name, SUM(l.debit - l.credit) as balance
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type IN ('asset', 'liability', 'equity') AND e.date <= $2
		GROUP BY c.type, c.name
	`
	rows, err := DB.Query(r.Context(), query, tenantID, date)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBErrorBal})
		return
	}
	defer rows.Close()

	assets, liabilities, equity, details := aggregateBalanceSheet(rows)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"assets":      assets,
			"liabilities": liabilities,
			"equity":      equity,
			"details":     details,
			"is_balanced": assets == (liabilities + equity),
		},
	})
}

func aggregateBalanceSheet(rows pgx.Rows) (int64, int64, int64, []map[string]any) {
	var assets, liabilities, equity int64
	details := []map[string]any{}
	for rows.Next() {
		var typ, name string
		var balance int64
		if err := rows.Scan(&typ, &name, &balance); err == nil {
			if typ == "liability" || typ == "equity" {
				balance = -balance
				if typ == "liability" {
					liabilities += balance
				} else {
					equity += balance
				}
			} else {
				assets += balance
			}
			details = append(details, map[string]any{"type": typ, "name": name, "balance": balance})
		}
	}
	return assets, liabilities, equity, details
}

func handleCashFlow(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
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

	ctx := r.Context()
	totalInflow, totalOutflow, details := queryCashMovements(ctx, tenantID, from, to)
	operating, investing, financing := categorizeByActivity(ctx, tenantID, from, to)
	openingCash := getOpeningCash(ctx, tenantID, from)
	netCashFlow := totalInflow - totalOutflow

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"net_cash_flow": netCashFlow,
			"total_inflow":  totalInflow,
			"total_outflow": totalOutflow,
			"opening_cash":  openingCash,
			"closing_cash":  openingCash + netCashFlow,
			"details":       details,
			"activities": map[string]any{
				"operating": map[string]any{"inflow": operating.Inflow, "outflow": operating.Outflow, "net": operating.Inflow - operating.Outflow, "lines": operating.Lines},
				"investing": map[string]any{"inflow": investing.Inflow, "outflow": investing.Outflow, "net": investing.Inflow - investing.Outflow, "lines": investing.Lines},
				"financing": map[string]any{"inflow": financing.Inflow, "outflow": financing.Outflow, "net": financing.Inflow - financing.Outflow, "lines": financing.Lines},
			},
		},
	})
}

func queryCashMovements(ctx context.Context, tenantID, from, to string) (int64, int64, []map[string]any) {
	query := `
		SELECT e.id, e.date, e.description, l.debit, l.credit
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date >= $2 AND e.date <= $3
		ORDER BY e.date ASC
	`
	rows, err := DB.Query(ctx, query, tenantID, from, to)
	if err != nil {
		return 0, 0, []map[string]any{}
	}
	defer rows.Close()

	var totalInflow, totalOutflow int64
	details := []map[string]any{}
	for rows.Next() {
		var id, desc string
		var debit, credit int64
		var t time.Time
		if err := rows.Scan(&id, &t, &desc, &debit, &credit); err == nil {
			totalInflow += debit
			totalOutflow += credit
			details = append(details, map[string]any{"id": id, "date": t.Format(dateFormatBal), "description": desc, "inflow": debit, "outflow": credit})
		}
	}
	return totalInflow, totalOutflow, details
}

type categoryBucket struct {
	Inflow  int64
	Outflow int64
	Lines   []map[string]any
}

func categorizeByActivity(ctx context.Context, tenantID, from, to string) (*categoryBucket, *categoryBucket, *categoryBucket) {
	operating := &categoryBucket{Lines: []map[string]any{}}
	investing := &categoryBucket{Lines: []map[string]any{}}
	financing := &categoryBucket{Lines: []map[string]any{}}

	query := `
		SELECT e.id, e.date, e.description, e.reference, l.debit, l.credit, c.code, c.name, c.type
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date >= $2 AND e.date <= $3
		ORDER BY e.date ASC
	`
	rows, err := DB.Query(ctx, query, tenantID, from, to)
	if err != nil {
		return operating, investing, financing
	}
	defer rows.Close()

	for rows.Next() {
		var id, desc, ref, code, name, accType string
		var debit, credit int64
		var t time.Time
		if err := rows.Scan(&id, &t, &desc, &ref, &debit, &credit, &code, &name, &accType); err == nil {
			var cCode, cName, cType string
			DB.QueryRow(ctx, `
				SELECT c.code, c.name, c.type FROM journal_lines l
				JOIN chart_of_accounts c ON l.account_id = c.id
				WHERE l.entry_id = $1 AND l.account_id != (SELECT id FROM chart_of_accounts WHERE tenant_id = $2 AND code = $3 LIMIT 1) LIMIT 1
			`, id, tenantID, code).Scan(&cCode, &cName, &cType)

			bucket := classifyActivity(cCode, operating, investing, financing)
			bucket.Inflow += debit
			bucket.Outflow += credit
			bucket.Lines = append(bucket.Lines, map[string]any{
				"id": id, "date": t.Format(dateFormatBal), "description": desc,
				"counterpart_code": cCode, "counterpart_name": cName,
				"inflow": debit, "outflow": credit, "reference": ref,
			})
		}
	}
	return operating, investing, financing
}

func classifyActivity(cCode string, operating, investing, financing *categoryBucket) *categoryBucket {
	if cCode == "" {
		return operating
	}
	switch {
	case strings.HasPrefix(cCode, "1") && (cCode >= "150" && cCode <= "199"):
		return investing
	case strings.HasPrefix(cCode, "2"), strings.HasPrefix(cCode, "3"):
		return financing
	default:
		return operating
	}
}

func getOpeningCash(ctx context.Context, tenantID, from string) int64 {
	var openingCash int64
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit - l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date < $2
	`, tenantID, from).Scan(&openingCash)
	return openingCash
}

func handleExpenses(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		listExpenses(w, r, tenantID)
	case http.MethodPost:
		recordExpense(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func listExpenses(w http.ResponseWriter, r *http.Request, tenantID string) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	query := `
		SELECT e.id, e.date, e.description, c.name, l.debit
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'expense'
	`
	args := []any{tenantID}
	if from != "" && to != "" {
		query += " AND e.date >= $2 AND e.date <= $3 "
		args = append(args, from, to)
	}
	query += " ORDER BY e.date DESC"

	rows, err := DB.Query(r.Context(), query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBErrorBal})
		return
	}
	defer rows.Close()

	var totalExpense int64
	details := []map[string]any{}
	for rows.Next() {
		var id, desc, accountName string
		var debit int64
		var t time.Time
		if err := rows.Scan(&id, &t, &desc, &accountName, &debit); err == nil {
			totalExpense += debit
			details = append(details, map[string]any{"id": id, "date": t.Format(dateFormatBal), "description": desc, "account_name": accountName, "amount": debit})
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]any{"total_expense": totalExpense, "details": details}})
}

func recordExpense(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		Date        string `json:"date"`
		Description string `json:"description"`
		Amount      int64  `json:"amount"`
		ExpenseCOA  string `json:"expense_coa"`
		PaymentCOA  string `json:"payment_coa"`
		LineItems   []struct {
			Name   string `json:"name"`
			Amount int64  `json:"amount"`
		} `json:"line_items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	if req.Amount == 0 || req.ExpenseCOA == "" || req.PaymentCOA == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Amount, ExpenseCOA, and PaymentCOA required"})
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBErrorBal})
		return
	}
	defer tx.Rollback(ctx)

	var expID, payID string
	err = tx.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, req.ExpenseCOA).Scan(&expID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid Expense COA"})
		return
	}
	err = tx.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, req.PaymentCOA).Scan(&payID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid Payment COA"})
		return
	}

	metaBytes, _ := json.Marshal(map[string]any{"type": "expense", "line_items": req.LineItems})

	var entryID string
	err = tx.QueryRow(ctx, "INSERT INTO journal_entries (tenant_id, date, description, metadata) VALUES ($1, $2, $3, $4) RETURNING id",
		tenantID, req.Date, req.Description, string(metaBytes)).Scan(&entryID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert entry failed"})
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, $4)", entryID, expID, req.Amount, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert lines failed"})
		return
	}
	_, err = tx.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, $4)", entryID, payID, 0, req.Amount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert lines failed"})
		return
	}

	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Expense recorded", Data: map[string]string{"id": entryID}})
}
