package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"core_project/shared/sdk/response"
)

const (
	layoutDateISOCF     = "2006-01-02"
	layoutDateDisplayCF = "2 January 2006"
	layoutDateTimeFullCF = "2 January 2006 15:04 MST"
	labelDateDescSep    = "%s — %s"
	tidakAdaAktivitas   = "(tidak ada aktivitas)"
	errPDFGenFailedCF   = "PDF generation failed"
)

func handleCashFlowPDF(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(response.XTenantID)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if tenantID == "" || from == "" || to == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID, from, or to"})
		return
	}
	ctx := r.Context()
	businessName := getTenantBusinessNameCF(ctx, tenantID)
	opLines, invLines, finLines, totals, openCash, closeCash := queryCashFlowData(ctx, tenantID, from, to)
	renderCashFlowPDF(w, cashFlowRenderData{
		BusinessName: businessName,
		From:         from,
		To:           to,
		OpLines:      opLines,
		InvLines:     invLines,
		FinLines:     finLines,
		Totals:       totals,
		OpenCash:     openCash,
		CloseCash:    closeCash,
	})
}

func getTenantBusinessNameCF(ctx context.Context, tenantID string) string {
	var name *string
	DB.QueryRow(ctx, `SELECT COALESCE(business_name, name) FROM tenants WHERE id = $1`, tenantID).Scan(&name)
	if name != nil {
		return *name
	}
	return "UMKM WCH"
}

type cfLine struct {
	Date        string
	Description string
	Counterpart string
	Inflow      int64
	Outflow     int64
}

type cashFlowTotals struct {
	TotalIn, TotalOut int64
	OpIn, OpOut       int64
	InvIn, InvOut     int64
	FinIn, FinOut     int64
}

type cashFlowRenderData struct {
	BusinessName          string
	From, To              string
	OpLines, InvLines, FinLines []cfLine
	Totals                cashFlowTotals
	OpenCash, CloseCash   int64
}

