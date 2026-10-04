package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLaundry_IsValidLaundryStatus(t *testing.T) {
	validStatuses := []string{
		"received",
		"washing",
		"drying",
		"ironing",
		"ready",
		"completed",
		"cancelled",
	}

	for _, s := range validStatuses {
		if !isValidLaundryStatus(s) {
			t.Errorf("Expected status %q to be valid, got false", s)
		}
	}

	invalidStatuses := []string{
		"pending",
		"in_progress",
		"unknown",
		"done",
		"",
	}

	for _, s := range invalidStatuses {
		if isValidLaundryStatus(s) {
			t.Errorf("Expected status %q to be invalid, got true", s)
		}
	}
}

func TestLaundry_GenerateOrderNo(t *testing.T) {
	no1 := generateLaundryOrderNo()
	no2 := generateLaundryOrderNo()

	if !strings.HasPrefix(no1, "LND-") {
		t.Errorf("Expected order_no to start with LND-, got %q", no1)
	}
	if len(no1) != 17 { // LND-YYYYMMDD-XXXX (4 + 8 + 1 + 4 = 17)
		t.Errorf("Expected order_no length to be 17, got %d (%q)", len(no1), no1)
	}
	if no1 == no2 {
		t.Errorf("Expected consecutive order numbers to be different, got %q == %q", no1, no2)
	}
}

func TestHandleLaundryOrders_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/laundry/orders", nil)
	w := httptest.NewRecorder()

	handleLaundryOrders(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing tenant ID, got %d", w.Code)
	}
}

func TestHandleLaundryOrders_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/laundry/orders", nil)
	req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	handleLaundryOrders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestCreateLaundryOrder_Validation(t *testing.T) {
	tests := []struct {
		name       string
		body       CreateLaundryOrderReq
		wantStatus int
		wantMsg    string
	}{
		{
			name: "missing customer name",
			body: CreateLaundryOrderReq{
				CustomerName:  "",
				CustomerPhone: "081234567890",
				ServiceType:   "kiloan",
				TotalAmount:   2500000,
			},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "Nama pelanggan wajib diisi",
		},
		{
			name: "missing customer phone",
			body: CreateLaundryOrderReq{
				CustomerName:  "Budi Santoso",
				CustomerPhone: "",
				ServiceType:   "kiloan",
				TotalAmount:   2500000,
			},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "Nomor WhatsApp pelanggan wajib diisi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/laundry/orders", bytes.NewReader(b))
			req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
			w := httptest.NewRecorder()

			handleLaundryOrders(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.wantMsg) {
				t.Errorf("Expected error message to contain %q, got %s", tt.wantMsg, w.Body.String())
			}
		})
	}
}

func TestHandleLaundryOrderDetail_MissingOrderID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/laundry/orders/", nil)
	req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	handleLaundryOrderDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for missing order ID, got %d", w.Code)
	}
}

func TestRequireLaundryType_MissingTenantID(t *testing.T) {
	handler := requireLaundryType(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/laundry/orders", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}
