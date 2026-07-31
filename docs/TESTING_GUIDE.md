# Testing Guide — WCH Platform

Panduan lengkap untuk menjalankan dan menulis test di WCH Platform.

## Quick Start

```bash
# Run all tests
make check

# Test per service
go test ./services/auth-service/... -v
go test ./services/billing-service/... -v
go test ./services/wa-gateway/... -v

# Test per package
go test ./services/auth-service -v -run TestPassword

# Frontend tests
cd frontend/umkm-web && npm test
cd frontend/campaign-web && npm test
cd frontend/superadmin-web && npm test
```

## Test Coverage Requirements

Berdasarkan SonarQube standards:
- **Target:** 75% coverage minimum
- **Current:** ~17.8% (sedang ditingkatkan ke 75%)
- **Critical paths:** 90%+ coverage (auth, payment, data encryption)

## Backend Testing (Go)

### Test File Naming
```
main.go → main_test.go
handlers.go → handlers_test.go
security.go → security_test.go
```

### Test Structure (Table-Driven)

```go
func TestFeatureName_Scenario(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"Valid input", "test", "TEST", false},
        {"Empty input", "", "", true},
        {"Special chars", "test@123", "TEST@123", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := YourFunction(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
            }
            
            if result != tt.expected {
                t.Errorf("Expected %q, got %q", tt.expected, result)
            }
        })
    }
}
```

### HTTP Handler Testing

```go
func TestHandler_Success(t *testing.T) {
    // Setup request
    req := httptest.NewRequest("POST", "/api/endpoint", bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer token")
    req.Header.Set("X-Tenant-ID", "tenant-123")
    
    // Setup response recorder
    w := httptest.NewRecorder()
    
    // Call handler
    handler := http.HandlerFunc(yourHandler)
    handler.ServeHTTP(w, req)
    
    // Assert response
    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }
    
    var response Response
    json.NewDecoder(w.Body).Decode(&response)
    
    if !response.Success {
        t.Error("Expected success=true")
    }
}
```

### Database Testing (Skip if DB unavailable)

```go
func TestDatabaseOperation(t *testing.T) {
    if DB == nil {
        t.Skip("Skipping test: DB not available")
    }
    
    ctx := context.Background()
    
    // Test logic
    result, err := DB.Query(ctx, "SELECT * FROM table WHERE id = $1", id)
    if err != nil {
        t.Fatalf("Query failed: %v", err)
    }
    defer result.Close()
}
```

### Mocking External Services

```go
// Mock HTTP client
mockClient := &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
        return &http.Response{
            StatusCode: 200,
            Body:       io.NopCloser(bytes.NewBufferString(`{"success":true}`)),
        }, nil
    }),
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
    return f(req)
}
```

### Security Tests

```go
func TestInputValidation_SQLInjection(t *testing.T) {
    injectionAttempts := []string{
        "admin' OR '1'='1",
        "'; DROP TABLE users;--",
        "admin'--",
    }
    
    for _, attempt := range injectionAttempts {
        t.Run(attempt, func(t *testing.T) {
            // Validation should reject
            if isValidUsername(attempt) {
                t.Errorf("SQL injection %q passed validation", attempt)
            }
        })
    }
}

func TestEncryption_AES256(t *testing.T) {
    key := make([]byte, 32) // AES-256
    plaintext := []byte("sensitive data")
    
    ciphertext, err := encrypt(plaintext, key)
    if err != nil {
        t.Fatal(err)
    }
    
    // Ciphertext should differ from plaintext
    if bytes.Equal(ciphertext, plaintext) {
        t.Error("Encryption failed: ciphertext equals plaintext")
    }
    
    // Decrypt and verify
    decrypted, err := decrypt(ciphertext, key)
    if err != nil {
        t.Fatal(err)
    }
    
    if !bytes.Equal(decrypted, plaintext) {
        t.Error("Decryption failed")
    }
}
```

## Frontend Testing (Vitest + Vue Test Utils)

### Component Testing

