package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)


func parseUploadedFile(r *http.Request) (headers []string, rows [][]string, err error) {
	if err = r.ParseMultipartForm(10 << 20); err != nil {
		return nil, nil, fmt.Errorf("file too large or invalid multipart: %w", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, nil, fmt.Errorf("missing 'file' field: %w", err)
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		return nil, nil, fmt.Errorf("file exceeds 10MB limit")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".csv":
		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1
		all, err := reader.ReadAll()
		if err != nil {
			return nil, nil, fmt.Errorf("CSV parse error: %w", err)
		}
		if len(all) < 1 {
			return nil, nil, fmt.Errorf("empty file")
		}
		headers = all[0]
		rows = all[1:]
	case ".xlsx":
		xl, err := excelize.OpenReader(file)
		if err != nil {
			return nil, nil, fmt.Errorf("XLSX open error: %w", err)
		}
		defer xl.Close()
		sheetName := xl.GetSheetName(0)
		allRows, err := xl.GetRows(sheetName)
		if err != nil {
			return nil, nil, fmt.Errorf("XLSX read error: %w", err)
		}
		if len(allRows) < 1 {
			return nil, nil, fmt.Errorf("empty sheet")
		}
		headers = allRows[0]
		rows = allRows[1:]
	default:
		return nil, nil, fmt.Errorf("unsupported file extension: %s (use .csv or .xlsx)", ext)
	}

	if len(rows) > 5000 {
		return nil, nil, fmt.Errorf("file exceeds 5000 rows limit (got %d)", len(rows))
	}
	return headers, rows, nil
}

func indexHeaders(headers []string) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func writeFileResponse(w http.ResponseWriter, filename, format string, headers []string, rows [][]string) {
	if format == "xlsx" {
		xl := excelize.NewFile()
		sheet := "Sheet1"
		xl.SetSheetName("Sheet1", sheet)
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			xl.SetCellValue(sheet, cell, h)
		}
		for r, row := range rows {
			for c, val := range row {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
				xl.SetCellValue(sheet, cell, val)
			}
		}
		var buf bytes.Buffer
		xl.Write(&buf)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xlsx\"", filename))
		w.Write(buf.Bytes())
		return
	}
	// default: CSV
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", filename))
	cw := csv.NewWriter(w)
	cw.Write(headers)
	for _, row := range rows {
		cw.Write(row)
	}
	cw.Flush()
}

func handleExportProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "xlsx"
	}
	headers := []string{"name", "sku", "category", "price_rupiah", "stock", "description", "image_url"}
	var rows [][]string
	if DB != nil {
		dbRows, err := DB.Query(r.Context(), `SELECT name, sku, category, price_rupiah, stock, description, image_url FROM products WHERE tenant_id = $1 ORDER BY name`, tenantID)
		if err == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var name, sku, category, desc, img *string
				var price, stock int64
				if err := dbRows.Scan(&name, &sku, &category, &price, &stock, &desc, &img); err == nil {
					rows = append(rows, []string{
						derefStr(name), derefStr(sku), derefStr(category),
						strconv.FormatInt(price, 10), strconv.FormatInt(stock, 10),
						derefStr(desc), derefStr(img),
					})
				}
			}
		}
	}
	writeFileResponse(w, "products", format, headers, rows)
}

func handleExportContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "xlsx"
	}
	headers := []string{"name", "phone", "email", "role", "notes"}
	var rows [][]string
	if DB != nil {
		dbRows, err := DB.Query(r.Context(), `SELECT name, phone_number, email, role, notes FROM tenant_contacts WHERE tenant_id = $1 ORDER BY name`, tenantID)
		if err == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var name, phone, email, role, notes *string
				if err := dbRows.Scan(&name, &phone, &email, &role, &notes); err == nil {
					rows = append(rows, []string{derefStr(name), derefStr(phone), derefStr(email), derefStr(role), derefStr(notes)})
				}
			}
		}
	}
	writeFileResponse(w, "contacts", format, headers, rows)
}

func handleExportJournal(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if tenantID == "" || from == "" || to == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID, from, or to"})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "xlsx"
	}
	headers := []string{"date", "description", "reference", "account_code", "account_name", "debit", "credit"}
	var rows [][]string
	if DB != nil {
		dbRows, err := DB.Query(r.Context(), `
			SELECT e.date, e.description, e.reference, c.code, c.name, l.debit, l.credit
			FROM journal_lines l
			JOIN journal_entries e ON l.entry_id = e.id
			JOIN chart_of_accounts c ON l.account_id = c.id
			WHERE e.tenant_id = $1 AND e.date >= $2 AND e.date <= $3
			ORDER BY e.date ASC, e.created_at ASC
		`, tenantID, from, to)
		if err == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var date time.Time
				var desc, ref, code, name string
				var debit, credit int64
				if err := dbRows.Scan(&date, &desc, &ref, &code, &name, &debit, &credit); err == nil {
					rows = append(rows, []string{
						date.Format("2006-01-02"), desc, ref, code, name,
						strconv.FormatInt(debit, 10), strconv.FormatInt(credit, 10),
					})
				}
			}
		}
	}
	writeFileResponse(w, fmt.Sprintf("journal_%s_%s", from, to), format, headers, rows)
}

func handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	entity := r.URL.Query().Get("entity")
	templates := map[string][][]string{
		"products": {
			{"name", "sku", "category", "price_rupiah", "stock", "description", "image_url"},
			{"Contoh Produk", "SKU-001", "Makanan", "15000", "50", "Contoh deskripsi", ""},
		},
		"contacts": {
			{"name", "phone", "email", "role", "notes"},
			{"Contoh Pelanggan", "6281234567890", "contoh@email.com", "customer", ""},
		},
		"journal": {
			{"date", "description", "reference", "debit_account_code", "credit_account_code", "amount_rupiah"},
			{"2026-01-15", "Penjualan tunai", "BATCH-001", "100", "400", "100000"},
		},
	}
	tmpl, ok := templates[entity]
	if !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "entity harus products|contacts|journal"})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	writeFileResponse(w, "template_"+entity, format, tmpl[0], tmpl[1:])
}
