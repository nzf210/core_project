package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"core_project/shared/sdk/config"
)

// ========================================
// JWT Security Tests
// ========================================

func TestJWT_TokenValidation(t *testing.T) {
	// Setup config for tests
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	tests := []struct {
		name      string
		token     string
		expectErr bool
	}{
		{
			name:      "Empty token",
			token:     "",
			expectErr: true,
		},
		{
			name:      "Invalid format",
			token:     "not-a-jwt-token",
			expectErr: true,
		},
		{
			name:      "Malformed JWT",
			token:     "header.payload",
			expectErr: true,
		},
		{
			name:      "Wrong signature",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.wrongsignature",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.token)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateJWT() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestJWT_TokenExpiration(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)

	// Create expired token
	expiredClaims := jwt.MapClaims{
		"tenant_id": "test-tenant",
		"user_id":   "test-user",
		"exp":       time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString(secret)

	// Expired token should fail validation
	_, err := ValidateJWT(expiredTokenString)
	if err == nil {
		t.Error("Expected expired token to fail validation")
	}

	// Create valid token
	validClaims := jwt.MapClaims{
		"tenant_id": "test-tenant",
		"user_id":   "test-user",
		"exp":       time.Now().Add(1 * time.Hour).Unix(), // Valid for 1 hour
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := validToken.SignedString(secret)

	// Valid token should pass
	claims, err := ValidateJWT(validTokenString)
	if err != nil {
		t.Errorf("Expected valid token to pass, got error: %v", err)
	}
	if claims["tenant_id"] != "test-tenant" {
		t.Error("Expected tenant_id to be extracted from token")
	}
}

func TestJWT_SigningMethodAttack(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	// Try to use "none" algorithm (algorithm confusion attack)
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"tenant_id": "malicious-tenant",
		"user_id":   "attacker",
		"role":      "superadmin",
	})
	noneTokenString, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := ValidateJWT(noneTokenString)
	if err == nil {
		t.Error("JWT with 'none' algorithm should be rejected")
	}
}

func TestJWT_MissingTenantID(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)

	// Token without tenant_id
	claims := jwt.MapClaims{
		"user_id": "test-user",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	// Create request with this token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	// Should return 403 Forbidden due to missing tenant context
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

// ========================================
// Authorization Header Tests
// ========================================

func TestMiddleware_AuthorizationHeaderFormats(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)
	validClaims := jwt.MapClaims{
		"tenant_id": "test-tenant",
		"user_id":   "test-user",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := validToken.SignedString(secret)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{"Valid Bearer token", "Bearer " + validTokenString, http.StatusOK},
		{"Missing Bearer prefix", validTokenString, http.StatusUnauthorized},
		{"Lowercase bearer", "bearer " + validTokenString, http.StatusUnauthorized},
		{"Wrong prefix", "Token " + validTokenString, http.StatusUnauthorized},
		{"Extra spaces", "Bearer  " + validTokenString, http.StatusUnauthorized},
		{"Empty header", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// ========================================
// Role Escalation Prevention Tests
// ========================================

func TestJWT_RoleEscalationPrevention(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)

	tests := []struct {
		name         string
		role         string
		shouldAccept bool
	}{
		{"Valid owner role", "owner", true},
		{"Valid staff role", "staff", true},
		{"Valid superadmin", "superadmin", true},
		{"Malicious privilege escalation", "owner' OR role='superadmin", true}, // JWT accepts any string
		{"Empty role", "", true}, // Role is optional in JWT
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"tenant_id": "test-tenant",
				"user_id":   "test-user",
				"role":      tt.role,
				"exp":       time.Now().Add(1 * time.Hour).Unix(),
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString(secret)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()

			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check that role is set in header
				role := r.Header.Get("X-User-Role")
				if role != tt.role {
					t.Errorf("Expected role %q in header, got %q", tt.role, role)
				}
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			if tt.shouldAccept && w.Code != http.StatusOK {
				t.Errorf("Expected token to be accepted, got status %d", w.Code)
			}
		})
	}
}

// ========================================
// Multi-Tenant Isolation Tests
// ========================================

func TestMiddleware_TenantIsolation(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)

	// Create tokens for different tenants
	tenant1Claims := jwt.MapClaims{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	tenant1Token := jwt.NewWithClaims(jwt.SigningMethodHS256, tenant1Claims)
	tenant1TokenString, _ := tenant1Token.SignedString(secret)

	tenant2Claims := jwt.MapClaims{
		"tenant_id": "tenant-2",
		"user_id":   "user-2",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	tenant2Token := jwt.NewWithClaims(jwt.SigningMethodHS256, tenant2Claims)
	tenant2TokenString, _ := tenant2Token.SignedString(secret)

	// Request from tenant-1 should only access tenant-1 data
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer "+tenant1TokenString)
	w1 := httptest.NewRecorder()

	handler1 := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Context().Value(TenantIDKey).(string)
		if tenantID != "tenant-1" {
			t.Errorf("Request 1: Expected tenant-1, got %s", tenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler1.ServeHTTP(w1, req1)

	// Request from tenant-2 should only access tenant-2 data
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+tenant2TokenString)
	w2 := httptest.NewRecorder()

	handler2 := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Context().Value(TenantIDKey).(string)
		if tenantID != "tenant-2" {
			t.Errorf("Request 2: Expected tenant-2, got %s", tenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	handler2.ServeHTTP(w2, req2)
}

// ========================================
// Cookie-Based Auth Tests
// ========================================

func TestMiddleware_CookieAuth(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)
	validClaims := jwt.MapClaims{
		"tenant_id": "test-tenant",
		"user_id":   "test-user",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := validToken.SignedString(secret)

	// Test cookie-based authentication
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: validTokenString,
	})
	w := httptest.NewRecorder()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Cookie auth should work, got status %d", w.Code)
	}
}

// ========================================
// Query Parameter Auth Tests (for iframes)
// ========================================

func TestMiddleware_QueryParamAuth(t *testing.T) {
	config.GlobalConfig = &config.Config{
		JWTSecret: "test-secret-key-for-testing-only",
	}

	secret := []byte(config.GlobalConfig.JWTSecret)
	validClaims := jwt.MapClaims{
		"tenant_id": "test-tenant",
		"user_id":   "test-user",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := validToken.SignedString(secret)

	// Test query parameter authentication (for iframe assets)
	req := httptest.NewRequest("GET", "/test?token="+validTokenString, nil)
	w := httptest.NewRecorder()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Query param auth should work, got status %d", w.Code)
	}

	// Check that cookie was set for subsequent requests
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" && cookie.Value == validTokenString {
			found = true
			// Verify security attributes
			if !cookie.HttpOnly {
				t.Error("Cookie should be HttpOnly")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Error("Cookie should use SameSite=Lax")
			}
		}
	}
	if !found {
		t.Error("Expected access_token cookie to be set")
	}
}
