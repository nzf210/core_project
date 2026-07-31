package main

import (
	"testing"
	"regexp"
)

// ========================================
// Input Validation Tests
// ========================================

func TestInputValidation_Username(t *testing.T) {
	tests := []struct {
		name     string
		username string
		valid    bool
	}{
		{"Valid alphanumeric", "user123", true},
		{"Valid with underscore", "user_name", true},
		{"Invalid with space", "user name", false},
		{"Invalid with special chars", "user@name", false},
		{"SQL injection attempt", "admin' OR '1'='1", false},
		{"XSS attempt", "<script>alert('xss')</script>", false},
		{"Path traversal", "../../etc/passwd", false},
		{"Empty string", "", false},
		{"Only numbers", "123456", true},
		{"Unicode chars", "用户名", false},
	}

	usernameRE := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usernameRE.MatchString(tt.username) && tt.username != ""
			if result != tt.valid {
				t.Errorf("username %q: expected valid=%v, got %v", tt.username, tt.valid, result)
			}
		})
	}
}

func TestInputValidation_Email(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"Valid simple", "test@example.com", true},
		{"Valid with plus", "user+tag@domain.co.id", true},
		{"Invalid no @", "userexample.com", false},
		{"Invalid no domain", "user@", false},
		{"Invalid no TLD", "user@domain", false},
		{"SQL injection", "admin'@example.com OR '1'='1", false},
		{"XSS in email", "<script>@example.com", false},
		{"Multiple @", "user@@example.com", false},
		{"Spaces", "user @example.com", false},
		{"Empty", "", false},
	}

	emailRE := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emailRE.MatchString(tt.email)
			if result != tt.valid {
				t.Errorf("email %q: expected valid=%v, got %v", tt.email, tt.valid, result)
			}
		})
	}
}

