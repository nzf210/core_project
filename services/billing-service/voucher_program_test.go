package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/response"
)

func TestHandleAdminVoucherProgramsItem_RoleCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/voucher-programs/test-uuid", nil)
	// Non-superadmin role should be forbidden
	req.Header.Set(response.XUserRole, "owner")
	w := httptest.NewRecorder()

	handleAdminVoucherProgramsItem(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for non-superadmin, got %d", w.Code)
	}
}

func TestHandleAdminVoucherProgramsItem_MissingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/voucher-programs/", nil)
	req.Header.Set(response.XUserRole, "superadmin")
	w := httptest.NewRecorder()

	handleAdminVoucherProgramsItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request for missing ID, got %d", w.Code)
	}
}

func TestHandleAdminVoucherProgramsItem_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/admin/voucher-programs/11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set(response.XUserRole, "superadmin")
	w := httptest.NewRecorder()

	handleAdminVoucherProgramsItem(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 Method Not Allowed for PATCH, got %d", w.Code)
	}
}

func TestUpdateVoucherProgramItem_Validation(t *testing.T) {
	// Missing name and voucher_type
	body := bytes.NewReader([]byte(`{"discount_value": 10}`))
	req := httptest.NewRequest(http.MethodPut, "/admin/voucher-programs/11111111-1111-1111-1111-111111111111", body)
	req.Header.Set(response.XUserRole, "superadmin")
	w := httptest.NewRecorder()

	handleAdminVoucherProgramsItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request for missing name/type, got %d", w.Code)
	}
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"   ", false},
		{"invalid-date", false},
		{"2026-09-08T12:00:00Z", true},
		{"2026-09-08T12:00:00+07:00", true},
		{"2026-09-08T12:00:00", true},
		{"2026-09-08T12:00", true},
		{"2026-09-08 12:00:00", true},
		{"2026-09-08 12:00", true},
		{"2026-09-08", true},
	}

	for _, tt := range tests {
		parsed, ok := parseDateTime(tt.input)
		if ok != tt.expected {
			t.Errorf("parseDateTime(%q) ok = %v, want %v", tt.input, ok, tt.expected)
		}
		if ok && parsed.IsZero() {
			t.Errorf("parseDateTime(%q) returned zero time when ok=true", tt.input)
		}
	}
}
