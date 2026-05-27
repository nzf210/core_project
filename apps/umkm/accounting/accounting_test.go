package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTransactions_Validation_Unbalanced(t *testing.T) {
	payload := []byte(`{
		"date": "2026-05-22",
		"description": "Sale",
		"reference": "INV-001",
		"lines": [
			{"account_id": "acc-1", "debit": 10000, "credit": 0},
			{"account_id": "acc-2", "debit": 0, "credit": 9000}
		]
	}`)
	req, _ := http.NewRequest("POST", "/transactions", bytes.NewBuffer(payload))
	req.Header.Set("X-Tenant-ID", "tenant-1")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleTransactions)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for unbalanced transaction, got %d", rr.Code)
	}

	var response APIResponse
	json.NewDecoder(rr.Body).Decode(&response)
	if response.Message != "Debit and Credit must be equal" {
		t.Errorf("Expected debit=credit validation message, got: %s", response.Message)
	}
}

func TestHandleTransactions_Validation_Balanced(t *testing.T) {
	payload := []byte(`{
		"date": "2026-05-22",
		"description": "Sale",
		"reference": "INV-001",
		"lines": [
			{"account_id": "acc-1", "debit": 10000, "credit": 0},
			{"account_id": "acc-2", "debit": 0, "credit": 10000}
		]
	}`)
	req, _ := http.NewRequest("POST", "/transactions", bytes.NewBuffer(payload))
	req.Header.Set("X-Tenant-ID", "tenant-1")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleTransactions)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for balanced transaction, got %d", rr.Code)
	}
}

func TestHandleAccounts_MissingTenant(t *testing.T) {
	req, _ := http.NewRequest("GET", "/accounts", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleAccounts)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing tenant, got %d", rr.Code)
	}
}