```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Dashboard from '@/components/Dashboard.vue'

describe('Dashboard.vue', () => {
  it('renders dashboard layout', () => {
    const wrapper = mount(Dashboard)
    expect(wrapper.find('.dashboard').exists()).toBe(true)
  })
  
  it('fetches data on mount', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ success: true, data: {} })
      })
    ) as any
    
    const wrapper = mount(Dashboard)
    await wrapper.vm.$nextTick()
    
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/umkm/dashboard'),
      expect.any(Object)
    )
  })
})
```

### API Testing

```typescript
describe('API Utilities', () => {
  it('includes Authorization header', () => {
    localStorage.setItem('token', 'test-token')
    
    const headers = buildRequestHeaders()
    expect(headers['Authorization']).toBe('Bearer test-token')
  })
  
  it('handles 401 errors', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ message: 'Unauthorized' })
      })
    ) as any
    
    // Should clear token and redirect
    const response = await fetch('/api/test')
    expect(response.status).toBe(401)
  })
})
```

### Security Testing

```typescript
describe('XSS Prevention', () => {
  it('escapes HTML in user input', () => {
    const malicious = '<script>alert("xss")</script>'
    const escaped = escapeHTML(malicious)
    
    expect(escaped).not.toContain('<script>')
    expect(escaped).toContain('&lt;script&gt;')
  })
})
```

## Test Commands

### Backend

```bash
# All tests
go test ./...

# Specific package
go test ./services/auth-service

# Specific test
go test -run TestPassword_BcryptHashing

# With coverage
go test ./... -cover
go test ./services/auth-service -coverprofile=coverage.out
go tool cover -html=coverage.out

# Verbose output
go test ./... -v

# Parallel execution
go test ./... -parallel 4

# Timeout (default 10m)
go test ./... -timeout 30m

# Race detection
go test ./... -race
```

### Frontend

```bash
cd frontend/umkm-web

# Run tests
npm test

# Watch mode
npm test -- --watch

# Coverage
npm run test:coverage

# UI mode (browser-based)
npm run test:ui

# Specific test
npm test -- Dashboard.spec.ts
```

## CI/CD Testing

### GitHub Actions Workflow

```yaml
name: Test

on: [push, pull_request]

jobs:
  test-backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
      redis:
        image: redis:7
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Run tests
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/test
          REDIS_ADDR: localhost:6379
        run: |
          go test ./... -v -cover -race
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'
      
      - name: Install dependencies
        run: |
          cd frontend/umkm-web && npm ci
      
      - name: Run tests
        run: |
          cd frontend/umkm-web && npm test
```

## Test Categories

### Unit Tests
- **Scope:** Single function/method
- **Dependencies:** None or mocked
- **Speed:** Fast (<100ms per test)
- **Coverage target:** 80%+

**Example:** Input validation, data transformation, pure functions

### Integration Tests
- **Scope:** Multiple components working together
- **Dependencies:** Real DB/Redis (or test containers)
- **Speed:** Medium (100ms-1s per test)
- **Coverage target:** 60%+

**Example:** API endpoint → DB query → response

### Security Tests
- **Scope:** Attack vectors & vulnerabilities
- **Dependencies:** None (validation logic)
- **Speed:** Fast
- **Coverage:** All critical paths

**Example:** SQL injection, XSS, JWT validation, encryption

### End-to-End Tests
- **Scope:** Full user flows
- **Dependencies:** Running services
- **Speed:** Slow (1s-10s per test)
- **Coverage:** Critical user journeys only

**Example:** Register → Verify OTP → Login → Dashboard

## Test Data

### Mock Data Patterns

```go
// Use realistic but obviously fake data
const (
    mockTenantID  = "11111111-1111-1111-1111-111111111111"
    mockUserID    = "22222222-2222-2222-2222-222222222222"
    mockEmail     = "test@example.com"
    mockPhone     = "628123456789"
    mockUsername  = "testuser"
)

// Avoid magic numbers
const (
    testTimeout     = 5 * time.Second
    testAmount      = 10000000 // 100,000 IDR in sen
    testValidityDays = 30
)
```

