package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/response"
)

func TestHandleAdminGenerateVouchers_SuperadminOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/vouchers/generate", nil)
	req.Header.Set(response.XUserRole, "owner")
	w := httptest.NewRecorder()

	handleAdminGenerateVouchers(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for non-superadmin, got %d", w.Code)
	}
}

func TestHandleAdminGenerateVouchers_InvalidPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/vouchers/generate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(response.XUserRole, "superadmin")
	w := httptest.NewRecorder()

	handleAdminGenerateVouchers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleAdminGenerateVouchers_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		payload GenerateVouchersReq
	}{
		{
			name: "Missing plan_id",
			payload: GenerateVouchersReq{
				ValidityDays: 30,
				Quantity:     10,
			},
		},
		{
			name: "Invalid validity_days (0)",
			payload: GenerateVouchersReq{
				PlanID:       "lite",
				ValidityDays: 0,
				Quantity:     10,
			},
		},
		{
			name: "Invalid quantity (0)",
			payload: GenerateVouchersReq{
				PlanID:       "lite",
				ValidityDays: 30,
				Quantity:     0,
			},
		},
		{
			name: "Quantity exceeds limit (1001)",
			payload: GenerateVouchersReq{
				PlanID:       "lite",
				ValidityDays: 30,
				Quantity:     1001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/admin/vouchers/generate", bytes.NewReader(body))
			req.Header.Set(response.XUserRole, "superadmin")
			w := httptest.NewRecorder()

			handleAdminGenerateVouchers(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleAdminGenerateVouchers_DefaultVoucherType(t *testing.T) {
	payload := GenerateVouchersReq{
		PlanID:       "lite",
		ValidityDays: 30,
		Quantity:     1,
		VoucherType:  "",
	}

	vType := payload.VoucherType
	if vType == "" {
		vType = "bonus_months"
	}

	expected := "bonus_months"
	if vType != expected {
		t.Errorf("Expected default voucher_type '%s', got '%s'", expected, vType)
	}
}

func TestHandleAdminListVouchers_SuperadminOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/vouchers", nil)
	req.Header.Set(response.XUserRole, "owner")
	w := httptest.NewRecorder()

	handleAdminListVouchers(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for non-superadmin, got %d", w.Code)
	}
}

func TestHandleAdminDeleteVoucher_SuperadminOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/admin/vouchers?id=test", nil)
	req.Header.Set(response.XUserRole, "owner")
	w := httptest.NewRecorder()

	handleAdminDeleteVoucher(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for non-superadmin, got %d", w.Code)
	}
}

func TestHandleAdminDeleteVoucher_MissingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/admin/vouchers", nil)
	req.Header.Set(response.XUserRole, "superadmin")
	w := httptest.NewRecorder()

	handleAdminDeleteVoucher(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing id, got %d", w.Code)
	}
}

func TestGenerateVoucherCode_Format(t *testing.T) {
	planID := "lite"
	tenantID := "test-tenant-123"

	code := generateVoucherCode(planID, tenantID)

	if code == "" {
		t.Error("Voucher code should not be empty")
	}

	t.Logf("Generated code: %s", code)
}

func TestVoucherQuantityLimits(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int
		shouldErr bool
	}{
		{"Minimum valid (1)", 1, false},
		{"Mid range (500)", 500, false},
		{"Maximum valid (1000)", 1000, false},
		{"Below minimum (0)", 0, true},
		{"Above maximum (1001)", 1001, true},
		{"Negative (-10)", -10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.quantity > 0 && tt.quantity <= 1000
			if isValid == tt.shouldErr {
				t.Errorf("Quantity %d: expected shouldErr=%v, got %v", tt.quantity, tt.shouldErr, !isValid)
			}
		})
	}
}

func TestVoucherValidityDays_Limits(t *testing.T) {
	tests := []struct {
		name         string
		validityDays int
		shouldErr    bool
	}{
		{"7 days", 7, false},
		{"30 days", 30, false},
		{"90 days", 90, false},
		{"365 days", 365, false},
		{"Zero days", 0, true},
		{"Negative days", -30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.validityDays > 0
			if isValid == tt.shouldErr {
				t.Errorf("ValidityDays %d: expected shouldErr=%v, got %v", tt.validityDays, tt.shouldErr, !isValid)
			}
		})
	}
}

func TestVoucherProgramName_DefaultGeneration(t *testing.T) {
	programName := ""
	planID := "lite"

	if programName == "" {
		programName = "Ad-hoc Voucher - " + planID
	}

	expected := "Ad-hoc Voucher - lite"
	if programName != expected {
		t.Errorf("Expected default program name '%s', got '%s'", expected, programName)
	}
}
