package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/webhook"
	"golang.org/x/crypto/bcrypt"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	// Run database migrations automatically
	if err := runMigrations(DB); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("/accounts", handleAccounts)
	mux.HandleFunc("/transactions", handleTransactions)
	mux.HandleFunc("/reports/income-statement", handleIncomeStatement)
	mux.HandleFunc("/reports/balance-sheet", handleBalanceSheet)
	mux.HandleFunc("/reports/cash-flow", handleCashFlow)
	mux.HandleFunc("/expenses", handleExpenses)
	mux.HandleFunc("/seed", handleSeed) // Helper endpoint to seed a tenant
	mux.HandleFunc("/admin/tenants", handleAdminTenants)
	mux.HandleFunc("/settings", handleSettings)
	mux.HandleFunc("/products", handleProducts)
	mux.HandleFunc("/products/export", handleProductsExport)
	mux.HandleFunc("/products/import", handleProductsImport)
	mux.HandleFunc("/checkout", handleCheckout)
	mux.HandleFunc("/webhook/store-payment", handleStorePaymentWebhook)
	mux.HandleFunc("/transactions/status", handleTransactionStatus)
	mux.HandleFunc("/webhook/payment", handlePaymentWebhook)
	mux.HandleFunc("/ocr", handleOCR)
	mux.HandleFunc("/faqs", handleFaqs)
	mux.HandleFunc("/faqs/generate", handleFaqsGenerate)
	mux.HandleFunc("/forwarders", handleForwarders)
	mux.HandleFunc("/internal/scheduled-reports", handleInternalScheduledReports)
	mux.HandleFunc("/internal/reports/summary", handleInternalReportsSummary)
	mux.HandleFunc("/automations", handleAutomations)
	mux.HandleFunc("/internal/automations/due", handleInternalAutomationsDue)
	mux.HandleFunc("/internal/automations/execute", handleInternalAutomationExecute)

	// ── Chatbot / N8N Hybrid Endpoints ──────────────────────────────────
	mux.HandleFunc("/internal/tenant/{tenant_id}/chatbot-config", handleInternalChatbotConfig)
	mux.HandleFunc("/internal/tenant/{tenant_id}/rag/search", handleInternalRAGSearch)
	mux.HandleFunc("/internal/conversation/log", handleInternalConversationLog)
	mux.HandleFunc("/internal/escalation/log", handleInternalEscalationLog)
	mux.HandleFunc("/internal/tenant/{tenant_id}/faqs", handleInternalFAQs)
	mux.HandleFunc("/internal/tenant/{tenant_id}/products", handleInternalProducts)
	mux.HandleFunc("/internal/tenant/{tenant_id}/rag/single", handleInternalRAGSingle)

	server := &http.Server{
		Addr:    ":8201", // UMKM port
		Handler: corsMiddleware(loggingMiddleware(mux)),
	}

	slog.Info("UMKM Accounting Engine listening", "port", 8201)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

func CRC16CCITT(data []byte) string {
	crc := 0xFFFF
	for _, b := range data {
		crc ^= (int(b) << 8)
		for i := 0; i < 8; i++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc = crc << 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc&0xFFFF)
}

func generateDynamicQRIS(staticQRIS string, amount float64) string {
	if !strings.Contains(staticQRIS, "6304") {
		return staticQRIS
	}
	// Split at 6304
	parts := strings.Split(staticQRIS, "6304")
	base := parts[0]
	
	// Remove existing tag 54 if present (simplified: assume it's static and has no 54, but if we need, we could parse it)
	// Add amount
	amtStr := fmt.Sprintf("%.0f", amount)
	amtTag := fmt.Sprintf("54%02d%s", len(amtStr), amtStr)
	
	// Tag 01 (Point of Initiation Method) might be 010211 (static). Change to 010212 (dynamic)
	newBase := strings.Replace(base, "010211", "010212", 1)
	newBase = newBase + amtTag + "6304"
	
	crc := CRC16CCITT([]byte(newBase))
	return newBase + crc
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "latency_ms", time.Since(start).Milliseconds())
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// ==========================================
// Handlers
// ==========================================

