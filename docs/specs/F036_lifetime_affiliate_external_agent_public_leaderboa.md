# F036: Lifetime Affiliate, External Agent & Public Leaderboard

**Date:** 2026-06-14  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F054](F054_referral_system_discount_downline_commission_uplin.md) (Referral System Discount + Commission)

---

## 🎯 Objectives

Sistem komisi lifetime recurring untuk agen/affiliator eksternal dengan leaderboard publik untuk memicu kompetisi dan portal withdrawal komisi.

**Tujuan eksplisit:**
1. Agen eksternal (tidak harus subscriber) dapat refer tenant baru dan mendapat komisi lifetime recurring setiap renewal
2. Public leaderboard untuk ranking agen berdasarkan closing count dan revenue generated — memicu kompetisi dan social proof
3. Portal withdrawal untuk agen cairkan komisi cash dengan minimal Rp 100.000, diproses manual oleh superadmin

**Problem yang diselesaikan:**
- Platform tidak punya insentif untuk agen eksternal promote WCH — hanya rely on direct marketing
- Komisi one-time tidak sustainable untuk agen — butuh recurring income model
- Tidak ada transparency ranking agen → no social proof untuk recruit agen baru

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Referral Code Locking**
  - *Verification:* `tenants.referred_by_affiliate_id` di-set saat first subscribe, lifetime lock (tidak bisa diubah)
  - *Example:* Tenant register dengan `?ref=AGEN-BUDI` → `referred_by_affiliate_id` = affiliate UUID AGEN-BUDI forever

- [x] **AC-2: Lifetime Commission on Renewal**
  - *Verification:* Payment webhook cek `referred_by_affiliate_id` → credit commission ke `affiliates.cash_balance_cents` + INSERT `affiliate_earnings`
  - *Example:* Tenant bayar renewal Rp 100.000 → affiliate dapat komisi Rp 20.000 (20%)

- [x] **AC-3: Public Leaderboard API**
  - *Verification:* `GET /api/public/affiliate-leaderboard` return TOP 10 agen (monthly + all-time) tanpa auth, data masked
  - *Example:* Response: `{ monthly: [{name: "Budi S.", closings: 150, revenue: 15000000}] }`

- [x] **AC-4: Withdrawal Request**
  - *Verification:* `POST /api/affiliate/withdraw` create `affiliate_withdrawals` row dengan status `pending`, minimal Rp 100.000
  - *Example:* Agen request withdraw Rp 500.000 → superadmin approve → manual transfer → status `completed`

- [x] **AC-5: Affiliate Dashboard (Frontend)**
  - *Verification:* `/affiliate` page tampilkan referral link, saldo tersedia, riwayat komisi, tombol "Tarik Dana"
  - *Example:* Dashboard show: "Saldo: Rp 500.000 | Komisi bulan ini: Rp 150.000 | Referral link: wch.id/ref/AGEN-BUDI"

- [x] **AC-6: Superadmin Withdrawal Management**
  - *Verification:* Superadmin dashboard list `pending` withdrawals → approve/reject → update status + admin_note
  - *Example:* Superadmin approve withdraw → transfer via mBanking → set status `completed` + note "Transfer BCA 12345"

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│       Tenant Registration with ?ref=CODE            │
│  1. Register → input referral code                  │
│  2. First subscribe → lock referred_by_affiliate_id │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Payment Webhook (Subscription/Renewal)         │
│  1. Invoice paid → query referred_by_affiliate_id   │
│  2. If exists:                                      │
│     - Calculate commission (20% of invoice)         │
│     - INSERT affiliate_earnings                     │
│     - UPDATE affiliates.cash_balance_cents          │
│     - UPDATE affiliates.total_earnings_cents        │
└─────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────┐
│         Affiliate Dashboard (Frontend)              │
│  - Referral link generator                          │
│  - Balance display + earnings history               │
│  - Withdraw button (min Rp 100k)                    │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Superadmin Withdrawal Management               │
│  1. List pending withdrawals                        │
│  2. Manual transfer via mBanking                    │
│  3. Approve → status=completed + admin_note         │
└─────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────┐
│        Public Leaderboard (Landing Page)            │
│  GET /api/public/affiliate-leaderboard              │
│  - TOP 10 monthly + all-time                        │
│  - Data masked (name initial, closings, revenue)    │
└─────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: 000040_affiliate_system.up.sql