### Database Fixtures

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("postgres", "postgres://...")
    if err != nil {
        t.Fatal(err)
    }
    
    // Clean slate
    _, _ = db.Exec("TRUNCATE TABLE users, tenants CASCADE")
    
    // Insert fixtures
    _, _ = db.Exec("INSERT INTO tenants (id, name) VALUES ($1, $2)", mockTenantID, "Test Tenant")
    
    return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
    db.Close()
}
```

## Test Best Practices

### DO ✅

1. **Test behavior, not implementation**
   ```go
   // Good
   if result != expected {
       t.Errorf("Expected %v, got %v", expected, result)
   }
   
   // Bad (implementation detail)
   if len(internalCache) != 5 {
       t.Error("Cache size wrong")
   }
   ```

2. **Use table-driven tests**
   ```go
   tests := []struct{
       name string
       // inputs & expected outputs
   }{
       {"case 1", ...},
       {"case 2", ...},
   }
   ```

3. **Descriptive test names**
   ```go
   // Good
   TestPassword_BcryptCost12
   TestVoucher_RedemptionIdempotency
   
   // Bad
   TestFunc1
   TestCase2
   ```

4. **Arrange-Act-Assert pattern**
   ```go
   // Arrange
   input := "test"
   expected := "TEST"
   
   // Act
   result := Transform(input)
   
   // Assert
   if result != expected {
       t.Error(...)
   }
   ```

5. **Test edge cases**
   - Empty input
   - Null/nil values
   - Boundary values (0, max int)
   - Special characters
   - Very long inputs

### DON'T ❌

1. **❌ Test private functions directly**
   - Test through public interface

2. **❌ Depend on test execution order**
   - Each test must be independent

3. **❌ Use sleep for synchronization**
   ```go
   // Bad
   go doWork()
   time.Sleep(1 * time.Second)
   
   // Good
   done := make(chan bool)
   go func() {
       doWork()
       done <- true
   }()
   <-done
   ```

4. **❌ Ignore test failures**
   - Fix or skip with reason: `t.Skip("reason")`

5. **❌ Write flaky tests**
   - No random values without seed
   - No timing dependencies
   - Deterministic test data

## Debugging Tests

### Verbose Output

```bash
# Go
go test -v ./services/auth-service

# Frontend
npm test -- --reporter=verbose
```

### Run Single Test

```bash
# Go
go test -run TestPassword_BcryptHashing ./services/auth-service

# Frontend
npm test -- Dashboard.spec.ts
```

### Print Debug Info

```go
func TestDebug(t *testing.T) {
    result := someFunc()
    t.Logf("Result: %+v", result)  // Only prints if test fails or -v flag
}
```

### Test with Debugger

```bash
# Delve (Go debugger)
dlv test ./services/auth-service -- -test.run TestPassword

# VS Code: Add breakpoint, run "Debug Test"
```

## Performance Testing

### Benchmark Tests (Go)

```go
func BenchmarkPasswordHash(b *testing.B) {
    password := []byte("TestPassword123")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bcrypt.GenerateFromPassword(password, 12)
    }
}

// Run: go test -bench=. -benchmem
```

### Load Testing

```bash
# hey (HTTP load generator)
hey -n 10000 -c 100 http://localhost:8000/api/endpoint

# Expected: 1000 RPS, p95 < 100ms
```

## Coverage Goals by Service

| Service | Current | Target | Priority |
|:--------|:--------|:-------|:---------|
| auth-service | 45% | 85% | Critical |
| billing-service | 60% | 80% | High |
| wa-gateway | 30% | 75% | High |
| api-gateway | 20% | 70% | Medium |
| umkm-accounting | 15% | 75% | High |
| campaign-api | 10% | 70% | Medium |

## Troubleshooting

### Tests hang
**Cause:** Deadlock, missing timeout, or waiting for channel  
**Solution:** Add `-timeout 30s` flag

### Race condition detected
**Cause:** Concurrent access to shared data  
**Solution:** Use mutexes or channels, run with `-race` to detect

### DB connection error
**Cause:** PostgreSQL/Redis not running  
**Solution:**
```bash
docker compose up -d postgres redis
```

### Import cycle
**Cause:** Circular dependencies between packages  
**Solution:** Refactor to break cycle, or move shared code to separate package

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Vitest Documentation](https://vitest.dev/)
- [Vue Test Utils](https://test-utils.vuejs.org/)
- [Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