// Seed SAK-EMKM standard accounts for a tenant
func handleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	// Basic SAK-EMKM
	accounts := []struct {
		Code, Name, Type string
	}{
		{"100", "Kas", "asset"},
		{"101", "Bank / QRIS", "asset"},
		{"110", "Piutang Usaha", "asset"},
		{"120", "Persediaan", "asset"},
		{"200", "Hutang Usaha", "liability"},
		{"300", "Modal", "equity"},
		{"400", "Pendapatan Usaha", "revenue"},
		{"500", "Beban Operasional", "expense"},
	}

	ctx := context.Background()
	for _, acc := range accounts {
		_, err := DB.Exec(ctx, 
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type) VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, code) DO NOTHING",
			tenantID, acc.Code, acc.Name, acc.Type,
		)
		if err != nil {
			slog.Error("Failed to seed account", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Seeded successfully"})
}

type AccountReq struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID string `json:"parent_id"`
}

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if r.Method == http.MethodGet {
		// List accounts
		rows, err := DB.Query(r.Context(), "SELECT id, code, name, type FROM chart_of_accounts WHERE tenant_id = $1", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var id, code, name, typ string
			if err := rows.Scan(&id, &code, &name, &typ); err == nil {
				results = append(results, map[string]interface{}{
					"id": id, "code": code, "name": name, "type": typ,
				})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
		return
	}

	if r.Method == http.MethodPost {
		var req AccountReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		var parent interface{}
		if req.ParentID != "" {
			parent = req.ParentID
		}

		var id string
		err := DB.QueryRow(r.Context(),
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type, parent_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			tenantID, req.Code, req.Name, req.Type, parent).Scan(&id)
		
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": id}})
		return
	}
}

type TransactionReq struct {
	Date        string `json:"date"` // YYYY-MM-DD
	Description string `json:"description"`
	Reference   string `json:"reference"`
	Lines       []struct {
		AccountID string `json:"account_id"`
		Debit     int64  `json:"debit"`
		Credit    int64  `json:"credit"`
	} `json:"lines"`
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handleGetTransactions(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var req TransactionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	// Validate Double-Entry
	var totalDebit, totalCredit int64
	for _, l := range req.Lines {
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if totalDebit != totalCredit {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Debit and Credit must be equal"})
		return
	}
	if totalDebit == 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Transaction must have value"})
		return
	}

	if DB == nil {
		if isTest {
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Transaction recorded", Data: map[string]string{"id": "mock-entry-id"}})
			return
		}
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database connection error"})
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer tx.Rollback(ctx)

	var entryID string
	err = tx.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference) VALUES ($1, $2, $3, $4) RETURNING id",
		tenantID, req.Date, req.Description, req.Reference).Scan(&entryID)
	
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert entry failed"})
		return
	}

	for _, l := range req.Lines {
		_, err = tx.Exec(ctx,
			"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, $4)",
			entryID, l.AccountID, l.Debit, l.Credit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert lines failed"})
			return
		}
	}

	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Transaction recorded", Data: map[string]string{"id": entryID}})
}

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
			"revenue": totalRevenue,
			"expense": totalExpense,
			"details": details,
		},
	})
}

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
				if typ == "liability" { liabilities += balance }
				if typ == "equity" { equity += balance }
			} else {
				assets += balance
			}
			details = append(details, map[string]interface{}{"type": typ, "name": name, "balance": balance})
		}
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Data: map[string]interface{}{
			"assets": assets,
			"liabilities": liabilities,
			"equity": equity,
			"details": details,
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
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Data: map[string]interface{}{
			"net_cash_flow": netCashFlow,
			"total_inflow": totalInflow,
			"total_outflow": totalOutflow,
			"details": details,
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
			"details": details,
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
			"type": "expense",
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

func handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{}})
		return
	}

	query := `
		SELECT e.id, e.date, e.description, e.reference, e.metadata, l.account_id, c.name, l.debit, l.credit
		FROM journal_entries e
		JOIN journal_lines l ON e.id = l.entry_id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1
		ORDER BY e.date DESC, e.created_at DESC
		LIMIT 50
	`
	rows, err := DB.Query(r.Context(), query, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	type Line struct {
		AccountID   string  `json:"account_id"`
		AccountName string  `json:"account_name"`
		Debit       float64 `json:"debit"`
		Credit      float64 `json:"credit"`
	}
	type Entry struct {
		ID          string                 `json:"id"`
		Date        string                 `json:"date"`
		Description string                 `json:"description"`
		Reference   string                 `json:"reference"`
		Metadata    map[string]interface{} `json:"metadata"`
		Lines       []Line                 `json:"lines"`
	}

	entriesMap := make(map[string]*Entry)
	var order []string

	for rows.Next() {
		var id, desc, ref, accID, accName string
		var debit, credit float64
		var dateRaw time.Time
		var metaRaw []byte
		if err := rows.Scan(&id, &dateRaw, &desc, &ref, &metaRaw, &accID, &accName, &debit, &credit); err == nil {
			date := dateRaw.Format("2006-01-02")
			
			var meta map[string]interface{}
			if metaRaw != nil {
				json.Unmarshal(metaRaw, &meta)
			}

			if _, exists := entriesMap[id]; !exists {
				entriesMap[id] = &Entry{ID: id, Date: date, Description: desc, Reference: ref, Metadata: meta, Lines: []Line{}}
				order = append(order, id)
			}
			entriesMap[id].Lines = append(entriesMap[id].Lines, Line{
				AccountID: accID, AccountName: accName, Debit: debit, Credit: credit,
			})
		}
	}

	var result []Entry
	for _, id := range order {
		result = append(result, *entriesMap[id])
	}
	if result == nil {
		result = []Entry{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

func handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createAdminTenant(w, r)
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" || DB == nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
			return
		}
		
		ctx := r.Context()
		tx, err := DB.Begin(ctx)
		if err != nil {
			slog.Error("Failed to start transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
			return
		}
		defer tx.Rollback(ctx)

		// 1. Manual cascade for journal_lines because it references chart_of_accounts without ON DELETE CASCADE
		_, err = tx.Exec(ctx, "DELETE FROM journal_lines WHERE entry_id IN (SELECT id FROM journal_entries WHERE tenant_id = $1)", id)
		if err != nil {
			slog.Error("Failed to delete journal_lines", "error", err, "tenant_id", id)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to clean up tenant journal data"})
			return
		}

		// 2. Delete tenant (will cascade to journal_entries, chart_of_accounts, users, etc.)
		_, err = tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
		if err != nil {
			slog.Error("Failed to delete tenant row", "error", err, "tenant_id", id)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to delete tenant"})
			return
		}

		tx.Commit(ctx)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant deleted"})
		return
	}

	if r.Method == http.MethodPut {
		if DB == nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
			return
		}
		var req struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Plan        string `json:"plan"`
			Username    string `json:"username"`
			Email       string `json:"email"`
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}
		
		ctx := r.Context()
		tx, err := DB.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
			return
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "UPDATE tenants SET name = $1, plan = $2, updated_at = NOW() WHERE id = $3", req.Name, req.Plan, req.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update tenant"})
			return
		}
		
		_, err = tx.Exec(ctx, "UPDATE users SET username = $1, email = $2, phone_number = $3, updated_at = NOW() WHERE tenant_id = $4", req.Username, req.Email, req.PhoneNumber, req.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update user"})
			return
		}
		
		tx.Commit(ctx)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant updated"})
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{}})
		return
	}

	rows, err := DB.Query(r.Context(), `
		SELECT t.id, t.name, t.plan, t.created_at, 
		       COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.phone_number, '')
		FROM tenants t
		LEFT JOIN (
			SELECT DISTINCT ON (tenant_id) tenant_id, username, email, phone_number 
			FROM users ORDER BY tenant_id, created_at ASC
		) u ON u.tenant_id = t.id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id, name, plan, username, email, phone string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &plan, &createdAt, &username, &email, &phone); err == nil {
			result = append(result, map[string]interface{}{
				"id": id,
				"name": name,
				"plan": plan,
				"username": username,
				"email": email,
				"phone_number": phone,
				"expiry": "2027-12-31", // Mock expiry for now
			})
		}
	}
	if result == nil {
		result = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

func createAdminTenant(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	var req struct {
		Name         string `json:"name"`
		Username     string `json:"username"`
		Email        string `json:"email"`
		PhoneNumber  string `json:"phone_number"`
		Plan         string `json:"plan"`
		Password     string `json:"password"`
		Subdomain    string `json:"subdomain"`
		CustomDomain string `json:"custom_domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	if req.Name == "" || req.Username == "" || req.Email == "" || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "All fields are required"})
		return
	}
	if req.Plan == "" {
		req.Plan = "lite"
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB tx error"})
		return
	}
	defer tx.Rollback(ctx)

	// 1. Create Tenant
	var tenantID string
	err = tx.QueryRow(ctx, "INSERT INTO tenants (name, plan, subdomain, custom_domain) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, '')) RETURNING id", req.Name, req.Plan, req.Subdomain, req.CustomDomain).Scan(&tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create tenant"})
		return
	}

	// Use provided password or fallback to default
	password := req.Password
	if password == "" {
		password = "password123"
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	passwordHash := string(hashBytes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to hash password"})
		return
	}
	
	var userID string
	err = tx.QueryRow(ctx, "INSERT INTO users (tenant_id, username, email, password_hash, phone_number) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, req.Username, req.Email, passwordHash, req.PhoneNumber).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create user (phone/email/username might exist)"})
		return
	}

	// 3. Seed Accounting (Default COA)
	accounts := []struct{ Code, Name, Type string }{
		{"100", "Kas", "asset"},
		{"101", "Bank / QRIS", "asset"},
		{"110", "Piutang Usaha", "asset"},
		{"120", "Persediaan", "asset"},
		{"200", "Hutang Usaha", "liability"},
		{"300", "Modal", "equity"},
		{"400", "Pendapatan Usaha", "revenue"},
		{"500", "Beban Operasional", "expense"},
	}
	for _, acc := range accounts {
		_, err = tx.Exec(ctx,
			"INSERT INTO chart_of_accounts (tenant_id, code, name, type) VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, code) DO NOTHING",
			tenantID, acc.Code, acc.Name, acc.Type,
		)
		if err != nil {
			slog.Error("Failed to seed account during admin tenant creation", "error", err)
		}
	}

	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Tenant and user created successfully!"})
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	if r.Method == http.MethodGet {
		var waNumber, xenditApiKey, xenditWebhookToken, waProvider, reportTime *string
		var qrisEnabled, reportEnabled *bool
		err := DB.QueryRow(r.Context(), "SELECT wa_number, xendit_api_key, xendit_webhook_token, wa_provider, qris_enabled, report_enabled, report_time FROM tenants WHERE id = $1", tenantID).Scan(&waNumber, &xenditApiKey, &xenditWebhookToken, &waProvider, &qrisEnabled, &reportEnabled, &reportTime)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}

		wNum, xApiKey, xWebToken, provider, rTime := "", "", "", "internal", "07:00"
		qEnabled, rEnabled := false, false
		if waNumber != nil { wNum = *waNumber }
		if xenditApiKey != nil { xApiKey = *xenditApiKey }
		if xenditWebhookToken != nil { xWebToken = *xenditWebhookToken }
		if waProvider != nil && *waProvider != "" { provider = *waProvider }
		if qrisEnabled != nil { qEnabled = *qrisEnabled }
		if reportEnabled != nil { rEnabled = *reportEnabled }
		if reportTime != nil { rTime = *reportTime }

		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"wa_number": wNum,
				"wa_provider": provider,
				"xendit_api_key": xApiKey,
				"xendit_webhook_token": xWebToken,
				"qris_enabled": qEnabled,
				"report_enabled": rEnabled,
				"report_time": rTime,
			},
		})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			WaNumber           string `json:"wa_number"`
			XenditApiKey       string `json:"xendit_api_key"`
			XenditWebhookToken string `json:"xendit_webhook_token"`
			QrisEnabled        bool   `json:"qris_enabled"`
			ReportEnabled      bool   `json:"report_enabled"`
			ReportTime         string `json:"report_time"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		// Preserve existing WhatsApp number if not provided by frontend
		var existingWNum *string
		_ = DB.QueryRow(r.Context(), "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&existingWNum)
		if req.WaNumber == "" && existingWNum != nil {
			req.WaNumber = *existingWNum
		}

		// WA provider is always 'internal' — legacy Fonnte token is no longer supported.
		_, err := DB.Exec(r.Context(), "UPDATE tenants SET wa_number = $1, wa_provider = 'internal', xendit_api_key = $2, xendit_webhook_token = $3, qris_enabled = $4, report_enabled = $5, report_time = $6, updated_at = NOW() WHERE id = $7", req.WaNumber, req.XenditApiKey, req.XenditWebhookToken, req.QrisEnabled, req.ReportEnabled, req.ReportTime, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update settings"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Settings updated successfully"})
		return
	}
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), "SELECT id, name, price, description, COALESCE(photo_url, ''), COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0), COALESCE(additional_photos, '[]'::jsonb) FROM products WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var id, name, desc, photoURL, category string
			var price float64
			var stockQuantity int
			var additionalPhotosJSON []byte
			if err := rows.Scan(&id, &name, &price, &desc, &photoURL, &category, &stockQuantity, &additionalPhotosJSON); err == nil {
				var addPhotos []string
				json.Unmarshal(additionalPhotosJSON, &addPhotos)
				if addPhotos == nil {
					addPhotos = []string{}
				}
				results = append(results, map[string]interface{}{
					"id": id, "name": name, "price": price, "description": desc, "photo_url": photoURL, "category": category, "stock_quantity": stockQuantity, "additional_photos": addPhotos,
				})
			}
		}
		if results == nil {
			results = []map[string]interface{}{}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name          string   `json:"name"`
			Price         float64  `json:"price"`
			Description   string   `json:"description"`
			PhotoURL      string   `json:"photo_url"`
			Category      string   `json:"category"`
			StockQuantity int      `json:"stock_quantity"`
			AdditionalPhotos []string `json:"additional_photos"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		if req.Name == "" || req.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Name and Price must be valid"})
			return
		}

		if req.Category == "" {
			req.Category = "Umum"
		}
		if req.AdditionalPhotos == nil {
			req.AdditionalPhotos = []string{}
		}
		addPhotosBytes, _ := json.Marshal(req.AdditionalPhotos)

		var id string
		err := DB.QueryRow(r.Context(),
			"INSERT INTO products (tenant_id, name, price, description, photo_url, category, stock_quantity, additional_photos) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
			tenantID, req.Name, req.Price, req.Description, req.PhotoURL, req.Category, req.StockQuantity, addPhotosBytes).Scan(&id)
		
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": id}})
		return
	}

	if r.Method == http.MethodDelete {
		productID := r.URL.Query().Get("id")
		if productID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id parameter"})
			return
		}
		_, err := DB.Exec(r.Context(), "DELETE FROM products WHERE id = $1 AND tenant_id = $2", productID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Delete failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Product deleted"})
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			ID            string   `json:"id"`
			Name          string   `json:"name"`
			Price         float64  `json:"price"`
			Description   string   `json:"description"`
			PhotoURL      string   `json:"photo_url"`
			Category      string   `json:"category"`
			StockQuantity int      `json:"stock_quantity"`
			AdditionalPhotos []string `json:"additional_photos"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		if req.ID == "" || req.Name == "" || req.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "ID, Name, and Price must be valid"})
			return
		}

		if req.Category == "" {
			req.Category = "Umum"
		}
		if req.AdditionalPhotos == nil {
			req.AdditionalPhotos = []string{}
		}
		addPhotosBytes, _ := json.Marshal(req.AdditionalPhotos)

		_, err := DB.Exec(r.Context(),
			"UPDATE products SET name = $1, price = $2, description = $3, photo_url = $4, category = $5, stock_quantity = $6, additional_photos = $7 WHERE id = $8 AND tenant_id = $9",
			req.Name, req.Price, req.Description, req.PhotoURL, req.Category, req.StockQuantity, addPhotosBytes, req.ID, tenantID)
		
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Update failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Product updated"})
		return
	}
}

func handleProductsImport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only POST allowed"})
		return
	}
	
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Could not parse form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing file"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid CSV format"})
		return
	}

	if len(records) <= 1 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "CSV is empty or only contains header"})
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer tx.Rollback(ctx)

	// Detect if first column is ID
	var hasIDCol bool
	if len(records) > 0 && strings.ToLower(records[0][0]) == "id" {
		hasIDCol = true
	}

	successCount := 0
	skipCount := 0
	var skippedIDs []string

	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		
		var id, name, desc, category, photoURL, addPhotosStr string
		var price float64
		var stock int
		var additionalPhotos []string

		if hasIDCol {
			if len(row) < 6 {
				continue
			}
			id = row[0]
			name = row[1]
			price, _ = strconv.ParseFloat(row[2], 64)
			desc = row[3]
			category = row[4]
			stock, _ = strconv.Atoi(row[5])
			if len(row) >= 7 {
				photoURL = row[6]
			}
			if len(row) >= 8 {
				addPhotosStr = row[7]
			}
		} else {
			if len(row) < 5 {
				continue
			}
			name = row[0]
			price, _ = strconv.ParseFloat(row[1], 64)
			desc = row[2]
			category = row[3]
			stock, _ = strconv.Atoi(row[4])
			if len(row) >= 6 {
				photoURL = row[5]
			}
			if len(row) >= 7 {
				addPhotosStr = row[6]
			}
		}
		
		hasPhotoCol := (hasIDCol && len(row) >= 7) || (!hasIDCol && len(row) >= 6)
		
		if addPhotosStr != "" {
			additionalPhotos = strings.Split(addPhotosStr, "|")
		} else {
			additionalPhotos = []string{}
		}
		addPhotosBytes, _ := json.Marshal(additionalPhotos)

		if name == "" || price <= 0 {
			if id != "" {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
			continue
		}
		if category == "" {
			category = "Umum"
		}

		if id != "" {
			var exists bool
			err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND tenant_id = $2)", id, tenantID).Scan(&exists)
			if err != nil || !exists {
				skipCount++
				skippedIDs = append(skippedIDs, id)
				continue
			}
			
			if hasPhotoCol {
				_, err = tx.Exec(ctx,
					"UPDATE products SET name = $1, price = $2, description = $3, category = $4, stock_quantity = $5, photo_url = $8, additional_photos = $9 WHERE id = $6 AND tenant_id = $7",
					name, price, desc, category, stock, id, tenantID, photoURL, addPhotosBytes)
			} else {
				_, err = tx.Exec(ctx,
					"UPDATE products SET name = $1, price = $2, description = $3, category = $4, stock_quantity = $5 WHERE id = $6 AND tenant_id = $7",
					name, price, desc, category, stock, id, tenantID)
			}
			
			if err == nil {
				successCount++
			} else {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
		} else {
			if hasPhotoCol {
				_, err = tx.Exec(ctx, 
					"INSERT INTO products (tenant_id, name, price, description, category, stock_quantity, photo_url, additional_photos) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
					tenantID, name, price, desc, category, stock, photoURL, addPhotosBytes)
			} else {
				_, err = tx.Exec(ctx, 
					"INSERT INTO products (tenant_id, name, price, description, category, stock_quantity) VALUES ($1, $2, $3, $4, $5, $6)",
					tenantID, name, price, desc, category, stock)
			}
			if err == nil {
				successCount++
			}
		}
	}
	
	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true, 
		"message": fmt.Sprintf("Berhasil mengimpor %d produk. %d dilewati.", successCount, skipCount),
		"successCount": successCount,
		"skipCount": skipCount,
		"skippedIDs": skippedIDs,
	})
}

func handleProductsExport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only GET allowed"})
		return
	}

	rows, err := DB.Query(r.Context(), "SELECT id, name, price, description, COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0), COALESCE(photo_url, ''), COALESCE(additional_photos, '[]'::jsonb) FROM products WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=products_export.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"ID", "Name", "Price", "Description", "Category", "Stock", "Photo_URL", "Additional_Photos"})

	for rows.Next() {
		var id, name, desc, category, photoURL string
		var price float64
		var stock int
		var additionalPhotosJSON []byte
		if err := rows.Scan(&id, &name, &price, &desc, &category, &stock, &photoURL, &additionalPhotosJSON); err == nil {
			var addPhotos []string
			json.Unmarshal(additionalPhotosJSON, &addPhotos)
			writer.Write([]string{
				id, name, fmt.Sprintf("%.2f", price), desc, category, strconv.Itoa(stock), photoURL, strings.Join(addPhotos, "|"),
			})
		}
	}

}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant"})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			PaymentMethod string  `json:"payment_method"` // "cash" or "qris"
			TotalAmount   float64 `json:"total_amount"`
			Items         []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Quantity int    `json:"quantity"`
				Price    float64`json:"price"`
			} `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
			return
		}

		ctx := r.Context()

		var realTotalAmount float64
		for _, item := range req.Items {
			var dbPrice float64
			err := DB.QueryRow(ctx, "SELECT price FROM products WHERE id = $1 AND tenant_id = $2", item.ID, tenantID).Scan(&dbPrice)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: fmt.Sprintf("Produk tidak valid: %s", item.Name)})
				return
			}
			realTotalAmount += dbPrice * float64(item.Quantity)
		}

		// Security Check: Compare calculated total with requested total
		if req.TotalAmount != realTotalAmount {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Harga tidak valid. Kemungkinan terdeteksi manipulasi."})
			return
		}

		if realTotalAmount <= 0 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid amount"})
			return
		}
		
		// Determine accounts
		var debitAccCode, creditAccCode string
		creditAccCode = "400" // Pendapatan Usaha
		
		var xenditApiKey *string
		var qrisEnabled *bool

		if req.PaymentMethod == "qris" {
			// Validate QRIS settings
			err := DB.QueryRow(ctx, "SELECT xendit_api_key, qris_enabled FROM tenants WHERE id = $1", tenantID).Scan(&xenditApiKey, &qrisEnabled)
			if err != nil || qrisEnabled == nil || !*qrisEnabled {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Pembayaran QRIS belum diaktifkan oleh toko ini"})
				return
			}
			if xenditApiKey == nil || *xenditApiKey == "" {
				writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Toko belum mengatur Xendit API Key untuk menerima pembayaran QRIS."})
				return
			}
			debitAccCode = "101" // Bank / QRIS
		} else {
			debitAccCode = "100" // Kas (Cash)
		}

		var debitAccID, creditAccID string
		err := DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, debitAccCode).Scan(&debitAccID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Debit account not found. Try relogin to re-seed accounts."})
			return
		}

		err = DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2", tenantID, creditAccCode).Scan(&creditAccID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Credit account not found"})
			return
		}

		// Insert Transaction
		dateStr := time.Now().Format("2006-01-02")
		description := "Penjualan via " + req.PaymentMethod
		reference := "INV-" + time.Now().Format("060102150405")

		itemsJSON, _ := json.Marshal(map[string]interface{}{
			"items": req.Items,
		})

		status := "paid"
		if req.PaymentMethod == "qris" {
			status = "pending"
		}

		_, err = DB.Exec(ctx, "INSERT INTO pos_transactions (tenant_id, reference, total_amount, payment_method, status, items_json) VALUES ($1, $2, $3, $4, $5, $6)",
			tenantID, reference, realTotalAmount, req.PaymentMethod, status, itemsJSON)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menyimpan transaksi POS"})
			return
		}

		// Xendit QRIS/Invoice implementation
		if req.PaymentMethod == "qris" {
			xClient := xendit.NewClient(*xenditApiKey)

			externalID := reference
			createInvoiceReq := invoice.NewCreateInvoiceRequest(externalID, realTotalAmount)
			desc := "Pembayaran Toko UMKM: " + reference
			createInvoiceReq.Description = &desc

			resp, _, err := xClient.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*createInvoiceReq).Execute()
			if err != nil {
				slog.Error("Failed to create store invoice", "error", err)
				writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal membuat invoice Xendit. Cek API Key Anda."})
				return
			}

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":   true,
				"message":   "Menunggu pembayaran via Xendit",
				"status":    "pending",
				"qris_url":  resp.InvoiceUrl,
				"reference": reference,
			})
			return
		}

		// If CASH (Paid immediately), proceed to journal and stock deduction
		tx, err := DB.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer tx.Rollback(ctx)

		var entryID string
		err = tx.QueryRow(ctx,
			"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			tenantID, dateStr, description, reference, itemsJSON).Scan(&entryID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create entry"})
			return
		}

		// Insert Debit Line
		_, err = tx.Exec(ctx,
			"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)",
			entryID, debitAccID, realTotalAmount)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to insert debit line"})
			return
		}

		// Insert Credit Line
		_, err = tx.Exec(ctx,
			"INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)",
			entryID, creditAccID, realTotalAmount)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to insert credit line"})
			return
		}

		err = tx.Commit(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to commit transaction"})
			return
		}

		for _, item := range req.Items {
			DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
		}

		webhook.DispatchEvent("pos_checkout_completed", tenantID, map[string]interface{}{
			"reference":      reference,
			"total_amount":   realTotalAmount,
			"payment_method": req.PaymentMethod,
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Transaksi berhasil dicatat",
			"status": "paid",
			"qris_url": "",
			"reference": reference,
		})
		return
	}
	
	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}

