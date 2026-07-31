# F054: Referral System — Discount Downline + Commission Upline

**Date:** 2026-06-16  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F036](../FEATURE_MAP.md) (Lifetime Affiliate), [F053](F053_admin-configurable_addon_pricing_addon_purchase_fl.md) (Addon Purchase)

---

## 🎯 Objectives

Lifetime referral system yang memberikan discount ke downline dan commission ke upline untuk semua transaksi di platform WCH.

**Tujuan eksplisit:**
1. Tenant yang daftar via kode referral mendapat potongan harga **seumur hidup** untuk semua pembelian (subscription renewal, addon purchase, campaign checkout)
2. Upline (agen/affiliator) mendapat komisi **seumur hidup** setiap downlinenya melakukan pembayaran apa pun
3. Superadmin dapat configure discount percentage dan commission rate via `referral_config` table

**Problem yang diselesaikan:**
- Platform tidak punya insentif untuk word-of-mouth marketing — user tidak termotivasi refer teman/keluarga
- Discount one-time (voucher) tidak sustainable untuk long-term retention
- Manual tracking referral via spreadsheet tidak scalable dan error-prone

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Referral Code Registration**
  - *Verification:* User register dengan query param `?ref=AFFILIATE_CODE` → `tenants.referred_by_affiliate_id` populated
  - *Example:* `POST /api/auth/register?ref=AGENT123` → tenant baru link ke affiliate AGENT123

- [x] **AC-2: Lifetime Discount Calculation**
  - *Verification:* Setiap checkout (subscription/addon/campaign) → cek `tenants.referred_by_affiliate_id` → apply discount sebelum payment
  - *Example:* Subscription Rp 100.000 → discount 10% → final price Rp 90.000

- [x] **AC-3: Discount Config via DB**
  - *Verification:* Superadmin update `referral_config.downline_discount_percent` → berlaku untuk semua transaksi berikutnya (no code change)
  - *Example:* `UPDATE referral_config SET downline_discount_percent = 15` → discount naik dari 10% ke 15%

- [x] **AC-4: Commission Calculation**
  - *Verification:* Setiap downline bayar → hitung commission dari `final_price` (setelah discount) → credit ke `affiliates.cash_balance_cents`
  - *Example:* Downline bayar Rp 90.000 → commission 20% → Rp 18.000 masuk ke affiliate balance

- [x] **AC-5: Commission Config via DB**
  - *Verification:* Superadmin update `referral_config.upline_commission_percent` → berlaku untuk transaksi berikutnya
  - *Example:* `UPDATE referral_config SET upline_commission_percent = 25` → commission naik dari 20% ke 25%

- [x] **AC-6: Audit Trail via invoice_referrals**
  - *Verification:* Setiap transaksi dengan referral discount → INSERT `invoice_referrals` (tenant_id, affiliate_id, discount_amount, invoice_id)
  - *Example:* `SELECT * FROM invoice_referrals WHERE tenant_id = '...'` → list semua discount yang pernah diterima tenant

- [x] **AC-7: Audit Trail via affiliate_earnings**
  - *Verification:* Setiap commission payout → INSERT `affiliate_earnings` (affiliate_id, amount, transaction_type, tenant_id)
  - *Example:* `SELECT * FROM affiliate_earnings WHERE affiliate_id = '...'` → list semua commission yang diterima affiliate

- [x] **AC-8: Multi-Product Support**
  - *Verification:* Referral discount berlaku untuk **semua produk WCH** (UMKM subscription/addon, Campaign checkout)
  - *Example:* Tenant beli UMKM addon → discount 10% | Tenant bayar Campaign tools → discount 10%

- [x] **AC-9: Commission Tidak Stacking**
  - *Verification:* Jika downline juga jadi affiliate dan refer orang lain → commission hanya ke direct upline (1 level, bukan multi-level marketing)
  - *Example:* A refer B, B refer C → B bayar → A dapat commission | C bayar → B dapat commission, A tidak dapat apa-apa