CREATE TABLE affiliates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(50) UNIQUE NOT NULL,
    bank_name VARCHAR(100),
    bank_account_number VARCHAR(50),
    bank_account_name VARCHAR(200),
    cash_balance_cents BIGINT NOT NULL DEFAULT 0,
    total_earnings_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_affiliates_referral_code ON affiliates(referral_code);
CREATE INDEX idx_affiliates_user_id ON affiliates(user_id);

CREATE TABLE affiliate_earnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id VARCHAR(255),
    amount_cents BIGINT NOT NULL,
    transaction_type VARCHAR(50) DEFAULT 'subscription',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_affiliate_earnings_affiliate ON affiliate_earnings(affiliate_id);
CREATE INDEX idx_affiliate_earnings_tenant ON affiliate_earnings(tenant_id);
CREATE INDEX idx_affiliate_earnings_created_at ON affiliate_earnings(created_at);

CREATE TABLE affiliate_withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    admin_note TEXT,
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_affiliate_withdrawals_affiliate ON affiliate_withdrawals(affiliate_id);
CREATE INDEX idx_affiliate_withdrawals_status ON affiliate_withdrawals(status);

ALTER TABLE tenants
ADD COLUMN referred_by_affiliate_id UUID REFERENCES affiliates(id) ON DELETE SET NULL;

CREATE INDEX idx_tenants_referred_by ON tenants(referred_by_affiliate_id);
```

### API Endpoints

#### `GET /api/public/affiliate-leaderboard`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "monthly": [
      {
        "name": "Budi S.",
        "closings": 150,
        "revenue_cents": 1500000000,
        "rank": 1
      },
      {
        "name": "Ani W.",
        "closings": 120,
        "revenue_cents": 1200000000,
        "rank": 2
      }
    ],
    "all_time": [
      {
        "name": "Dewi K.",
        "closings": 500,
        "revenue_cents": 5000000000,
        "rank": 1
      }
    ]
  }
}
```

**Data Masking:**
- Name: First name + initial (e.g., "Budi Santoso" → "Budi S.")
- No phone, email, referral_code exposed

**Error Cases:**
- None — always returns 200 OK (empty array if no data)

#### `GET /api/affiliate/dashboard`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "referral_code": "AGEN-BUDI",
    "referral_link": "https://wch.id/register?ref=AGEN-BUDI",
    "cash_balance_cents": 50000000,
    "total_earnings_cents": 200000000,
    "this_month_earnings_cents": 15000000,
    "total_closings": 150,
    "this_month_closings": 12
  }
}
```

**Error Cases:**
- `401 Unauthorized` — Not affiliate user
- `404 Not Found` — Affiliate record not exists

#### `GET /api/affiliate/earnings?limit=50&offset=0`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "earnings": [
      {
        "id": "uuid",
        "tenant_name": "Warung Bu Siti",
        "amount_cents": 2000000,
        "transaction_type": "subscription",
        "created_at": "2026-06-14T10:30:00Z"
      }
    ],
    "total": 150
  }
}
```

#### `POST /api/affiliate/withdraw`

**Request:**
```json
{
  "amount_cents": 50000000,
  "bank_name": "BCA",
  "bank_account_number": "1234567890",
  "bank_account_name": "BUDI SANTOSO"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Withdrawal request submitted. Will be processed within 1-3 business days.",
  "data": {
    "withdrawal_id": "uuid",
    "amount_cents": 50000000,
    "status": "pending"
  }
}
```

**Error Cases:**
- `400 Bad Request` — Amount < Rp 100.000 (10000000 cents) or > cash_balance_cents
- `401 Unauthorized` — Not affiliate user

