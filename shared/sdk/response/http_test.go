package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	
	data := map[string]string{"foo": "bar"}
	JSON(recorder, http.StatusOK, "Request successful", data)
	
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
	
	var resp APIResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if resp.Status != http.StatusOK {
		t.Errorf("Expected Status 200")
	}
	if resp.Message != "Request successful" {
		t.Errorf("Expected message 'Request successful', got %s", resp.Message)
	}
}

func TestError(t *testing.T) {
	recorder := httptest.NewRecorder()
	
	Error(recorder, http.StatusBadRequest, "Invalid input", errors.New("bad field"))
	
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}
	
	var resp APIResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if resp.Status != http.StatusBadRequest {
		t.Errorf("Expected Status 400")
	}
	if resp.Message != "Invalid input" {
		t.Errorf("Expected message 'Invalid input', got %s", resp.Message)
	}
	if resp.Error != "bad field" {
		t.Errorf("Expected error 'bad field', got %s", resp.Error)
	}
}
