package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"core_project/shared/sdk/config"
	"net/url"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var AIGatewayURL = "http://localhost:8002/v1/chat"
var AccountingURL = "http://localhost:8201"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if os.Getenv("AI_GATEWAY_URL") != "" {
		AIGatewayURL = os.Getenv("AI_GATEWAY_URL")
	}
	if os.Getenv("ACCOUNTING_URL") != "" {
		AccountingURL = os.Getenv("ACCOUNTING_URL")
	} else if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		AccountingURL = "http://umkm-accounting:8201"
		AIGatewayURL = "http://ai-gateway:8002/v1/chat"
	}

	cfg := config.LoadConfig(".env")
	if err := initDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/webhook/wa", handleWAWebhook)

	server := &http.Server{
		Addr:    ":8203", // Chatbot port
		Handler: loggingMiddleware(mux),
	}

	// Start worker pool for handling WA webhooks concurrently
	startWorkerPool(100) // 100 concurrent workers

	slog.Info("UMKM Chatbot listening", "port", 8203)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
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

type ChatReq struct {
	SessionID string `json:"session_id"` // Optional
	Message   string `json:"message"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var req ChatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	ctx := r.Context()

	// Manage Session
	sessionID := req.SessionID
	if sessionID == "" && DB != nil {
		err := DB.QueryRow(ctx, "INSERT INTO chat_sessions (tenant_id, title) VALUES ($1, $2) RETURNING id", tenantID, "New Chat").Scan(&sessionID)
		if err != nil {
			slog.Error("Failed to create session", "err", err)
		}
	}

	// Save User Message
	if sessionID != "" && DB != nil {
		DB.Exec(ctx, "INSERT INTO chat_messages (session_id, role, content) VALUES ($1, $2, $3)", sessionID, "user", req.Message)
	}

	tenantName := "UMKM WCH"
	if DB != nil {
		DB.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
	}
	systemPrompt := buildSystemPrompt(ctx, tenantID, tenantName, req.Message, "owner")
	// Call AI Gateway
	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    req.Message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)

	aiReqHTTP, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	aiReqHTTP.Header.Set("Content-Type", "application/json")

	aiClient := &http.Client{Timeout: 30 * time.Second}
	aiRespHTTP, err := aiClient.Do(aiReqHTTP)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to contact AI Gateway"})
		return
	}
	defer aiRespHTTP.Body.Close()

	var aiGatewayResp struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	json.NewDecoder(aiRespHTTP.Body).Decode(&aiGatewayResp)

	if !aiGatewayResp.Success {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway returned error"})
		return
	}

	aiAnswer := processAIAnswer(ctx, tenantID, aiGatewayResp.Text, "Web UI", "owner")

	// Save Assistant Message
	if sessionID != "" && DB != nil {
		DB.Exec(ctx, "INSERT INTO chat_messages (session_id, role, content) VALUES ($1, $2, $3)", sessionID, "assistant", aiAnswer)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"reply":      aiAnswer,
		},
	})
}

// Queue setup
type ChatJob struct {
	Sender   string
	Message  string
	TenantID string
}

var JobQueue = make(chan ChatJob, 5000)

func startWorkerPool(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for job := range JobQueue {
				processChatJob(job)
			}
		}(i)
	}
}

// handleWAWebhook processes incoming WhatsApp messages from third party like Fonnte
func handleWAWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read raw body to parse JSON
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	slog.Info("Raw Webhook Body", "body", string(bodyBytes))
	// Restore body for any subsequent reader if needed
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload struct {
		Sender  string `json:"sender"`
		Message string `json:"message"`
	}

	sender := ""
	message := ""

	// Try parsing as JSON first
	if err := json.Unmarshal(bodyBytes, &payload); err == nil && payload.Sender != "" {
		sender = payload.Sender
		message = payload.Message
	} else {
		// Fallback to FormValue if not JSON
		r.ParseMultipartForm(10 << 20)
		sender = r.FormValue("sender")
		message = r.FormValue("message")
	}

	slog.Info("Received WA Webhook", "sender", sender, "message", message)

	tenantID := r.URL.Query().Get("tenant_id")

	// Respond immediately to avoid timeout from webhook provider
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":true,"message":"queued"}`))

	// Enqueue the job for async processing
	select {
	case JobQueue <- ChatJob{Sender: sender, Message: message, TenantID: tenantID}:
		slog.Info("Job queued successfully", "sender", sender)
	default:
		slog.Error("JobQueue is full! Dropping message from", "sender", sender)
	}
}

