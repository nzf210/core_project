package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"core_project/shared/sdk/response"
)

const (
	headerTenantIDImport = "X-Tenant-ID"
	errMissingXTenantID  = response.MissingXTenantID
)

func handleImportProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantIDImport)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingXTenantID})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	imported, skipped, errs := processProductRows(r.Context(), tenantID, headers, rows)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]any{"imported": imported, "skipped": skipped, "errors": errs}})
}

func processProductRows(ctx context.Context, tenantID string, headers []string, rows [][]string) (int, int, []map[string]any) {
	idx := indexHeaders(headers)
	for _, c := range []string{"name", "sku", "price_rupiah"} {
		if _, ok := idx[c]; !ok {
			return 0, 0, []map[string]any{{"error": "Kolom wajib hilang: " + c}}
		}
	}
	var imported, skipped int
	var errs []map[string]any
	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		name, sku, price, ok := validateProductRow(idx, row)
		if !ok {
			skipped++
			if name != "" {
				errs = append(errs, map[string]any{"row": rowIdx, "error": name})
			}
			continue
		}
		stock, _ := strconv.ParseInt(cellVal(idx, row, "stock"), 10, 64)
		err := upsertProduct(ctx, upsertProductParams{
			TenantID: tenantID,
			Name:     name,
			SKU:      sku,
			Price:    price,
			Stock:    stock,
			Category: cellVal(idx, row, "category"),
			Desc:     cellVal(idx, row, "description"),
			ImageURL: cellVal(idx, row, "image_url"),
		})
		if err != nil {
			skipped++
			errs = append(errs, map[string]any{"row": rowIdx, "error": err.Error()})
		} else {
			imported++
		}
	}
	return imported, skipped, errs
}

func validateProductRow(idx map[string]int, row []string) (name, sku string, price int64, ok bool) {
	name = cellVal(idx, row, "name")
	sku = cellVal(idx, row, "sku")
	if name == "" || sku == "" {
		name = "name atau sku kosong"
		return
	}
	price, _ = strconv.ParseInt(cellVal(idx, row, "price_rupiah"), 10, 64)
	if price < 0 {
		name = "price_rupiah tidak valid"
		return
	}
	ok = true
	return
}

func cellVal(idx map[string]int, row []string, col string) string {
	i, ok := idx[col]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

type upsertProductParams struct {
	TenantID string
	Name     string
	SKU      string
	Price    int64
	Stock    int64
	Category string
	Desc     string
	ImageURL string
}

func upsertProduct(ctx context.Context, p upsertProductParams) error {
	_, err := DB.Exec(ctx, `
		INSERT INTO products (tenant_id, name, sku, category, price_rupiah, stock, description, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, sku) DO UPDATE SET
			name = EXCLUDED.name, category = EXCLUDED.category,
			price_rupiah = EXCLUDED.price_rupiah, stock = EXCLUDED.stock,
			description = EXCLUDED.description, image_url = EXCLUDED.image_url
	`, p.TenantID, p.Name, p.SKU, nullString(p.Category), p.Price, p.Stock, nullString(p.Desc), nullString(p.ImageURL))
	return err
}

func handleImportContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantIDImport)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingXTenantID})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	imported, skipped, errs := processContactRows(r.Context(), tenantID, headers, rows)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]any{"imported": imported, "skipped": skipped, "errors": errs}})
}

func processContactRows(ctx context.Context, tenantID string, headers []string, rows [][]string) (int, int, []map[string]any) {
	idx := indexHeaders(headers)
	if _, ok := idx["phone"]; !ok {
		return 0, 0, []map[string]any{{"error": "Kolom wajib hilang: phone"}}
	}
	var imported, skipped int
	var errs []map[string]any
	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		phone := cellVal(idx, row, "phone")
		if phone == "" {
			skipped++
			errs = append(errs, map[string]any{"row": rowIdx, "error": "phone kosong"})
			continue
		}
		role := cellVal(idx, row, "role")
		if role == "" {
			role = "customer"
		}
		_, err := DB.Exec(ctx, `
			INSERT INTO tenant_contacts (tenant_id, name, phone_number, email, role, notes)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, phone_number) DO UPDATE SET
				name = EXCLUDED.name, email = EXCLUDED.email,
				role = EXCLUDED.role, notes = EXCLUDED.notes
		`, tenantID, nullString(cellVal(idx, row, "name")), phone, nullString(cellVal(idx, row, "email")), role, nullString(cellVal(idx, row, "notes")))
		if err != nil {
			skipped++
			errs = append(errs, map[string]any{"row": rowIdx, "error": err.Error()})
		} else {
			imported++
		}
	}
	return imported, skipped, errs
}

func handleImportJournal(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantIDImport)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingXTenantID})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	imported, skipped, errs := processJournalRows(r.Context(), tenantID, headers, rows)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]any{"imported": imported, "skipped": skipped, "errors": errs}})
}

func processJournalRows(ctx context.Context, tenantID string, headers []string, rows [][]string) (int, int, []map[string]any) {
	idx := indexHeaders(headers)
	for _, c := range []string{"date", "description", "debit_account_code", "credit_account_code", "amount_rupiah"} {
		if _, ok := idx[c]; !ok {
			return 0, 0, []map[string]any{{"error": "Kolom wajib hilang: " + c}}
		}
	}

	groups, order, importErrs := buildJournalGroups(idx, rows)
	accountCache := map[string]string{}
	imported, skipped := importJournalGroups(ctx, tenantID, groups, order, accountCache, importErrs)
	return imported, skipped, importErrs
}

