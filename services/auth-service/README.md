# Auth Service

**Port:** 8001  
**Database:** PostgreSQL (via `shared/sdk/db`)  
**Cache:** Redis (OTP & refresh tokens)

## Deskripsi

Service autentikasi multi-tenant untuk WCH Platform. Menangani registrasi, login, JWT generation, OTP via WhatsApp/Telegram, dan RBAC (Role-Based Access Control).

## Fitur Utama

- 🔐 **Multi-tenant Authentication** — JWT dengan `tenant_id` dan `role`
- 📱 **OTP Login** — Via WhatsApp (whatsmeow/Cloud API) atau Telegram
- 🔑 **Password Hashing** — bcrypt cost=12
- 🎫 **Refresh Token** — SHA-256 hash di Redis + PostgreSQL
- 👥 **Role Management** — Owner, Admin, Staff, Kasir, Superadmin
- 📞 **Phone Login** — Alternatif username/password
- 🔄 **Telegram Bot** — Registrasi & login via Telegram
- 🎭 **Impersonation** — Superadmin dapat login sebagai tenant untuk troubleshooting

## Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5433/wch_platform

# Redis
REDIS_ADDR=localhost:6381
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-secret-key-min-32-chars

# WhatsApp Gateway
WA_GATEWAY_URL=http://localhost:8202

# Telegram Bot
TELEGRAM_BOT_TOKEN=your-bot-token

# Server
PORT=8001
ENV=development  # or production
```

## API Endpoints

### Registrasi & Login

#### POST `/register`
Registrasi user baru (Owner tenant baru).

**Request:**
```json
{
  "username": "user123",
  "password": "SecurePass123",
  "email": "user@example.com",
  "phoneNumber": "6281234567890",
  "businessName": "Toko Berkah",
  "businessType": "warung",
  "referralCode": "PROMO2024" // optional
}
```

**Response:**
```json
{
  "success": true,
  "message": "Registration successful. OTP sent to WhatsApp.",
  "data": {
    "userId": "uuid",
    "tenantId": "uuid",
    "phoneNumber": "6281234567890"
  }
}
```

#### POST `/verify-otp`
Verifikasi OTP setelah registrasi.

**Request:**
```json
{
  "phoneNumber": "6281234567890",
  "otp": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "message": "OTP verified",
  "data": {
    "token": "jwt-token",
    "refreshToken": "refresh-token",
    "userId": "uuid",
    "tenantId": "uuid",
    "role": "owner"
  }
}
```

#### POST `/login`
Login dengan username/email + password.

**Request:**
```json
{
  "username": "user123",  // or email
  "password": "SecurePass123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "jwt-token",
    "refreshToken": "refresh-token",
    "userId": "uuid",
    "tenantId": "uuid",
    "role": "owner"
  }
}
```

#### POST `/phone-login`
Request OTP untuk login via phone number.

**Request:**
```json
{
  "phoneNumber": "6281234567890"
}
```

**Response:**
```json
{
  "success": true,
  "message": "OTP sent to WhatsApp"
}
```

#### POST `/verify-phone-login`
Verifikasi OTP phone login.

**Request:**
```json
{
  "phoneNumber": "6281234567890",
  "otp": "123456"
}
```

**Response:** (sama dengan `/verify-otp`)

### Telegram Auth

#### POST `/telegram/register`
Registrasi via Telegram Bot.

**Request:**
```json
{
  "telegramChatId": "123456789",
  "username": "user123",
  "password": "SecurePass123",
  "email": "user@example.com",
  "phoneNumber": "6281234567890",
  "businessName": "Toko Berkah",
  "businessType": "warung"
}
```

#### POST `/telegram/login`
Login via Telegram Bot.

**Request:**
```json
{
  "telegramChatId": "123456789",
  "phoneNumber": "6281234567890"
}
```

#### POST `/telegram/webhook`
Webhook endpoint untuk Telegram Bot (internal).

### Token Management

#### POST `/refresh`
Refresh JWT token.

**Request:**
```json
{
  "refreshToken": "your-refresh-token"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "new-jwt-token",
    "refreshToken": "new-refresh-token"
  }
}
```

#### POST `/logout`
Logout dan hapus refresh token.

**Headers:**
```
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