func queryCashFlowData(ctx context.Context, tenantID, from, to string) ([]cfLine, []cfLine, []cfLine, cashFlowTotals, int64, int64) {
	var totalInflow, totalOutflow, openingCash int64
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit), 0), COALESCE(SUM(l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date >= $2 AND e.date <= $3
	`, tenantID, from, to).Scan(&totalInflow, &totalOutflow)
	DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit - l.credit), 0)
		FROM journal_lines l
		JOIN journal_entries e ON l.entry_id = e.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e.tenant_id = $1 AND c.type = 'asset' AND c.code IN ('100', '101') AND e.date < $2
	`, tenantID, from).Scan(&openingCash)

	var opIn, opOut, invIn, invOut, finIn, finOut int64
	var opLines, invLines, finLines []cfLine

	rows, err := DB.Query(ctx, `
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
				cCode, cName := queryCounterpartAccount(ctx, tenantID, t, desc)
				line := cfLine{Date: t.Format(layoutDateISOCF), Description: desc, Counterpart: cCode + " " + cName, Inflow: debit, Outflow: credit}
				switch classifyAccountFlow(cCode) {
				case "invest":
					invIn += debit
					invOut += credit
					invLines = append(invLines, line)
				case "finance":
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
	totals := cashFlowTotals{
		TotalIn: totalInflow, TotalOut: totalOutflow,
		OpIn: opIn, OpOut: opOut, InvIn: invIn, InvOut: invOut, FinIn: finIn, FinOut: finOut,
	}
	return opLines, invLines, finLines, totals, openingCash, closingCash
}

func queryCounterpartAccount(ctx context.Context, tenantID string, date time.Time, desc string) (string, string) {
	var cCode, cName string
	_ = DB.QueryRow(ctx, `
		SELECT c.code, c.name
		FROM journal_lines l
		JOIN journal_entries e2 ON l.entry_id = e2.id
		JOIN chart_of_accounts c ON l.account_id = c.id
		WHERE e2.tenant_id = $1 AND e2.date = $2 AND e2.description = $3
		  AND c.code NOT IN ('100', '101')
		LIMIT 1
	`, tenantID, date, desc).Scan(&cCode, &cName)
	return cCode, cName
}

func classifyAccountFlow(cCode string) string {
	if strings.HasPrefix(cCode, "1") && cCode >= "150" && cCode <= "199" {
		return "invest"
	}
	if strings.HasPrefix(cCode, "2") || strings.HasPrefix(cCode, "3") {
		return "finance"
	}
	return "operational"
}

func renderCashFlowPDF(w http.ResponseWriter, d cashFlowRenderData) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 8, d.BusinessName)
	pdf.Ln(8)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 7, "Laporan Arus Kas")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	fromDate, _ := time.Parse(layoutDateISOCF, d.From)
	toDate, _ := time.Parse(layoutDateISOCF, d.To)
	pdf.Cell(0, 5, fmt.Sprintf("Periode: %s – %s", fromDate.Format(layoutDateDisplayCF), toDate.Format(layoutDateDisplayCF)))
	pdf.Ln(5)
	pdf.Cell(0, 5, "Dicetak: "+time.Now().Format(layoutDateTimeFullCF))
	pdf.Ln(8)

	cfRow := func(label string, amount int64) {
		pdf.Cell(110, 5, "  "+label)
		pdf.CellFormat(60, 5, formatIDR(amount), "", 0, "R", false, 0, "")
		pdf.Ln(5)
	}
	cfTotal := func(label string, amount int64) {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(110, 5, label)
		pdf.CellFormat(60, 5, formatIDR(amount), "", 0, "R", false, 0, "")
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 10)
	}
	cfSub := func(title string) {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 5, title)
		pdf.Ln(5)
		pdf.SetFont("Arial", "", 10)
	}
	cfSection := func(title string) {
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 7, title)
		pdf.Ln(7)
		pdf.SetFont("Arial", "", 10)
	}

	cfSection("I. ARUS KAS DARI AKTIVITAS OPERASIONAL")
	cfSub("Kas Masuk:")
	for _, l := range d.OpLines {
		if l.Inflow > 0 {
			cfRow(fmt.Sprintf(labelDateDescSep, l.Date, l.Description), l.Inflow)
		}
	}
	if len(d.OpLines) == 0 {
		cfRow(tidakAdaAktivitas, 0)
	}
	cfTotal("Total Kas Masuk", d.Totals.OpIn)
	cfSub("Kas Keluar:")
	for _, l := range d.OpLines {
		if l.Outflow > 0 {
			cfRow(fmt.Sprintf(labelDateDescSep, l.Date, l.Description), -l.Outflow)
		}
	}
	cfTotal("Total Kas Keluar", -d.Totals.OpOut)
	cfTotal("Arus Kas Operasional", d.Totals.OpIn-d.Totals.OpOut)
	pdf.Ln(3)

	cfSection("II. ARUS KAS DARI AKTIVITAS INVESTASI")
	for _, l := range d.InvLines {
		cfRow(fmt.Sprintf(labelDateDescSep, l.Date, l.Description), l.Inflow-l.Outflow)
	}
	if len(d.InvLines) == 0 {
		cfRow(tidakAdaAktivitas, 0)
	}
	cfTotal("Arus Kas Investasi", d.Totals.InvIn-d.Totals.InvOut)
	pdf.Ln(3)

	cfSection("III. ARUS KAS DARI AKTIVITAS PENDANAAN")
	for _, l := range d.FinLines {
		cfRow(fmt.Sprintf(labelDateDescSep, l.Date, l.Description), l.Inflow-l.Outflow)
	}
	if len(d.FinLines) == 0 {
		cfRow(tidakAdaAktivitas, 0)
	}
	cfTotal("Arus Kas Pendanaan", d.Totals.FinIn-d.Totals.FinOut)
	pdf.Ln(3)

	pdfSummaryCashFlow(pdf, d.Totals.TotalIn-d.Totals.TotalOut, d.OpenCash, d.CloseCash)

	pdf.SetFont("Arial", "I", 8)
	pdf.SetY(-20)
	pdf.Cell(0, 4, "Generated by WCH Platform")
	pdf.Ln(4)
	pdf.Cell(0, 4, fmt.Sprintf("Halaman %d dari {nb}", pdf.PageNo()))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errPDFGenFailedCF})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"cash-flow_%s_%s.pdf\"", d.From, d.To))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

func pdfSummaryCashFlow(pdf *gofpdf.Fpdf, netCash, openingCash, closingCash int64) {
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
}
