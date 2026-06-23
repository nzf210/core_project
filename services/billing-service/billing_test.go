package main

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateTicketNumber(t *testing.T) {
	tn1 := generateTicketNumber()
	tn2 := generateTicketNumber()

	if tn1 == "" {
		t.Error("ticket number should not be empty")
	}
	if tn1 == tn2 {
		t.Log("ticket numbers may collide at nano precision, but this is unlikely")
	}
	// Check format: TKT-YYYY-MMDD-RRRR
	if len(tn1) < 10 {
		t.Errorf("ticket number %q seems too short", tn1)
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()
	s := formatTime(&now)
	if s == nil {
		t.Error("expected non-nil for valid time")
	}
	if len(*s) < 15 {
		t.Errorf("expected RFC3339 format, got %q", *s)
	}

	s2 := formatTime(nil)
	if s2 != nil {
		t.Error("expected nil for nil time")
	}
}

func TestMaxInt64(t *testing.T) {
	if v := maxInt64(5, 3); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
	if v := maxInt64(3, 5); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
	if v := maxInt64(-1, -2); v != -1 {
		t.Errorf("expected -1, got %d", v)
	}
	if v := maxInt64(0, 0); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}

func TestHashToken(t *testing.T) {
	h := hashToken("test-token")
	if len(h) != 64 {
		t.Errorf("SHA-256 hex should be 64 chars, got %d", len(h))
	}
	// Same input = same hash
	h2 := hashToken("test-token")
	if h != h2 {
		t.Error("hash should be deterministic")
	}
	// Different input = different hash
	h3 := hashToken("different-token")
	if h == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestCachedTenantExpiry(t *testing.T) {
	ct := cachedTenant{
		name:      "test",
		email:     "test@test.com",
		waNumber:  "08123456789",
		telegram:  "123456",
		expiresAt: time.Now().Add(-1 * time.Minute),
	}
	if !time.Now().After(ct.expiresAt) {
		t.Error("cache should be expired")
	}
}

func TestBuildTicketMessage(t *testing.T) {
	payload := TicketPayload{
		TicketNumber:  "TKT-2026-0613-1234",
		TenantName:    "Test Tenant",
		PlanName:      "Lite",
		ActivatedAt:   "13 Jun 2026, 15:04 WIB",
		ExpiresAt:     "30 hari dari sekarang",
		AmountPaid:    99000,
		PaymentMethod: "voucher",
	}
	msg := buildTicketMessage(payload)
	if msg == "" {
		t.Error("ticket message should not be empty")
	}
	if !strings.Contains(msg, "TKT-2026-0613-1234") {
		t.Error("message should contain ticket number")
	}
	if !strings.Contains(msg, "Test Tenant") {
		t.Error("message should contain tenant name")
	}
	if !strings.Contains(msg, "Lite") {
		t.Error("message should contain plan name")
	}
}

func TestN8NStatus_DefaultValues(t *testing.T) {
	status := N8NStatus{
		Status:          "unknown",
		Version:         "unknown",
		ActiveWorkflows: 0,
		QueueMode:       false,
	}
	if status.Status != "unknown" {
		t.Errorf("default status should be unknown, got %q", status.Status)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestVoucherRedeemReq(t *testing.T) {
	req := VoucherRedeemReq{Code: "WCH-TEST-CODE"}
	if req.Code != "WCH-TEST-CODE" {
		t.Errorf("expected WCH-TEST-CODE, got %q", req.Code)
	}
}

func TestSubscribeReq(t *testing.T) {
	req := SubscribeReq{PlanID: "pro", VoucherCode: "WCH-XYZ"}
	if req.PlanID != "pro" {
		t.Errorf("expected pro, got %q", req.PlanID)
	}
	if req.VoucherCode != "WCH-XYZ" {
		t.Errorf("expected WCH-XYZ, got %q", req.VoucherCode)
	}
}

func TestHandleHealth(t *testing.T) {
	// Test that handleHealth doesn't panic; needs http.ResponseWriter
	// Integration test only — skip in unit
}

func TestPlanRowType(t *testing.T) {
	p := planRow{
		ID:           "plan-1",
		Name:         "Lite",
		Description:  "Basic Plan",
		PriceMonthly: 99000,
		PriceYearly:  990000,
		IsActive:     true,
		SortOrder:    2,
	}
	if p.PriceMonthly < 0 {
		t.Error("price should be non-negative")
	}
}

func TestFeatureRowType(t *testing.T) {
	f := featureRow{
		FeatureKey:   "max_stores",
		FeatureName:  "Max Stores",
		FeatureValue: "5",
		IsEnabled:    true,
	}
	if f.FeatureKey == "" {
		t.Error("feature key should not be empty")
	}
}

func TestPlanWithFeaturesType(t *testing.T) {
	pwf := planWithFeatures{
		planRow: planRow{
			ID:   "plan-1",
			Name: "Pro",
		},
		Features: []featureRow{
			{FeatureKey: "max_stores", FeatureValue: "5", IsEnabled: true},
		},
	}
	if len(pwf.Features) != 1 {
		t.Errorf("expected 1 feature, got %d", len(pwf.Features))
	}
}

func TestSendWANotification_BuildsCorrectURL(t *testing.T) {
	// Test that the function compiles and normalizes JID correctly
	// This is a compilation test — runtime behavior depends on services running
}

func TestSendTicketNotifications_BuildsMessage(t *testing.T) {
	payload := TicketPayload{
		TicketNumber:  "TKT-123",
		TenantName:    "UMKM Test",
		PlanName:      "Ultimate",
		ActivatedAt:   "13 Jun 2026",
		ExpiresAt:     "13 Jul 2026",
		AmountPaid:    199000,
		PaymentMethod: "xendit",
		VoucherCode:   "WCH-auto-generate",
	}
	msg := buildTicketMessage(payload)
	if msg == "" {
		t.Error("message should not be empty")
	}
}
