package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	headerTenantPrompt = "X-Tenant-ID"
	layoutDatePrompt   = "2006-01-02"
	headerContentType  = "Content-Type"
)

func buildSystemPrompt(ctx context.Context, tenantID, tenantName, message, role string, cfg *chatConfigCache) string {
	systemPrompt := baseSystemPrompt(tenantName, role)

	if cfg != nil {
		systemPrompt = applyPromptOverrides(systemPrompt, cfg)
	}

	systemPrompt = enrichWithCOA(ctx, tenantID, role, systemPrompt)
	systemPrompt = enrichWithProducts(ctx, tenantID, systemPrompt)
	systemPrompt = enrichWithFAQs(ctx, tenantID, systemPrompt)
	systemPrompt += instructionBlock()

	msgLower := strings.ToLower(message)
	systemPrompt = applyFinancialReports(ctx, tenantID, role, msgLower, systemPrompt)

	return systemPrompt
}

func baseSystemPrompt(tenantName, role string) string {
	switch role {
	case "customer":
		return fmt.Sprintf("Anda adalah asisten virtual (Customer Service) untuk toko bernama '%s'. Jawab dengan ramah dan sopan kepada pelanggan. Jika pelanggan menanyakan daftar harga barang/produk, berikan harga sesuai katalog. Jika pelanggan marah, ada keluhan komplain, atau secara spesifik meminta bicara dengan admin/pemilik, Anda WAJIB merespon dengan mengancam awali pesan Anda menggunakan format `[FORWARD_TO_ADMIN] {Isi keluhan/pesan pelanggan agar admin tahu}`. Contoh: `[FORWARD_TO_ADMIN] Tolong cek keluhan pelanggan ini mengenai barang rusak.`", tenantName)
	case "kasir", "staff":
		return fmt.Sprintf("Anda adalah asisten Kasir untuk toko '%s' (UMKM WCH). Tugas Anda HANYA membantu mencatat transaksi masuk/keluar harian dan menghitung jumlah kas hari ini. \n\nPERINGATAN: DILARANG KERAS memberikan informasi rahasia seperti laporan Laba/Rugi, Modal, atau Total Neraca jika ditanya. Jika ditanya soal Laba/Rugi, katakan bahwa Anda tidak memiliki hak akses untuk itu.", tenantName)
	case "owner", "admin", "user":
		return fmt.Sprintf("Anda adalah asisten keuangan pintar untuk toko '%s' (UMKM WCH). Anda memiliki akses penuh ke laporan keuangan dan operasional. Jawab dalam bahasa Indonesia yang ramah.", tenantName)
	default:
		return fmt.Sprintf("Anda adalah asisten toko '%s'.", tenantName)
	}
}

func applyPromptOverrides(prompt string, cfg *chatConfigCache) string {
	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		return cfg.SystemPrompt
	}
	hint := languageHint(cfg.Language) + toneHint(cfg.Tone)
	if hint != "" {
		prompt += hint
	}
	return prompt
}

func languageHint(lang string) string {
	switch lang {
	case "en":
		return " Respond in English."
	case "id":
		return " Jawab dalam bahasa Indonesia."
	}
	return ""
}

func toneHint(tone string) string {
	switch tone {
	case "formal":
		return " Gunakan nada formal dan profesional."
	case "casual":
		return " Gunakan nada santai dan akrab."
	case "professional":
		return " Gunakan nada profesional dan solutif."
	case "friendly":
		return " Gunakan nada ramah, hangat, dan bersahabat."
	}
	return ""
}