- [x] **AC-10: Prevent Self-Referral**
  - *Verification:* User tidak bisa register dengan kode referral sendiri
  - *Example:* User AGENT123 coba register dengan `?ref=AGENT123` → reject atau ignore (tidak simpan `referred_by_affiliate_id`)

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│         User Registration with ?ref=CODE            │
│  1. Validate affiliate_code exists in affiliates    │
│  2. INSERT tenants { referred_by_affiliate_id }     │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│          Checkout Flow (Subscription/Addon)         │
│  1. Query: SELECT referred_by_affiliate_id          │
│  2. If exists:                                      │
│     - Query referral_config (discount %)            │
│     - discount_amount = price * discount % / 100    │
│     - final_price = price - discount_amount         │
│     - INSERT invoice_referrals (audit trail)        │
│  3. Process payment (Xendit/Wallet) with final_price│
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│        Commission Payout (After Payment Success)    │
│  1. Query referral_config (commission %)            │
│  2. commission = final_price * commission % / 100   │
│  3. INSERT affiliate_earnings (audit trail)         │
│  4. UPDATE affiliates.cash_balance_cents += commission│
└─────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: 000045_referral_system.up.sql

-- Config table (single row, superadmin editable)
CREATE TABLE referral_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    downline_discount_percent INT NOT NULL DEFAULT 10,  -- 10% discount default
    upline_commission_percent INT NOT NULL DEFAULT 20,  -- 20% commission default
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed default config
INSERT INTO referral_config (downline_discount_percent, upline_commission_percent) 
VALUES (10, 20);

-- Add referral link to tenants table
ALTER TABLE tenants
ADD COLUMN referred_by_affiliate_id UUID REFERENCES affiliates(id) ON DELETE SET NULL;

CREATE INDEX idx_tenants_referred_by ON tenants(referred_by_affiliate_id);

-- Audit trail: discount history
CREATE TABLE invoice_referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    invoice_id VARCHAR(255),  -- Xendit invoice_id atau subscription_id
    discount_amount_cents BIGINT NOT NULL,  -- Discount yang diterima tenant (sen)
    original_price_cents BIGINT NOT NULL,
    final_price_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_invoice_referrals_tenant ON invoice_referrals(tenant_id);
CREATE INDEX idx_invoice_referrals_affiliate ON invoice_referrals(affiliate_id);
CREATE INDEX idx_invoice_referrals_invoice ON invoice_referrals(invoice_id);

-- Audit trail: commission history (reuse existing affiliate_earnings table)
-- ALTER TABLE affiliate_earnings ADD transaction_type column (already exists from F036)
-- transaction_type values: 'subscription', 'addon_purchase', 'campaign_checkout'
```

### API Endpoints

#### `POST /api/auth/register?ref=AFFILIATE_CODE`

**Request:**
```json
{
  "phone_number": "08123456789",
  "business_name": "Warung Bu Siti",
  "password": "secure123"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "uuid",
    "tenant_id": "uuid",
    "referred_by_affiliate_id": "affiliate-uuid",
    "message": "Selamat! Anda terdaftar dengan kode referral AGENT123. Dapatkan diskon 10% untuk semua pembelian."
  }
}
```

**Error Cases:**
- `400 Bad Request` — Referral code tidak valid (affiliate tidak ditemukan)
- `409 Conflict` — Self-referral attempt (user coba pakai kode sendiri)

#### `POST /api/umkm/subscribe` (modified untuk referral)

**Backend Logic (pseudo-code):**
```go
// 1. Hitung base price
basePrice := plan.PriceCents

// 2. Cek voucher discount (existing F002)
voucherDiscount := applyVoucherDiscount(basePrice, voucherCode)
priceAfterVoucher := basePrice - voucherDiscount

