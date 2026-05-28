package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/config"
	"core_project/shared/sdk/response"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

var xenditClient *xendit.APIClient

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load shared configuration
	cfg := config.LoadConfig(".env")
	if err := initDB(cfg); err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	if err := ensureSchema(); err != nil {
		slog.Error("Failed to ensure schema", "error", err)
		os.Exit(1)
	}

	xKey := os.Getenv("XENDIT_API_KEY")
	if xKey == "" {
		xKey = "xnd_development_mock_key_1234567890"
	}
	xenditClient = xendit.NewClient(xKey)

	mux := http.NewServeMux()

	// Protected routes
	mux.Handle("/subscribe", auth.Middleware(http.HandlerFunc(handleSubscribe)))
	
	// Public route (Midtrans/Xendit Webhook)
	mux.HandleFunc("/webhook/payment", handlePaymentWebhook)

	server := &http.Server{
		Addr:    ":8003", // Billing service port
		Handler: mux,
	}

	slog.Info("Billing Service listening", "port", 8003)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start Billing Service", "error", err)
	}
}

type SubscribeReq struct {
	PlanID string `json:"plan_id"`
}

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing Tenant ID in token context", nil)
		return
	}
	
	var req SubscribeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// 1. Validate Plan
	planPrices := map[string]float64{
		"lite": 150000,
		"pro":  450000,
	}

	if req.PlanID == "free" {
		// Update DB for free plan
		_, err := DB.Exec(r.Context(), "UPDATE tenants SET plan = 'free' WHERE id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to update tenant plan to free", "tenant_id", tenantID, "error", err)
		}
		_, err = DB.Exec(r.Context(), `
			INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, current_period_end, updated_at)
			VALUES ($1, 'free', 'active', NOW() + INTERVAL '30 days', NOW())
			ON CONFLICT (tenant_id)
			DO UPDATE SET plan_id = 'free', status = 'active', current_period_end = NOW() + INTERVAL '30 days', updated_at = NOW()`,
			tenantID)
		if err != nil {
			slog.Error("Failed to upsert tenant subscription to free", "tenant_id", tenantID, "error", err)
		}

		response.JSON(w, http.StatusOK, "Subscribed to free plan successfully", nil)
		return
	}

	amount, exists := planPrices[req.PlanID]
	if !exists {
		response.Error(w, http.StatusBadRequest, "Invalid Plan ID", nil)
		return
	}

	// 2. Generate Xendit Invoice
	invoiceID := uuid.NewString()[:8]
	externalID := fmt.Sprintf("INV-%s|%s", invoiceID, tenantID)

	createInvoiceReq := invoice.NewCreateInvoiceRequest(externalID, amount)
	desc := fmt.Sprintf("Subscription for %s plan", req.PlanID)
	createInvoiceReq.Description = &desc

	resp, _, err := xenditClient.InvoiceApi.CreateInvoice(r.Context()).
		CreateInvoiceRequest(*createInvoiceReq).Execute()

	if err != nil {
		slog.Error("Failed to create xendit invoice", "error", err)
		response.Error(w, http.StatusInternalServerError, "Failed to create invoice", nil)
		return
	}

	paymentURL := resp.InvoiceUrl

	// 3. Persist invoice to database
	_, dbErr := DB.Exec(r.Context(),
		"INSERT INTO invoices (id, tenant_id, plan_id, amount, status, payment_url) VALUES ($1, $2, $3, $4, $5, $6)",
		externalID, tenantID, req.PlanID, amount, "pending", paymentURL)
	if dbErr != nil {
		slog.Error("Failed to save invoice to database", "invoice_id", externalID, "error", dbErr)
	}

	slog.Info("Created subscription invoice", "tenant_id", tenantID, "plan", req.PlanID, "invoice_id", *resp.Id)

	response.JSON(w, http.StatusOK, "Invoice created. Please complete payment.", map[string]interface{}{
		"invoice_id":  *resp.Id,
		"payment_url": paymentURL,
		"plan_id":     req.PlanID,
	})
}

// handlePaymentWebhook processes payment success notifications
func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify Xendit Callback Token
	callbackToken := r.Header.Get("x-callback-token")
	expectedToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")
	if expectedToken != "" && callbackToken != expectedToken {
		slog.Warn("Unauthorized webhook callback attempt", "received_token", callbackToken)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status, _ := payload["status"].(string)
	externalID, _ := payload["external_id"].(string)

	if status == "PAID" || status == "SETTLED" {
		parts := strings.Split(externalID, "|")
		if len(parts) == 2 {
			tenantID := parts[1]
			slog.Info("Payment received!", "tenant_id", tenantID, "status", status)

			// Query invoice detail to get plan_id
			planID := "lite" // fallback
			var dbAmount float64
			err := DB.QueryRow(r.Context(), "SELECT plan_id, amount FROM invoices WHERE id = $1", externalID).Scan(&planID, &dbAmount)
			if err != nil {
				slog.Warn("Invoice not found in DB, using fallback", "invoice_id", externalID, "error", err)
			}

			// 1. Update invoice status
			_, err = DB.Exec(r.Context(), "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", externalID)
			if err != nil {
				slog.Error("Failed to update invoice status in DB", "invoice_id", externalID, "error", err)
			}

			// 2. Upsert subscription status
			_, err = DB.Exec(r.Context(), `
				INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, current_period_end, updated_at)
				VALUES ($1, $2, 'active', NOW() + INTERVAL '30 days', NOW())
				ON CONFLICT (tenant_id)
				DO UPDATE SET plan_id = EXCLUDED.plan_id, status = 'active', current_period_end = EXCLUDED.current_period_end, updated_at = NOW()`,
				tenantID, planID)
			if err != nil {
				slog.Error("Failed to upsert tenant subscription in DB", "tenant_id", tenantID, "error", err)
			}

			// 3. Update main tenants table
			_, err = DB.Exec(r.Context(), "UPDATE tenants SET plan = $1 WHERE id = $2", planID, tenantID)
			if err != nil {
				slog.Error("Failed to update tenant plan in tenants table", "tenant_id", tenantID, "plan", planID, "error", err)
			}

			// Send automated WA notification
			go sendWANotification(tenantID, fmt.Sprintf("Halo! Pembayaran tagihan Anda untuk %s (%s) telah berhasil kami terima. Layanan Anda kini sudah aktif. Terima kasih!", externalID, planID))
		}
	}

	w.WriteHeader(http.StatusOK)
}

func sendWANotification(tenantID, message string) {
	target := os.Getenv("TEST_WA_NUMBER")
	if target == "" {
		target = "6281234567890" // Default mock target
	}
	targetJID := target + "@s.whatsapp.net"

	sysTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if sysTenant == "" {
		sysTenant = "system"
	}

	data := url.Values{}
	data.Set("tenant_id", sysTenant)
	data.Set("target", targetJID)
	data.Set("message", message)

	waURL := "http://localhost:8202/api/wa/send"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		waURL = "http://wa-gateway:8202/api/wa/send"
	}

	resp, err := http.PostForm(waURL, data)
	if err != nil {
		slog.Error("Failed to send WA notification via Gateway", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("WA Gateway API returned non-OK status", "status", resp.Status)
		return
	}

	slog.Info("WA Notification sent successfully via Gateway", "tenant_id", tenantID, "target", targetJID)
}