func handleOCR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Failed to parse form"})
		return
	}
	
	file, _, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "No image provided"})
		return
	}
	defer file.Close()

	// Simulate AI Vision OCR processing latency
	time.Sleep(1500 * time.Millisecond)

	draft := map[string]interface{}{
		"date": time.Now().Format("2006-01-02"),
		"description": "Pembelian Bahan Baku (Hasil Scan OCR)",
		"reference": "OCR-" + time.Now().Format("150405"),
		"lines": []map[string]interface{}{
			{"account_id": "beban_bahan_baku", "debit": 150000, "credit": 0},
			{"account_id": "kas_kecil", "debit": 0, "credit": 150000},
		},
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Nota berhasil dipindai oleh AI",
		Data: draft,
	})
}

// ---- FAQ Handlers ----

func handleFaqs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, "SELECT id, question, answer FROM tenant_faqs WHERE tenant_id = $1 ORDER BY created_at ASC", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var faqs []map[string]string
		for rows.Next() {
			var id, question, answer string
			if err := rows.Scan(&id, &question, &answer); err == nil {
				faqs = append(faqs, map[string]string{"id": id, "question": question, "answer": answer})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: faqs})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid input"})
			return
		}
		var newID string
		err := DB.QueryRow(ctx, "INSERT INTO tenant_faqs (tenant_id, question, answer) VALUES ($1, $2, $3) RETURNING id", tenantID, req.Question, req.Answer).Scan(&newID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert error"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": newID}})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		DB.Exec(ctx, "DELETE FROM tenant_faqs WHERE id = $1 AND tenant_id = $2", id, tenantID)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Deleted"})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}

func handleFaqsGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")

	// Get tenant profile
	var tenantName string
	DB.QueryRow(r.Context(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)

	prompt := fmt.Sprintf("Buatkan 3 pertanyaan FAQ dan jawabannya untuk toko bernama '%s'. Outputkan HANYA dalam format JSON array seperti: [{\"question\": \"...\", \"answer\": \"...\"}] tanpa markdown tambahan.", tenantName)

	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    prompt,
		"system_msg": "Anda adalah asisten pembuat FAQ.",
		"tenant_id":  tenantID,
	}
	
	payloadBytes, _ := json.Marshal(aiReqBody)
	reqHTTP, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://ai-gateway:8002/v1/chat", bytes.NewBuffer(payloadBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Gagal menyiapkan request ke AI"})
		return
	}
	reqHTTP.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(reqHTTP)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "AI Gateway tidak merespon"})
		return
	}
	defer resp.Body.Close()
	
	var aiResp struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil || !aiResp.Success {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Gagal memproses respon dari AI Gateway"})
		return
	}

	var generated []map[string]string
	if err := json.Unmarshal([]byte(aiResp.Text), &generated); err != nil {
		// Fallback to simple parse or default if AI returned malformed JSON
		generated = []map[string]string{
			{"question": "Berapa jam operasional toko?", "answer": "Kami buka dari jam 08:00 pagi hingga 20:00 malam."},
		}
	}
	
	for _, f := range generated {
		DB.Exec(r.Context(), "INSERT INTO tenant_faqs (tenant_id, question, answer) VALUES ($1, $2, $3)", tenantID, f["question"], f["answer"])
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "FAQ awal berhasil di-generate."})
}

