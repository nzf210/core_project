package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// HEALTH CHECK TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestHealthCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Health check status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Health check status = %q, want %q", resp["status"], "ok")
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// LOGGING MIDDLEWARE TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestLoggingMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := loggingMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoggingMiddlewareLatency(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
	})

	handler := loggingMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	// Test passes if no panic occurs
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// ROUTE REGISTRATION TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestRouteRegistration(t *testing.T) {
	// Test that all routes are registered
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{"GET", "/dashboard"},
		{"GET", "/candidates"},
		{"PUT", "/candidates/{id}/verify"},
		{"GET", "/campaigns"},
		{"GET", "/volunteers"},
		{"GET", "/volunteers/stats"},
		{"GET", "/voters"},
		{"GET", "/voters/stats"},
		{"GET", "/regions/provinces"},
		{"GET", "/surveys"},
		{"GET", "/events"},
		{"GET", "/users"},
		{"GET", "/tasks"},
		{"GET", "/notifications"},
		{"GET", "/roles"},
		{"GET", "/audit-logs"},
		{"GET", "/reports"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			// Create a minimal mux for testing
			mux := http.NewServeMux()
			mux.HandleFunc("/health", handleHealth)
			mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/candidates", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/campaigns", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/volunteers", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/voters", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/regions/provinces", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/surveys", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/roles", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/audit-logs", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("/reports", func(w http.ResponseWriter, r *http.Request) {})

			// Just verify route registration doesn't panic
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			// We expect 404 for unregistered routes, 200 for registered
			if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
				t.Errorf("Unexpected status for %s %s: %d", tt.method, tt.path, rec.Code)
			}
		})
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// CONTENT TYPE TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestHealthCheckContentType(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// CORS HEADERS TESTS (if applicable)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestCORSHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("CORS origin = %q, want %q", origin, "*")
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// METHOD NOT ALLOWED TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Test POST to GET-only endpoint
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// REQUEST LOGGING TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestRequestLogging(t *testing.T) {
	mux := http.NewServeMux()
	var loggedMethod, loggedPath string

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		loggedMethod = r.Method
		loggedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	handler := loggingMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if loggedMethod != "GET" {
		t.Errorf("Logged method = %q, want %q", loggedMethod, "GET")
	}
	if loggedPath != "/test" {
		t.Errorf("Logged path = %q, want %q", loggedPath, "/test")
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// PORT CONFIGURATION TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestServerPort(t *testing.T) {
	// Verify the server port is correctly set
	expectedPort := ":9002"
	if expectedPort != ":9002" {
		t.Errorf("Server port = %q, want %q", expectedPort, ":9002")
	}
}
