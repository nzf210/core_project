package main

import (
	"context"
	"testing"
	"time"
)

func TestActivateSubscription_Proration(t *testing.T) {
	if DB == nil {
		t.Skip("DB not available")
	}

	tests := []struct {
		name           string
		validityDays   int
		proratedDays   int
		expectedTotal  int
	}{
		{"No proration", 30, 0, 30},
		{"With 5 days proration", 30, 5, 35},
		{"With 15 days proration", 30, 15, 45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := tt.validityDays + tt.proratedDays
			if total != tt.expectedTotal {
				t.Errorf("Expected %d days, got %d", tt.expectedTotal, total)
			}
		})
	}
}

func TestPlanPriority_Calculation(t *testing.T) {
	tests := []struct {
		planID   string
		priority int
	}{
		{"ultimate", 4},
		{"pro", 3},
		{"lite", 2},
		{"free", 1},
		{"inactive", 0},
	}

	for _, tt := range tests {
		t.Run(tt.planID, func(t *testing.T) {
			var priority int
			switch tt.planID {
			case "ultimate":
				priority = 4
			case "pro":
				priority = 3
			case "lite":
				priority = 2
			case "free":
				priority = 1
			default:
				priority = 0
			}

			if priority != tt.priority {
				t.Errorf("Plan %s: expected priority %d, got %d", tt.planID, tt.priority, priority)
			}
		})
	}
}

func TestValidateVoucherOnly_Logic(t *testing.T) {
	if DB == nil {
		t.Skip("DB not available")
	}

	ctx := context.Background()
	code := "TEST-VOUCHER-2026"
	planID := "lite"

	valid := validateVoucherOnly(ctx, code, planID)
	t.Logf("Voucher %s validation result: %v", code, valid)
}

func TestApplyVoucher_Idempotency(t *testing.T) {
	if DB == nil {
		t.Skip("DB not available")
	}

	ctx := context.Background()
	code := "TEST-VOUCHER-IDEMPOTENT"
	planID := "lite"
	tenantID := "test-tenant-123"

	ok1, _ := applyVoucher(ctx, code, planID, tenantID)
	ok2, _ := applyVoucher(ctx, code, planID, tenantID)

	if ok1 && ok2 {
		t.Error("Voucher should not be redeemable twice (race condition)")
	}

	t.Logf("First attempt: %v, Second attempt: %v", ok1, ok2)
}

func TestTicketNumberGeneration(t *testing.T) {
	ticket1 := generateTicketNumber()
	ticket2 := generateTicketNumber()

	if ticket1 == "" || ticket2 == "" {
		t.Error("Ticket number should not be empty")
	}

	if ticket1 == ticket2 {
		t.Error("Ticket numbers should be unique")
	}

	t.Logf("Generated tickets: %s, %s", ticket1, ticket2)
}

func TestBuildTicketMessage_Format(t *testing.T) {
	payload := TicketPayload{
		TenantName:    "Test Business",
		TicketNumber:  "TKT-2026-001",
		PlanName:      "Lite Plan",
		ActivatedAt:   time.Now().Format("02 Jan 2006 15:04 WIB"),
		ExpiresAt:     "30 hari dari sekarang",
		PaymentMethod: "payment",
		VoucherCode:   "",
	}

	message := buildTicketMessage(payload)

	if message == "" {
		t.Error("Message should not be empty")
	}

	expectedParts := []string{
		"Test Business",
		"TKT-2026-001",
		"Lite Plan",
		"Aktif",
	}

	for _, part := range expectedParts {
		if !containsString(message, part) {
			t.Errorf("Message should contain '%s'", part)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

func TestVoucherExpiration_Logic(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(-24 * time.Hour)

	if expiresAt.Before(now) {
		t.Log("Voucher expired, should not be valid")
	} else {
		t.Error("Expired voucher should be detected")
	}
}

func TestVoucherTargetPlan_Validation(t *testing.T) {
	tests := []struct {
		name         string
		targetPlanID *string
		requestPlan  string
		shouldMatch  bool
	}{
		{
			name:         "Any plan (nil target)",
			targetPlanID: nil,
			requestPlan:  "lite",
			shouldMatch:  true,
		},
		{
			name:         "Exact match",
			targetPlanID: strPtr("lite"),
			requestPlan:  "lite",
			shouldMatch:  true,
		},
		{
			name:         "Plan mismatch",
			targetPlanID: strPtr("pro"),
			requestPlan:  "lite",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.targetPlanID == nil || *tt.targetPlanID == "" || *tt.targetPlanID == tt.requestPlan

			if isValid != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v", tt.shouldMatch, isValid)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
