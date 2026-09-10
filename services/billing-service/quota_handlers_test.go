package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/auth"
)

func TestHandlePurchaseAddon_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/addons/purchase", nil)
	w := httptest.NewRecorder()
	handlePurchaseAddon(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 MethodNotAllowed, got %d", w.Code)
	}
}

func TestHandlePurchaseAddon_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/addons/purchase", bytes.NewBufferString(`{"addon_key":"ai_vision"}`))
	w := httptest.NewRecorder()
	handlePurchaseAddon(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestHandlePurchaseAddon_MissingAddonKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/addons/purchase", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()
	handlePurchaseAddon(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 BadRequest, got %d", w.Code)
	}
}

func TestAddonTierPriorityComparison(t *testing.T) {
	tests := []struct {
		tenantTier string
		minTier    string
		allowed    bool
	}{
		{"lite", "pro", false},
		{"lite", "ultimate", false},
		{"pro", "pro", true},
		{"pro", "lite", true},
		{"ultimate", "pro", true},
		{"ultimate", "ultimate", true},
		{"superadmin", "ultimate", true},
		{"inactive", "lite", false},
	}

	for _, tt := range tests {
		tenantPriority := auth.TierPriority(tt.tenantTier)
		minPriority := auth.TierPriority(tt.minTier)
		canBuy := tenantPriority >= minPriority
		if canBuy != tt.allowed {
			t.Errorf("tenantTier %s vs minTier %s: expected allowed=%v, got %v", tt.tenantTier, tt.minTier, tt.allowed, canBuy)
		}
	}
}
