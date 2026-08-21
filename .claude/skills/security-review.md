# security-review

Security review untuk perubahan code sebelum deploy.

## When to invoke

- User mengetik `/security-review`
- Sebelum deploy ke production
- Setelah implement autentikasi/autorisasi
- Saat handle user input atau data sensitif

## What this does

Comprehensive security review untuk:
1. **Authentication & Authorization** - JWT, session, RBAC
2. **Input Validation** - SQL injection, XSS, path traversal
3. **Data Protection** - Encryption, hashing, secrets management
4. **API Security** - Rate limiting, CORS, CSRF
5. **Infrastructure** - Environment variables, dependencies

## Security Checklist

### 1. Authentication & Authorization

```bash
# Check JWT implementation
grep -rn "jwt.NewWithClaims\|jwt.Parse" --include="*.go"

# Check password hashing (must use bcrypt cost >= 12)
grep -rn "bcrypt.GenerateFromPassword" --include="*.go"

# Check RBAC implementation
grep -rn "role.*==\|HasPermission" --include="*.go"
```

**Common issues:**
- JWT tanpa expiry
- Password plain text atau weak hash (MD5, SHA1)
- Missing role validation

### 2. Input Validation

```bash
# SQL injection risk
grep -rn 'Query.*+\|Exec.*fmt.Sprintf.*SELECT' --include="*.go"

# Path traversal risk
grep -rn 'filepath.Join.*tenantID\|os.Open.*req\.' --include="*.go"

# Command injection
grep -rn 'exec.Command.*+\|os.system' --include="*.go"
```

**Common issues:**
- String concatenation di SQL queries
- User input langsung ke file path
- Unvalidated tenant ID

### 3. Data Protection

```bash
# Check encryption (must AES-256-GCM)
grep -rn 'aes.NewCipher\|cipher.NewGCM' --include="*.go"

# Check hardcoded secrets
grep -rn 'password.*=.*"\|apiKey.*=.*"' --include="*.go" --include="*.env*"

# Check PII handling (NIK, phone, email)
grep -rn 'encrypted_nik\|encrypted_' --include="*.go"
```

**Common issues:**
- Secrets di hardcode (API keys, passwords)
- PII tanpa enkripsi
- Weak encryption (AES-128, DES)

### 4. API Security

```bash
# Check CORS config
grep -rn 'AllowOrigins\|Access-Control-Allow-Origin' --include="*.go"

# Check rate limiting
grep -rn 'RateLimiter\|TokenBucket' --include="*.go"

# Check authentication middleware
grep -rn 'AuthMiddleware\|VerifyJWT' --include="*.go"
```

**Common issues:**
- CORS `*` di production
- No rate limiting
- Public endpoints tanpa auth

### 5. Infrastructure

```bash
# Check .env usage
grep -rn 'os.Getenv\|godotenv.Load' --include="*.go"

# Check dependencies dengan known vulnerabilities
go list -m all | grep -E 'vulnerable|outdated'

# Check exposed ports di docker-compose
grep -A2 'ports:' docker-compose*.yml
```

## Output Format

```
=== Security Review: /api/auth/login ===

❌ CRITICAL: Password stored in plain text
   File: services/auth-service/main.go:245
   Fix: Use bcrypt.GenerateFromPassword with cost 12

⚠️  HIGH: Missing rate limiting on login endpoint
   File: services/api-gateway/main.go:89
   Fix: Add rate limiter (5 attempts per minute)

✅ SQL Injection: All queries use parameterization
✅ XSS: All user inputs properly escaped
✅ Secrets: No hardcoded credentials found

=== Summary ===
Critical: 1
High: 1
Medium: 0

❌ DEPLOY BLOCKED - Fix critical issues first
```

## Integration

- Auto-invoke before production deploy
- Block deploy jika ada CRITICAL findings
- Generate security report untuk audit
