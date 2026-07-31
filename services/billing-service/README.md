# Billing Service

**Port:** 8003  
**Database:** PostgreSQL (subscriptions, invoices, vouchers, topup)  
**Payment Gateway:** Xendit

## Deskripsi

Service billing multi-tenant untuk WCH Platform. Menangani subscription management, payment processing, voucher redemption, wallet topup, dan affiliate commission.

## Fitur Utama

- 💳 **Xendit Integration** — Per-tenant API keys & merchant accounts
- 📅 **Subscription Management** — Activation, renewal, upgrade/downgrade
- 🎫 **Voucher System** — Generate, redeem, validate vouchers
- 💰 **Wallet Topup** — E-wallet balance management
- 📊 **Usage Tracking** — AI usage, transaction quotas per plan
- 🤝 **Affiliate Program** — Commission tracking & payout
- 🔄 **Webhook Handler** — Xendit payment callbacks
- 🔐 **Per-Tenant Xendit** — Isolated payment accounts

## Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5433/wch_platform

# Redis (caching Xendit clients)
REDIS_ADDR=localhost:6381
REDIS_PASSWORD=
REDIS_DB=0

# Xendit (fallback global)
XENDIT_API_KEY=xnd_development_...
XENDIT_WEBHOOK_TOKEN=verify_token_...

# Encryption (untuk encrypt Xendit API keys)
ENCRYPTION_KEY=32-byte-key-here  # WAJIB 32 bytes

# Server
PORT=8003
ENV=development
```

## Per-Tenant Xendit Architecture

Setiap tenant memiliki kredensial Xendit sendiri di tabel `tenants`:

```sql
CREATE TABLE tenants (
    ...
    xendit_api_key VARCHAR(255),         -- Encrypted
    xendit_merchant_id VARCHAR(255),
    xendit_webhook_token VARCHAR(255),   -- Encrypted
    ...
);
```

**Flow Payment:**
```
Tenant request topup/subscription
    ↓
getTenantXenditClient(tenantID)  ← Cache 5-min TTL
    ↓
Baca xendit_api_key dari DB
    ↓
CreateInvoice di merchantID tenant
    ↓
Dana masuk ke bank account tenant ✅
```

**Webhook Verification:**
```
Xendit callback → /webhooks/xendit/invoice.paid
    ↓
Extract tenantID dari external_id
    ↓
Verify webhook token:
      Priority 1: tenant.xendit_webhook_token (DB)
      Fallback: env XENDIT_WEBHOOK_TOKEN (global)
    ↓
Process payment
```

## API Endpoints

### Subscription Management

#### GET `/subscriptions`
Ambil subscription aktif tenant.

**Headers:**
```
Authorization: Bearer <jwt-token>
X-Tenant-ID: <tenant-uuid>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "planId": "lite",
    "status": "active",
    "startDate": "2026-01-01T00:00:00Z",
    "endDate": "2026-02-01T00:00:00Z",
    "autoRenew": true
  }
}
```

#### POST `/subscriptions/activate`
Aktivasi subscription setelah payment.

**Headers:**
```
Authorization: Bearer <jwt-token>
X-Tenant-ID: <tenant-uuid>
```

**Request:**
```json
{
  "planId": "lite",
  "durationMonths": 1,
  "voucherCode": "WCH-LITE-ABC123"  // optional
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "invoiceUrl": "https://checkout.xendit.co/...",
    "externalId": "INV-{uuid}|{tenantId}",
    "amount": 50000
  }
}
```

#### POST `/subscriptions/upgrade`
Upgrade plan (dengan proration).

**Request:**
```json
{
  "newPlanId": "pro"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "proratedDays": 15,  // Sisa hari dari plan lama
    "newEndDate": "2026-02-15T00:00:00Z",
    "message": "Plan upgraded to Pro. 15 days from Lite preserved."
  }
}
```

#### POST `/subscriptions/cancel`
Cancel auto-renewal (subscription tetap aktif sampai end date).

**Response:**
```json
{
  "success": true,
  "message": "Auto-renewal cancelled. Subscription active until 2026-02-01."
}
```

### Wallet & Topup

#### GET `/wallet`
Cek saldo wallet tenant.

**Headers:**
```
Authorization: Bearer <jwt-token>
X-Tenant-ID: <tenant-uuid>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "balance": 500000,  // dalam sen (5000 rupiah)
    "currency": "IDR"
  }
}
```

#### POST `/topup`
Topup wallet via Xendit invoice.

**Request:**
```json
{
  "amount": 100000  // dalam sen (1000 rupiah)
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "invoiceUrl": "https://checkout.xendit.co/...",
    "externalId": "{uuid}-wallet-topup-{tenantId}",
    "amount": 100000
  }
}
```

### Voucher Management

#### POST `/vouchers/redeem`
Redeem voucher code.

**Headers:**
```
Authorization: Bearer <jwt-token>
X-Tenant-ID: <tenant-uuid>
```

**Request:**
```json
{
  "code": "WCH-LITE-ABC123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "planId": "lite",
    "validityDays": 30,
    "activatedUntil": "2026-08-31T00:00:00Z"
  }
}
```

**Error Responses:**
```json
// 404 - Voucher tidak ditemukan
{
  "success": false,
  "message": "Voucher code not found"
}