// 3. Cek referral discount (F054 NEW)
referralDiscount := 0
var affiliateID *string
row := DB.QueryRow("SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID)
row.Scan(&affiliateID)

if affiliateID != nil {
    config := getReferralConfig() // SELECT * FROM referral_config LIMIT 1
    referralDiscount = priceAfterVoucher * config.DownlineDiscountPercent / 100
}

finalPrice := priceAfterVoucher - referralDiscount

// 4. Process payment (Xendit/Wallet) dengan finalPrice
// ...

// 5. Audit trail
if affiliateID != nil {
    DB.Exec(`INSERT INTO invoice_referrals 
        (tenant_id, affiliate_id, invoice_id, discount_amount_cents, original_price_cents, final_price_cents)
        VALUES ($1, $2, $3, $4, $5, $6)`,
        tenantID, affiliateID, invoiceID, referralDiscount, basePrice, finalPrice)
    
    // 6. Commission payout
    commission := finalPrice * config.UplineCommissionPercent / 100
    DB.Exec(`INSERT INTO affiliate_earnings 
        (affiliate_id, amount_cents, transaction_type, tenant_id)
        VALUES ($1, $2, 'subscription', $3)`,
        affiliateID, commission, tenantID)
    DB.Exec(`UPDATE affiliates SET cash_balance_cents = cash_balance_cents + $1 WHERE id = $2`,
        commission, affiliateID)
}
```

#### `GET /api/superadmin/referral-config`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "downline_discount_percent": 10,
    "upline_commission_percent": 20,
    "is_active": true,
    "updated_at": "2026-06-16T10:30:00Z"
  }
}
```

#### `PUT /api/superadmin/referral-config`

**Request:**
```json
{
  "downline_discount_percent": 15,
  "upline_commission_percent": 25
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Referral config updated successfully",
  "data": {
    "downline_discount_percent": 15,
    "upline_commission_percent": 25
  }
}
```

**Error Cases:**
- `400 Bad Request` — Invalid percentage (< 0 atau > 100)
- `401 Unauthorized` — Not superadmin

#### `GET /api/umkm/referral-discount`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "has_referral": true,
    "affiliate_code": "AGENT123",
    "discount_percent": 10,
    "lifetime_savings_cents": 250000,
    "transaction_count": 5
  }
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (billing-service):**
```go
// referral_test.go
func TestApplyReferralDiscount_ValidAffiliate(t *testing.T) {
    // Mock DB: tenant has referred_by_affiliate_id
    // Mock config: 10% discount
    // basePrice = 100000
    // Expect: discount = 10000, finalPrice = 90000
}

func TestApplyReferralDiscount_NoAffiliate(t *testing.T) {
    // Mock DB: tenant.referred_by_affiliate_id = NULL
    // Expect: discount = 0, finalPrice = basePrice
}

func TestCalculateCommission_ValidTransaction(t *testing.T) {
    // finalPrice = 90000, commission 20%
    // Expect: commission = 18000
}

func TestPreventSelfReferral(t *testing.T) {
    // User coba register dengan own affiliate code
    // Expect: referred_by_affiliate_id = NULL (ignore)
}
```

### Integration Tests

```bash
# 1. Register dengan referral code
curl -X POST "http://localhost:8000/api/auth/register?ref=AGENT123" \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "08123456789",
    "business_name": "Warung Test",
    "password": "test123"
  }'
# → 200 OK, referred_by_affiliate_id populated

# 2. Subscribe dengan referral discount
TENANT_TOKEN=$(curl -X POST http://localhost:8000/api/auth/login ...)
curl -X POST http://localhost:8000/api/umkm/subscribe \
  -H "Authorization: Bearer $TENANT_TOKEN" \
  -H "X-Tenant-ID: ..." \
  -d '{"plan_id":"lite","payment_method":"wallet"}'
# → final_price dikurangi 10%

# 3. Verify audit trail
psql -d wch -c "SELECT * FROM invoice_referrals WHERE tenant_id = '...'"
# → discount_amount_cents = 10000

# 4. Verify commission payout
psql -d wch -c "SELECT * FROM affiliate_earnings WHERE affiliate_id = 'AGENT123-uuid'"
# → amount_cents = 18000 (20% dari 90000)

# 5. Superadmin update config
curl -X PUT http://localhost:8000/api/superadmin/referral-config \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
  -d '{"downline_discount_percent":15,"upline_commission_percent":25}'
# → 200 OK

# 6. New subscription dengan updated config
# → discount 15%, commission 25%
```

### Manual Testing Checklist

- [ ] User A register dengan `?ref=AGENT123` → `referred_by_affiliate_id` populated
- [ ] User A subscribe → discount 10% applied
- [ ] Affiliate AGENT123 balance bertambah 20% dari final_price
- [ ] `invoice_referrals` dan `affiliate_earnings` table populated correctly
- [ ] Superadmin update discount 10%→15% → subscription berikutnya pakai 15%
- [ ] User B (no referral) subscribe → no discount applied
- [ ] User A beli addon → discount 10% applied (multi-product support)
- [ ] User A refer User C → User C bayar → User A dapat commission (not User B)
- [ ] User A coba register dengan own code → self-referral prevented

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Referral discount applied", 
  "tenant_id", tenantID,
  "affiliate_id", affiliateID,
  "discount_amount_cents", discountAmount,
  "final_price_cents", finalPrice)

slog.Info("Commission payout", 
  "affiliate_id", affiliateID,
  "commission_cents", commission,
  "transaction_type", "subscription",
  "tenant_id", tenantID)
