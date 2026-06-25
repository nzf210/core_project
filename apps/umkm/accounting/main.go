package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"core_project/shared/sdk/config"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type AccountReq struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID string `json:"parent_id"`
}

type TransactionReq struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	Reference   string `json:"reference"`
	Lines       []struct {
		AccountID string `json:"account_id"`
		Debit     int64  `json:"debit"`
		Credit    int64  `json:"credit"`
	} `json:"lines"`
}

type ChatbotConfig struct {
	LLMProvider                   string   `json:"llm_provider"`
	LLMModel                      string   `json:"llm_model"`
	Temperature                   float64  `json:"temperature"`
	MaxTokens                     int      `json:"max_tokens"`
	SystemPrompt                  string   `json:"system_prompt"`
	Tone                          string   `json:"tone"`
	Language                      string   `json:"language"`
	MaxContextMessages            int      `json:"max_context_messages"`
	WelcomeMessage                string   `json:"welcome_message"`
	FallbackMessage               string   `json:"fallback_message"`
	OutsideHoursMessage           string   `json:"outside_hours_message"`
	BusinessHoursStart            string   `json:"business_hours_start"`
	BusinessHoursEnd              string   `json:"business_hours_end"`
	BusinessDays                  []int    `json:"business_days"`
	EscalationEnabled             bool     `json:"escalation_enabled"`
	EnableVision                  bool     `json:"enable_vision"`
	EnableVoiceReply              bool     `json:"enable_voice_reply"`
	VoiceModel                    string   `json:"voice_model"`
	EscalationKeywords            []string `json:"escalation_keywords"`
	EscalationConfidenceThreshold float64  `json:"escalation_confidence_threshold"`
	AutoEscalateAfterMinutes      int      `json:"auto_escalate_after_minutes"`
	RAGEnabled                    bool     `json:"rag_enabled"`
	RAGTopK                       int      `json:"rag_top_k"`
	RAGSimilarityThreshold        float64  `json:"rag_similarity_threshold"`
	ChannelsEnabled               []string `json:"channels_enabled"`
	IsActive                      bool     `json:"is_active"`
	WAProviderPreference           string   `json:"wa_provider_preference"`
}

var AIGatewayURL = "http://localhost:8002/v1/chat"

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if os.Getenv("AI_GATEWAY_URL") != "" {
		AIGatewayURL = os.Getenv("AI_GATEWAY_URL")
	} else if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		AIGatewayURL = "http://ai-gateway:8002/v1/chat"
	}

	cfg := config.LoadConfig(".env")
	if err := initDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

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
	mux.HandleFunc("/reports/sales-chart", handleSalesChart)
	mux.HandleFunc("/reports/top-products", handleTopProducts)
	mux.HandleFunc("/reports/recent-transactions", handleRecentTransactions)
	mux.HandleFunc("/reports/cash-flow/pdf", handleCashFlowPDF)
	mux.HandleFunc("/reports/income-statement/pdf", handleIncomeStatementPDF)
	mux.HandleFunc("/reports/balance-sheet/pdf", handleBalanceSheetPDF)
	mux.HandleFunc("/expenses", handleExpenses)
	mux.HandleFunc("/seed", handleSeed)
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
	mux.HandleFunc("/internal/tenant/{tenant_id}/chatbot-config", handleInternalChatbotConfig)
	mux.HandleFunc("/chatbot/config", handleChatbotConfig)
	mux.HandleFunc("/chatbot/config/test", handleChatbotConfigTest)
	mux.HandleFunc("/chatbot/permissions", handleChatbotPermissions)
	mux.HandleFunc("/wa/setup", handleWASetup)
	mux.HandleFunc("/wa/connect", handleWAConnect)
	mux.HandleFunc("/wa/cloud-api-credential", handleWACloudAPICredential)
	mux.HandleFunc("/clinic/settings", requireClinicType(handleClinicSettings))
	mux.HandleFunc("/clinic/appointments/book", requireClinicType(handleClinicBook))
	mux.HandleFunc("/clinic/appointments/cancel", requireClinicType(handleClinicCancel))
	mux.HandleFunc("/clinic/appointments/queue", requireClinicType(handleClinicQueue))
	mux.HandleFunc("/clinic/appointments/call", requireClinicType(handleClinicCall))
	mux.HandleFunc("/clinic/medical-records", requireClinicType(handleClinicMedicalRecords))
	mux.HandleFunc("/clinic/doctors", requireClinicType(handleClinicDoctors))
	mux.HandleFunc("/export/products", handleExportProducts)
	mux.HandleFunc("/export/contacts", handleExportContacts)
	mux.HandleFunc("/import/products", handleImportProducts)
	mux.HandleFunc("/import/contacts", handleImportContacts)
	mux.HandleFunc("/import/journal", handleImportJournal)
	mux.HandleFunc("/import/template", handleImportTemplate)
	mux.HandleFunc("/internal/tenant/{tenant_id}/rag/search", handleInternalRAGSearch)
	mux.HandleFunc("/internal/conversation/log", handleInternalConversationLog)
	mux.HandleFunc("/internal/escalation/log", handleInternalEscalationLog)
	mux.HandleFunc("/internal/tenant/{tenant_id}/faqs", handleInternalFAQs)
	mux.HandleFunc("/internal/tenant/{tenant_id}/products", handleInternalProducts)
	mux.HandleFunc("/internal/tenant/{tenant_id}/rag/single", handleInternalRAGSingle)

	server := &http.Server{
		Addr:    ":8201",
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
	parts := strings.Split(staticQRIS, "6304")
	base := parts[0]
	amtStr := strconv.FormatFloat(amount, 'f', 0, 64)
	amtTag := fmt.Sprintf("54%02d%s", len(amtStr), amtStr)
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

