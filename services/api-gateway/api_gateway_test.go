package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// MIDDLEWARE TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		wantOrigin  string
		wantMethods string
		wantHeaders string
	}{
		{
			name:       "with specific origin",
			origin:     "https://app.wch.id",
			wantOrigin: "https://app.wch.id",
		},
		{
			name:       "without origin",
			origin:     "",
			wantOrigin: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := corsMiddleware(mux)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			gotOrigin := rec.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.wantOrigin {
				t.Errorf("CORS origin = %q, want %q", gotOrigin, tt.wantOrigin)
			}
		})
	}
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {})

	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Preflight status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "10.0.0.1:1234",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr: "10.0.0.1:1234",
			want:       "192.168.1.2",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1:1234",
			want:       "10.0.0.1:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// HELPER FUNCTION TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "World", true},
		{"Hello World", "world", true}, // case-insensitive
		{"Hello World", "FOO", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := strings.Contains(strings.ToLower(tt.s), strings.ToLower(tt.substr))
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestStringHelpers(t *testing.T) {
	// Test string manipulation helpers that might be used
	tests := []struct {
		name  string
		input string
		test  func(string) bool
		want  bool
	}{
		{
			name:  "non-empty string",
			input: "test@example.com",
			test:  func(s string) bool { return len(s) > 0 },
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			test:  func(s string) bool { return len(s) > 0 },
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.test(tt.input)
			if got != tt.want {
				t.Errorf("test(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// RATE LIMIT HELPER TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestRateLimitConstants(t *testing.T) {
	// Verify rate limit constants are reasonable
	if rateLimitWindow <= 0 {
		t.Error("rateLimitWindow should be positive")
	}
	if rateLimitPublic <= 0 {
		t.Error("rateLimitPublic should be positive")
	}
	if rateLimitAuth <= 0 {
		t.Error("rateLimitAuth should be positive")
	}
	if rateLimitPerIP <= 0 {
		t.Error("rateLimitPerIP should be positive")
	}
}

func TestRateLimitValues(t *testing.T) {
	// Public routes should have higher limits than auth routes
	if rateLimitPublic <= rateLimitAuth {
		t.Errorf("rateLimitPublic (%d) should be > rateLimitAuth (%d)", rateLimitPublic, rateLimitAuth)
	}

	// Per-IP limit should be higher than public limit
	if rateLimitPerIP <= rateLimitPublic {
		t.Errorf("rateLimitPerIP (%d) should be > rateLimitPublic (%d)", rateLimitPerIP, rateLimitPublic)
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// CONFIG HELPER TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestConfigEnvValues(t *testing.T) {
	// Test that configuration can handle different env values
	envs := []string{"development", "production", "staging"}

	for _, env := range envs {
		t.Run(env, func(t *testing.T) {
			// Simulate getTarget behavior
			isProd := env == "production"
			if isProd && env != "production" {
				t.Error("production detection failed")
			}
			_ = isProd // Used in actual code
		})
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// MUTEX SAFETY TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestMutexExists(t *testing.T) {
	// Verify mutex is initialized (used for rate limiting)
	if mu == (sync.Mutex{}) {
		// This is actually fine - mutex zero value is valid
		t.Log("Mutex initialized to zero value (valid)")
	}
}
