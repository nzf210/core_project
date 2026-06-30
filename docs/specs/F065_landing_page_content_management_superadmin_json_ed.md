# F065: Landing Page Content Management — Superadmin JSON Editor


### 🎯 Tujuan (Goals)

1. **Subscription via wallet**: Jika wallet balance cukup → bypass Xendit → deduct wallet → activateSubscription langsung
2. **Topup+Subscribe flow**: Jika balance kurang → partial pay? (opsi lanjutan). MVP: full pay dari wallet atau full Xendit.
3. **Wallet balance indicator di UI**: Saat checkout subscription, tampilkan "Saldo wallet: Rp X. Bayar via Wallet?"
4. **Auto-renew subscription via wallet**: Setiap bulan, auto-deduct wallet untuk perpanjang subscription
5. **Referral discount tetap jalan**: Baik bayar via Xendit maupun wallet, referral discount dihitung sama


### 🔄 Flow End-to-End

#### A. Topup Wallet (EXISTING — didokumentasikan untuk kelengkapan)

```
Tenant klik "Topup Wallet" → /wallet/topup { amount_cents }
         │
         ├─ 1. Validasi: min Rp 10.000 (10000 cents) ✅
         ├─ 2. CREATE Xendit invoice:
         │     external_id = {uuid}-wallet-topup-{tenantID}
         │     amount = req.AmountCents
         │     → Return invoice_url ke tenant
         │
         ├─ 3. Tenant bayar via Xendit
         │
         └─ 4. Xendit webhook → handlePaymentWebhook:
              ├─ Deteksi "-wallet-topup-" di external_id ✅
              ├─ Dedup: cek existing wallet_transactions.reference ✅
              ├─ Transaction: UPSERT wallet_credits (balance += amount)
              ├─ INSERT wallet_transactions (type='topup')
              └─ WA notification: "Topup Rp X berhasil. Saldo: Rp Y"
```

#### B. Subscription via Wallet (NEW)

```
Tenant pilih paket → handleSubscribe
         │
         ├─ 1. Hitung final_price:
         │     base_price → voucher_discount → referral_discount
         │     (sama seperti sekarang, line 617-649)
         │
         ├─ 2. Cek bayar_via_wallet flag dari request body:
         │     req.PayViaWallet = true
         │     → Jika tidak dikirim, default false (Xendit seperti biasa)
         │
         ├─ 3. Jika PayViaWallet = true:
         │     ├─ Cek wallet balance >= final_price
         │     │   CheckWalletBalance(tenantID, final_price)
         │     │
         │     ├─ Jika CUKUP:
         │     │   ├─ DeductWalletBalance(tenantID, final_price,
         │     │   │   "subscription:{planID}:{ts}",
         │     │   │   "Pembayaran langganan {planName} via Wallet")
         │     │   │   → transaction_type = 'subscription'
         │     │   │
         │     │   ├─ activateSubscription(..., activatedBy="wallet")
         │     │   │
         │     │   ├─ F054: Affiliate commission (sama seperti Xendit
         │     │   │   → hitung dari final_price)
         │     │   │
         │     │   └─ Response: { status: 'activated', method: 'wallet' }
         │     │
         │     └─ Jika TIDAK CUKUP:
         │         └─ Response 402: { message: "Saldo tidak cukup",
         │             balance_cents, required_cents,
         │             topup_url: "/wallet" }
         │
         └─ 4. Jika PayViaWallet = false (default):
             └─ CREATE Xendit invoice seperti biasa (existing flow)
```

#### C. Auto-Renew Subscription via Wallet (NEW — background worker)

