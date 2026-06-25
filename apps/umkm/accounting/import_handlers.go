package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)




func handleImportProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	idx := indexHeaders(headers)
	requiredCols := []string{"name", "sku", "price_rupiah"}
	for _, c := range requiredCols {
		if _, ok := idx[c]; !ok {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Kolom wajib hilang: " + c})
			return
		}
	}
	var imported, skipped int
	var errs []map[string]interface{}
	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		get := func(col string) string {
			i, ok := idx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		name := get("name")
		sku := get("sku")
		priceStr := get("price_rupiah")
		if name == "" || sku == "" {
			skipped++
			errs = append(errs, map[string]interface{}{"row": rowIdx, "error": "name atau sku kosong"})
			continue
		}
		price, _ := strconv.ParseInt(priceStr, 10, 64)
		if price < 0 {
			skipped++
			errs = append(errs, map[string]interface{}{"row": rowIdx, "error": "price_rupiah tidak valid"})
			continue
		}
		stock, _ := strconv.ParseInt(get("stock"), 10, 64)
		category := get("category")
		desc := get("description")
		img := get("image_url")
		_, err := DB.Exec(r.Context(), `
			INSERT INTO products (tenant_id, name, sku, category, price_rupiah, stock, description, image_url)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (tenant_id, sku) DO UPDATE SET
				name = EXCLUDED.name, category = EXCLUDED.category,
				price_rupiah = EXCLUDED.price_rupiah, stock = EXCLUDED.stock,
				description = EXCLUDED.description, image_url = EXCLUDED.image_url
		`, tenantID, name, sku, nullString(category), price, stock, nullString(desc), nullString(img))
		if err != nil {
			skipped++
			errs = append(errs, map[string]interface{}{"row": rowIdx, "error": err.Error()})
		} else {
			imported++
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"imported": imported, "skipped": skipped, "errors": errs}})
}

func handleImportContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	idx := indexHeaders(headers)
	if _, ok := idx["phone"]; !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Kolom wajib hilang: phone"})
		return
	}
	var imported, skipped int
	var errs []map[string]interface{}
	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		get := func(col string) string {
			i, ok := idx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		phone := get("phone")
		if phone == "" {
			skipped++
			errs = append(errs, map[string]interface{}{"row": rowIdx, "error": "phone kosong"})
			continue
		}
		role := get("role")
		if role == "" {
			role = "customer"
		}
		_, err := DB.Exec(r.Context(), `
			INSERT INTO tenant_contacts (tenant_id, name, phone_number, email, role, notes)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, phone_number) DO UPDATE SET
				name = EXCLUDED.name, email = EXCLUDED.email,
				role = EXCLUDED.role, notes = EXCLUDED.notes
		`, tenantID, nullString(get("name")), phone, nullString(get("email")), role, nullString(get("notes")))
		if err != nil {
			skipped++
			errs = append(errs, map[string]interface{}{"row": rowIdx, "error": err.Error()})
		} else {
			imported++
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"imported": imported, "skipped": skipped, "errors": errs}})
}

