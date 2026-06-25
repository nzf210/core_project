package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)


func handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
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
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var assets, liabilities, equity int64
	var details []map[string]interface{}
	for rows.Next() {
		var typ, name string
		var balance int64
		if err := rows.Scan(&typ, &name, &balance); err == nil {
			if typ == "liability" || typ == "equity" {
				balance = -balance // Natural balance is credit
				if typ == "liability" {
					liabilities += balance
				}
				if typ == "equity" {
					equity += balance
				}
			} else {
				assets += balance
			}
			details = append(details, map[string]interface{}{"type": typ, "name": name, "balance": balance})
		}
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"assets":      assets,
			"liabilities": liabilities,
			"equity":      equity,
			"details":     details,
			"is_balanced": assets == (liabilities + equity),
		},
	})
}

func handleCashFlow(w http.ResponseWriter, r *http.Request) {
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

	// Simplistic Cash Flow: approximated based on transactions affecting '100' (Kas) or '101' (Bank)
	query := `
		SELECT e.id, e.date, e.description, l.debit, l.credit
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date >= $2 AND e.date <= $3
		ORDER BY e.date ASC
	`
	rows, err := DB.Query(r.Context(), query, tenantID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var totalInflow, totalOutflow int64
	var details []map[string]interface{}
	for rows.Next() {
		var id, desc string
		var debit, credit int64
		var t time.Time
		if err := rows.Scan(&id, &t, &desc, &debit, &credit); err == nil {
			// For cash accounts (asset), debit is inflow, credit is outflow
			totalInflow += debit
			totalOutflow += credit
			details = append(details, map[string]interface{}{
				"id": id, "date": t.Format("2006-01-02"), "description": desc, "inflow": debit, "outflow": credit,
			})
		}
	}

	netCashFlow := totalInflow - totalOutflow

	// F021: per-counterpart breakdown by SAK-EMKM activity category
	type categoryBucket struct {
		Inflow  int64
		Outflow int64
		Lines   []map[string]interface{}
	}
	operating := &categoryBucket{}
	investing := &categoryBucket{}
	financing := &categoryBucket{}

	// Re-query to get the counterpart (non-cash) account per cash line
	counterpartQuery := `
		SELECT e.id, e.date, e.description, e.reference,
		       l.debit, l.credit, c.code, c.name, c.type
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101')
		  AND e.date >= $2 AND e.date <= $3
		ORDER BY e.date ASC
	`
	rows2, err := DB.Query(r.Context(), counterpartQuery, tenantID, from, to)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, desc, ref, code, name, accType string
			var debit, credit int64
			var t time.Time
			if err := rows2.Scan(&id, &t, &desc, &ref, &debit, &credit, &code, &name, &accType); err == nil {
				// Get counterpart account (the other line in the same entry)
				var cCode, cName, cType string
				DB.QueryRow(r.Context(), `
					SELECT c.code, c.name, c.type FROM journal_lines l
					JOIN chart_of_accounts c ON l.account_id = c.id
					WHERE l.entry_id = $1 AND l.account_id != (
						SELECT id FROM chart_of_accounts WHERE tenant_id = $2 AND code = $3 LIMIT 1
					) LIMIT 1
				`, id, tenantID, code).Scan(&cCode, &cName, &cType)

				bucket := operating
				// Classify by counterpart account code prefix
				if cCode != "" {
					switch {
					case strings.HasPrefix(cCode, "1") && (cCode >= "150" && cCode <= "199"):
						bucket = investing
					case strings.HasPrefix(cCode, "2"):
						bucket = financing
					case strings.HasPrefix(cCode, "3"):
						bucket = financing
					default:
						bucket = operating
					}
				}
				bucket.Inflow += debit
				bucket.Outflow += credit
				bucket.Lines = append(bucket.Lines, map[string]interface{}{
					"id": id, "date": t.Format("2006-01-02"), "description": desc,
					"counterpart_code": cCode, "counterpart_name": cName,
					"inflow": debit, "outflow": credit, "reference": ref,
				})
			}
		}
	}

	// Get opening cash balance (sum of all cash account movements before `from`)
	var openingCash int64
	DB.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(l.debit - l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date < $2
	`, tenantID, from).Scan(&openingCash)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"net_cash_flow": netCashFlow,
			"total_inflow":  totalInflow,
			"total_outflow": totalOutflow,
			"opening_cash":  openingCash,
			"closing_cash":  openingCash + netCashFlow,
			"details":       details,
			"activities": map[string]interface{}{
				"operating": map[string]interface{}{
					"inflow": operating.Inflow, "outflow": operating.Outflow,
					"net":   operating.Inflow - operating.Outflow,
					"lines": operating.Lines,
				},
				"investing": map[string]interface{}{
					"inflow": investing.Inflow, "outflow": investing.Outflow,
					"net":   investing.Inflow - investing.Outflow,
					"lines": investing.Lines,
				},
				"financing": map[string]interface{}{
					"inflow": financing.Inflow, "outflow": financing.Outflow,
					"net":   financing.Inflow - financing.Outflow,
					"lines": financing.Lines,
				},
			},
		},
	})
}

func handleExpenses(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		query := `
			SELECT e.id, e.date, e.description, c.name, l.debit
			FROM journal_lines l
			JOIN journal_entries e ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND c.type = 'expense'
		`
		args := []interface{}{tenantID}
		if from != "" && to != "" {
			query += " AND e.date >= $2 AND e.date <= $3 "
			args = append(args, from, to)
		}
		query += " ORDER BY e.date DESC"

		rows, err := DB.Query(r.Context(), query, args...)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var totalExpense int64
		var details []map[string]interface{}
		for rows.Next() {
			var id, desc, accountName string
			var debit int64
			var t time.Time
			if err := rows.Scan(&id, &t, &desc, &accountName, &debit); err == nil {
				totalExpense += debit
				details = append(details, map[string]interface{}{
					"id": id, "date": t.Format("2006-01-02"), "description": desc, "account_name": accountName, "amount": debit,
				})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"total_expense": totalExpense,
			"details":       details,
		}})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Date        string `json:"date"` // YYYY-MM-DD
			Description string `json:"description"`
			Amount      int64  `json:"amount"`
			ExpenseCOA  string `json:"expense_coa"` // e.g. "500"
			PaymentCOA  string `json:"payment_coa"` // e.g. "100" (Kas) or "101" (Bank)
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
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
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

		metaBytes, _ := json.Marshal(map[string]interface{}{
			"type":       "expense",
			"line_items": req.LineItems,
		})

		var entryID string
		err = tx.QueryRow(ctx,
			"INSERT INTO journal_entries (tenant_id, date, description, metadata) VALUES ($1, $2, $3, $4) RETURNING id",
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
		return
	}
}