```

**Metrics to track:**
- Referral conversion rate (register dengan ref code / total register)
- Average lifetime savings per referred tenant
- Average commission per affiliate per month
- Referral discount total per month (cost untuk platform)

**Alerts:**
- Referral discount > 50% total revenue → investigate config misconfiguration atau abuse
- Commission payout failed → DB transaction issue

**Grafana Dashboard:**
- Panel 1: Referral funnel (register dengan ref → first purchase → repeat purchase)
- Panel 2: Top 10 affiliates by commission earned
- Panel 3: Discount vs Commission ratio (sustainability check)

---

## 🚀 Rollout Plan

### Phase 1: Backend + Migration (Done ✅)
- Migration 000045: `referral_config`, `invoice_referrals` tables, `tenants.referred_by_affiliate_id` column
- Deploy billing-service dengan referral discount + commission logic
- Seed default config (10% discount, 20% commission)

### Phase 2: Frontend Registration (Done ✅)
- Umkm-web `/register` detect `?ref=` query param → pass ke backend
- Display referral benefit message setelah register success

### Phase 3: Frontend Referral Display (Done ✅)
- Settings page → "Referral Info" section → show affiliate code, discount percent, lifetime savings
- Dashboard → banner "Anda hemat Rp X dengan kode referral AGENT123"

### Phase 4: Superadmin Config UI (Done ✅)
- Superadmin dashboard → "Referral Settings" page
- Form edit `downline_discount_percent` dan `upline_commission_percent`
- Display impact preview (e.g., "New discount: 15% → Rp 15.000 off per Rp 100.000")

### Phase 5: Analytics Dashboard (Future)
- Affiliate leaderboard (public) → top earners by month
- Affiliate personal dashboard → commission breakdown, downline list, performance metrics

### Rollback
- **Phase 1 rollback:** Revert migration 000045 → `referred_by_affiliate_id` column dropped, referral logic disabled
- **Emergency:** Set `referral_config.is_active = false` → disable all discount + commission (feature flag)
- **Partial rollback:** Set discount/commission percent ke 0 → system tetap jalan, tapi no incentive

---

## 🔮 Future Enhancements (Out of Scope)

- **Multi-Level Referral:** 2-level commission (A refer B, B refer C → A dapat commission dari C juga) — butuh consensus apakah ini MLM scheme
- **Tiered Commission:** Commission rate berbeda berdasarkan total sales affiliate (e.g., 20% untuk <Rp 10jt, 25% untuk >Rp 10jt)
- **Referral Leaderboard Gamification:** Badge, rank, bonus untuk top affiliate per bulan
- **Referral Code Customization:** Affiliate bisa pilih custom code (e.g., `WARUNGSITI` bukan `AGENT123`)
- **Referral Expiry:** Discount/commission hanya berlaku X bulan pertama (bukan lifetime) — reduce long-term cost
- **Referral Clawback:** Jika downline refund/chargeback → clawback commission dari affiliate balance

---

## 📚 References

- [F036: Lifetime Affiliate System](../FEATURE_MAP.md) — Affiliate dashboard + public leaderboard
- [F002: Voucher Link Subscription](../FEATURE_MAP.md) — Voucher discount logic (stacking dengan referral)
- [F053: Addon Purchase Flow](F053_admin-configurable_addon_pricing_addon_purchase_fl.md) — Addon purchase dengan referral discount
- [Stripe Referral Program Design](https://stripe.com/docs/connect/referrals) — Inspiration untuk audit trail + commission structure

---

## 📝 Notes & Decisions

**2026-06-16:** Decision: Referral discount applied **after voucher discount** (not before) untuk max benefit ke tenant. Flow: base price → voucher discount → referral discount → final price.  
**2026-06-16:** Commission calculated dari **final price** (setelah discount) bukan base price — align incentive agar affiliate promosikan adoption, bukan hanya upsell.  
**2026-06-16:** Single-level commission only (bukan multi-level) untuk avoid MLM regulatory risk. A refer B, B refer C → A tidak dapat commission dari C.  
**2026-06-16:** Self-referral prevention via backend check (bukan frontend validation) — user tidak bisa bypass via API call langsung. Jika detected → ignore `referred_by_affiliate_id`, tidak reject registration (better UX).  
**2026-06-16:** `referral_config` single row table (bukan per-affiliate custom rate) untuk simplicity. Future enhancement: tiered commission berdasarkan performance.
