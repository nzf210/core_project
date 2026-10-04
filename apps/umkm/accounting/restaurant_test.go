package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRestaurant_IsValidRestaurantStatus(t *testing.T) {
	validStatuses := []string{
		"pending",
		"cooking",
		"ready_to_serve",
		"served",
		"completed",
		"cancelled",
	}

	for _, s := range validStatuses {
		if !isValidRestaurantStatus(s) {
			t.Errorf("Expected status %q to be valid, got false", s)
		}
	}

	invalidStatuses := []string{
		"received",
		"washing",
		"in_progress",
		"unknown",
		"done",
		"",
	}

	for _, s := range invalidStatuses {
		if isValidRestaurantStatus(s) {
			t.Errorf("Expected status %q to be invalid, got true", s)
		}
	}
}

func TestRestaurant_GenerateOrderNo(t *testing.T) {
	no1 := generateRestaurantOrderNo()
	no2 := generateRestaurantOrderNo()

	if !strings.HasPrefix(no1, "KOT-") {
		t.Errorf("Expected order_no to start with KOT-, got %q", no1)
	}
	if len(no1) != 17 { // KOT-YYYYMMDD-XXXX (4 + 8 + 1 + 4 = 17)
		t.Errorf("Expected order_no length to be 17, got %d (%q)", len(no1), no1)
	}
	if no1 == no2 {
		t.Errorf("Expected consecutive order numbers to be different, got %q == %q", no1, no2)
	}
}

func TestHandleRestaurantOrders_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/restaurant/orders", nil)
	w := httptest.NewRecorder()

	handleRestaurantOrders(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing tenant ID, got %d", w.Code)
	}
}

func TestHandleRestaurantOrders_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/restaurant/orders", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleRestaurantOrders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestHandleRestaurantOrders_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/restaurant/orders", bytes.NewBufferString("invalid-json"))
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleRestaurantOrders(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for malformed JSON, got %d", w.Code)
	}
}

func TestHandleRestaurantOrderDetail_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/restaurant/orders/some-id", nil)
	w := httptest.NewRecorder()

	handleRestaurantOrderDetail(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing tenant ID, got %d", w.Code)
	}
}

func TestHandleRestaurantOrderDetail_InvalidAction(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/restaurant/orders/some-id/invalid-action", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleRestaurantOrderDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found for invalid action, got %d", w.Code)
	}
}

func TestHandleRestaurantOrderDetail_InvalidStatusUpdatePayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/restaurant/orders/some-id/status", bytes.NewBufferString("not-json"))
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleRestaurantOrderDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for malformed JSON in status update, got %d", w.Code)
	}
}

func TestHandleRestaurantOrderDetail_UnknownStatusUpdate(t *testing.T) {
	body, _ := json.Marshal(UpdateRestaurantStatusReq{Status: "non_existent_status"})
	req := httptest.NewRequest(http.MethodPatch, "/restaurant/orders/some-id/status", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()

	handleRestaurantOrderDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for invalid status value, got %d", w.Code)
	}
}