#### `GET /api/superadmin/affiliate/withdrawals?status=pending`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "withdrawals": [
      {
        "id": "uuid",
        "affiliate_name": "Budi Santoso",
        "affiliate_code": "AGEN-BUDI",
        "amount_cents": 50000000,
        "bank_name": "BCA",
        "bank_account_number": "1234567890",
        "bank_account_name": "BUDI SANTOSO",
        "status": "pending",
        "created_at": "2026-06-14T10:30:00Z"
      }
    ],
    "total": 5
  }
}
```

#### `PUT /api/superadmin/affiliate/withdrawals/:id`

**Request:**
```json
{
  "status": "completed",
  "admin_note": "Transfer BCA 12345 completed at 2026-06-14 15:30"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Withdrawal updated successfully"
}
```

**Error Cases:**
- `400 Bad Request` — Invalid status (not `approved` or `rejected`)
- `401 Unauthorized` — Not superadmin
- `404 Not Found` — Withdrawal ID not exists

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (billing-service):**
```go
// affiliate_test.go
func TestCalculateCommission_ValidInvoice(t *testing.T) {
    // Invoice Rp 100.000, commission 20%
    // Expect: commission = Rp 20.000
}

func TestWithdrawRequest_BelowMinimum(t *testing.T) {
    // Amount = Rp 50.000 (< Rp 100.000)
    // Expect: 400 Bad Request
}

func TestWithdrawRequest_InsufficientBalance(t *testing.T) {
    // Balance = Rp 100.000, withdraw = Rp 150.000
    // Expect: 400 Bad Request
}
```

### Integration Tests

```bash
# 1. Register affiliate
curl -X POST http://localhost:8003/affiliate/register \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"bank_name":"BCA","bank_account_number":"1234567890"}'
# → Returns referral_code

# 2. Tenant register dengan referral code
curl -X POST "http://localhost:8001/register?ref=AGEN-BUDI" \
  -d '{"phone":"08123456789","business_name":"Warung Test"}'
# → tenants.referred_by_affiliate_id populated

# 3. Tenant subscribe → commission credited
curl -X POST http://localhost:8003/subscribe \
  -H "Authorization: Bearer $TENANT_TOKEN" \
  -d '{"plan_id":"lite","payment_method":"wallet"}'
# → affiliate_earnings INSERT, cash_balance_cents updated

# 4. Public leaderboard
curl http://localhost:8000/api/public/affiliate-leaderboard
# → 200 OK, masked data

# 5. Affiliate request withdraw
curl -X POST http://localhost:8003/affiliate/withdraw \
  -H "Authorization: Bearer $AFFILIATE_TOKEN" \
  -d '{"amount_cents":10000000}'
# → status pending

# 6. Superadmin approve
curl -X PUT http://localhost:8003/admin/affiliate/withdrawals/$WITHDRAWAL_ID \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
  -d '{"status":"completed","admin_note":"Transfer done"}'
# → 200 OK
```

### Manual Testing Checklist

- [ ] Affiliate register → referral_code generated
- [ ] Tenant register dengan `?ref=CODE` → `referred_by_affiliate_id` locked
- [ ] Tenant first subscribe → affiliate dapat commission
- [ ] Tenant renewal (month 2, 3, ...) → affiliate tetap dapat commission (lifetime)
- [ ] Public leaderboard show TOP 10 monthly + all-time
- [ ] Affiliate dashboard show balance, earnings history, referral link
- [ ] Affiliate withdraw below Rp 100k → error
- [ ] Affiliate withdraw above balance → error
- [ ] Affiliate withdraw valid amount → status pending
- [ ] Superadmin list pending withdrawals
- [ ] Superadmin approve → status completed + admin_note

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Affiliate commission credited", 
  "affiliate_id", affiliateID,
  "tenant_id", tenantID,
  "amount_cents", commission,
  "invoice_id", invoiceID)

slog.Info("Withdrawal requested", 
  "affiliate_id", affiliateID,
  "amount_cents", amount,
  "current_balance_cents", balance)
```

**Metrics to track:**
- Affiliate count (active vs inactive)
- Commission payout per month (cost untuk platform)
- Withdrawal request count + approval rate
- Average time-to-approval for withdrawals
- Top 10 affiliates by revenue generated

**Alerts:**
- Withdrawal pending > 7 days → reminder untuk superadmin process
- Commission payout > 30% total revenue → investigate affiliate fraud or abuse

