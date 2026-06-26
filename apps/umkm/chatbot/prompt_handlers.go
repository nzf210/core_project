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

func buildSystemPrompt(ctx context.Context, tenantID, tenantName, message, role string, cfg *chatConfigCache) string {
	var systemPrompt string
	switch role {
	case "customer":
		systemPrompt = fmt.Sprintf("Anda adalah asisten virtual (Customer Service) untuk toko bernama '%s'. Jawab dengan ramah dan sopan kepada pelanggan. Jika pelanggan menanyakan daftar harga barang/produk, berikan harga sesuai katalog. Jika pelanggan marah, ada keluhan komplain, atau secara spesifik meminta bicara dengan admin/pemilik, Anda WAJIB merespon dengan mengawali pesan Anda menggunakan format `[FORWARD_TO_ADMIN] {Isi keluhan/pesan pelanggan agar admin tahu}`. Contoh: `[FORWARD_TO_ADMIN] Tolong cek keluhan pelanggan ini mengenai barang rusak.`", tenantName)
	case "kasir", "staff":
		systemPrompt = fmt.Sprintf("Anda adalah asisten Kasir untuk toko '%s' (UMKM WCH). Tugas Anda HANYA membantu mencatat transaksi masuk/keluar harian dan menghitung jumlah kas hari ini. \n\nPERINGATAN: DILARANG KERAS memberikan informasi rahasia seperti laporan Laba/Rugi, Modal, atau Total Neraca jika ditanya. Jika ditanya soal Laba/Rugi, katakan bahwa Anda tidak memiliki hak akses untuk itu.", tenantName)
	case "owner", "admin", "user":
		systemPrompt = fmt.Sprintf("Anda adalah asisten keuangan pintar untuk toko '%s' (UMKM WCH). Anda memiliki akses penuh ke laporan keuangan dan operasional. Jawab dalam bahasa Indonesia yang ramah.", tenantName)
	default:
		systemPrompt = fmt.Sprintf("Anda adalah asisten toko '%s'.", tenantName)
	}

	// F020: Apply per-tenant config overrides (language, tone, custom prompt)
	if cfg != nil {
		if strings.TrimSpace(cfg.SystemPrompt) == "" {
			// No custom prompt — augment with language + tone hints
			langHint := ""
			switch cfg.Language {
			case "en":
				langHint = " Respond in English."
			case "id":
				langHint = " Jawab dalam bahasa Indonesia."
			}
			toneHint := ""
			switch cfg.Tone {
			case "formal":
				toneHint = " Gunakan nada formal dan profesional."
			case "casual":
				toneHint = " Gunakan nada santai dan akrab."
			case "professional":
				toneHint = " Gunakan nada profesional dan solutif."
			case "friendly":
				toneHint = " Gunakan nada ramah, hangat, dan bersahabat."
			}
			if langHint != "" || toneHint != "" {
				systemPrompt += langHint + toneHint
			}
		} else {
			// Custom prompt replaces the base entirely
			systemPrompt = cfg.SystemPrompt
		}
	}

	// Fetch COA if NOT a customer
	if role != "customer" {
		coaURL := AccountingURL + "/accounts"
		coaReq, _ := http.NewRequestWithContext(ctx, "GET", coaURL, nil)
		coaReq.Header.Set("X-Tenant-ID", tenantID)
		coaResp, err := http.DefaultClient.Do(coaReq)
		if err == nil {
			defer coaResp.Body.Close()
			coaBody, _ := io.ReadAll(coaResp.Body)
			systemPrompt += "\n\nData Chart of Accounts (COA) tenant ini (format JSON):\n" + string(coaBody)
		}
	}

	// Fetch Products (Catalog) for EVERYONE
	productsURL := AccountingURL + "/products"
	prodReq, _ := http.NewRequestWithContext(ctx, "GET", productsURL, nil)
	prodReq.Header.Set("X-Tenant-ID", tenantID)
	prodResp, err := http.DefaultClient.Do(prodReq)
	if err == nil {
		defer prodResp.Body.Close()
		prodBody, _ := io.ReadAll(prodResp.Body)
		systemPrompt += "\n\nKatalog Produk & Harga (format JSON):\n" + string(prodBody) + "\n\nGunakan data katalog ini jika pengguna bertanya tentang produk atau harga."
	}

	// Fetch FAQs
	if DB != nil {
		rows, err := DB.Query(ctx, "SELECT question, answer FROM tenant_faqs WHERE tenant_id = $1", tenantID)
		if err == nil {
			defer rows.Close()
			systemPrompt += "\n\nDaftar FAQ (Tanya Jawab Umum) Toko ini:\n"
			hasFaq := false
			for rows.Next() {
				var q, a string
				if err := rows.Scan(&q, &a); err == nil {
					systemPrompt += fmt.Sprintf("Q: %s\nA: %s\n", q, a)
					hasFaq = true
				}
			}
			if !hasFaq {
				systemPrompt += "(Belum ada FAQ khusus)\n"
			}
		}
	}

	systemPrompt += `
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

	// RAG Logic
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "laba") || strings.Contains(msgLower, "rugi") || strings.Contains(msgLower, "pendapatan") {
		if role == "kasir" || role == "staff" {
			systemPrompt += "\n\n[SISTEM]: Akses ke Laporan Laba/Rugi ditolak untuk role Kasir/Staff."
		} else {
			from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
			to := time.Now().Format("2006-01-02")
			url := fmt.Sprintf("%s/reports/income-statement?from=%s&to=%s", AccountingURL, from, to)
			httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			httpReq.Header.Set("X-Tenant-ID", tenantID)
			resp, err := http.DefaultClient.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				systemPrompt += fmt.Sprintf("\n\nData Laba/Rugi aktual: %s", string(body))
			}
		}
	} else if strings.Contains(msgLower, "kas") || strings.Contains(msgLower, "saldo") || strings.Contains(msgLower, "uang") {
		from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
		to := time.Now().Format("2006-01-02")
		url := fmt.Sprintf("%s/reports/cash-flow?from=%s&to=%s", AccountingURL, from, to)
		httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		httpReq.Header.Set("X-Tenant-ID", tenantID)
		resp, err := http.DefaultClient.Do(httpReq)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			systemPrompt += fmt.Sprintf("\n\nData Arus Kas aktual: %s", string(body))
		}
	} else if strings.Contains(msgLower, "aset") || strings.Contains(msgLower, "hutang") || strings.Contains(msgLower, "modal") || strings.Contains(msgLower, "neraca") {
		if role == "kasir" || role == "staff" {
			systemPrompt += "\n\n[SISTEM]: Akses ke Neraca Keuangan ditolak untuk role Kasir/Staff."
		} else {
			from := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
			to := time.Now().Format("2006-01-02")
			url := fmt.Sprintf("%s/reports/balance-sheet?from=%s&to=%s", AccountingURL, from, to)
			httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			httpReq.Header.Set("X-Tenant-ID", tenantID)
			resp, err := http.DefaultClient.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				systemPrompt += fmt.Sprintf("\n\nData Neraca (Balance Sheet) aktual: %s", string(body))
			}
		}
	}

	return systemPrompt
}

func processAIAnswer(ctx context.Context, tenantID, answer, sender, _ string) string {
	// 1. Process [FORWARD_TO_ADMIN]
	if strings.Contains(answer, "[FORWARD_TO_ADMIN]") {
		startIdx := strings.Index(answer, "[FORWARD_TO_ADMIN]")
		msgToAdmin := strings.TrimSpace(answer[startIdx+18:])
		// Find end of line or next bracket if any
		endIdx := strings.Index(msgToAdmin, "\n")
		if endIdx != -1 {
			msgToAdmin = msgToAdmin[:endIdx]
		}

		// Clean up the answer shown to user
		answer = strings.Replace(answer, answer[startIdx:startIdx+18+len(msgToAdmin)], "Mohon ditunggu ya, pesan Anda sedang kami teruskan ke Admin.", 1)

		// Forward message to all forwarders or owner
		go func() {
			if DB == nil {
				return
			}

			// Get forwarders
			var forwarders []string
			rows, err := DB.Query(context.Background(), "SELECT phone_number FROM tenant_forwarders WHERE tenant_id = $1", tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var phone string
					if err := rows.Scan(&phone); err == nil {
						forwarders = append(forwarders, phone)
					}
				}
			}

			// Fallback to owner if no forwarders defined
			if len(forwarders) == 0 {
				var ownerPhone string
				err = DB.QueryRow(context.Background(), "SELECT phone_number FROM users WHERE tenant_id = $1 AND role = 'owner' LIMIT 1", tenantID).Scan(&ownerPhone)
				if err == nil && ownerPhone != "" {
					forwarders = append(forwarders, ownerPhone)
				}
			}

			// Send to all
			for _, phone := range forwarders {
				if strings.HasPrefix(phone, "0") {
					phone = "62" + phone[1:]
				}
				phone = strings.TrimPrefix(phone, "+")

				waGatewayURL := waSendURL()
				data := url.Values{}
				data.Set("target", phone)
				data.Set("message", fmt.Sprintf("⚠️ *ESKALASI OTOMATIS DARI BOT* ⚠️\nPelanggan dengan nomor %s memerlukan bantuan.\n\nKonteks: %s", sender, msgToAdmin))
				data.Set("tenant_id", tenantID)
				reqWA, _ := http.NewRequest("POST", waGatewayURL, strings.NewReader(data.Encode()))
				reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				http.DefaultClient.Do(reqWA)
			}
		}()
	}

	// 2. Process JSON Expense blocks
	startExpIdx := strings.Index(answer, "```json:expense")
	if startExpIdx != -1 {
		endBlock := answer[startExpIdx+15:]
		endIdx := strings.Index(endBlock, "```")
		if endIdx != -1 {
			jsonStr := endBlock[:endIdx]

			// POST expense
			txReq, _ := http.NewRequestWithContext(ctx, "POST", AccountingURL+"/expenses", strings.NewReader(jsonStr))
			txReq.Header.Set("X-Tenant-ID", tenantID)
			txReq.Header.Set("Content-Type", "application/json")
			txResp, err := http.DefaultClient.Do(txReq)

			cleanMsg := strings.Replace(answer, answer[startExpIdx:startExpIdx+15+endIdx+3], "", 1)
			cleanMsg = strings.TrimSpace(cleanMsg)

			if err == nil && txResp.StatusCode == http.StatusOK {
				cleanMsg += "\n\n✅ Pengeluaran telah berhasil dicatat ke sistem akuntansi Anda!"
			} else {
				errDetail := ""
				if txResp != nil {
					b, _ := io.ReadAll(txResp.Body)
					errDetail = string(b)
					txResp.Body.Close()
				} else if err != nil {
					errDetail = err.Error()
				}
				cleanMsg += "\n\n❌ Gagal mencatat pengeluaran. " + errDetail
			}
			return cleanMsg
		}
	}

	// 3. Process JSON Transaction blocks
	startIdx := strings.Index(answer, "```json")
	if startIdx != -1 {
		// skip if it's json:expense
		if !strings.HasPrefix(answer[startIdx:], "```json:expense") {
			endBlock := answer[startIdx+7:]
			endIdx := strings.Index(endBlock, "```")
			if endIdx != -1 {
				jsonStr := endBlock[:endIdx]

				// POST transaction
				txReq, _ := http.NewRequestWithContext(ctx, "POST", AccountingURL+"/transactions", strings.NewReader(jsonStr))
				txReq.Header.Set("X-Tenant-ID", tenantID)
				txReq.Header.Set("Content-Type", "application/json")
				txResp, err := http.DefaultClient.Do(txReq)

				cleanMsg := strings.Replace(answer, answer[startIdx:startIdx+7+endIdx+3], "", 1)
				cleanMsg = strings.TrimSpace(cleanMsg)

				if err == nil && txResp.StatusCode == http.StatusOK {
					cleanMsg += "\n\n✅ Transaksi telah berhasil dicatat ke sistem akuntansi Anda!"
				} else {
					errDetail := ""
					if txResp != nil {
						b, _ := io.ReadAll(txResp.Body)
						errDetail = string(b)
						txResp.Body.Close()
					} else if err != nil {
						errDetail = err.Error()
					}
					cleanMsg += "\n\n❌ Gagal mencatat transaksi. " + errDetail
				}
				return cleanMsg
			}
		}
	}
	return answer
}