// ---- Forwarders Handlers ----

func handleForwarders(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	ctx := r.Context()

	if r.Method == http.MethodGet {
		rows, err := DB.Query(ctx, "SELECT id, phone_number FROM tenant_forwarders WHERE tenant_id = $1", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var list []map[string]string
		for rows.Next() {
			var id, phone string
			if err := rows.Scan(&id, &phone); err == nil {
				list = append(list, map[string]string{"id": id, "phone_number": phone})
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: list})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid input"})
			return
		}
		var newID string
		err := DB.QueryRow(ctx, "INSERT INTO tenant_forwarders (tenant_id, phone_number) VALUES ($1, $2) RETURNING id", tenantID, req.PhoneNumber).Scan(&newID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert error"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": newID}})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		DB.Exec(ctx, "DELETE FROM tenant_forwarders WHERE id = $1 AND tenant_id = $2", id, tenantID)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Deleted"})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}

func handleTransactionStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	reference := r.URL.Query().Get("reference")
	if reference == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing reference"})
		return
	}

	var status string
	err := DB.QueryRow(r.Context(), "SELECT status FROM pos_transactions WHERE reference = $1 AND tenant_id = $2", reference, tenantID).Scan(&status)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status": status,
	})
}

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	ctx := r.Context()
	
	var tenantID, currentStatus, paymentMethod, itemsJSONStr string
	var totalAmount float64
	err := DB.QueryRow(ctx, "SELECT tenant_id, status, payment_method, total_amount, items_json::text FROM pos_transactions WHERE reference = $1 FOR UPDATE", req.Reference).Scan(&tenantID, &currentStatus, &paymentMethod, &totalAmount, &itemsJSONStr)
	
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
		return
	}

	if currentStatus == "paid" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Already paid"})
		return
	}

	// Update to paid
	_, err = DB.Exec(ctx, "UPDATE pos_transactions SET status = 'paid', updated_at = NOW() WHERE reference = $1", req.Reference)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update status"})
		return
	}

	// Create journal entries
	var debitAccID, creditAccID string
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '101'", tenantID).Scan(&debitAccID)
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '400'", tenantID).Scan(&creditAccID)

	dateStr := time.Now().Format("2006-01-02")
	var entryID string
	err = DB.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, dateStr, "Pembayaran Webhook: "+req.Reference, req.Reference, itemsJSONStr).Scan(&entryID)
	
	if err == nil {
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)", entryID, debitAccID, totalAmount)
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)", entryID, creditAccID, totalAmount)
	}

	// Deduct Stock
	var parsedItems struct {
		Items []struct {
			ID       string `json:"id"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	json.Unmarshal([]byte(itemsJSONStr), &parsedItems)
	
	for _, item := range parsedItems.Items {
		DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
	}

	// Send WA Notification
	var waNumber *string
	DB.QueryRow(ctx, "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&waNumber)
	if waNumber != nil && *waNumber != "" {
		go func(tenantID, phone, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA* ✅\n\nRef: %s\nNominal: Rp %.0f\nMetode: QRIS\n\nTerima kasih, dana telah masuk ke rekening Anda dan sistem telah mencatat transaksi ini.", ref, amount)
			
			// Format phone to JID
			target := phone
			if strings.HasPrefix(target, "0") {
				target = "62" + target[1:]
			}
			if !strings.Contains(target, "@") {
				target = target + "@s.whatsapp.net"
			}

			// Internal WA Gateway call
			data := url.Values{}
			data.Set("tenant_id", tenantID)
			data.Set("target", target)
			data.Set("message", msg)

			req, _ := http.NewRequest("POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Message-Type", "subscription")
			req.Header.Set("X-Source", "umkm-accounting")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(tenantID, *waNumber, req.Reference, totalAmount)
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Payment processed successfully"})
}

func handleStorePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request"})
		return
	}

	status, _ := payload["status"].(string)
	reference, _ := payload["external_id"].(string)

	if status != "PAID" && status != "SETTLED" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Ignored"})
		return
	}

	ctx := r.Context()
	
	var tenantID, currentStatus, paymentMethod, itemsJSONStr string
	var totalAmount float64
	err := DB.QueryRow(ctx, "SELECT tenant_id, status, payment_method, total_amount, items_json::text FROM pos_transactions WHERE reference = $1 FOR UPDATE", reference).Scan(&tenantID, &currentStatus, &paymentMethod, &totalAmount, &itemsJSONStr)
	
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Transaction not found"})
		return
	}

	// Verify Xendit Webhook Token
	callbackToken := r.Header.Get("x-callback-token")
	var xenditWebhookToken *string
	DB.QueryRow(ctx, "SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID).Scan(&xenditWebhookToken)
	if xenditWebhookToken != nil && *xenditWebhookToken != "" && callbackToken != *xenditWebhookToken {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Message: "Invalid webhook token"})
		return
	}

	if currentStatus == "paid" {
		writeJSON(w, http.StatusOK, APIResponse{Message: "Already paid"})
		return
	}

	// Update to paid
	_, err = DB.Exec(ctx, "UPDATE pos_transactions SET status = 'paid', updated_at = NOW() WHERE reference = $1", reference)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update status"})
		return
	}

	webhook.DispatchEvent("pos_checkout_completed", tenantID, map[string]interface{}{
		"reference":      reference,
		"total_amount":   totalAmount,
		"payment_method": paymentMethod,
	})

	// Create journal entries
	var debitAccID, creditAccID string
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '101'", tenantID).Scan(&debitAccID)
	DB.QueryRow(ctx, "SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = '400'", tenantID).Scan(&creditAccID)

	dateStr := time.Now().Format("2006-01-02")
	var entryID string
	err = DB.QueryRow(ctx,
		"INSERT INTO journal_entries (tenant_id, date, description, reference, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, dateStr, "Pembayaran Xendit: "+reference, reference, itemsJSONStr).Scan(&entryID)
	
	if err == nil {
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)", entryID, debitAccID, totalAmount)
		DB.Exec(ctx, "INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)", entryID, creditAccID, totalAmount)
	}

	// Deduct Stock
	var parsedItems struct {
		Items []struct {
			ID       string `json:"id"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}
	json.Unmarshal([]byte(itemsJSONStr), &parsedItems)
	
	for _, item := range parsedItems.Items {
		DB.Exec(ctx, "UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2 AND tenant_id = $3", item.Quantity, item.ID, tenantID)
	}

	// Send WA Notification
	var waNumber *string
	DB.QueryRow(ctx, "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&waNumber)
	if waNumber != nil && *waNumber != "" {
		go func(tenantID, phone, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA VIA XENDIT* ✅\n\nRef: %s\nNominal: Rp %.0f\nMetode: Xendit (QRIS/VA)\n\nTerima kasih, dana otomatis tercatat dalam sistem POS Anda.", ref, amount)
			
			target := phone
			if strings.HasPrefix(target, "0") {
				target = "62" + target[1:]
			}
			if !strings.Contains(target, "@") {
				target = target + "@s.whatsapp.net"
			}

			data := url.Values{}
			data.Set("tenant_id", tenantID)
			data.Set("target", target)
			data.Set("message", msg)

			req, _ := http.NewRequest("POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Message-Type", "subscription")
			req.Header.Set("X-Source", "umkm-accounting")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(tenantID, *waNumber, reference, totalAmount)
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Store payment webhook processed"})
}

func handleInternalScheduledReports(w http.ResponseWriter, r *http.Request) {
	timeParam := r.URL.Query().Get("time")
	if timeParam == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing time parameter"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	rows, err := DB.Query(r.Context(), "SELECT id, name, wa_number FROM tenants WHERE report_enabled = true AND report_time = $1", timeParam)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var id, name string
		var waNumber *string
		if err := rows.Scan(&id, &name, &waNumber); err == nil {
			wn := ""
			if waNumber != nil {
				wn = *waNumber
			}
			tenants = append(tenants, map[string]interface{}{
				"id":        id,
				"name":      name,
				"wa_number": wn,
			})
		}
	}

	if tenants == nil {
		tenants = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: tenants})
}

func handleInternalReportsSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}

	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	// Today's boundaries
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")

	query := `
		SELECT c.type, SUM(l.credit - l.debit) as balance
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date = $2
		GROUP BY c.type
	`
	rows, err := DB.Query(r.Context(), query, tenantID, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var totalRevenue int64
	for rows.Next() {
		var typ string
		var balance int64
		if err := rows.Scan(&typ, &balance); err == nil {
			totalRevenue += balance
		}
	}

	// Count transactions
	var txCount int
	err = DB.QueryRow(r.Context(), "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date = $2", tenantID, startOfDay).Scan(&txCount)
	if err != nil {
		txCount = 0
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"tenant_id":     tenantID,
			"date":          startOfDay,
			"total_revenue": totalRevenue,
			"tx_count":      txCount,
		},
	})
}