**Grafana Dashboard:**
- Panel 1: Affiliate funnel (register → first refer → first commission → first withdrawal)
- Panel 2: Commission trend (monthly payout)
- Panel 3: Leaderboard distribution (Pareto: 20% affiliates generate 80% revenue?)

---

## 🚀 Rollout Plan

### Phase 1: Backend + Migration (Done ✅)
- Migration 000040: `affiliates`, `affiliate_earnings`, `affiliate_withdrawals` tables
- Deploy billing-service dengan affiliate endpoints
- Payment webhook integration (commission calculation + credit)

### Phase 2: Frontend Affiliate Dashboard (Done ✅)
- Deploy umkm-web `/affiliate` page (referral link, balance, earnings, withdraw)
- Test: affiliate register → refer tenant → receive commission → request withdraw

### Phase 3: Superadmin Withdrawal Management (Done ✅)
- Superadmin dashboard page untuk list + approve/reject withdrawals
- Test: superadmin approve withdrawal → manual transfer → mark completed

### Phase 4: Public Leaderboard (Done ✅)
- API endpoint `/api/public/affiliate-leaderboard`
- Landing page integration (TOP 10 display)
- Social proof: "Bergabung dengan 500+ agen sukses"

### Phase 5: Analytics & Gamification (Future)
- Email notification untuk affiliate milestone (first refer, 10 closings, Rp 1jt earned)
- Badge system (Bronze/Silver/Gold/Platinum based on total earnings)
- Tiered commission (20% → 25% setelah 50 closings)

### Rollback
- **Phase 1 rollback:** Revert migration 000040 → `referred_by_affiliate_id` column dropped, commission logic disabled
- **Emergency:** Set commission rate ke 0% via config → no new earnings (existing balance tetap bisa di-withdraw)
- **Withdrawal freeze:** Disable `/api/affiliate/withdraw` endpoint → prevent new withdrawal requests

---

## 🔮 Future Enhancements (Out of Scope)

- **Multi-Level Marketing (MLM):** 2-level commission (A refer B, B refer C → A dapat commission dari C juga) — butuh legal review MLM regulations
- **Auto-Withdrawal:** Auto-transfer via payment gateway API (Xendit disbursement) saat reach threshold — reduce manual work
- **Referral Analytics Dashboard:** Affiliate personal dashboard dengan funnel metrics (clicks, registers, conversions, LTV)
- **Referral Contests:** Monthly contest dengan hadiah tambahan untuk top 3 affiliates (e.g., bonus Rp 5jt untuk rank 1)
- **White-Label Affiliate:** Affiliate bisa customize landing page dengan branding sendiri (subdomain: agent-budi.wch.id)

---

## 📚 References

- [F054: Referral System Discount + Commission](F054_referral_system_discount_downline_commission_uplin.md) — Referral discount untuk tenant + commission calculation logic
- [Payment Webhook Implementation](../../services/billing-service/webhook.go) — Commission credit logic
- [Affiliate Dashboard Component](../../frontend/umkm-web/src/components/AffiliateDashboard.vue) — Frontend UI
- [Stripe Connect Referrals](https://stripe.com/docs/connect/referrals) — Inspiration untuk commission structure

---

## 📝 Notes & Decisions

**2026-06-14:** Decision: Commission 20% dari invoice value (setelah discount) bukan base price — align incentive agar affiliate promote adoption, bukan hanya upsell high-tier plan.  
**2026-06-14:** Withdrawal manual process (superadmin approval + mBanking transfer) untuk MVP — auto-disbursement via Xendit API defer ke future karena butuh Xendit business account approval + API key provisioning.  
**2026-06-14:** Public leaderboard data masked (name initial only) untuk privacy — full name hanya visible untuk superadmin dashboard.  
**2026-06-14:** Minimal withdraw Rp 100.000 untuk reduce admin overhead — lower threshold (e.g., Rp 50.000) would increase manual transfer frequency 2x.  
**2026-06-14:** `referred_by_affiliate_id` lifetime lock on first subscribe (tidak bisa diubah) — prevent fraud (tenant bisa ganti referral code setiap renewal untuk avoid commission).