func enrichWithCOA(ctx context.Context, tenantID, role, prompt string) string {
	if role == "customer" {
		return prompt
	}
	coaReq, _ := http.NewRequestWithContext(ctx, "GET", AccountingURL+"/accounts", nil)
	coaReq.Header.Set(headerTenantPrompt, tenantID)
	resp, err := http.DefaultClient.Do(coaReq)
	if err != nil {
		return prompt
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return prompt + "\n\nData Chart of Accounts (COA) tenant ini (format JSON):\n" + string(body)
}

func enrichWithProducts(ctx context.Context, tenantID, prompt string) string {
	prodReq, _ := http.NewRequestWithContext(ctx, "GET", AccountingURL+"/products", nil)
	prodReq.Header.Set(headerTenantPrompt, tenantID)
	resp, err := http.DefaultClient.Do(prodReq)
	if err != nil {
		return prompt
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return prompt + "\n\nKatalog Produk & Harga (format JSON):\n" + string(body) + "\n\nGunakan data katalog ini jika pengguna bertanya tentang produk atau harga."
}

func enrichWithFAQs(ctx context.Context, tenantID, prompt string) string {
	if DB == nil {
		return prompt
	}
	rows, err := DB.Query(ctx, "SELECT question, answer FROM tenant_faqs WHERE tenant_id = $1", tenantID)
	if err != nil {
		return prompt
	}
	defer rows.Close()
	prompt += "\n\nDaftar FAQ (Tanya Jawab Umum) Toko ini:\n"
	hasFaq := false
	for rows.Next() {
		var q, a string
		if rows.Scan(&q, &a) == nil {
			prompt += fmt.Sprintf("Q: %s\nA: %s\n", q, a)
			hasFaq = true
		}
	}
	if !hasFaq {
		prompt += "(Belum ada FAQ khusus)\n"
	}
	return prompt
}

func instructionBlock() string {
	return `
Jika user bermaksud mencatat PENGELUARAN (expense) secara spesifik (misal bayar listrik, beli bahan, dll), Anda WAJIB menyertakan blok kode JSON khusus dengan format:
` + "```json:expense\n" + `{
  "date": "2026-05-24",
  "description": "Pembayaran operasional",
  "amount": 100000,
  "expense_coa": "500",
  "payment_coa": "100",
  "line_items": [
    {"name": "Beli barang A", "amount": 100000}
  ]
}` + "\n```\n" + `
Untuk pencatatan transaksi selain pengeluaran (misal pemasukan, penjualan), gunakan format standar:
` + "```json\n" + `{
  "date": "2026-05-24",
  "description": "Catatan singkat",
  "reference": "AUTO",
  "lines": [
    {"account_id": "ID_AKUN_DEBIT", "debit": 100000, "credit": 0},
    {"account_id": "ID_AKUN_KREDIT", "debit": 0, "credit": 100000}
  ]
}` + "\n```\n" + `PENTING: Gunakan tipe data integer (angka bulat).

Jika ada pertanyaan yang TIDAK BISA ANDA JAWAB (tidak ada di FAQ, produk, atau wewenang Anda), DILARANG mengarang jawaban. Anda WAJIB membalas dengan format:
[FORWARD_TO_ADMIN] {Isi keluhan/pertanyaan user secara ringkas}
`
}

func applyFinancialReports(ctx context.Context, tenantID, role, msgLower, prompt string) string {
	dateRange := dateRangeArg()
	switch {
	case strings.Contains(msgLower, "laba") || strings.Contains(msgLower, "rugi") || strings.Contains(msgLower, "pendapatan"):
		if role == "kasir" || role == "staff" {
			return prompt + "\n\n[SISTEM]: Akses ke Laporan Laba/Rugi ditolak untuk role Kasir/Staff."
		}
		return prompt + fetchFinancialReport(ctx, tenantID, "/reports/income-statement", dateRange)

	case strings.Contains(msgLower, "kas") || strings.Contains(msgLower, "saldo") || strings.Contains(msgLower, "uang"):
		return prompt + fetchFinancialReport(ctx, tenantID, "/reports/cash-flow", dateRange)

	case strings.Contains(msgLower, "aset") || strings.Contains(msgLower, "hutang") || strings.Contains(msgLower, "modal") || strings.Contains(msgLower, "neraca"):
		if role == "kasir" || role == "staff" {
			return prompt + "\n\n[SISTEM]: Akses ke Neraca Keuangan ditolak untuk role Kasir/Staff."
		}
		return prompt + fetchFinancialReport(ctx, tenantID, "/reports/balance-sheet", dateRange)
	}
	return prompt
}

func dateRangeArg() string {
	from := time.Now().AddDate(0, -1, 0).Format(layoutDatePrompt)
	to := time.Now().Format(layoutDatePrompt)
	return fmt.Sprintf("from=%s&to=%s", from, to)
}

func fetchFinancialReport(ctx context.Context, tenantID, path, query string) string {
	url := fmt.Sprintf("%s%s?%s", AccountingURL, path, query)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set(headerTenantPrompt, tenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return fmt.Sprintf("\n\nData %s aktual: %s", pathLabel(path), string(body))
}

func pathLabel(path string) string {
	switch path {
	case "/reports/income-statement":
		return "Laba/Rugi"
	case "/reports/cash-flow":
		return "Arus Kas"
	case "/reports/balance-sheet":
		return "Neraca (Balance Sheet)"
	}
	return path
}

func processAIAnswer(ctx context.Context, tenantID, answer, sender, _ string) string {
	if hasForward := handleAdminForward(ctx, tenantID, sender, answer); hasForward {
		return cleanForwardedAnswer(answer)
	}
	if clean := processExpenseBlock(ctx, tenantID, answer); clean != "" {
		return clean
	}
	if clean := processTransactionBlock(ctx, tenantID, answer); clean != "" {
		return clean
	}
	return answer
}

func handleAdminForward(ctx context.Context, tenantID, sender, answer string) bool {
	if !strings.Contains(answer, "[FORWARD_TO_ADMIN]") {
		return false
	}
	msgToAdmin := extractForwardMessage(answer)

	go func() {
		if DB == nil {
			return
		}
		forwarders := collectForwarders(ctx, tenantID)
		if len(forwarders) == 0 {
			return
		}
		for _, phone := range forwarders {
			sendForwardMessage(ctx, tenantID, normalizePhone(phone), sender, msgToAdmin)
		}
	}()
	return true
}

func extractForwardMessage(answer string) string {
	idx := strings.Index(answer, "[FORWARD_TO_ADMIN]")
	if idx == -1 {
		return ""
	}
	msg := strings.TrimSpace(answer[idx+18:])
	if e := strings.Index(msg, "\n"); e != -1 {
		msg = msg[:e]
	}
	return msg
}

func collectForwarders(ctx context.Context, tenantID string) []string {
	var forwarders []string
	rows, _ := DB.Query(ctx, "SELECT phone_number FROM tenant_forwarders WHERE tenant_id = $1", tenantID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var phone string
			if rows.Scan(&phone) == nil {
				forwarders = append(forwarders, phone)
			}
		}
	}
	if len(forwarders) == 0 {
		var ownerPhone string
		if DB.QueryRow(ctx, "SELECT phone_number FROM users WHERE tenant_id = $1 AND role = 'owner' LIMIT 1", tenantID).Scan(&ownerPhone) == nil && ownerPhone != "" {
			forwarders = append(forwarders, ownerPhone)
		}
	}
	return forwarders
}

func sendForwardMessage(_ context.Context, tenantID, phone, sender, msg string) {
	data := url.Values{}
	data.Set("target", phone)
	data.Set("message", fmt.Sprintf("⚠️ *ESKALASI OTOMATIS DARI BOT* ⚠️\nPelanggan dengan nomor %s memerlukan bantuan.\n\nKonteks: %s", sender, msg))
	data.Set("tenant_id", tenantID)
	req, _ := http.NewRequest("POST", waSendURL(), strings.NewReader(data.Encode()))
	req.Header.Set(headerContentType, headerContentTypeForm)
	http.DefaultClient.Do(req)
}

func cleanForwardedAnswer(_ string) string {
	return "Mohon ditunggu ya, pesan Anda sedang kami teruskan ke Admin."
}

func processExpenseBlock(ctx context.Context, tenantID, answer string) string {
	idx := strings.Index(answer, "```json:expense")
	if idx == -1 {
		return ""
	}
	block := answer[idx+15:]
	end := strings.Index(block, "```")
	if end == -1 {
		return ""
	}
	jsonStr := block[:end]
	cleanMsg := stripBlockFrom(answer, idx, 15+end+3)

	txReq, _ := http.NewRequestWithContext(ctx, "POST", AccountingURL+"/expenses", strings.NewReader(jsonStr))
	txReq.Header.Set(headerTenantPrompt, tenantID)
	txReq.Header.Set(headerContentType, "application/json")
	txResp, err := http.DefaultClient.Do(txReq)

	if err == nil && txResp.StatusCode == http.StatusOK {
		return strings.TrimSpace(cleanMsg) + "\n\n✅ Pengeluaran telah berhasil dicatat ke sistem akuntansi Anda!"
	}
	return strings.TrimSpace(cleanMsg) + "\n\n❌ Gagal mencatat pengeluaran."
}

func processTransactionBlock(ctx context.Context, tenantID, answer string) string {
	idx := strings.Index(answer, "```json")
	if idx == -1 || strings.HasPrefix(answer[idx:], "```json:expense") {
		return ""
	}
	block := answer[idx+7:]
	end := strings.Index(block, "```")
	if end == -1 {
		return ""
	}
	jsonStr := block[:end]
	cleanMsg := stripBlockFrom(answer, idx, 7+end+3)

	txReq, _ := http.NewRequestWithContext(ctx, "POST", AccountingURL+"/transactions", strings.NewReader(jsonStr))
	txReq.Header.Set(headerTenantPrompt, tenantID)
	txReq.Header.Set(headerContentType, "application/json")
	txResp, err := http.DefaultClient.Do(txReq)

	if err == nil && txResp.StatusCode == http.StatusOK {
		return strings.TrimSpace(cleanMsg) + "\n\n✅ Transaksi telah berhasil dicatat ke sistem akuntansi Anda!"
	}
	return strings.TrimSpace(cleanMsg) + "\n\n❌ Gagal mencatat transaksi."
}

func stripBlockFrom(answer string, start, end int) string {
	if start < 0 || end > len(answer) || start >= end {
		return answer
	}
	return answer[:start] + answer[end:]
}