```
Cron job (di billing-service) — setiap jam:
         │
         ├─ SELECT FROM tenant_subscriptions ts
         │   JOIN wallet_credits wc ON wc.tenant_id = ts.tenant_id
         │   WHERE ts.status = 'active'
         │     AND ts.remaining_days <= 3       -- 3 hari sebelum expired
         │     AND wc.balance_cents >= (
         │       SELECT price_monthly FROM saas_plans sp
         │       WHERE sp.id = ts.plan_id
         │     )
         │     AND ts.auto_renew_via_wallet = true   -- flag baru!
         │
         ├─ Untuk setiap row:
         │   ├─ Deduct wallet: price_monthly
         │   │   → transaction_type = 'subscription_auto_renew'
         │   │
         │   ├─ Extend subscription: remaining_days += 30
         │   │   current_plan_expires_at = NOW() + 30 days
         │   │
         │   ├─ F054: Affiliate commission dari amount
         │   │
         │   └─ Kirim notifikasi: "Langganan diperpanjang otomatis via Wallet"
         │
         └─ Logging: subscription_auto_renew_logs (atau reuse wallet_transactions)
```

#### D. Referral Discount + Wallet Integration

```
handleSubscribe (patch):
         │
         ├─ 1. base_price = priceMonthly ✅ (existing)
         ├─ 2. Voucher discount ✅ (existing)
         ├─ 3. Referral discount ✅ (existing — line 620-631)
         │
         ├─ 4. Jika PayViaWallet:
         │     └─ Deduct final_price dari wallet (setelah semua diskon)
         │
         └─ 5. Jika !PayViaWallet (Xendit):
             └─ CREATE invoice dengan final_price (existing)
```

**PENTING:** Referral discount dihitung SEBELUM decide bayar via wallet atau Xendit. Jadi discount_amount sama dalam kedua kasus.


### ✅ Acceptance Criteria (AC)

- [x] AC-1: POST `/subscribe` dengan `pay_via_wallet=true` dan balance cukup → deduct wallet → activateSubscription → response `{status:'activated', payment_method:'wallet'}`
- [x] AC-2: POST `/subscribe` dengan `pay_via_wallet=true` dan balance kurang → response 402 + `{required_cents, balance_cents, topup_url}`
- [x] AC-3: POST `/subscribe` tanpa `pay_via_wallet` (default false) → Xendit invoice seperti biasa
- [x] AC-4: Referral discount tetap di-apply baik via wallet maupun Xendit
- [x] AC-5: Wallet auto-renew cron deferred — manual renew via POST /addons/purchase sufficient for MVP.
- [x] AC-6: Frontend checkout menampilkan wallet balance + opsi "Bayar dari Wallet"
- [x] AC-7: Wallet page menampilkan transaksi subscription — DeductWalletBalance INSERT type='consume' + description. Wallet.vue render non-topup tx sebagai negatif.
- [x] AC-8: `go build`, `go vet`, `vue-tsc` clean ✅


### Notes:

- Wallet subscription adalah **opsi**, bukan kewajiban. Tenant bisa tetap pakai Xendit.
- Auto-renew flag (`auto_renew_via_wallet`) terpisah dari `auto_renew` addon — jangan campur.
- Referral discount priority: voucher → referral → final_price. Konsisten dengan F054 fix.
- Untuk MVP: wallet deduct full amount. Partial payment (wallet + Xendit) terlalu kompleks untuk sekarang— bisa jadi F059 jika diperlukan.
- Race condition: `DeductWalletBalance` sudah pakai `SELECT ... FOR UPDATE` di dalam transaksi (via `UPDATE ... WHERE balance_cents >= $1`). Aman untuk concurrent request.


### 📌 Background — State Saat Ini

```
现状 (Current):
  - User buka domain → langsung ke /login
  - Tidak ada halaman "Apa itu WCH?" atau daftar fitur
  - Tidak ada pricing display sebelum register
  - Satu-satunya jalur konversi: orang daftar karena dikasih tahu (word of mouth)
  
  Route yang ada:
  - /login → Login.vue
  - /register → Register.vue
  - Semua route lain requires auth

  Theme: dark/light mode SUDAH ada (F056)
  CSS variables: SUDAH ada di main.css

  Masalah:
  - Bounce rate tinggi: orang yang dikasih link WCH langsung lihat login → close
  - Tidak ada SEO: Google tidak index halaman kosong
  - Tidak ada conversion path: landing → CTA → register adalah funnel standar
```