// ==========================================
// Automation Handlers
// ==========================================

func getAutomationLimit(plan string) int {
	switch plan {
	case "free":
		return 0
	case "lite":
		return 3
	case "pro":
		return 10
	default:
		return 999 // enterprise, superadmin
	}
}

// cronMatchesNow checks if a cron expression matches the current time (minute + hour + day-of-week level)
func cronMatchesNow(cronExpr string, now time.Time) bool {
	parts := strings.Fields(cronExpr)
	if len(parts) < 5 {
		return false
	}
	minute := now.Minute()
	hour := now.Hour()
	dayOfMonth := now.Day()
	month := int(now.Month())
	dayOfWeek := int(now.Weekday()) // 0=Sunday

	return fieldMatches(parts[0], minute) &&
		fieldMatches(parts[1], hour) &&
		fieldMatches(parts[2], dayOfMonth) &&
		fieldMatches(parts[3], month) &&
		fieldMatches(parts[4], dayOfWeek)
}

func fieldMatches(field string, value int) bool {
	if field == "*" {
		return true
	}
	// Handle */N step values
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step == 0 {
			return false
		}
		return value%step == 0
	}
	// Handle comma-separated values
	for _, part := range strings.Split(field, ",") {
		// Handle range N-M
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				low, err1 := strconv.Atoi(rangeParts[0])
				high, err2 := strconv.Atoi(rangeParts[1])
				if err1 == nil && err2 == nil && value >= low && value <= high {
					return true
				}
			}
			continue
		}
		v, err := strconv.Atoi(part)
		if err == nil && v == value {
			return true
		}
	}
	return false
}

func handleAutomations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		rows, err := DB.Query(ctx, `SELECT id, type, name, enabled, cron_expression, config, target_wa, last_run_at, created_at FROM tenant_automations WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
		if err != nil {
			slog.Error("Failed to query tenant_automations", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var id, typ, name, cronExpr string
			var enabled bool
			var configJSON []byte
			var targetWA *string
			var lastRunAt *time.Time
			var createdAt time.Time
			if err := rows.Scan(&id, &typ, &name, &enabled, &cronExpr, &configJSON, &targetWA, &lastRunAt, &createdAt); err == nil {
				var cfg map[string]interface{}
				json.Unmarshal(configJSON, &cfg)
				if cfg == nil {
					cfg = map[string]interface{}{}
				}
				tw := ""
				if targetWA != nil {
					tw = *targetWA
				}
				var lra string
				if lastRunAt != nil {
					lra = lastRunAt.Format(time.RFC3339)
				}
				results = append(results, map[string]interface{}{
					"id":              id,
					"type":            typ,
					"name":            name,
					"enabled":         enabled,
					"cron_expression": cronExpr,
					"config":          cfg,
					"target_wa":       tw,
					"last_run_at":     lra,
					"created_at":      createdAt.Format(time.RFC3339),
				})
			}
		}
		if results == nil {
			results = []map[string]interface{}{}
		}

		// Also get plan info for limit display
		var plan string
		DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
		limit := getAutomationLimit(plan)

		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"automations": results,
				"plan":        plan,
				"limit":       limit,
				"count":       len(results),
			},
		})

	case http.MethodPost:
		// Check plan limit
		var plan string
		DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
		limit := getAutomationLimit(plan)

		var currentCount int
		DB.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_automations WHERE tenant_id = $1", tenantID).Scan(&currentCount)

		if currentCount >= limit {
			msg := "Paket Anda tidak mendukung fitur automasi. Upgrade ke paket Lite atau lebih tinggi."
			if limit > 0 {
				msg = fmt.Sprintf("Batas automasi untuk paket %s adalah %d. Hapus automasi lama atau upgrade paket.", plan, limit)
			}
			writeJSON(w, http.StatusForbidden, APIResponse{Message: msg})
			return
		}

		var req struct {
			Type           string                 `json:"type"`
			Name           string                 `json:"name"`
			CronExpression string                 `json:"cron_expression"`
			Config         map[string]interface{} `json:"config"`
			TargetWA       string                 `json:"target_wa"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}
		if req.Type == "" || req.Name == "" || req.CronExpression == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "type, name, dan cron_expression wajib diisi"})
			return
		}

		configJSON, _ := json.Marshal(req.Config)
		if req.Config == nil {
			configJSON = []byte("{}")
		}

		var id string
		err := DB.QueryRow(ctx,
			"INSERT INTO tenant_automations (tenant_id, type, name, cron_expression, config, target_wa) VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')) RETURNING id",
			tenantID, req.Type, req.Name, req.CronExpression, configJSON, req.TargetWA).Scan(&id)
		if err != nil {
			slog.Error("Failed to create automation", "error", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal membuat automasi"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil dibuat", Data: map[string]string{"id": id}})

	case http.MethodPut:
		automationID := r.URL.Query().Get("id")
		if automationID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id"})
			return
		}
		var req struct {
			Name           string                 `json:"name"`
			Enabled        *bool                  `json:"enabled"`
			CronExpression string                 `json:"cron_expression"`
			Config         map[string]interface{} `json:"config"`
			TargetWA       string                 `json:"target_wa"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
			return
		}

		// Build dynamic update
		sets := []string{"updated_at = NOW()"}
		args := []interface{}{}
		argIdx := 1

		if req.Name != "" {
			sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
			args = append(args, req.Name)
			argIdx++
		}
		if req.Enabled != nil {
			sets = append(sets, fmt.Sprintf("enabled = $%d", argIdx))
			args = append(args, *req.Enabled)
			argIdx++
		}
		if req.CronExpression != "" {
			sets = append(sets, fmt.Sprintf("cron_expression = $%d", argIdx))
			args = append(args, req.CronExpression)
			argIdx++
		}
		if req.Config != nil {
			configJSON, _ := json.Marshal(req.Config)
			sets = append(sets, fmt.Sprintf("config = $%d", argIdx))
			args = append(args, configJSON)
			argIdx++
		}
		if req.TargetWA != "" {
			sets = append(sets, fmt.Sprintf("target_wa = $%d", argIdx))
			args = append(args, req.TargetWA)
			argIdx++
		}

		query := fmt.Sprintf("UPDATE tenant_automations SET %s WHERE id = $%d AND tenant_id = $%d",
			strings.Join(sets, ", "), argIdx, argIdx+1)
		args = append(args, automationID, tenantID)

		_, err := DB.Exec(ctx, query, args...)
		if err != nil {
			slog.Error("Failed to update automation", "error", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal mengupdate automasi"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil diupdate"})

	case http.MethodDelete:
		automationID := r.URL.Query().Get("id")
		if automationID == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id"})
			return
		}
		_, err := DB.Exec(ctx, "DELETE FROM tenant_automations WHERE id = $1 AND tenant_id = $2", automationID, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menghapus automasi"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Automasi berhasil dihapus"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

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
	var due []map[string]interface{}

	for rows.Next() {
		var id, tenantID, typ, name, cronExpr, waNumber, tenantName string
		var configJSON []byte
		if err := rows.Scan(&id, &tenantID, &typ, &name, &cronExpr, &configJSON, &waNumber, &tenantName); err == nil {
			if cronMatchesNow(cronExpr, now) {
				var cfg map[string]interface{}
				json.Unmarshal(configJSON, &cfg)
				due = append(due, map[string]interface{}{
					"automation_id": id,
					"tenant_id":    tenantID,
					"type":         typ,
					"name":         name,
					"wa_number":    waNumber,
					"tenant_name":  tenantName,
					"config":       cfg,
				})
			}
		}
	}

	if due == nil {
		due = []map[string]interface{}{}
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
	today := now.Format("2006-01-02")
	var message string

	switch req.Type {
	case "daily_report":
		var totalRevenue int64
		var txCount int
		DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(l.credit - l.debit), 0)
			FROM journal_lines l
			JOIN journal_entries e ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date = $2
		`, req.TenantID, today).Scan(&totalRevenue)
		DB.QueryRow(ctx, "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date = $2", req.TenantID, today).Scan(&txCount)

		message = fmt.Sprintf("📊 *LAPORAN HARIAN*\n📅 %s\n\n💰 Total Pendapatan: Rp %s\n🧾 Jumlah Transaksi: %d\n\n_Laporan otomatis dari SaaS UMKM WCH_",
			today, formatRupiah(totalRevenue), txCount)

	case "weekly_report":
		weekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
		var totalRevenue int64
		var txCount int
		DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(l.credit - l.debit), 0)
			FROM journal_lines l
			JOIN journal_entries e ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND c.type = 'revenue' AND e.date >= $2 AND e.date <= $3
		`, req.TenantID, weekAgo, today).Scan(&totalRevenue)
		DB.QueryRow(ctx, "SELECT COUNT(id) FROM journal_entries WHERE tenant_id = $1 AND date >= $2 AND date <= $3", req.TenantID, weekAgo, today).Scan(&txCount)

		message = fmt.Sprintf("📊 *LAPORAN MINGGUAN*\n📅 %s s/d %s\n\n💰 Total Pendapatan: Rp %s\n🧾 Jumlah Transaksi: %d\n\n_Laporan otomatis dari SaaS UMKM WCH_",
			weekAgo, today, formatRupiah(totalRevenue), txCount)

	case "monthly_report":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
		var totalRevenue, totalExpense int64
		DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(CASE WHEN c.type='revenue' THEN l.credit-l.debit ELSE 0 END), 0),
			       COALESCE(SUM(CASE WHEN c.type='expense' THEN l.debit-l.credit ELSE 0 END), 0)
			FROM journal_lines l
			JOIN journal_entries e ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND c.type IN ('revenue', 'expense') AND e.date >= $2 AND e.date <= $3
		`, req.TenantID, firstOfMonth, today).Scan(&totalRevenue, &totalExpense)

		netIncome := totalRevenue - totalExpense
		message = fmt.Sprintf("📊 *LAPORAN BULANAN*\n📅 %s s/d %s\n\n💰 Pendapatan: Rp %s\n💸 Beban: Rp %s\n📈 Laba Bersih: Rp %s\n\n_Laporan otomatis dari SaaS UMKM WCH_",
			firstOfMonth, today, formatRupiah(totalRevenue), formatRupiah(totalExpense), formatRupiah(netIncome))

	case "low_stock_alert":
		threshold := 5
		// Read threshold from automation config if available
		var configJSON []byte
		DB.QueryRow(ctx, "SELECT config FROM tenant_automations WHERE id = $1", req.AutomationID).Scan(&configJSON)
		if configJSON != nil {
			var cfg map[string]interface{}
			json.Unmarshal(configJSON, &cfg)
			if t, ok := cfg["threshold"]; ok {
				if tf, ok := t.(float64); ok {
					threshold = int(tf)
				}
			}
		}

		rows, err := DB.Query(ctx, "SELECT name, COALESCE(stock_quantity, 0) FROM products WHERE tenant_id = $1 AND COALESCE(stock_quantity, 0) <= $2 ORDER BY stock_quantity ASC LIMIT 20", req.TenantID, threshold)
		if err == nil {
			defer rows.Close()
			var items []string
			for rows.Next() {
				var pName string
				var stock int
				if rows.Scan(&pName, &stock) == nil {
					items = append(items, fmt.Sprintf("• %s (stok: %d)", pName, stock))
				}
			}
			if len(items) > 0 {
				message = fmt.Sprintf("⚠️ *ALERT STOK RENDAH*\n📅 %s\n\nProduk dengan stok ≤ %d:\n%s\n\n_Segera restok agar tidak kehabisan!_",
					today, threshold, strings.Join(items, "\n"))
			} else {
				message = fmt.Sprintf("✅ *STOK AMAN*\n📅 %s\n\nSemua produk memiliki stok di atas %d. Tidak ada yang perlu restok saat ini.", today, threshold)
			}
		}

	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Unknown automation type: " + req.Type})
		return
	}

	// Update last_run_at
	DB.Exec(ctx, "UPDATE tenant_automations SET last_run_at = NOW() WHERE id = $1", req.AutomationID)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":       message,
			"automation_id": req.AutomationID,
			"tenant_id":     req.TenantID,
			"type":          req.Type,
		},
	})
}

