# Security Best Practices — WCH Platform

Panduan praktik keamanan untuk development dan production WCH Platform.

## 🔐 Authentication & Authorization

### Password Security

**✅ DO:**
```go
// Gunakan bcrypt cost=12 (WAJIB sesuai CLAUDE.md)
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// Validate password complexity
if len(password) < 6 {
    return errors.New("Password minimum 6 karakter")
}
```

**❌ DON'T:**
```go
// JANGAN gunakan MD5/SHA untuk password
hash := md5.Sum([]byte(password))  // TIDAK AMAN!

// JANGAN simpan plaintext
db.Exec("INSERT INTO users (password) VALUES ($1)", password)  // BAHAYA!

// JANGAN bcrypt cost terlalu rendah
hash, _ := bcrypt.GenerateFromPassword([]byte(password), 4)  // Terlalu lemah!
```

### JWT Token Management

**✅ DO:**
```go
// Include tenant_id, user_id, role, exp
claims := jwt.MapClaims{
    "tenant_id": tenantID,
    "user_id":   userID,
    "role":      role,
    "exp":       time.Now().Add(24 * time.Hour).Unix(),
}

// Sign dengan HS256
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenString, _ := token.SignedString([]byte(jwtSecret))
```

**❌ DON'T:**
```go
// JANGAN gunakan algorithm "none"
token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)

// JANGAN skip expiry validation
// JANGAN hardcode JWT secret
jwtSecret := "mysecret123"  // BAHAYA!

// JANGAN store JWT di localStorage tanpa XSS protection
localStorage.setItem('token', token)  // Vulnerable to XSS
```

**Best Practice:**
- JWT secret minimal 32 karakter random
- Token expiry 24 jam
- Refresh token stored di Redis + PostgreSQL
- HttpOnly cookies lebih aman dari localStorage

### Multi-Tenant Isolation

**✅ DO:**
```go
// Selalu validate tenant_id dari JWT
tenantID := r.Header.Get("X-Tenant-ID")
if tenantID == "" {
    return http.StatusBadRequest
}

// WHERE clause dengan tenant_id
rows, err := DB.Query(ctx, 
    "SELECT * FROM transactions WHERE tenant_id = $1 AND id = $2",
    tenantID, transactionID)
```

**❌ DON'T:**
```go
// JANGAN query tanpa tenant_id filter
rows, err := DB.Query(ctx, "SELECT * FROM transactions WHERE id = $1", id)

// JANGAN trust tenant_id dari request body
tenantID := req.TenantID  // User bisa manipulasi!
```

## 🛡️ Input Validation

### Parameterized Queries (Anti SQL Injection)

**✅ DO:**
```go
// Parameterized query
rows, err := DB.Query(ctx, 
    "SELECT * FROM users WHERE username = $1 AND tenant_id = $2",
    username, tenantID)
```

**❌ DON'T:**
```go
// String concatenation — SQL INJECTION!
query := "SELECT * FROM users WHERE username = '" + username + "'"
rows, err := DB.Query(ctx, query)
```

### Input Sanitization

**✅ DO:**
```go
// Regex validation
var (
    usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
    emailRE    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
    phoneRE    = regexp.MustCompile(`^62[0-9]{9,13}$`)
)

if !usernameRE.MatchString(username) {
    return errors.New("Invalid username format")
}

// Length validation
if len(businessName) < 3 || len(businessName) > 255 {
    return errors.New("Business name must be 3-255 characters")
}

// Enum validation
validTypes := []string{"umum", "warung", "clinic"}
if !contains(validTypes, businessType) {
    return errors.New("Invalid business type")
}
```

**❌ DON'T:**
```go
// JANGAN terima input mentah tanpa validasi
db.Exec("INSERT INTO tenants (name) VALUES ($1)", userInput)

// JANGAN skip validation untuk "trusted" input
// Semua input user harus divalidasi!
```

### XSS Prevention (Frontend)

**✅ DO:**
```typescript
// Escape HTML
function escapeHTML(str: string): string {
    const div = document.createElement('div')
    div.textContent = str
    return div.innerHTML
}

// Sanitize sebelum v-html
import DOMPurify from 'dompurify'
const clean = DOMPurify.sanitize(dirty)

// Validate URLs
const dangerousProtocols = ['javascript:', 'data:', 'vbscript:']
if (dangerousProtocols.some(p => url.toLowerCase().startsWith(p))) {
    return false
}
```

**❌ DON'T:**
```typescript
// JANGAN v-html tanpa sanitization
<div v-html="userContent"></div>

// JANGAN innerHTML tanpa escape
element.innerHTML = userInput

// JANGAN eval() user input
eval(userCode)  // SANGAT BERBAHAYA!
```

## 🔒 Data Encryption

### AES-256-GCM Encryption

**✅ DO:**
```go
// Encrypt sensitive data (NIK, API keys)
func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)  // key WAJIB 32 bytes
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}
```