func handleImportJournal(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	headers, rows, err := parseUploadedFile(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}
	idx := indexHeaders(headers)
	for _, c := range []string{"date", "description", "debit_account_code", "credit_account_code", "amount_rupiah"} {
		if _, ok := idx[c]; !ok {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Kolom wajib hilang: " + c})
			return
		}
	}
	type jLine struct {
		rowIdx   int
		date     time.Time
		desc     string
		ref      string
		debit    int64
		credit   int64
		debitAcc string
		credAcc string
	}
	groups := map[string][]jLine{}
	order := []string{}
	seenRef := map[string]bool{}
	var imported, skipped int
	var importErrs []map[string]interface{}
	for rowNum, row := range rows {
		rowIdx := rowNum + 2
		get := func(col string) string {
			i, ok := idx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		dateStr := get("date")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			skipped++
			importErrs = append(importErrs, map[string]interface{}{"row": rowIdx, "error": "date tidak valid"})
			continue
		}
		amount, _ := strconv.ParseInt(get("amount_rupiah"), 10, 64)
		if amount <= 0 {
			skipped++
			importErrs = append(importErrs, map[string]interface{}{"row": rowIdx, "error": "amount_rupiah harus > 0"})
			continue
		}
		ref := get("reference")
		if ref == "" {
			ref = fmt.Sprintf("IMP-%s-%d", time.Now().Format("20060102"), rowIdx)
		}
		groups[ref] = append(groups[ref], jLine{
			rowIdx: rowIdx, date: t, desc: get("description"),
			ref: ref, debit: amount, credit: amount,
			debitAcc: get("debit_account_code"), credAcc: get("credit_account_code"),
		})
		if _, seen := seenRef[ref]; !seen {
			seenRef[ref] = true
			order = append(order, ref)
		}
	}
	accountCache := map[string]string{}
	resolveAcc := func(ctx context.Context, code string) (string, error) {
		if id, ok := accountCache[code]; ok {
			return id, nil
		}
		var id string
		err := DB.QueryRow(ctx, `SELECT id FROM chart_of_accounts WHERE tenant_id = $1 AND code = $2`, tenantID, code).Scan(&id)
		if err != nil {
			return "", err
		}
		accountCache[code] = id
		return id, nil
	}
	for _, ref := range order {
		group := groups[ref]
		var totalDebit, totalCredit int64
		for _, l := range group {
			totalDebit += l.debit
			totalCredit += l.credit
		}
		if totalDebit != totalCredit {
			skipped += len(group)
			for _, l := range group {
				importErrs = append(importErrs, map[string]interface{}{
					"row": l.rowIdx, "error": fmt.Sprintf("reference %s tidak balance (debit %d != credit %d)", ref, totalDebit, totalCredit),
				})
			}
			continue
		}
		entryDesc := group[0].desc
		entryDate := group[0].date
		var entryID string
		err := DB.QueryRow(r.Context(), `
			INSERT INTO journal_entries (tenant_id, date, description, reference)
			VALUES ($1, $2, $3, $4) RETURNING id
		`, tenantID, entryDate, entryDesc, ref).Scan(&entryID)
		if err != nil {
			skipped += len(group)
			for _, l := range group {
				importErrs = append(importErrs, map[string]interface{}{"row": l.rowIdx, "error": "Gagal create entry: " + err.Error()})
			}
			continue
		}
		allOk := true
		for _, l := range group {
			debitID, err1 := resolveAcc(r.Context(), l.debitAcc)
			creditID, err2 := resolveAcc(r.Context(), l.credAcc)
			if err1 != nil || err2 != nil {
				allOk = false
				importErrs = append(importErrs, map[string]interface{}{"row": l.rowIdx, "error": "Akun tidak ditemukan: " + l.debitAcc + " atau " + l.credAcc})
				continue
			}
			_, err = DB.Exec(r.Context(), `INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, $3, 0)`, entryID, debitID, l.debit)
			if err != nil {
				allOk = false
				importErrs = append(importErrs, map[string]interface{}{"row": l.rowIdx, "error": err.Error()})
				continue
			}
			_, err = DB.Exec(r.Context(), `INSERT INTO journal_lines (entry_id, account_id, debit, credit) VALUES ($1, $2, 0, $3)`, entryID, creditID, l.credit)
			if err != nil {
				allOk = false
				importErrs = append(importErrs, map[string]interface{}{"row": l.rowIdx, "error": err.Error()})
			}
		}
		if allOk {
			imported += len(group)
		} else {
			skipped += len(group)
			DB.Exec(r.Context(), `DELETE FROM journal_lines WHERE entry_id = $1`, entryID)
			DB.Exec(r.Context(), `DELETE FROM journal_entries WHERE id = $1`, entryID)
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"imported": imported, "skipped": skipped, "errors": importErrs}})
}