// processChatJob handles the heavy logic asynchronously
func processChatJob(job ChatJob) {
	ctx := context.Background()
	sender := job.Sender
	message := job.Message
	tenantID := job.TenantID

	userRole := "customer"
	tenantName := "UMKM WCH"

	if tenantID != "" {
		if DB != nil {
			// Fetch tenant name
			err := DB.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
			if err != nil {
				slog.Warn("Failed to get tenant name", "error", err)
			}

			// Check user role
			cleanSender := strings.Split(sender, "@")[0]
			rows, err := DB.Query(ctx, "SELECT phone_number, role FROM users WHERE tenant_id = $1", tenantID)
			if err == nil {
				for rows.Next() {
					var dbPhone, dbRole string
					if err := rows.Scan(&dbPhone, &dbRole); err == nil {
						if strings.HasPrefix(dbPhone, "0") {
							dbPhone = "62" + dbPhone[1:]
						}
						dbPhone = strings.TrimPrefix(dbPhone, "+")
						if dbPhone == cleanSender {
							userRole = dbRole
							break
						}
					}
				}
				rows.Close()
			}
			
			// Auto-save contact
			_, errSave := DB.Exec(ctx, "INSERT INTO tenant_contacts (tenant_id, phone_number) VALUES ($1, $2) ON CONFLICT (tenant_id, phone_number) DO NOTHING", tenantID, cleanSender)
			if errSave != nil {
				slog.Warn("Failed to auto-save contact", "error", errSave, "phone", cleanSender)
			}
		}
	} else {
		// Global webhook (Tenant owner chatting with the central bot)
		if DB != nil {
			err := DB.QueryRow(ctx, "SELECT tenant_id FROM users WHERE phone_number = $1 LIMIT 1", sender).Scan(&tenantID)
			if err != nil {
				slog.Warn("Unregistered phone number attempted to chat", "sender", sender)
				
				// Auto-reply to unregistered user via WA Gateway
				waGatewayURL := "http://wa-gateway:8202/api/wa/send"
				data := url.Values{}
				data.Set("target", sender)
				data.Set("message", "Mohon maaf, nomor WhatsApp Anda belum terdaftar sebagai pengguna sistem UMKM WCH. Silakan mendaftar melalui aplikasi web kami terlebih dahulu.")
				data.Set("tenant_id", "global") 
				
				reqWA, _ := http.NewRequestWithContext(ctx, "POST", waGatewayURL, strings.NewReader(data.Encode()))
				reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				respWA, errF := http.DefaultClient.Do(reqWA)
				if errF == nil {
					respWA.Body.Close()
				}
				return
			}
		} else {
			// DB is down
			slog.Error("Database unavailable while processing async chat")
			return
		}
	}

	// 2. Call AI Gateway
	systemPrompt := buildSystemPrompt(ctx, tenantID, tenantName, message, userRole)
	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)
	aiReqHTTP, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	aiReqHTTP.Header.Set("Content-Type", "application/json")

	aiRespHTTP, err := http.DefaultClient.Do(aiReqHTTP)
	if err == nil {
		defer aiRespHTTP.Body.Close()
		var aiGatewayResp struct {
			Success bool   `json:"success"`
			Text    string `json:"text"`
		}
		json.NewDecoder(aiRespHTTP.Body).Decode(&aiGatewayResp)

		if aiGatewayResp.Success && aiGatewayResp.Text != "" {
			finalText := processAIAnswer(ctx, tenantID, aiGatewayResp.Text, sender, userRole)
			// 3. Post reply back to WA Gateway API
			waGatewayURL := "http://wa-gateway:8202/api/wa/send"
			data := url.Values{}
			data.Set("target", sender)
			data.Set("message", finalText)
			data.Set("tenant_id", tenantID)

			reqWA, _ := http.NewRequestWithContext(ctx, "POST", waGatewayURL, strings.NewReader(data.Encode()))
			reqWA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			respWA, errWA := http.DefaultClient.Do(reqWA)
			if errWA == nil {
				respWA.Body.Close()
			} else {
				slog.Error("Failed to send WA reply", "error", errWA)
			}
		}
	} else {
		slog.Error("Failed to contact AI Gateway in worker", "error", err)
	}
}