### Profile & Password

#### GET `/profile`
Ambil profil user.

**Headers:**
```
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "userId": "uuid",
    "username": "user123",
    "email": "user@example.com",
    "phoneNumber": "6281234567890",
    "role": "owner",
    "tenantId": "uuid"
  }
}
```

#### PUT `/profile`
Update profil user.

**Headers:**
```
Authorization: Bearer <jwt-token>
```

**Request:**
```json
{
  "email": "newemail@example.com",
  "fullName": "John Doe"
}
```

#### POST `/change-password`
Ganti password.

**Headers:**
```
Authorization: Bearer <jwt-token>
```

**Request:**
```json
{
  "oldPassword": "OldPass123",
  "newPassword": "NewPass456"
}
```

#### POST `/reset-password`
Reset password ke default (phone number).

**Request:**
```json
{
  "username": "user123",
  "phoneNumber": "6281234567890"
}
```

### Superadmin

#### POST `/superadmin/login`
Login khusus superadmin.

**Request:**
```json
{
  "username": "superadmin",
  "password": "admin-password"
}
```

#### POST `/superadmin/tenants/{tenantId}/impersonate`
Generate JWT token sebagai owner tenant (untuk troubleshooting).

**Headers:**
```
Authorization: Bearer <superadmin-jwt>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "impersonated-jwt-token",
    "tenantId": "uuid",
    "expiresIn": "12h"
  }
}
```

## Testing

```bash
# Run all tests
go test ./services/auth-service/... -v

# Run specific test
go test -run TestPassword_BcryptHashing -v

# Run security tests
go test -run TestInputValidation -v
go test -run TestJWT -v
go test -run TestEncryption -v
```

## Database Schema

### Table: `users`
```sql
- id UUID PRIMARY KEY
- tenant_id UUID REFERENCES tenants(id)
- username VARCHAR(50) UNIQUE
- email VARCHAR(255)
- password_hash TEXT
- phone_number VARCHAR(20)
- role VARCHAR(20)  -- owner, admin, staff, kasir, superadmin
- telegram_chat_id VARCHAR(50)
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ
```

### Table: `refresh_tokens`
```sql
- id UUID PRIMARY KEY
- user_id UUID REFERENCES users(id)
- token_hash TEXT  -- SHA-256 hash
- expires_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
```

## Security Best Practices

### Password
- ✅ **Bcrypt cost=12** (WAJIB sesuai CLAUDE.md)
- ✅ Minimum 6 karakter
- ✅ Unique salt per hash
- ❌ JANGAN simpan plaintext password

### JWT
- ✅ HS256 signing
- ✅ Include `tenant_id`, `user_id`, `role`
- ✅ Expiry 24 jam
- ✅ Refresh token via dedicated endpoint
- ❌ JANGAN expose JWT secret

### OTP
- ✅ 6 digit random
- ✅ Expire 1 jam
- ✅ Stored di Redis dengan TTL
- ✅ Anti-spam: max 1 OTP per phone dalam 1 jam
- ❌ JANGAN kirim OTP via email (gunakan WA/Telegram)

### Input Validation
- ✅ Username: `^[a-zA-Z0-9_]+$`
- ✅ Email: `^[^\s@]+@[^\s@]+\.[^\s@]+$`
- ✅ Phone: `^62[0-9]{6,15}$` (Indonesian format)
- ✅ Role: whitelist `[owner, admin, staff, kasir, superadmin]`

### Anti SQL Injection
```go
// ✅ BENAR — Parameterized query
rows, err := DB.Query(ctx, "SELECT * FROM users WHERE username = $1", username)

// ❌ SALAH — String concatenation
rows, err := DB.Query(ctx, "SELECT * FROM users WHERE username = '" + username + "'")
```

