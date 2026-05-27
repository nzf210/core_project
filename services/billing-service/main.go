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
	"core_project/shared/sdk/response"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

// In a real app, this connects to DB
var mockDB = map[string]string{}
var xenditClient *xendit.APIClient

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	xenditClient = xendit.NewClient(os.Getenv("XENDIT_API_KEY"))

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
	server.ListenAndServe()
}

type SubscribeReq struct {
	PlanID string `json:"plan_id"`
}

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	tenantID := r.Context().Value(auth.TenantIDKey).(string)
	
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
		response.JSON(w, http.StatusOK, "Subscribed to free plan", nil)
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
			
			// Update tenant_subscriptions DB table to 'active'
			mockDB[tenantID] = "active"

			// Send automated WA notification
			go sendWANotification(tenantID, fmt.Sprintf("Halo! Pembayaran tagihan Anda untuk %s telah berhasil kami terima. Layanan Anda kini sudah aktif. Terima kasih!", externalID))
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

	resp, err := http.PostForm("http://localhost:8202/api/wa/send", data)
	if err != nil {
		slog.Error("Failed to send WA notification", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("WA notification API returned non-OK status", "status", resp.Status)
		return
	}

	slog.Info("WA Notification sent successfully", "tenant_id", tenantID, "target", targetJID)
}
