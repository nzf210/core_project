package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCheckout_MissingTenant(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	handleCheckout(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleCheckout_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
	req.Header.Set("X-Tenant-ID", "tenant-test-123")
	w := httptest.NewRecorder()

	handleCheckout(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleCheckout_EmptyItems(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"payment_method": "cash",
		"items":          []CheckoutItem{},
	})
	req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader(payload))
	req.Header.Set("X-Tenant-ID", "tenant-test-123")
	w := httptest.NewRecorder()

	handleCheckout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleCheckoutConfirm_MissingTenant(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/checkout/confirm", bytes.NewReader([]byte(`{"reference":"INV-001"}`)))
	w := httptest.NewRecorder()

	handleCheckoutConfirm(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleCheckoutConfirm_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/checkout/confirm", nil)
	req.Header.Set("X-Tenant-ID", "tenant-test-123")
	w := httptest.NewRecorder()

	handleCheckoutConfirm(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleCheckoutConfirm_MissingReference(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/checkout/confirm", bytes.NewReader([]byte(`{"reference":""}`)))
	req.Header.Set("X-Tenant-ID", "tenant-test-123")
	w := httptest.NewRecorder()

	handleCheckoutConfirm(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDynamicQRIS_Tag54Insertion(t *testing.T) {
	staticPayload := "00020101021129300012ID.CO.BANK.BIJBS0286073309000010203053033605802ID5914TOKO SINAR04694000109030806114032160203SBI"
	amount := 50000.0 // Rp 50.000

	dynamicPayload := generateDynamicQRIS(staticPayload, amount)
	if dynamicPayload == "" {
		t.Fatal("dynamic payload is empty")
	}

	if !strings.Contains(dynamicPayload, "540550000") {
		t.Errorf("expected dynamic QRIS to contain Tag 54 amount '540550000', got: %s", dynamicPayload)
	}

	// Dynamic QRIS must end with 4 hex checksum digits after Tag 6304
	if !strings.Contains(dynamicPayload, "6304") {
		t.Errorf("expected dynamic QRIS to contain Tag 6304 checksum, got: %s", dynamicPayload)
	}
}