type jLine struct {
	rowIdx   int
	date     time.Time
	desc     string
	ref      string
	debit    int64
	credit   int64
	debitAcc string
	credAcc  string
}

func buildJournalGroups(idx map[string]int, rows [][]string) (map[string][]jLine, []string, []map[string]any) {
	groups := map[string][]jLine{}
	order := []string{}
	seenRef := map[string]bool{}
	var skipped int
	var importErrs []map[string]any

	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		line, ok := parseJournalRow(idx, row, rowIdx)
		if !ok {
			skipped++
			if line.desc != "" {
				importErrs = append(importErrs, map[string]any{"row": rowIdx, "error": line.desc})
			}
			continue
		}
		groups[line.ref] = append(groups[line.ref], line)
		if !seenRef[line.ref] {
			seenRef[line.ref] = true
			order = append(order, line.ref)
		}
	}
	_ = skipped
	return groups, order, importErrs
}

func parseJournalRow(idx map[string]int, row []string, rowIdx int) (jLine, bool) {
	dateStr := cellVal(idx, row, "date")
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return jLine{desc: "date tidak valid"}, false
	}
	amount, _ := strconv.ParseInt(cellVal(idx, row, "amount_rupiah"), 10, 64)
	if amount <= 0 {
		return jLine{desc: "amount_rupiah harus > 0"}, false
	}
	ref := cellVal(idx, row, "reference")
	if ref == "" {
		ref = fmt.Sprintf("IMP-%s-%d", time.Now().Format("20060102"), rowIdx)
	}
	t, _ := time.Parse("2006-01-02", dateStr)
	return jLine{
		rowIdx: rowIdx, date: t, desc: cellVal(idx, row, "description"),
		ref: ref, debit: amount, credit: amount,
		debitAcc: cellVal(idx, row, "debit_account_code"),
		credAcc: cellVal(idx, row, "credit_account_code"),
	}, true
}

func importJournalGroups(ctx context.Context, tenantID string, groups map[string][]jLine, order []string, accountCache map[string]string, importErrs []map[string]any) (int, int) {
	var imported, skipped int
	for _, ref := range order {
		group := groups[ref]
		totalDebit, totalCredit := sumJournalGroup(group)
		if totalDebit != totalCredit {
			skipped += len(group)
			for _, l := range group {
				importErrs = append(importErrs, map[string]any{
					"row": l.rowIdx, "error": fmt.Sprintf("reference %s tidak balance (debit %d != credit %d)", ref, totalDebit, totalCredit),
				})
			}
			continue
		}
		entryID := insertJournalEntry(ctx, tenantID, group, ref)
		if entryID == "" {
			skipped += len(group)
			for _, l := range group {
				importErrs = append(importErrs, map[string]any{"row": l.rowIdx, "error": "Gagal create entry"})
			}
			continue
		}
		allOk := insertJournalLines(ctx, tenantID, entryID, group, accountCache, importErrs)
		if allOk {
			imported += len(group)
		} else {
			skipped += len(group)
			DB.Exec(ctx, `DELETE FROM journal_lines WHERE entry_id = $1`, entryID)
			DB.Exec(ctx, `DELETE FROM journal_entries WHERE id = $1`, entryID)
		}
	}
	return imported, skipped
}

func sumJournalGroup(group []jLine) (debit, credit int64) {
	for _, l := range group {
		debit += l.debit
		credit += l.credit
	}
	return
}

func insertJournalEntry(ctx context.Context, tenantID string, group []jLine, ref string) string {
	var entryID string
	err := DB.QueryRow(ctx, `
		INSERT INTO journal_entries (tenant_id, date, description, reference)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, tenantID, group[0].date, group[0].desc, ref).Scan(&entryID)
	if err != nil {
		return ""
	}
	return entryID
}

func insertJournalLines(ctx context.Context, tenantID, entryID string, group []jLine, accountCache map[string]string, importErrs []map[string]any) bool {
	allOk := true
	for _, l := range group {
		debitID, err1 := resolveAccountID(ctx, tenantID, l.debitAcc, accountCache)
		creditID, err2 := resolveAccountID(ctx, tenantID, l.credAcc, accountCache)
		if err1 != nil || err2 != nil {
			allOk = false
			importErrs = append(importErrs, map[string]any{"row": l.rowIdx, "error": "Akun tidak ditemukan: " + l.debitAcc + " atau " + l.credAcc})
			continue
		}
		if !insertJournalLine(ctx, entryID, debitID, l.debit, 0) {
			allOk = false
			importErrs = append(importErrs, map[string]any{"row": l.rowIdx, "error": "Gagal insert debit line"})
		}
		if !insertJournalLine(ctx, entryID, creditID, 0, l.credit) {
			allOk = false
			importErrs = append(importErrs, map[string]any{"row": l.rowIdx, "error": "Gagal insert credit line"})
		}
	}
	return allOk
}

func resolveAccountID(ctx context.Context, tenantID, code string, cache map[string]string) (string, error) {
	if id, ok := cache[code]; ok {
		return id, nil
	}
	var id string
	err := DB.QueryRow(ctx, `SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2`, tenantID, code).Scan(&id)
	if err != nil {
		return "", err
	}
	cache[code] = id
	return id, nil
}

func insertJournalLine(ctx context.Context, entryID string, accountID string, debit, credit int64) bool {
	_, err := DB.Exec(ctx, `INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, $4)`, entryID, accountID, debit, credit)
	return err == nil
}