func formatRupiah(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Chatbot / N8N Hybrid Internal Handlers
// ─────────────────────────────────────────────────────────────────────────────

func handleInternalChatbotConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT llm_provider, llm_model, temperature, max_tokens, system_prompt,
		        tone, language, max_context_messages, welcome_message, fallback_message,
			outside_hours_message, business_hours_start, business_hours_end, business_days,
			escalation_enabled, escalation_keywords, escalation_confidence_threshold,
			auto_escalate_after_minutes, rag_enabled, rag_top_k, rag_similarity_threshold,
			channels_enabled, is_active
		 FROM tenant_chatbot_configs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Chatbot config not found"})
		return
	}

	var cfg ChatbotConfig
	var sysPrompt, welcome, fallback, outsideHrs *string
	var escalationKW []byte
	if err := rows.Scan(
		&cfg.LLMProvider, &cfg.LLMModel, &cfg.Temperature, &cfg.MaxTokens,
		&sysPrompt, &cfg.Tone, &cfg.Language, &cfg.MaxContextMessages,
		&welcome, &fallback, &outsideHrs,
		&cfg.BusinessHoursStart, &cfg.BusinessHoursEnd, &cfg.BusinessDays,
		&cfg.EscalationEnabled, &escalationKW, &cfg.EscalationConfidenceThreshold,
		&cfg.AutoEscalateAfterMinutes, &cfg.RAGEnabled, &cfg.RAGTopK, &cfg.RAGSimilarityThreshold,
		&cfg.ChannelsEnabled, &cfg.IsActive,
	); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Scan error"})
		return
	}

	if sysPrompt != nil {
		cfg.SystemPrompt = *sysPrompt
	}
	if welcome != nil {
		cfg.WelcomeMessage = *welcome
	}
	if fallback != nil {
		cfg.FallbackMessage = *fallback
	}
	if outsideHrs != nil {
		cfg.OutsideHoursMessage = *outsideHrs
	}
	json.Unmarshal(escalationKW, &cfg.EscalationKeywords)

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cfg})
}

type ChatbotConfig struct {
	LLMProvider                  string   `json:"llm_provider"`
	LLMModel                     string   `json:"llm_model"`
	Temperature                  float64  `json:"temperature"`
	MaxTokens                    int      `json:"max_tokens"`
	SystemPrompt                 string   `json:"system_prompt"`
	Tone                         string   `json:"tone"`
	Language                     string   `json:"language"`
	MaxContextMessages           int      `json:"max_context_messages"`
	WelcomeMessage               string   `json:"welcome_message"`
	FallbackMessage              string   `json:"fallback_message"`
	OutsideHoursMessage          string   `json:"outside_hours_message"`
	BusinessHoursStart           string   `json:"business_hours_start"`
	BusinessHoursEnd             string   `json:"business_hours_end"`
	BusinessDays                 []int    `json:"business_days"`
	EscalationEnabled            bool     `json:"escalation_enabled"`
	EscalationKeywords           []string `json:"escalation_keywords"`
	EscalationConfidenceThreshold float64 `json:"escalation_confidence_threshold"`
	AutoEscalateAfterMinutes     int      `json:"auto_escalate_after_minutes"`
	RAGEnabled                   bool     `json:"rag_enabled"`
	RAGTopK                      int      `json:"rag_top_k"`
	RAGSimilarityThreshold       float64  `json:"rag_similarity_threshold"`
	ChannelsEnabled              []string `json:"channels_enabled"`
	IsActive                     bool     `json:"is_active"`
}

func handleInternalRAGSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		Query     string  `json:"query"`
		TenantID  string  `json:"tenant_id"`
		TopK      int     `json:"top_k"`
		Threshold float64 `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "query cannot be empty"})
		return
	}
	if req.TenantID == "" {
		req.TenantID = tenantID
	}
	if req.TopK == 0 {
		req.TopK = 5
	}
	if req.Threshold == 0 {
		req.Threshold = 0.7
	}

	// Generate query embedding via AI gateway
	queryEmb, err := generateEmbedding(r.Context(), req.Query)
	if err != nil {
		slog.Warn("Failed to generate query embedding, returning empty RAG results", "error", err)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{}})
		return
	}

	rows, err := DB.Query(r.Context(), `
		SELECT id, source_type, source_id, content, metadata,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM vector_embeddings
		WHERE tenant_id = $2
		  AND (1 - (embedding <=> $1::vector)) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`, queryEmb, req.TenantID, req.Threshold, req.TopK)
	if err != nil {
		slog.Error("RAG search error", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Search error"})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, sourceType string
		var sourceID *string
		var content string
		var metaJSON []byte
		var similarity float64
		if err := rows.Scan(&id, &sourceType, &sourceID, &content, &metaJSON, &similarity); err == nil {
			var meta map[string]interface{}
			json.Unmarshal(metaJSON, &meta)
			sID := ""
			if sourceID != nil {
				sID = *sourceID
			}
			results = append(results, map[string]interface{}{
				"id": id, "source_type": sourceType, "source_id": sID,
				"content": content, "similarity": similarity, "metadata": meta,
			})
		}
	}
	if results == nil {
		results = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
}

func generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]interface{}{
		"input": text,
		"model": "text-embedding-ada-002",
	}
	body, _ := json.Marshal(payload)
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	// Use OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		reqHTTP.Header.Set("Authorization", "Bearer "+apiKey)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return result.Data[0].Embedding, nil
}

func handleInternalConversationLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		TenantID        string  `json:"tenant_id"`
		CustomerID      string  `json:"customer_id"`
		Channel         string  `json:"channel"`
		UserMessage     string  `json:"user_message"`
		AssistantMsg    string  `json:"assistant_message"`
		LLMProvider     string  `json:"llm_provider"`
		LLMModel        string  `json:"llm_model"`
		TokensUsed      int     `json:"tokens_used"`
		SessionID       string  `json:"session_id,omitempty"`
		Confidence      float64 `json:"confidence,omitempty"`
		RAGSources      []map[string]interface{} `json:"rag_sources,omitempty"`
		LatencyMs       int     `json:"latency_ms,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.TenantID == "" || req.CustomerID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "tenant_id and customer_id required"})
		return
	}
	if req.Channel == "" {
		req.Channel = "whatsapp"
	}

	ctx := r.Context()

	// Upsert conversation session
	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID
	} else {
		// Find or create active session
		err := DB.QueryRow(ctx,
			`SELECT id FROM conversation_sessions
			 WHERE tenant_id = $1 AND customer_id = $2 AND status = 'active'
			 ORDER BY last_message_at DESC LIMIT 1`,
			req.TenantID, req.CustomerID).Scan(&sessionID)
		if err != nil {
			// Create new session
			err = DB.QueryRow(ctx,
				`INSERT INTO conversation_sessions (tenant_id, customer_id, channel, status, last_message_at)
				 VALUES ($1, $2, $3, 'active', NOW()) RETURNING id`,
				req.TenantID, req.CustomerID, req.Channel).Scan(&sessionID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create session"})
				return
			}
		}
	}

	// Log user message
	var userLogID string
	ragSrcJSON, _ := json.Marshal(req.RAGSources)
	DB.QueryRow(ctx,
		`INSERT INTO conversation_logs (session_id, tenant_id, role, content, channel,
		 llm_provider, llm_model, tokens_used, latency_ms, confidence, rag_sources)
		 VALUES ($1, $2, 'user', $3, $4, NULL, NULL, 0, 0, NULL, '[]')
		 RETURNING id`,
		sessionID, req.TenantID, req.UserMessage, req.Channel).Scan(&userLogID)

	// Log assistant message
	var assistantLogID string
	DB.QueryRow(ctx,
		`INSERT INTO conversation_logs (session_id, tenant_id, role, content, channel,
		 llm_provider, llm_model, tokens_used, latency_ms, confidence, rag_sources)
		 VALUES ($1, $2, 'assistant', $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		sessionID, req.TenantID, req.AssistantMsg, req.Channel,
		req.LLMProvider, req.LLMModel, req.TokensUsed, req.LatencyMs,
		req.Confidence, string(ragSrcJSON)).Scan(&assistantLogID)

	// Update session
	DB.Exec(ctx,
		`UPDATE conversation_sessions
		 SET message_count = message_count + 2, last_message_at = NOW()
		 WHERE id = $1`,
		sessionID)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"session_id": sessionID},
	})
}

func handleInternalEscalationLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		SessionID               string `json:"session_id"`
		TenantID                string `json:"tenant_id"`
		Reason                  string `json:"reason"`
		TriggerMessage          string `json:"trigger_message"`
		ChatwootConversationID  string `json:"chatwoot_conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.TenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "tenant_id required"})
		return
	}

	ctx := r.Context()

	// Mark session as escalated
	if req.SessionID != "" {
		DB.Exec(ctx,
			`UPDATE conversation_sessions
			 SET status = 'escalated', escalated_to = $1, escalated_at = NOW()
			 WHERE id = $2`,
			req.ChatwootConversationID, req.SessionID)
	}

	var id string
	err := DB.QueryRow(ctx,
		`INSERT INTO escalation_history
		 (session_id, tenant_id, reason, trigger_message, chatwoot_conversation_id)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		 RETURNING id`,
		req.SessionID, req.TenantID, req.Reason, req.TriggerMessage,
		req.ChatwootConversationID).Scan(&id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to log escalation"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"escalation_id": id},
	})
}

func handleInternalFAQs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT id, question, answer FROM tenant_faqs WHERE tenant_id = $1 ORDER BY created_at ASC`,
		tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var faqs []map[string]interface{}
	for rows.Next() {
		var id, question, answer string
		if rows.Scan(&id, &question, &answer) == nil {
			faqs = append(faqs, map[string]interface{}{
				"id": id, "question": question, "answer": answer,
			})
		}
	}
	if faqs == nil {
		faqs = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: faqs})
}

func handleInternalProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT id, name, price, description, COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0)
		 FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var id, name, desc, category string
		var price float64
		var stock int
		if rows.Scan(&id, &name, &price, &desc, &category, &stock) == nil {
			products = append(products, map[string]interface{}{
				"id": id, "name": name, "price": int64(price * 100),
				"description": desc, "category": category, "stock": stock,
			})
		}
	}
	if products == nil {
		products = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: products})
}

func handleInternalRAGSingle(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req struct {
		TenantID  string `json:"tenant_id"`
		SourceType string `json:"source_type"`
		SourceID  string `json:"source_id"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.Content == "" || req.SourceType == "" || req.SourceID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "content, source_type, source_id required"})
		return
	}
	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	// Generate embedding
	emb, err := generateEmbedding(r.Context(), req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to generate embedding"})
		return
	}

	// Upsert into vector_embeddings
	var id string
	err = DB.QueryRow(r.Context(),
		`INSERT INTO vector_embeddings (tenant_id, source_type, source_id, content, embedding)
		 VALUES ($1, $2, $3, $4, $5::vector)
		 ON CONFLICT (tenant_id, source_type, source_id) DO UPDATE
		   SET content = EXCLUDED.content, embedding = EXCLUDED.embedding, updated_at = NOW()
		 RETURNING id`,
		req.TenantID, req.SourceType, req.SourceID, req.Content, vectorFromSlice(emb)).Scan(&id)
	if err != nil {
		slog.Error("Failed to upsert vector embedding", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to store embedding"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"id": id},
	})
}

func vectorFromSlice(v []float64) string {
	b := &strings.Builder{}
	b.WriteString("[")
	for i, f := range v {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%.6f", f))
	}
	b.WriteString("]")
	return b.String()
}

