package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusOK, APIResponse{Success: true, Message: "ok"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

func TestExtractTenantID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	if id := ExtractTenantID(req); id != "" {
		t.Errorf("expected empty, got %q", id)
	}
	req.Header.Set("X-Tenant-ID", "tenant-123")
	if id := ExtractTenantID(req); id != "tenant-123" {
		t.Errorf("expected tenant-123, got %q", id)
	}
}

func TestExtractUserID(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	if id := ExtractUserID(req); id != "" {
		t.Errorf("expected empty, got %q", id)
	}
	req.Header.Set("X-User-ID", "user-456")
	if id := ExtractUserID(req); id != "user-456" {
		t.Errorf("expected user-456, got %q", id)
	}
}

func TestExtractUserRole(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	if role := ExtractUserRole(req); role != "" {
		t.Errorf("expected empty, got %q", role)
	}
	req.Header.Set("X-User-Role", "admin")
	if role := ExtractUserRole(req); role != "admin" {
		t.Errorf("expected admin, got %q", role)
	}
}

func TestHandleCandidates_NoTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/candidates", nil)
	rr := httptest.NewRecorder()
	HandleCandidates(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCandidates_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/candidates", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleCandidates(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleCandidateVerify_NoTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/candidates/123/verify", nil)
	rr := httptest.NewRecorder()
	HandleCandidateVerify(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCandidateVerify_NoID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/candidates//verify", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleCandidateVerify(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCampaigns_NoTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	rr := httptest.NewRecorder()
	HandleCampaigns(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCampaigns_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/campaigns", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleCampaigns(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleVolunteers_NoTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/volunteers", nil)
	rr := httptest.NewRecorder()
	HandleVolunteers(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleVolunteers_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/volunteers", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleVolunteers(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleVolunteerStats_NoTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/volunteers/stats", nil)
	rr := httptest.NewRecorder()
	HandleVolunteerStats(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleVolunteerStats_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/volunteers/stats", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleVolunteerStats(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleVoters_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/voters", nil)
	rr := httptest.NewRecorder()
	HandleVoters(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleVoters_NoUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/voters", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	HandleVoters(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleVoters_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/voters", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	rr := httptest.NewRecorder()
	HandleVoters(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleVoterStats_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/voters/stats", nil)
	rr := httptest.NewRecorder()
	HandleVoterStats(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