// 400 - Voucher expired
{
  "success": false,
  "message": "Voucher expired at 2026-06-30"
}

// 409 - Voucher sudah digunakan
{
  "success": false,
  "message": "Voucher already redeemed by tenant X"
}
```

### Superadmin Voucher Management

#### POST `/admin/vouchers/generate`
Generate voucher codes (batch).

**Headers:**
```
Authorization: Bearer <superadmin-jwt>
```

**Request:**
```json
{
  "planId": "lite",
  "validityDays": 30,
  "quantity": 100,
  "programName": "Promo Ramadan 2026",  // optional
  "maxUses": 1  // optional, default 1
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "planId": "lite",
    "validityDays": 30,
    "count": 100,
    "codes": [
      {"code": "WCH-LITE-ABC123", "days": 30},
      {"code": "WCH-LITE-DEF456", "days": 30},
      ...
    ]
  }
}
```

#### GET `/admin/vouchers`
List semua vouchers dengan filter.

**Query Params:**
```
?planId=lite&used=false&limit=50
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total": 100,
    "used": 25,
    "unused": 75,
    "codes": [
      {
        "id": "uuid",
        "code": "WCH-LITE-ABC123",
        "planId": "lite",
        "programName": "Promo Ramadan",
        "isRedeemed": false,
        "redeemedBy": null,
        "redeemedAt": null,
        "createdAt": "2026-07-01T00:00:00Z"
      },
      ...
    ]
  }
}
```

#### DELETE `/admin/vouchers?id=<uuid>`
Hapus voucher yang belum digunakan.

**Response:**
```json
{
  "success": true,
  "message": "Voucher deleted"
}
```

**Error:**
```json
// 400 - Voucher sudah redeemed
{
  "success": false,
  "message": "Cannot delete redeemed voucher"
}
```

### Webhook (Xendit Callbacks)

#### POST `/webhook`
Xendit payment notification.

**Headers:**
```
X-Callback-Token: <xendit-webhook-token>
```

**Request (Invoice Paid):**
```json
{
  "id": "invoice-id",
  "external_id": "INV-{uuid}|{tenantId}",
  "status": "PAID",
  "amount": 50000,
  "paid_at": "2026-07-31T10:00:00Z",
  "payment_method": "BANK_TRANSFER"
}
```

**Processing Flow:**
```
Verify signature (X-Callback-Token)
    ↓
Extract tenantID dari external_id
    ↓
Check duplicate processing (idempotency)
    ↓
Validate amount matches expected
    ↓
    ├─ Subscription → Activate subscription
    └─ Topup → Credit wallet
    ↓
Calculate affiliate commission (if applicable)
    ↓
Send notification (Telegram/Email)
    ↓
Return 200 OK
```

## Database Schema

### Table: `subscriptions`
```sql
- id UUID PRIMARY KEY
- tenant_id UUID REFERENCES tenants(id)
- plan_id VARCHAR(50)  -- free, lite, pro, ultimate
- status VARCHAR(20)   -- active, expired, cancelled
- start_date TIMESTAMPTZ
- end_date TIMESTAMPTZ
- auto_renew BOOLEAN DEFAULT true
- created_at TIMESTAMPTZ
```

### Table: `invoices`
```sql
- id UUID PRIMARY KEY
- tenant_id UUID REFERENCES tenants(id)
- external_id VARCHAR(255) UNIQUE  -- Xendit invoice ID
- amount BIGINT  -- dalam sen
- status VARCHAR(20)  -- pending, paid, expired
- invoice_url TEXT
- paid_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
```

### Table: `voucher_codes`
```sql
- id UUID PRIMARY KEY
- code VARCHAR(50) UNIQUE
- plan_id VARCHAR(50)
- program_name VARCHAR(100)
- validity_days INTEGER
- is_redeemed BOOLEAN DEFAULT false
- redeemed_by UUID REFERENCES tenants(id)
- redeemed_at TIMESTAMPTZ
- expires_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
```

### Table: `wallet_transactions`
```sql
- id UUID PRIMARY KEY
- tenant_id UUID REFERENCES tenants(id)
- type VARCHAR(20)  -- topup, deduction, refund
- amount BIGINT  -- dalam sen
- balance_after BIGINT
- description TEXT
- created_at TIMESTAMPTZ
```

### Table: `affiliate_commissions`
```sql
- id UUID PRIMARY KEY
- referrer_tenant_id UUID REFERENCES tenants(id)
- referee_tenant_id UUID REFERENCES tenants(id)
- commission_amount BIGINT
- status VARCHAR(20)  -- pending, paid
- paid_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
```

## Security Best Practices

### Xendit API Key Encryption
```go
// Encrypt sebelum simpan ke DB
encrypted, err := encrypt(apiKey, encryptionKey)
if err != nil {
    return err
}

