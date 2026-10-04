package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestService_IsValidServiceStatus(t *testing.T) {
	validStatuses := []string{
		"received",
		"diagnosing",
		"working",
		"testing",
		"ready",
		"completed",
		"cancelled",
	}

	for _, s := range validStatuses {
		if !isValidServiceStatus(s) {
			t.Errorf("Expected status %q to be valid, got false", s)
		}
	}

	invalidStatuses := []string{
		"pending",
		"cooking",
		"in_progress",
		"unknown",
		"done",
		"",
	}

	for _, s := range invalidStatuses {
		if isValidServiceStatus(s) {
			t.Errorf("Expected status %q to be invalid, got true", s)
		}
	}
}

func TestService_GenerateOrderNo(t *testing.T) {
	no1 := generateServiceOrderNo()
	no2 := generateServiceOrderNo()

	if !strings.HasPrefix(no1, "SPK-") {
		t.Errorf("Expected order_no to start with SPK-, got %q", no1)
	}
	if len(no1) != 17 { // SPK-YYYYMMDD-XXXX (4 + 8 + 1 + 4 = 17)
		t.Errorf("Expected order_no length to be 17, got %d (%q)", len(no1), no1)
	}
	if no1 == no2 {
		t.Errorf("Expected consecutive order numbers to be different, got %q == %q", no1, no2)
	}
}

func TestHandleServiceOrders_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/service/orders", nil)
	w := httptest.NewRecorder()

	handleServiceOrders(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing tenant ID, got %d", w.Code)
	}
}

func TestHandleServiceOrders_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/service/orders", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleServiceOrders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestHandleServiceOrders_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload CreateServiceOrderReq
	}{
		{
			name: "Missing Customer Name",
			payload: CreateServiceOrderReq{
				CustomerPhone: "08123456789",
				UnitName:      "Honda Beat",
				Complaint:     "Ganti oli",
			},
		},
		{
			name: "Missing Customer Phone",
			payload: CreateServiceOrderReq{
				CustomerName: "Budi",
				UnitName:     "Honda Beat",
				Complaint:    "Ganti oli",
			},
		},
		{
			name: "Missing Unit Name",
			payload: CreateServiceOrderReq{
				CustomerName:  "Budi",
				CustomerPhone: "08123456789",
				Complaint:     "Ganti oli",
			},
		},
		{
			name: "Missing Complaint",
			payload: CreateServiceOrderReq{
				CustomerName:  "Budi",
				CustomerPhone: "08123456789",
				UnitName:      "Honda Beat",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/service/orders", bytes.NewReader(body))
			req.Header.Set("X-Tenant-ID", "tenant-123")
			w := httptest.NewRecorder()

			handleServiceOrders(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 Bad Request for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestHandleServiceOrderDetail_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/service/orders/some-id", nil)
	w := httptest.NewRecorder()

	handleServiceOrderDetail(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing tenant ID, got %d", w.Code)
	}
}

func TestHandleServiceOrderDetail_InvalidAction(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/service/orders/some-id/invalid-action", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleServiceOrderDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found for invalid action, got %d", w.Code)
	}
}

func TestHandleServiceOrderDetail_InvalidStatusUpdatePayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/service/orders/some-id/status", bytes.NewBufferString("not-json"))
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleServiceOrderDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for malformed JSON in status update, got %d", w.Code)
	}
}

func TestHandleServiceOrderDetail_UnknownStatusUpdate(t *testing.T) {
	body, _ := json.Marshal(UpdateServiceStatusReq{Status: "non_existent_status"})
	req := httptest.NewRequest(http.MethodPatch, "/service/orders/some-id/status", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleServiceOrderDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for invalid status value, got %d", w.Code)
	}
}