func buildSystemPrompt(ctx context.Context, tenantID, tenantName, message, role string) string {
	var systemPrompt string
	if role == "customer" {
		systemPrompt = fmt.Sprintf("Anda adalah asisten virtual (Customer Service) untuk toko bernama '%s'. Jawab dengan ramah dan sopan kepada pelanggan. Jika pelanggan menanyakan daftar harga barang/produk, berikan harga sesuai katalog. Jika pelanggan marah, ada keluhan komplain, atau secara spesifik meminta bicara dengan admin/pemilik, Anda WAJIB merespon dengan mengawali pesan Anda menggunakan format `[FORWARD_TO_ADMIN] {Isi keluhan/pesan pelanggan agar admin tahu}`. Contoh: `[FORWARD_TO_ADMIN] Tolong cek keluhan pelanggan ini mengenai barang rusak.`", tenantName)
		// We DO NOT return early anymore because we want to inject the Product Catalog for customers too!
	} else if role == "kasir" || role == "staff" {
		systemPrompt = fmt.Sprintf("Anda adalah asisten Kasir untuk toko '%s' (UMKM WCH). Tugas Anda HANYA membantu mencatat transaksi masuk/keluar harian dan menghitung jumlah kas hari ini. \n\nPERINGATAN: DILARANG KERAS memberikan informasi rahasia seperti laporan Laba/Rugi, Modal, atau Total Neraca jika ditanya. Jika ditanya soal Laba/Rugi, katakan bahwa Anda tidak memiliki hak akses untuk itu.", tenantName)
	} else if role == "owner" || role == "admin" || role == "user" {
		systemPrompt = fmt.Sprintf("Anda adalah asisten keuangan pintar untuk toko '%s' (UMKM WCH). Anda memiliki akses penuh ke laporan keuangan dan operasional. Jawab dalam bahasa Indonesia yang ramah.", tenantName)
	} else {
		systemPrompt = fmt.Sprintf("Anda adalah asisten toko '%s'.", tenantName)
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
Jika user bermaksud mencatat transaksi (misal ada pemasukan, pengeluaran, pembelian, penjualan), Anda WAJIB menyertakan blok kode JSON yang berisi struktur transaksi di dalam balasan Anda.
Gunakan format berikut secara presisi:
` + "```json\n" + `{
  "date": "2026-05-24",
  "description": "Catatan singkat",
  "reference": "AUTO",
  "lines": [
    {"account_id": "ID_AKUN_DEBIT_DARI_COA", "debit": 100000, "credit": 0},
    {"account_id": "ID_AKUN_KREDIT_DARI_COA", "debit": 0, "credit": 100000}
  ]
}` + "\n```\n" + `PENTING: Gunakan tipe data integer (angka bulat) untuk debit dan credit tanpa titik/koma.

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

func processAIAnswer(ctx context.Context, tenantID, answer, sender, role string) string {
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
			if DB == nil { return }
			
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
				
				waGatewayURL := "http://wa-gateway:8202/api/wa/send"
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

	// 2. Process JSON Transaction blocks
	startIdx := strings.Index(answer, "```json")
	if startIdx != -1 {
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
	return answer
}