// Simpan encrypted value
_, err = db.Exec("UPDATE tenants SET xendit_api_key = $1 WHERE id = $2", encrypted, tenantID)
```

### Webhook Signature Validation
```go
func verifyWebhookSignature(r *http.Request, tenantID string) bool {
    token := r.Header.Get("X-Callback-Token")
    
    // Priority 1: Per-tenant token
    tenantToken := getTenantWebhookToken(tenantID)
    if tenantToken != "" {
        return token == tenantToken
    }
    
    // Fallback: Global token
    return token == cfg.XenditWebhookToken
}
```

### Idempotency Check
```go
// Cek apakah invoice sudah diproses
var exists bool
err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM invoices WHERE external_id = $1 AND status = 'paid')", externalID).Scan(&exists)

if exists {
    // Sudah diproses, return 200 OK tanpa re-process
    return writeJSON(w, 200, Response{Success: true, Message: "Already processed"})
}
```

## Proration Logic

Saat upgrade plan, sisa hari dari plan lama di-preserve:

```go
func calculateProration(currentEndDate time.Time, newPlanID string) (int, time.Time) {
    now := time.Now()
    remainingDays := int(currentEndDate.Sub(now).Hours() / 24)
    
    // New plan duration (contoh: 30 hari)
    newDuration := 30
    
    // New end date = now + newDuration + remainingDays
    newEndDate := now.AddDate(0, 0, newDuration + remainingDays)
    
    return remainingDays, newEndDate
}
```

**Example:**
- Current plan: Lite (end: 2026-08-15) — sisa 15 hari
- Upgrade to: Pro (30 hari)
- New end date: 2026-09-15 (30 + 15 = 45 hari total)

## Affiliate Commission

**Rules:**
- **5% commission** dari pembayaran referee
- **Max 50,000 per transaksi**
- Commission credited ke referrer wallet
- Tracked di tabel `affiliate_commissions`

**Calculation:**
```go
func calculateCommission(amount int64) int64 {
    commission := amount * 5 / 100
    maxCommission := int64(5000000) // 50,000 rupiah = 5,000,000 sen
    
    if commission > maxCommission {
        return maxCommission
    }
    return commission
}
```

## Testing

```bash
# Run all tests
go test ./services/billing-service/... -v

# Specific test suites
go test -run TestWebhook -v
go test -run TestVoucher -v
go test -run TestSubscription -v
go test -run TestProration -v
```

## Monitoring

### Metrics (Prometheus)
```
billing_payments_total{status, plan}
billing_subscriptions_active{plan}
billing_vouchers_redeemed_total{plan}
billing_commissions_total{status}
billing_xendit_errors_total{type}
```

### Logs (slog JSON)
- Payment received (tenant, amount, plan)
- Subscription activated/expired
- Voucher redeemed
- Commission credited
- Webhook signature mismatch

## Troubleshooting

### Webhook 401 Unauthorized
**Penyebab:** `X-Callback-Token` tidak match.  
**Solusi:**
```bash
# Cek token di DB
psql -h localhost -p 5433 -U wch_admin -d wch_platform
SELECT id, xendit_webhook_token FROM tenants WHERE id = '<tenantId>';

# Update token jika salah
UPDATE tenants SET xendit_webhook_token = 'correct-token' WHERE id = '<tenantId>';
```

### Payment success tapi subscription tidak aktif
**Penyebab:** Webhook gagal process atau `external_id` tidak valid.  
**Solusi:**
```bash
# Cek invoices table
SELECT * FROM invoices WHERE external_id = 'INV-...';

# Manual activate (development only)
UPDATE subscriptions SET status = 'active', end_date = NOW() + INTERVAL '30 days' WHERE tenant_id = '<uuid>';
```

### Voucher redemption gagal
**Penyebab:** Voucher expired atau sudah redeemed.  
**Solusi:**
```bash
# Cek voucher
SELECT * FROM voucher_codes WHERE code = 'WCH-LITE-ABC123';

# Reset voucher (development only)
UPDATE voucher_codes SET is_redeemed = false, redeemed_by = NULL, redeemed_at = NULL WHERE code = 'WCH-LITE-ABC123';
```

### Xendit client cache issue
**Penyebab:** Tenant update API key tapi cache belum refresh.  
**Solusi:**
```bash
# Restart billing service untuk clear cache
docker compose restart billing-service
# atau make dev-billing (native)
```

## Production Checklist

- [ ] Set `ENCRYPTION_KEY` (32 bytes) untuk encrypt Xendit keys
- [ ] Verify Xendit webhook URL registered di dashboard
- [ ] Test webhook dengan Xendit simulator
- [ ] Set up monitoring untuk failed payments
- [ ] Configure alerts untuk subscription expiry (7 days before)
- [ ] Test proration logic dengan real transactions
- [ ] Verify commission calculation
- [ ] Backup database harian (subscriptions & invoices critical)
- [ ] Test idempotency (kirim webhook duplicate)
- [ ] Rate limit webhook endpoint (500 req/min via API Gateway)

## Related Services

- **Auth Service** (8001) — Tenant & user data
- **API Gateway** (8000) — Webhook routing & rate limiting
- **Notification Service** (8005) — Payment notifications
- **Subscription Worker** (8006) — Auto-freeze expired tenants