**❌ DON'T:**
```go
// JANGAN simpan data sensitif plaintext
db.Exec("INSERT INTO campaign_voters (nik) VALUES ($1)", nik)

// JANGAN gunakan ECB mode (tidak aman!)
cipher.NewCipher(key)  // tanpa GCM/CBC

// JANGAN reuse nonce
nonce := []byte("fixed-nonce-123")  // BAHAYA!
```

**Data yang WAJIB dienkripsi:**
- NIK (Nomor Induk Kependudukan)
- Xendit API keys per-tenant
- WhatsApp Cloud API access tokens
- Refresh tokens (hash SHA-256)

### Encryption Key Management

**✅ DO:**
```bash
# Generate secure key (32 bytes untuk AES-256)
openssl rand -base64 32 > encryption.key

# Store di environment variable
export ENCRYPTION_KEY=$(cat encryption.key)

# JANGAN commit ke git!
echo "encryption.key" >> .gitignore
```

**❌ DON'T:**
```go
// JANGAN hardcode encryption key
encryptionKey := []byte("mysecretkey12345")  // BAHAYA!

// JANGAN commit .env ke git
git add .env  // JANGAN!

// JANGAN share key via Slack/email
// Gunakan secret management (HashiCorp Vault, AWS Secrets Manager)
```

## 🚦 Rate Limiting & Anti-Abuse

### API Rate Limiting

**✅ DO:**
```go
// Per-IP rate limiting (public endpoints)
const rateLimitPublic = 100  // req/min

// Per-tenant rate limiting (authenticated)
const rateLimitTenant = 1000  // req/min

// Redis-based sliding window
func checkRateLimit(key string, limit int) bool {
    count, _ := redisClient.Incr(ctx, key).Result()
    if count == 1 {
        redisClient.Expire(ctx, key, time.Minute)
    }
    return count <= int64(limit)
}
```

**WhatsApp Rate Limiting (Anti-Ban):**
```go
// Token bucket: 5 msg/min per tenant
type TenantRateLimiter struct {
    buckets map[string]*tokenBucket
    rate    int  // 5 msg/min
}

// WAJIB untuk whatsmeow (unofficial WA client)
if !rateLimiter.Allow(tenantID) {
    return http.StatusTooManyRequests
}
```

**❌ DON'T:**
```go
// JANGAN skip rate limiting
// JANGAN rate limit secara global (harus per-tenant/per-IP)
// JANGAN blast WhatsApp >100 msg/hari via whatsmeow
```

## 🔑 Secrets Management

### Environment Variables

**✅ DO:**
```bash
# .env (local development)
JWT_SECRET=$(openssl rand -base64 32)
DATABASE_URL=postgres://user:$(openssl rand -base64 16)@localhost:5433/db
ENCRYPTION_KEY=$(openssl rand -base64 32)

# Production: Gunakan secret manager
# - AWS Secrets Manager
# - HashiCorp Vault
# - Google Secret Manager
```

**❌ DON'T:**
```bash
# JANGAN commit secrets
git add .env  # BAHAYA!

# JANGAN hardcode di code
apiKey := "xnd_production_abc123"  // JANGAN!

# JANGAN log secrets
log.Printf("API Key: %s", apiKey)  // BAHAYA!

# JANGAN share via chat
"Here's the prod DB password: xxx"  // JANGAN!
```

### Masking Sensitive Data

**✅ DO:**
```go
// Mask di logs
func maskSensitive(field, value string) string {
    sensitive := []string{"password", "api_key", "token", "secret"}
    for _, s := range sensitive {
        if field == s {
            return "***"
        }
    }
    return value
}

// Mask credit card
func maskCreditCard(cc string) string {
    if len(cc) < 4 {
        return "****"
    }
    return "****" + cc[len(cc)-4:]
}
```

**❌ DON'T:**
```go
// JANGAN log plaintext password
slog.Info("User login", "username", username, "password", password)

// JANGAN return secrets di error message
return fmt.Errorf("DB connection failed: %s", databaseURL)
```

## 🌐 Network Security

### HTTPS/TLS

**✅ DO:**
```nginx
# Force HTTPS
server {
    listen 80;
    server_name app.wch-platform.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    
    ssl_certificate /etc/letsencrypt/live/app.wch-platform.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/app.wch-platform.com/privkey.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
}
```

**❌ DON'T:**
```nginx
# JANGAN allow HTTP di production
listen 80;  # Tanpa redirect ke HTTPS

# JANGAN gunakan TLS 1.0/1.1 (deprecated)
ssl_protocols TLSv1 TLSv1.1;
```

### CORS Configuration

**✅ DO:**
```go
// Whitelist allowed origins
allowedOrigins := []string{
    "https://app.wch-platform.com",
    "https://campaign.wch-platform.com",
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if contains(allowedOrigins, origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        next.ServeHTTP(w, r)
    })
}
```

**❌ DON'T:**
```go
// JANGAN allow wildcard origin di production
w.Header().Set("Access-Control-Allow-Origin", "*")

// JANGAN trust semua origins
origin := r.Header.Get("Origin")
w.Header().Set("Access-Control-Allow-Origin", origin)  // Unsafe!
```

### Webhook Signature Verification

