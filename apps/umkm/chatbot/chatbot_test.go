package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleChat_Validation(t *testing.T) {
	req, _ := http.NewRequest("POST", "/chat", bytes.NewBufferString(`{"message": "Halo"}`))
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleChat)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for missing tenant, got %d", rr.Code)
	}
}