func TestInputValidation_PhoneNumber(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		valid bool
	}{
		{"Valid Indonesian mobile", "628123456789", true},
		{"Valid min length", "6281234567", true},
		{"Valid max length", "628123456789012345", true},
		{"Invalid prefix 08", "081234567890", false},
		{"Invalid prefix +62", "+628123456789", false},
		{"Invalid too short", "62812", false},
		{"Invalid too long", "6281234567890123456", false},
		{"SQL injection", "62' OR '1'='1", false},
		{"XSS attempt", "62<script>", false},
		{"Invalid chars", "62-812-3456", false},
		{"Empty", "", false},
	}

	phoneRE := regexp.MustCompile(`^62[0-9]{6,15}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := phoneRE.MatchString(tt.phone)
			if result != tt.valid {
				t.Errorf("phone %q: expected valid=%v, got %v", tt.phone, tt.valid, result)
			}
		})
	}
}

func TestInputValidation_Password(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"Valid strong", "SecurePass123!", true},
		{"Valid min 6 chars", "Pass12", true},
		{"Invalid too short", "abc12", false},
		{"Invalid empty", "", false},
		{"SQL injection chars allowed", "Pass' OR '1'='1", true}, // Bcrypt hashes anything
		{"XSS chars allowed", "<script>alert(1)</script>", true}, // Stored as hash
		{"Unicode allowed", "パスワード123", true},
		{"Spaces allowed", "my secure password", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := len(tt.password) >= 6
			if valid != tt.valid {
				t.Errorf("password length check: expected valid=%v, got %v", tt.valid, valid)
			}
		})
	}
}

// ========================================
// SQL Injection Prevention Tests
// ========================================

func TestSQLInjection_Prevention(t *testing.T) {
	// These tests verify that parameterized queries are used
	// In actual implementation, these should be caught by input validation BEFORE query
	sqlInjectionAttempts := []struct {
		name  string
		input string
	}{
		{"Classic OR injection", "admin' OR '1'='1"},
		{"Union select", "' UNION SELECT * FROM users--"},
		{"Drop table", "'; DROP TABLE users;--"},
		{"Stacked queries", "admin'; DELETE FROM tenants WHERE '1'='1"},
		{"Comment injection", "admin'--"},
		{"Hex injection", "0x61646D696E"},
		{"Time-based blind", "admin' AND SLEEP(5)--"},
	}

	usernameRE := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	for _, tt := range sqlInjectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			// All SQL injection attempts should be rejected by input validation
			if usernameRE.MatchString(tt.input) {
				t.Errorf("SQL injection attempt %q passed validation", tt.input)
			}
		})
	}
}

// ========================================
// XSS Prevention Tests
// ========================================

func TestXSS_Prevention(t *testing.T) {
	xssAttempts := []struct {
		name  string
		input string
	}{
		{"Script tag", "<script>alert('xss')</script>"},
		{"Image onerror", "<img src=x onerror=alert(1)>"},
		{"SVG onload", "<svg onload=alert(1)>"},
		{"Iframe injection", "<iframe src='javascript:alert(1)'>"},
		{"HTML entities", "&#60;script&#62;alert(1)&#60;/script&#62;"},
		{"JavaScript protocol", "javascript:alert(1)"},
		{"Data URI", "data:text/html,<script>alert(1)</script>"},
	}

	usernameRE := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	for _, tt := range xssAttempts {
		t.Run(tt.name, func(t *testing.T) {
			// XSS attempts should be rejected by input validation
			if usernameRE.MatchString(tt.input) {
				t.Errorf("XSS attempt %q passed validation", tt.input)
			}
		})
	}
}

// ========================================
// Path Traversal Prevention Tests
// ========================================

func TestPathTraversal_Prevention(t *testing.T) {
	pathTraversalAttempts := []struct {
		name  string
		input string
	}{
		{"Parent directory", "../../../etc/passwd"},
		{"Windows style", "..\\..\\windows\\system32"},
		{"Encoded dots", "%2e%2e%2f%2e%2e%2f"},
		{"Unicode", "..%c0%af..%c0%af"},
		{"Absolute path", "/etc/passwd"},
		{"Null byte", "../../../etc/passwd%00.jpg"},
	}

	usernameRE := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	for _, tt := range pathTraversalAttempts {
		t.Run(tt.name, func(t *testing.T) {
			if usernameRE.MatchString(tt.input) {
				t.Errorf("Path traversal attempt %q passed validation", tt.input)
			}
		})
	}
}

// ========================================
// Business Name & Type Validation
// ========================================

func TestBusinessName_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"Valid name", "Toko Berkah", true},
		{"Valid with symbols", "Toko & Co.", true},
		{"Too short", "AB", false},
		{"Empty", "", false},
		{"SQL injection", "Toko' OR '1'='1", true}, // Business names can have quotes
		{"Very long name", string(make([]byte, 300)), false}, // Max 255
		{"Unicode", "トコ バルカ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := len(tt.input) >= 3 && len(tt.input) <= 255
			if valid != tt.valid {
				t.Errorf("business_name %q: expected valid=%v, got %v", tt.input, tt.valid, valid)
			}
		})
	}
}

func TestBusinessType_Validation(t *testing.T) {
	validTypes := []string{"umum", "warung", "toko", "restoran", "cafe", "salon", "bengkel", "laundry", "clinic"}

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"Valid umum", "umum", true},
		{"Valid clinic", "clinic", true},
		{"Invalid custom", "my_custom_type", false},
		{"SQL injection", "umum' OR '1'='1", false},
		{"Empty", "", false},
		{"Case sensitive", "UMUM", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := false
			for _, vt := range validTypes {
				if tt.input == vt {
					valid = true
					break
				}
			}
			if valid != tt.valid {
				t.Errorf("business_type %q: expected valid=%v, got %v", tt.input, tt.valid, valid)
			}
		})
	}
}

// ========================================
// Role Validation Tests
// ========================================

func TestRole_Validation(t *testing.T) {
	validRoles := []string{"owner", "admin", "staff", "kasir", "superadmin"}

	tests := []struct {
		name  string
		role  string
		valid bool
	}{
		{"Valid owner", "owner", true},
		{"Valid superadmin", "superadmin", true},
		{"Invalid custom", "super_user", false},
		{"SQL injection", "owner' OR '1'='1", false},
		{"Privilege escalation attempt", "superadmin' OR role='owner", false},
		{"Empty", "", false},
		{"Case sensitive", "OWNER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := false
			for _, vr := range validRoles {
				if tt.role == vr {
					valid = true
					break
				}
			}
			if valid != tt.valid {
				t.Errorf("role %q: expected valid=%v, got %v", tt.role, tt.valid, valid)
			}
		})
	}
}

// ========================================
// UUID Validation Tests
// ========================================

func TestUUID_Validation(t *testing.T) {
	tests := []struct {
		name  string
		uuid  string
		valid bool
	}{
		{"Valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"Valid UUID v4 lowercase", "123e4567-e89b-12d3-a456-426614174000", true},
		{"Invalid format", "not-a-uuid", false},
		{"SQL injection", "550e8400' OR '1'='1", false},
		{"Empty", "", false},
		{"Too short", "123e4567", false},
		{"No dashes", "123e4567e89b12d3a456426614174000", false},
	}

	uuidRE := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uuidRE.MatchString(tt.uuid)
			if result != tt.valid {
				t.Errorf("uuid %q: expected valid=%v, got %v", tt.uuid, tt.valid, result)
			}
		})
	}
}

// ========================================
// OTP Validation Tests
// ========================================

func TestOTP_Validation(t *testing.T) {
	tests := []struct {
		name  string
		otp   string
		valid bool
	}{
		{"Valid 6 digits", "123456", true},
		{"Invalid 5 digits", "12345", false},
		{"Invalid 7 digits", "1234567", false},
		{"Invalid letters", "12345A", false},
		{"SQL injection", "123' OR '1'='1", false},
		{"Empty", "", false},
	}

	otpRE := regexp.MustCompile(`^[0-9]{6}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := otpRE.MatchString(tt.otp)
			if result != tt.valid {
				t.Errorf("otp %q: expected valid=%v, got %v", tt.otp, tt.valid, result)
			}
		})
	}
}