**✅ DO:**
```go
// Verify Xendit webhook signature
func verifyWebhookSignature(r *http.Request, tenantID string) bool {
    receivedToken := r.Header.Get("X-Callback-Token")
    
    // Per-tenant token (priority)
    tenantToken := getTenantWebhookToken(tenantID)
    if tenantToken != "" {
        return subtle.ConstantTimeCompare(
            []byte(receivedToken),
            []byte(tenantToken),
        ) == 1
    }
    
    // Fallback global token
    return subtle.ConstantTimeCompare(
        []byte(receivedToken),
        []byte(globalWebhookToken),
    ) == 1
}
```

**❌ DON'T:**
```go
// JANGAN skip signature verification
// Process webhook tanpa verify token — BAHAYA!

// JANGAN gunakan == untuk compare secrets (timing attack)
if receivedToken == expectedToken {  // Vulnerable!
    // Use crypto/subtle.ConstantTimeCompare instead
}
```

## 🗄️ Database Security

### Connection Security

**✅ DO:**
```go
// Use SSL/TLS untuk DB connection
connStr := "postgres://user:pass@host:5432/db?sslmode=require"

// Connection pooling dengan limits
config, _ := pgxpool.ParseConfig(connStr)
config.MaxConns = 25
config.MinConns = 5
pool, _ := pgxpool.NewWithConfig(ctx, config)
```

**❌ DON'T:**
```go
// JANGAN disable SSL di production
connStr := "postgres://user:pass@host:5432/db?sslmode=disable"

// JANGAN unlimited connections
config.MaxConns = 10000  // Resource exhaustion!
```

### Principle of Least Privilege

**✅ DO:**
```sql
-- Create role dengan permission minimal
CREATE ROLE wch_app_user WITH LOGIN PASSWORD 'strong_password';

-- Grant hanya permission yang dibutuhkan
GRANT SELECT, INSERT, UPDATE ON tenants TO wch_app_user;
GRANT SELECT, INSERT, UPDATE ON users TO wch_app_user;

-- JANGAN grant DELETE ke app user (hanya admin)
-- JANGAN grant TRUNCATE, DROP
```

**❌ DON'T:**
```sql
-- JANGAN grant SUPERUSER ke app
CREATE ROLE wch_app_user WITH SUPERUSER;

-- JANGAN grant ALL PRIVILEGES
GRANT ALL PRIVILEGES ON DATABASE wch_platform TO wch_app_user;
```

## 📝 Secure Coding Checklist

### Before Commit

- [ ] Tidak ada hardcoded secrets (API keys, passwords, tokens)
- [ ] Tidak ada `.env` file di git
- [ ] Semua input user divalidasi (regex, length, type)
- [ ] Parameterized queries untuk semua DB operations
- [ ] Sensitive data di-encrypt (AES-256-GCM)
- [ ] Password di-hash dengan bcrypt cost=12
- [ ] JWT include tenant_id dan expiry
- [ ] Rate limiting aktif untuk API endpoints
- [ ] Webhook signature verified
- [ ] Error messages tidak expose internals

### Before Deployment

- [ ] `.env` production memiliki strong secrets (32+ chars)
- [ ] HTTPS/TLS enabled dan certificate valid
- [ ] Firewall configured (UFW/iptables)
- [ ] Database user tidak SUPERUSER
- [ ] Redis password set
- [ ] Backup automated (daily cron)
- [ ] Monitoring & alerting aktif
- [ ] Security headers set (X-Content-Type-Options, X-Frame-Options)
- [ ] CORS whitelist configured (no wildcard)
- [ ] Rate limiting tested (simulate attack)

## 🚨 Incident Response

### Data Breach Response Plan

**If credentials compromised:**
```bash
# 1. Revoke compromised secrets IMMEDIATELY
# 2. Rotate all secrets
export JWT_SECRET=$(openssl rand -base64 32)
export ENCRYPTION_KEY=$(openssl rand -base64 32)

# 3. Force logout all users
redis-cli -p 6381 FLUSHDB  # Clear session cache

# 4. Notify affected users
# 5. Review logs untuk unauthorized access
grep "401\|403\|500" logs/*.log | less
```

**If database breach:**
```bash
# 1. Isolate database (block external connections)
sudo ufw deny 5433/tcp

# 2. Audit logs
docker compose exec postgres psql -U wch_admin -d wch_platform \
  -c "SELECT * FROM pg_stat_activity;"

# 3. Change DB password
ALTER USER wch_admin WITH PASSWORD 'new_strong_password';

# 4. Review access logs untuk suspicious queries
```

## 📚 Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [PostgreSQL Security](https://www.postgresql.org/docs/current/security.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

## 🎓 Training

**Internal Security Training:**
- Monthly security review meeting
- Quarterly penetration testing
- Annual security audit

**External Resources:**
- OWASP Security Training
- Cloud Security Certification (AWS/GCP)
- Go Security Workshop

---

**Last Updated:** 2026-07-31  
**Next Review:** 2026-10-31  
**Owner:** Security Team

**Questions?** Contact: security@wch-platform.com
