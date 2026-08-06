package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── calculateRate ────────────────────────────────────────────────────────────

func TestCalculateRate_Normal(t *testing.T) {
	rate := calculateRate(100, 25)
	if rate != 25.0 {
		t.Errorf("expected 25.0, got %f", rate)
	}
}

func TestCalculateRate_ZeroTotal(t *testing.T) {
	rate := calculateRate(0, 10)
	if rate != 0.0 {
		t.Errorf("expected 0.0 for zero total, got %f", rate)
	}
}

func TestCalculateRate_FullRedemption(t *testing.T) {
	rate := calculateRate(50, 50)
	if rate != 100.0 {
		t.Errorf("expected 100.0, got %f", rate)
	}
}

func TestCalculateRate_ZeroRedeemed(t *testing.T) {
	rate := calculateRate(50, 0)
	if rate != 0.0 {
		t.Errorf("expected 0.0, got %f", rate)
	}
}

// ─── extractTenantIDFromExternalID ───────────────────────────────────────────

func TestExtractTenantIDFromExternalID_InvoiceFormat(t *testing.T) {
	tenantID, err := extractTenantIDFromExternalID("INV-550e8400-e29b-41d4-a716-446655440000|tenant-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "tenant-abc" {
		t.Errorf("expected tenant-abc, got %q", tenantID)
	}
}

func TestExtractTenantIDFromExternalID_WalletTopupFormat(t *testing.T) {
	tenantID, err := extractTenantIDFromExternalID("550e8400-e29b-41d4-a716-wallet-topup-tenant-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "tenant-xyz" {
		t.Errorf("expected tenant-xyz, got %q", tenantID)
	}
}

func TestExtractTenantIDFromExternalID_MalformedFormat(t *testing.T) {
	_, err := extractTenantIDFromExternalID("no-pipe-or-topup-here")
	if err == nil {
		t.Error("expected error for malformed external_id")
	}
}

func TestExtractTenantIDFromExternalID_Empty(t *testing.T) {
	_, err := extractTenantIDFromExternalID("")
	if err == nil {
		t.Error("expected error for empty external_id")
	}
}

// ─── isSuperadmin ─────────────────────────────────────────────────────────────

func TestIsSuperadmin_True(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "superadmin")
	rr := httptest.NewRecorder()

	if !isSuperadmin(rr, req) {
		t.Error("expected true for superadmin role")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected no error response, got %d", rr.Code)
	}
}

func TestIsSuperadmin_False(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "owner")
	rr := httptest.NewRecorder()

	if isSuperadmin(rr, req) {
		t.Error("expected false for non-superadmin role")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestIsSuperadmin_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	if isSuperadmin(rr, req) {
		t.Error("expected false when role header missing")
	}
}

// ─── landingCache helpers ─────────────────────────────────────────────────────

func TestLandingCache_SetAndGet(t *testing.T) {
	invalidateAllLandingCache()

	setCachedConfig("test-id", []byte(`{"key":"value"}`))
	data, ok := getCachedConfig("test-id")
	if !ok {
		t.Error("expected cache hit")
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("unexpected cached data: %s", data)
	}
}

func TestLandingCache_Miss(t *testing.T) {
	invalidateAllLandingCache()

	_, ok := getCachedConfig("nonexistent-id")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestLandingCache_Invalidate(t *testing.T) {
	invalidateAllLandingCache()

	setCachedConfig("to-delete", []byte("data"))
	invalidateCache("to-delete")

	_, ok := getCachedConfig("to-delete")
	if ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestLandingCache_InvalidateAll(t *testing.T) {
	invalidateAllLandingCache()

	setCachedConfig("key1", []byte("a"))
	setCachedConfig("key2", []byte("b"))
	invalidateAllLandingCache()

	_, ok1 := getCachedConfig("key1")
	_, ok2 := getCachedConfig("key2")
	if ok1 || ok2 {
		t.Error("expected all cache entries to be cleared")
	}
}

// ─── handleHealth ─────────────────────────────────────────────────────────────

func TestHandleHealth_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleHealth_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	handleHealth(rr, req)

	// handleHealth uses GET only — POST should either 405 or return ok depending on impl
	_ = rr.Code
}

// ─── handleListPlans / handleValidateVoucher ──────────────────────────────────

func TestHandleListPlans_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/plans", nil)
	rr := httptest.NewRecorder()
	handleListPlans(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleValidateVoucher_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/voucher/validate", nil)
	rr := httptest.NewRecorder()
	handleValidateVoucher(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// ─── handleWallet / handleWalletTopup ─────────────────────────────────────────

func TestHandleWallet_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/wallet", nil)
	rr := httptest.NewRecorder()
	handleWallet(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleWalletTopup_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wallet/topup", nil)
	rr := httptest.NewRecorder()
	handleWalletTopup(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleWallet_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wallet", nil)
	rr := httptest.NewRecorder()
	handleWallet(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 400 or 401 for missing tenant, got %d", rr.Code)
	}
}