### 🎯 Tujuan

1. Halaman publik yang menjelaskan produk dengan jelas: "Aplikasi Kasir, Pembukuan & AI untuk UMKM"
2. Menampilkan fitur unggulan: POS, Akuntansi Double-Entry, AI Chatbot, Laporan
3. Pricing table (Lite/Pro/Ultimate) — ambil data dari backend (opsional: static fallback)
4. CTA yang jelas: "Coba Gratis" → register
5. SEO-friendly: meta tags, semantic HTML
6. Fully responsive (mobile-first)
7. Dark/light mode compatible (reuse F056 theme system)
8. User yang sudah login → redirect ke dashboard


## F065: Landing Page Content Management — Superadmin JSON Editor

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Tanggal:** 2026-06-29

### Deskripsi

Landing page content (hero, features, steps, testimonials, CTA, footer) sekarang **dinamis dari database** via `landing_configs` table. Superadmin bisa edit konten via JSON editor di dashboard, dan perubahan langsung tampil di landing page.

### Architecture

```
Superadmin UI (JSON Editor)
  → PUT /api/superadmin/landing-configs/?id=hero
  → billing-service:8003/admin/landing-configs
  → PostgreSQL landing_configs table
  → Invalidate cache (in-memory, 6h TTL)

Landing Page (public)
  → GET /landing-configs
  → api-gateway:8000 → billing-service:8003
  → Check cache → DB fallback
  → Return all configs as JSON
  → LandingPage.vue render with fallback
```

### Database

```sql
CREATE TABLE landing_configs (
    id         VARCHAR(50) PRIMARY KEY,  -- 'hero', 'features', 'steps', 'testimonials', 'cta', 'footer'
    content    JSONB NOT NULL DEFAULT '{}',
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### API Endpoints

| Method | Path | Auth | Deskripsi |
|:-------|:-----|:-----|:----------|
| GET | `/landing-configs` | Public | Semua config (cached 6 jam) |
| GET | `/api/superadmin/landing-configs` | Superadmin | List semua config + metadata |
| PUT | `/api/superadmin/landing-configs/?id=hero` | Superadmin | Update config + auto invalidate cache |

### Caching

- **In-memory cache (sync.RWMutex)**, 6 jam TTL
- Tidak perlu Redis — landing page load frequency rendah
- Cache auto-invalidated saat superadmin update config
- Response header `X-Cache: HIT/MISS` untuk debugging

### Acceptance Criteria

- [x] AC-1: Landing page fetch content dari `GET /landing-configs` (public)
- [x] AC-2: Perubahan content via superadmin langsung tampil (cache invalidate)
- [x] AC-3: Fallback static content jika API gagal (landing tidak pernah blank)
- [x] AC-4: Hanya superadmin yang bisa edit (via `/api/superadmin/` route)
- [x] AC-5: `vue-tsc` clean, `go build` clean

### Files Changed

- `shared/migrations/000083_landing_configs.{up,down}.sql` — NEW migration
- `services/billing-service/landing_config_handlers.go` — NEW handlers (public + admin)
- `services/billing-service/main.go` — register `/landing-config`, `/admin/landing-configs`
- `services/api-gateway/main.go` — proxy `/landing-configs` (public) + `/api/superadmin/landing-configs` (admin)
- `frontend/umkm-web/src/api.ts` — `getLandingConfigs()`
- `frontend/umkm-web/src/superadminApi.ts` — `getLandingConfigs()`, `updateLandingConfig()`
- `frontend/umkm-web/src/components/LandingPage.vue` — dynamic content via computed getters + fallback

### Migration

```bash
make migrate-new NAME=landing_configs
# → shared/migrations/000083_landing_configs.up.sql / .down.sql
```