## Flow Diagram

### Registrasi Flow
```
User submit form
    ↓
Validate input (username, email, phone, password)
    ↓
Check duplicate username/email
    ↓
Create tenant (new business)
    ↓
Hash password (bcrypt cost=12)
    ↓
Insert user ke DB (role: owner)
    ↓
Generate OTP (6 digit)
    ↓
Store OTP di Redis (TTL 1 jam)
    ↓
Send OTP via WA Gateway
    ↓
Return userId, tenantId ke frontend
```

### OTP Verification Flow
```
User input OTP
    ↓
Get OTP dari Redis (key: otp:{phone})
    ↓
Compare OTP
    ↓
Delete OTP dari Redis
    ↓
Generate JWT (HS256, exp: 24h)
    ↓
Generate refresh token (SHA-256 hash)
    ↓
Store refresh token hash di DB & Redis
    ↓
Return token + refreshToken
```

### Login Flow (Username/Password)
```
User input username + password
    ↓
Query user dari DB (WHERE username = $1 OR email = $1)
    ↓
bcrypt.CompareHashAndPassword(hash, password)
    ↓
Generate JWT + refresh token
    ↓
Return tokens
```

## Rate Limiting

Auth endpoints di-rate limit via API Gateway:
- **Public endpoints** (`/auth/*`): 100 req/min per IP
- **Login/Register**: Extra protection anti brute-force

## Troubleshooting

### Error: "OTP sudah dikirim"
**Penyebab:** Redis masih menyimpan OTP aktif (belum expired).  
**Solusi:** Tunggu 1 jam atau hapus manual via Redis CLI:
```bash
redis-cli -p 6381
DEL otp:6281234567890
```

### Error: "Missing Authorization header"
**Penyebab:** Frontend tidak mengirim token di header.  
**Solusi:** Pastikan request header:
```javascript
headers: {
  'Authorization': `Bearer ${token}`
}
```

### Error: "Token does not contain tenant context"
**Penyebab:** JWT tidak memiliki `tenant_id` claim.  
**Solusi:** Re-generate token dengan claim lengkap atau cek JWT secret.

### Error: "Invalid or expired token"
**Penyebab:** JWT expired (>24 jam) atau signature salah.  
**Solusi:** Gunakan refresh token untuk generate JWT baru.

### Database Connection Failed
**Solusi:**
```bash
# Cek PostgreSQL running
docker ps | grep postgres

# Test connection
psql -h localhost -p 5433 -U wch_admin -d wch_platform
```

## Development

```bash
# Install dependencies
go mod download

# Run migrations (auto-run saat startup)
# Manual: psql -f shared/migrations/000001_initial.up.sql

# Run service
go run services/auth-service/*.go

# Hot reload (air)
cd services/auth-service
air
```

## Production Checklist

- [ ] Set `JWT_SECRET` min 32 karakter
- [ ] Set `ENV=production`
- [ ] Enable HTTPS
- [ ] Set rate limiting di API Gateway
- [ ] Backup PostgreSQL harian
- [ ] Monitor failed login attempts
- [ ] Set Redis password
- [ ] Review logs untuk suspicious activity
- [ ] Test OTP delivery (WA & Telegram)
- [ ] Verify bcrypt cost=12 aktif

## Monitoring

Metrics yang diexpose (Prometheus):
- `auth_logins_total{method, success}` — Total login attempts
- `auth_active_sessions` — Active user sessions

Log yang di-track (slog JSON):
- Login success/failed
- OTP sent/verified
- Token refresh
- Password changes
- Impersonation events (superadmin)

## Related Services

- **API Gateway** (8000) — Routing & rate limiting
- **WA Gateway** (8202) — OTP delivery via WhatsApp
- **Notification Service** (8005) — OTP delivery via Telegram
- **Billing Service** (8003) — Subscription status check
