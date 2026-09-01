package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// postJSONWithTenant is an alias for postJSONWithAuth with tenant context.
func postJSONWithTenant(baseURL, path string, body interface{}, token, tenantID string) (*http.Response, error) {
	return postJSONWithAuth(baseURL, path, body, token, tenantID)
}

// patchJSONWithAuth sends a PATCH request with JSON body and auth/tenant headers.
func patchJSONWithAuth(baseURL, path string, body interface{}, token, tenantID string) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("PATCH", baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	return http.DefaultClient.Do(req)
}

// postJSONWithHeader posts JSON with a single custom header (e.g. x-callback-token for webhook).
func postJSONWithHeader(baseURL, path string, body interface{}, headerKey, headerVal string) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if headerKey != "" {
		req.Header.Set(headerKey, headerVal)
	}
	return http.DefaultClient.Do(req)
}

// registerAndLogin creates a fresh tenant and returns a populated TestState.
// Returns nil and calls t.FailNow on any hard failure.
func registerAndLogin(t *testing.T, log *TestLogger, prefix string) *TestState {
	t.Helper()
	nano := time.Now().UnixNano()
	phone := fmt.Sprintf("628%d", nano%9000000000)
	username := fmt.Sprintf("%s%d", prefix, nano)

	log.Start(fmt.Sprintf("Registering user %s (phone: %s)...", username, phone))
	resp, err := postJSON(authServiceURL, "/register", RegisterReq{
		Username:    username,
		Password:    "TestPassword123!",
		Email:       fmt.Sprintf("%s%d@example.com", prefix, nano),
		PhoneNumber: phone,
	})
	if err != nil {
		log.Error("Register failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var regResp Response
	json.NewDecoder(resp.Body).Decode(&regResp)
	if !regResp.Success {
		log.Error("Register failed: " + regResp.Message)
		t.FailNow()
	}
	log.Success("Registered")

	// verify OTP (dev mode accepts 000000)
	resp, err = postJSON(authServiceURL, "/verify-otp", map[string]string{
		"phoneNumber": phone,
		"otp":         "000000",
	})
	if err != nil {
		log.Error("Verify OTP failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	// login
	resp, err = postJSON(authServiceURL, "/login", LoginReq{
		Username: username,
		Password: "TestPassword123!",
	})
	if err != nil {
		log.Error("Login failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		log.Error("Login failed: " + loginResp.Message)
		t.FailNow()
	}

	data, _ := loginResp.Data.(map[string]interface{})
	state := &TestState{
		AccessToken: extractStringField(loginResp.Data, "accessToken"),
		TenantID:    extractStringField(loginResp.Data, "tenantId"),
		Phone:       phone,
		Username:    username,
	}
	_ = data
	if state.AccessToken == "" || state.TenantID == "" {
		log.Error("Login response missing accessToken or tenantId")
		t.FailNow()
	}
	log.Auth(fmt.Sprintf("Logged in (tenant: %s)", state.TenantID))
	return state
}

// extractStringField pulls a string field from an interface{} that is map[string]interface{}.
func extractStringField(data interface{}, key string) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

// extractInt64Field pulls an int64 field from an interface{} map.
func extractInt64Field(data interface{}, key string) int64 {
	m, ok := data.(map[string]interface{})
	if !ok {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// extractFirstPlanID returns the first plan ID from a list plans response data.
func extractFirstPlanID(data interface{}) string {
	// data may be []interface{} or map with a "plans" key
	var plans []interface{}
	switch v := data.(type) {
	case []interface{}:
		plans = v
	case map[string]interface{}:
		if p, ok := v["plans"].([]interface{}); ok {
			plans = p
		}
	}
	for _, p := range plans {
		if pm, ok := p.(map[string]interface{}); ok {
			if id, ok := pm["id"].(string); ok && id != "" {
				// prefer non-free plans when possible
				if id != "lite" && id != "free" {
					return id
				}
			}
		}
	}
	// fallback: return first regardless
	for _, p := range plans {
		if pm, ok := p.(map[string]interface{}); ok {
			if id, ok := pm["id"].(string); ok && id != "" {
				return id
			}
		}
	}
	return ""
}

// extractVoucherCode extracts the first voucher code from a generate-vouchers response.
func extractVoucherCode(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	codes, ok := m["codes"].([]interface{})
	if !ok || len(codes) == 0 {
		return ""
	}
	first, ok := codes[0].(map[string]interface{})
	if !ok {
		return ""
	}
	code, _ := first["code"].(string)
	return code
}

// getEnvOrDefault returns the env var value or a default.
func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// containsStr reports whether substr appears in s (case-sensitive).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
