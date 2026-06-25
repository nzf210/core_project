package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func handleCashFlowPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if tenantID == "" || from == "" || to == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID, from, or to"})
		return
	}

	businessName := "UMKM WCH"
	if DB != nil {
		var name *string
		DB.QueryRow(r.Context(), `SELECT COALESCE(business_name, name) FROM tenants WHERE id = $1`, tenantID).Scan(&name)
		if name != nil {
			businessName = *name
		}
	}

	var totalInflow, totalOutflow, openingCash int64
	DB.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(l.debit), 0), COALESCE(SUM(l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date >= $2 AND e.date <= $3
	`, tenantID, from, to).Scan(&totalInflow, &totalOutflow)
	DB.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(l.debit - l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date < $2
	`, tenantID, from).Scan(&openingCash)

	type cfLine struct {
		Date        string
		Description string
		Counterpart string
		Inflow      int64
		Outflow     int64
	}
	var opIn, opOut, invIn, invOut, finIn, finOut int64
	var opLines, invLines, finLines []cfLine

	rows, err := DB.Query(r.Context(), `
		SELECT e.date, e.description, l.debit, l.credit
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101')
		  AND e.date >= $2 AND e.date <= $3
		ORDER BY e.date ASC
	`, tenantID, from, to)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t time.Time
			var desc string
			var debit, credit int64
			if err := rows.Scan(&t, &desc, &debit, &credit); err == nil {
				var cCode, cName string
				_ = DB.QueryRow(r.Context(), `
					SELECT c.code, c.name
					FROM journal_lines l
					JOIN journal_entries e2 ON l.entry_id = e2.id
					JOIN chart_of_accounts c ON l.account_id = c.id
					WHERE e2.tenant_id = $1 AND e2.date = $2 AND e2.description = $3
					  AND c.code NOT IN ('100', '101')
					LIMIT 1
				`, tenantID, t, desc).Scan(&cCode, &cName)

				counterpartLabel := cCode + " " + cName
				line := cfLine{Date: t.Format("2006-01-02"), Description: desc, Counterpart: counterpartLabel, Inflow: debit, Outflow: credit}
				switch {
				case strings.HasPrefix(cCode, "1") && cCode >= "150" && cCode <= "199":
					invIn += debit
					invOut += credit
					invLines = append(invLines, line)
				case strings.HasPrefix(cCode, "2") || strings.HasPrefix(cCode, "3"):
					finIn += debit
					finOut += credit
					finLines = append(finLines, line)
				default:
					opIn += debit
					opOut += credit
					opLines = append(opLines, line)
				}
			}
		}
	}

	netCash := totalInflow - totalOutflow
	closingCash := openingCash + netCash

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 8, businessName)
	pdf.Ln(8)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 7, "Laporan Arus Kas")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	fromDate, _ := time.Parse("2006-01-02", from)
	toDate, _ := time.Parse("2006-01-02", to)
	pdf.Cell(0, 5, fmt.Sprintf("Periode: %s – %s", fromDate.Format("2 January 2006"), toDate.Format("2 January 2006")))
	pdf.Ln(5)
	pdf.Cell(0, 5, "Dicetak: "+time.Now().Format("2 January 2006 15:04 MST"))
	pdf.Ln(8)

	sectionHeader := func(title string) {
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 7, title)
		pdf.Ln(7)
		pdf.SetFont("Arial", "", 10)
	}
	subHeader := func(title string) {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 5, title)
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 10)
	}
	row := func(label string, amount int64) {
		pdf.Cell(110, 5, "  "+label)
		pdf.CellFormat(60, 5, formatIDR(amount), "", 0, "R", false, 0, "")
		pdf.Ln(5)
	}
	totalRow := func(label string, amount int64) {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(110, 5, label)
		pdf.CellFormat(60, 5, formatIDR(amount), "", 0, "R", false, 0, "")
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 10)
	}

	sectionHeader("I. ARUS KAS DARI AKTIVITAS OPERASIONAL")
	subHeader("Kas Masuk:")
	for _, l := range opLines {
		if l.Inflow > 0 {
			row(fmt.Sprintf("%s — %s", l.Date, l.Description), l.Inflow)
		}
	}
	if len(opLines) == 0 {
		row("(tidak ada aktivitas)", 0)
	}
	totalRow("Total Kas Masuk", opIn)
	subHeader("Kas Keluar:")
	for _, l := range opLines {
		if l.Outflow > 0 {
			row(fmt.Sprintf("%s — %s", l.Date, l.Description), -l.Outflow)
		}
	}
	totalRow("Total Kas Keluar", -opOut)
	totalRow("Arus Kas Operasional", opIn-opOut)
	pdf.Ln(3)

	sectionHeader("II. ARUS KAS DARI AKTIVITAS INVESTASI")
	for _, l := range invLines {
		row(fmt.Sprintf("%s — %s", l.Date, l.Description), l.Inflow-l.Outflow)
	}
	if len(invLines) == 0 {
		row("(tidak ada aktivitas)", 0)
	}
	totalRow("Arus Kas Investasi", invIn-invOut)
	pdf.Ln(3)

	sectionHeader("III. ARUS KAS DARI AKTIVITAS PENDANAAN")
	for _, l := range finLines {
		row(fmt.Sprintf("%s — %s", l.Date, l.Description), l.Inflow-l.Outflow)
	}
	if len(finLines) == 0 {
		row("(tidak ada aktivitas)", 0)
	}
	totalRow("Arus Kas Pendanaan", finIn-finOut)
	pdf.Ln(3)

	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(0, 7, "RINGKASAN")
	pdf.Ln(7)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(110, 6, "Kenaikan/(Penurunan) Bersih Kas")
	pdf.CellFormat(60, 6, formatIDR(netCash), "", 0, "R", false, 0, "")
	pdf.Ln(6)
	pdf.Cell(110, 6, "Kas Awal Periode")
	pdf.CellFormat(60, 6, formatIDR(openingCash), "", 0, "R", false, 0, "")
	pdf.Ln(6)
	pdf.Cell(110, 6, "Kas Akhir Periode")
	pdf.CellFormat(60, 6, formatIDR(closingCash), "", 0, "R", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Arial", "I", 8)
	pdf.SetY(-20)
	pdf.Cell(0, 4, "Generated by WCH Platform")
	pdf.Ln(4)
	pdf.Cell(0, 4, fmt.Sprintf("Halaman %d dari {nb}", pdf.PageNo()))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "PDF generation failed: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"cash-flow_%s_%s.pdf\"", from, to))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}
