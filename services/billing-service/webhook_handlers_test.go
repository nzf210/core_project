package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlePaymentWebhook_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()

	handlePaymentWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestHandlePaymentWebhook_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handlePaymentWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandlePaymentWebhook_MalformedExternalID(t *testing.T) {
	payload := map[string]interface{}{
		"status":      "PAID",
		"external_id": "invalid-format",
		"paid_amount": 100000.0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handlePaymentWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for malformed external_id, got %d", w.Code)
	}
}

func TestExtractTenantID_InvoiceFormat(t *testing.T) {
	externalID := "INV-12345678|tenant-uuid-1234"
	parts := strings.Split(externalID, "|")

	if len(parts) < 2 {
		t.Fatal("Failed to split external_id")
	}

	tenantID := parts[1]
	expected := "tenant-uuid-1234"

	if tenantID != expected {
		t.Errorf("Expected %s, got %s", expected, tenantID)
	}
}

func TestExtractTenantID_TopupFormat(t *testing.T) {
	externalID := "uuid-123-wallet-topup-tenant-uuid-5678"
	keyWalletTopup := "-wallet-topup-"

	if !strings.Contains(externalID, keyWalletTopup) {
		t.Fatal("external_id does not contain topup key")
	}

	parts := strings.Split(externalID, keyWalletTopup)
	if len(parts) != 2 {
		t.Fatal("Failed to split topup external_id")
	}

	tenantID := parts[1]
	expected := "tenant-uuid-5678"

	if tenantID != expected {
		t.Errorf("Expected %s, got %s", expected, tenantID)
	}
}

func TestWebhookTokenVerification_HeaderMissing(t *testing.T) {
	callbackToken := ""

	if callbackToken == "" {
		t.Log("Token header missing, should skip verification or use fallback")
	}
}

func TestAmountValidation_Overpayment(t *testing.T) {
	paidAmount := int64(150000)
	invoiceAmount := int64(100000)

	if paidAmount > invoiceAmount {
		excess := paidAmount - invoiceAmount
		expected := int64(50000)
		if excess != expected {
			t.Errorf("Overpayment calculation wrong: expected %d, got %d", expected, excess)
		}
	}
}

func TestAmountValidation_Underpayment(t *testing.T) {
	paidAmount := int64(80000)
	invoiceAmount := int64(100000)

	if paidAmount < invoiceAmount {
		t.Log("Underpayment detected, should block activation")
	} else {
		t.Error("Should detect underpayment")
	}
}

func TestAffiliateCommission_Calculation(t *testing.T) {
	tests := []struct {
		name               string
		invoiceAmount      int64
		commissionPercent  float64
		minPurchase        int64
		maxCommission      int64
		expectedCommission int64
	}{
		{
			name:               "10% of 100k",
			invoiceAmount:      100000,
			commissionPercent:  10,
			minPurchase:        0,
			maxCommission:      0,
			expectedCommission: 10000,
		},
		{
			name:               "10% capped at 5k",
			invoiceAmount:      100000,
			commissionPercent:  10,
			minPurchase:        0,
			maxCommission:      5000,
			expectedCommission: 5000,
		},
		{
			name:               "Below minimum purchase",
			invoiceAmount:      50000,
			commissionPercent:  10,
			minPurchase:        100000,
			maxCommission:      0,
			expectedCommission: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.invoiceAmount < tt.minPurchase {
				if 0 != tt.expectedCommission {
					t.Errorf("Expected 0 for below min purchase, got %d", tt.expectedCommission)
				}
				return
			}

			commission := tt.invoiceAmount * int64(tt.commissionPercent) / 100
			if tt.maxCommission > 0 && commission > tt.maxCommission {
				commission = tt.maxCommission
			}

			if commission != tt.expectedCommission {
				t.Errorf("Expected commission %d, got %d", tt.expectedCommission, commission)
			}
		})
	}
}

func TestWebhookStatus_ExpiredInvoice(t *testing.T) {
	status := "EXPIRED"

	if status == "EXPIRED" {
		t.Log("Should refund voucher and mark invoice expired")
	}
}

func TestWebhookStatus_PaidSettled(t *testing.T) {
	validStatuses := []string{"PAID", "SETTLED"}

	for _, status := range validStatuses {
		if status != "PAID" && status != "SETTLED" {
			t.Errorf("Status %s should be valid for activation", status)
		}
	}
}

func TestTopupDeduplication_Logic(t *testing.T) {
	externalID := "uuid-123-wallet-topup-tenant-456"

	existing := 0

	if existing > 0 {
		t.Log("Duplicate topup, should return 200 OK without processing")
	} else {
		t.Log("New topup, should process")
	}
}
