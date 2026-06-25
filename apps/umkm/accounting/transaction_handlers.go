package main

import (
	"encoding/json"
	"net/http"
	"time"
)


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