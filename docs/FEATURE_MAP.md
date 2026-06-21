# 🗺️ WCH Platform — Feature Map & Specification

> **Dokumen utama untuk AI governance.** Setiap fitur baru/wubah WAJIB ada SPEC di sini.
> User approve SPEC duluan, baru AI implement.

---

## 🔄 Spec-First Workflow

```
USER menulis SPEC      →       AI review & clarify      →       USER approve
     ↓                         ↓                                  ↓
 FEATURE_MAP.md         AI tanya clarifications           USER comment/approve
                              ↓                                  ↓
                      AI wait for approval          AI implement dari SPEC
                                                            ↓
                                                    USER review diff\n                                                            ↓\n                                                    JALANKAN TESTING
```

### Aturan untuk AI:
1. Baca FEATURE_MAP.md sebelum coding
2. Kalau ada feature baru/wubah, tanya USER dulu:
   - "Ada SPEC untuk fitur ini?" → kalau belum, buat draft SPEC
   - "SPEC ini sudah diapprove?" → kalau belum, jangan implement
3. Ambiguitas? → Tanya clarification dulu
4. **Setiap feature baru WAJIB punya plan dulu**:
   - Buat file plan di `docs/plans/<YYYY-MM-DD>-<feature-name>.md`
   - Plan harus bite-sized, copy-pasteable, dan siap dieksekusi oleh subagent
   - Jangan coding sebelum plan selesai di-review/approve
5. Implementasi selesai? → Update kolom `Implementation` di tabel
6. **Testing Wajib** — Setiap kali ada *perubahan*, *tambah fungsi*, atau *hapus fungsi*, JALANKAN TEST sebelum menyelesaikan task:
   - `make check` (untuk menjalankan linter, build, dan semua test)
   - Atau `go test ./apps/umkm/... -v` (untuk test spesifik)
7. Setelah selesai → Update kembali `FEATURE_MAP.md` (status + testing result)

---

## 📋 Feature Specifications

Format per feature:
```markdown
### FXXX: [Nama Feature]

**Spec Status:** ✅ Approved | 🔍 In Review | ✅ Approved | ❌ Rejected
**Implementation:** ✅ Done | 🔨 In Progress | ✅ Done | ❌ Cancelled

**Deskripsi:** Apa yang fitur ini lakukan

**Spec:**
- Bullet point spesifikasi detail
- Include business rules
- Include validasi yang perlu

**Acceptance Criteria (AC):**
- [ ] AC-1: Kriteria yang bisa diverifikasi
- [ ] AC-2: User bisa test apakah fitur jalan

**Files yang perlu diubah:**
- `path/to/file.go` — deskripsi perubahan

**Notes:**
- Catatan implementasi jika ada
```

---

## 📊 Feature Registry

| ID | Feature | Spec Status | Implementation | Last Updated |
|:---|:--------|:------------|:---------------|:-------------|
| F001 | Multi-Store Quota | ✅ Approved | ✅ Done | 2026-06-12 |
| F002 | Voucher Link Subscription | ✅ Approved | ✅ Done | 2026-06-12 |
| F003 | Subscription Hold Worker | ✅ Approved | ✅ Done | 2026-06-12 |
| F004 | Read-only Enforcement (Frozen) | ✅ Approved | ✅ Done | 2026-06-12 |
| F005 | Superadmin Dashboard | ✅ Approved | ✅ Done | 2026-06-12 |
| F006 | Multi-Tenant WA Session Pool | ✅ Approved | ✅ Done | 2026-06-01 |
| F007 | Chatbot with RAG | ✅ Approved | ✅ Done | 2026-06-01 |
| F008 | Escalation to Chatwoot | ✅ Approved | ✅ Done | 2026-06-01 |
| F009 | N8N Queue Mode Automation | ✅ Approved | ✅ Done | 2026-06-01 |
| F010 | Campaign Volunteer Management | ✅ Approved | ✅ Done | 2026-06-12 |
| F011 | Campaign Voter Onboarding | ✅ Approved | ✅ Done | 2026-06-12 |
| F012 | Sidebar Navigation UI | ✅ Approved | ✅ Done | 2026-06-12 |
| F013 | N8N Integration via Super Admin | ❌ Removed | — | — |
| F014 | Flexible LLM Model System | ✅ Approved | ✅ Done | 2026-06-12 |
| F015 | Onboarding Activation Flow | ✅ Approved | ✅ Done | 2026-06-13 (UI: 2026-06-14) |
| F016 | Hybrid WhatsApp (Cloud API + whatsmeow) | ✅ Approved | ✅ Done | 2026-06-13 |
| F017 | OTP 1-Hour Reuse Window | ✅ Approved | ✅ Done | 2026-06-13 |
| F018 | Telegram Auth (Register & Login) | ✅ Approved | ✅ Done | 2026-06-13 |
| F019 | Onboarding Sync via /me (Fix Lite Tier) | ✅ Approved | ✅ Done | 2026-06-14 |
| F020 | AI CS Setup Wizard (Per-Tenant Config UI) | ✅ Approved | ✅ Done | 2026-06-14 |
| F021 | Cash Flow PDF Export | ✅ Approved | ✅ Done | 2026-06-14 |
| F022 | Excel/Google Sheet Import & Export | ✅ Approved | ✅ Done | 2026-06-14 |
| F023 | FAQ Bot AI — Edit & Generate | ✅ Approved | ✅ Done | 2026-06-14 |
| F024 | Paid-Only Enforcement (Hardening) | ✅ Approved | ✅ Done | 2026-06-14 |
| F025 | Tier Restrictions Overhaul + AI Multimodal | ✅ Approved | ✅ Done (Phase 1+2) / ⏳ Pending (Phase 3) | 2026-06-14 |
| F026 | N8N Notification Webhooks & Workflows | ✅ Approved | ✅ Done | 2026-06-14 |
| F027 | Core Business Flow Fixes & Optimizations | ✅ Approved | ✅ Done | 2026-06-14 |
| F029 | Dynamic Multimodal Guardrails | ✅ Approved | ✅ Done | 2026-06-14 |
| F030 | GetPlanFeatures DB Integration | ✅ Approved | ✅ Done | 2026-06-14 |
| F031 | Campaign Anti-Double Validation | ✅ Approved | ✅ Done | 2026-06-17 |
| F032 | Modul Saksi & Real Count C1 | ✅ Approved | ✅ Done | 2026-06-17 |
| F033 | Campaign Logistics Tracking | ✅ Approved | ✅ Done | 2026-06-17 |
| F034 | Add-on Wallet & Meta API Connector | ✅ Approved | ✅ Done | 2026-06-20 |
| F035 | Discount Vouchers (Percent & Fixed) | ✅ Approved | ✅ Done | 2026-06-20 |
| F036 | Lifetime Affiliate, External Agent & Public Leaderboard | ✅ Approved | ✅ Done | 2026-06-20 |
| F037 | Dashboard Sentimen Isu Harian (AI NLP) | ✅ Approved | ✅ Done | 2026-06-17 |
| F038 | Wargame & Simulasi Kemenangan | ✅ Approved | ✅ Done | 2026-06-17 |
| F039 | Peta Kerawanan & Pelaporan Pelanggaran | ✅ Approved | ✅ Done | 2026-06-17 |
| F040 | WA Bot FAQ Panduan Kampanye (RAG) | ✅ Approved | ✅ Done | 2026-06-17 |
| F041 | Gamification & Leaderboard Relawan | ✅ Approved | ✅ Done | 2026-06-17 |
| F042 | Auto-Scan KTP (AI OCR Vision) | ✅ Approved | ✅ Done | 2026-06-17 |
| F043 | Multi-Level Election & Sainte-Laguë Simulator | ✅ Approved | ✅ Done | 2026-06-20 |
| F044 | Campaign Modular License & Payment System | ✅ Approved | ✅ Done | 2026-06-20 |
| F045 | UMKM Healthcare Clinic Queue System | ✅ Approved | ✅ Done | 2026-06-17 |
| F046 | Hierarchical Coordinator Assignment | ✅ Approved | ✅ Done | 2026-06-20 |
| F047 | Hardening Migration (F024 cleanup) | ✅ Approved | ✅ Done | 2026-06-17 |
| F048 | WA Provider Preferences & Activation Guard | ✅ Approved | ✅ Done (v2) | 2026-06-20 |
| F049 | Container Overhaul & Infrastructure Optimization | ✅ Approved | ✅ Done | 2026-06-17 |
| F050 | WCH E2E MCP Server (UI Testing & Browser Automation) | ✅ Approved | ✅ Done | 2026-06-20 |
| F051 | AI Quota Per-Modalitas (Text/Vision/Image) | ✅ Approved | ✅ Done | 2026-06-20 |
| F052 | Tier-First Feature System + Per-Tenant Addon Guard | ✅ Approved | ✅ Done | 2026-06-20 |
| F053 | Admin-Configurable Addon Pricing + Addon Purchase Flow | ✅ Approved | ✅ Done | 2026-06-22 |
| F054 | Referral System: Discount Downline + Commission Upline | ✅ Approved | ✅ Done | 2026-06-22 |
| F055 | Force Password Change (Reset Default + Wajib Ganti) | ✅ Approved | ✅ Done | 2026-06-20 |
| F056 | Theme Management (Dark/Light/System) | ✅ Approved | ✅ Done | 2026-06-21 |
| F057 | Superadmin Feature Matrix + Addon Tier Gating | ✅ Approved | ✅ Done | 2026-06-22 |
| F058 | Wallet Payment untuk Subscription & Topup | ✅ Approved | 🔨 In Progress | 2026-06-22 |
| F059 | Landing Page — Marketing & Onboarding | ✅ Approved | ⏳ Pending | 2026-06-22 |
| F060 | Sales Dashboard Chart — Visual Penjualan | ✅ Approved | ⏳ Pending | 2026-06-22 |

## F056: Theme Management (Dark/Light/System)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Fitur perpindahan tema (Dark, Light, System Default) di UMKM frontend menggunakan CSS Variables di `:root` dan class `.theme-light`.

---

## F057: Superadmin Feature Matrix + Addon Tier Gating
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Superadmin bisa mengatur feature per tier (toggle is_enabled per plan × feature) dan addon gating (set `min_tier` — tier minimum yang dibutuhkan untuk membeli addon). Enforcement BE membaca `min_tier` saat `CanUseAddon()`.

**⚠️ KNOWN ISSUE — Dual Data Source Addon:** Ada 2 tabel addon yang tumpang tindih:
- `available_features` (F052/F057 — primary) — berisi: `ai_vision`, `ai_audio`, `wa_blast`, `extra_store`, `extra_user`
- `addon_prices` (F034 legacy) — berisi: `ai_vision`, `ai_audio_stt`, `wa_blast_api`, `wa_session_meta`
- **Overlap:** `ai_vision` ada di kedua tabel.
- **Dampak:**
  1. Addon gating, feature matrix, dan marketplace hanya baca `available_features` → addon legacy (`ai_audio_stt`, `wa_blast_api`, `wa_session_meta`) **tidak muncul** di matrix/gating/marketplace
  2. Harga bisa berbeda antara 2 sumber
- **Solusi:** Konsolidasi migration — INSERT addon legacy ke `available_features`, lalu deprecate endpoint `handleAdminAddonPrices` (arahkan superadmin UI ke available_features)

**⚠️ KNOWN ISSUE — PATCH min_tier Bug:**
Di `handleAdminAddonGating` (line 2508-2511), query UPDATE punya `WHERE min_tier IS NULL`:
```sql
UPDATE plan_features SET min_tier=$1 WHERE feature_key=$2 AND min_tier IS NULL
```
Akibatnya: jika `min_tier` sudah di-set "lite" lalu diganti "pro", query tidak mengupdate apapun karena `min_tier` tidak NULL.
**Fix:** Hapus `AND min_tier IS NULL` — UPDATE tanpa kondisi itu.

**Spec (Opsi C — Hybrid):**

### Backend Enforcement
- `CanUseAddon()` di `shared/sdk/auth/can_use.go` — baca `min_tier` dari `plan_features`, enforce tier priority sebelum check `tenant_addons`
- `tierPriority()` — helper: superadmin=100, ultimate=4, pro=3, lite=2, inactive=0

### Superadmin BE Endpoints (billing-service)
| Method | Path | Deskripsi |
|:-------|:-----|:----------|
| `GET` | `/admin/feature-matrix` | Return full matrix: plans × features (is_enabled, min_tier, feature_value) |
| `PATCH` | `/admin/feature-matrix` | Toggle `is_enabled` per plan × feature (upsert) |
| `GET` | `/admin/available-features` | List semua feature/addon registry |
| `POST` | `/admin/available-features` | Upsert feature/addon definition |
| `PATCH` | `/admin/available-features/{key}` | Partial update feature metadata |
| `DELETE` | `/admin/available-features/{key}` | Delete feature dari registry |
| `GET` | `/admin/addon-gating` | List addon + min_tier + default_enabled |
| `PATCH` | `/admin/addon-gating` | Update min_tier + default_enabled per addon |

### Konsolidasi Data Source
1. Migration baru: INSERT addon legacy (`ai_audio_stt`, `wa_blast_api`, `wa_session_meta`) dari `addon_prices` ke `available_features` (set `is_addon=true`, `addon_price_cents`, `addon_unit`)
2. Deprecate: `handleAdminAddonPrices` → redirect superadmin UI ke available_features saja
3. Opsional: tambah kolom `legacy_from TEXT` di `available_features` untuk tracking

### PATCH min_tier Fix
```diff
- UPDATE plan_features SET min_tier=$1 WHERE feature_key=$2 AND min_tier IS NULL
+ UPDATE plan_features SET min_tier=$1 WHERE feature_key=$2
```
Tanpa kondisi `AND min_tier IS NULL` — agar bisa raise maupun lower tier kapan saja.

### Frontend UI (SuperAdminDashboard.vue)
- Tombol "Feature Matrix" di header card daftar tenant
- Modal dengan tabel matrix: baris = feature keys, kolom = paket (lite/pro/ultimate)
- Toggle checkbox per cell → PATCH langsung saat diklik (optimistic update)
- Section kedua: addon gating — select `min_tier` per addon
- Sesi ended: Tutup modal → semua cache invalidation berjalan di BE

### Cache Invalidation
- Toggle feature → `cache.Client.Del("plan_features:"+planID)` + `auth.InvalidateFeatureDefCache(featureKey)`
- Update addon gating → invalidates all plan tiers + feature def cache

**Acceptance Criteria (AC):**
- [x] AC-1: `CanUseAddon()` enforce `min_tier` — tenant tier < min_tier → false (即使有tenant_addons purchase)
- [x] AC-2: Superadmin toggle feature is_enabled → instant cache invalidation
- [ ] AC-3: **Fix PATCH min_tier** — hapus `AND min_tier IS NULL` agar bisa naik/turun tier kapan saja
- [ ] AC-4: **Konsolidasi addon** — INSERT legacy addon ke `available_features`, deprecate `addon_prices` endpoints
- [x] AC-5: Feature Matrix UI — checkbox toggle per cell
- [x] AC-6: Addon Gating UI — select min_tier per addon
- [ ] AC-7: Semua addon (termasuk legacy) muncul di matrix setelah konsolidasi
- [x] AC-8: `go build ./...` clean ✅ (2026-06-21)

**Files Changed:**
- `shared/sdk/auth/can_use.go` — `tierPriority()` + `min_tier` enforcement in `CanUseAddon()`
- `services/billing-service/main.go` — 4 new handlers: `handleAdminAvailableFeaturesCollection`, `handleAdminAvailableFeaturesItem`, `handleAdminFeatureMatrix`, `handleAdminAddonGating` + 4 new routes
- `services/billing-service/main.go` — Fix PATCH min_tier (hapus `AND min_tier IS NULL`)
- `shared/migrations/NNNNNN_consolidate_addon_sources.up.sql` — INSERT legacy addon ke available_features
- `frontend/umkm-web/src/superadminApi.ts` — 7 new API methods
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Feature Matrix modal + state + logic
- `docs/FEATURE_MAP.md` — F057 entry

**Notes:**
- `available_features` registry + `plan_features` toggle matrix = clean separation between "apa yang ada" vs "siapa dapat apa"
- `min_tier` sudah ada di schema (migration 000068), enforcement baru diaktifkan di F057
- Addon gating per tier: tidak semua addon bisa dibeli oleh semua tier (misal: wa_blast hanya untuk pro+)
- **PENTING:** Setelah konsolidasi, hapus routing `handleAdminAddonPrices` dari billing-service main.go untuk mencegah confusion

---

## F058: Wallet Payment untuk Subscription & Topup

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

**Deskripsi:** Tenant bisa bayar subscription menggunakan saldo wallet (setelah topup via Xendit), bypass Xendit invoice. Juga standardisasi topup flow + referral discount integration di subscription payment.

---

### 📌 Background — State Saat Ini

```
现状 (Current):
  wallet_credits table EXISTS (migration 000055):
    tenant_id UUID PK, balance_cents BIGINT, updated_at
    → Hanya dipakai untuk addon purchase

  wallet_transactions table EXISTS:
    id, tenant_id, amount_cents, transaction_type ('topup'|'consume'),
    reference, description, created_at
    → transaction_type belum ada 'subscription'

  handleWalletTopup EXISTS (billing-service line 3951):
    POST /wallet/topup → Xendit invoice → webhook → INSERT wallet_credits

  handleSubscribe (line 577-767):
    ✅ Referral discount applied
    ✅ Voucher handling
    ❌ Wallet bypass: final_price > 0 → selalu CREATE Xendit invoice
    ❌ Tidak ada opsi "bayar dari wallet" untuk subscription

  handlePaymentWebhook (line 1020-1330):
    ✅ Deteksi wallet topup via -wallet-topup- external_id
    ✅ Overpayment → kelebihan masuk wallet
    ❌ Tidak handle subscription-via-wallet (karena handleSubscribe belum kirim wallet payment)

  Addon purchase (handlePurchaseAddon):
    ✅ Wallet deduction SUDAH
    ✅ Referral commission SUDAH
    ❌ Referral discount BELUM

问题 (Gaps):
  1. Subscription selalu lewat Xendit — tidak bisa pakai wallet
  2. Tidak ada transaksi type 'subscription' di wallet_transactions
  3. Frontend tidak ada indikator wallet balance saat checkout subscription
  4. Auto-deduct: tenant bisa milih "bayar dari wallet" otomatis tiap bulan (auto-renew pake wallet)
  5. Referral discount hanya untuk subscription Xendit — kalau wallet bypass, discount harus tetap jalan
```

---

### 🎯 Tujuan (Goals)

1. **Subscription via wallet**: Jika wallet balance cukup → bypass Xendit → deduct wallet → activateSubscription langsung
2. **Topup+Subscribe flow**: Jika balance kurang → partial pay? (opsi lanjutan). MVP: full pay dari wallet atau full Xendit.
3. **Wallet balance indicator di UI**: Saat checkout subscription, tampilkan "Saldo wallet: Rp X. Bayar via Wallet?"
4. **Auto-renew subscription via wallet**: Setiap bulan, auto-deduct wallet untuk perpanjang subscription
5. **Referral discount tetap jalan**: Baik bayar via Xendit maupun wallet, referral discount dihitung sama

---

### 📊 Data Model — Ekstensi

#### `wallet_transactions` — tambah enum `transaction_type`
```
ALTER TABLE wallet_transactions
  DROP CONSTRAINT IF EXISTS valid_transaction_type;
  
-- Sebelumnya hanya 'topup' dan 'consume'
-- Tambah: 'subscription', 'subscription_auto_renew', 'subscription_refund'
```

#### `wallet_credits` — trigger untuk auto-renew opt-in (opsional)
```
ALTER TABLE wallet_credits
  ADD COLUMN IF NOT EXISTS auto_renew_subscription BOOLEAN DEFAULT false;
  -- Jika true, sistem auto-deduct wallet setiap bulan untuk perpanjang
```

**TIDAK perlu tabel baru** — reuse existing `wallet_credits` + `wallet_transactions`.

---

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

---

### 🖥️ UI Specs

#### Checkout Subscription — wallet indicator

Di halaman pilih paket / checkout subscription (saat ini di `Subscribe.vue` atau modal activation):

```
┌─────────────────────────────────────────────┐
│  Langganan Pro                             │
│  Harga: Rp 100.000/bulan                   │
│                                             │
│  👜 Saldo Wallet: Rp 250.000               │ ← NEW
│                                             │
│  Metode Pembayaran:                         │
│  ○ Xendit (Bank Transfer / QRIS / EWallet)  │ ← default
│  ● Bayar dari Wallet (Rp 100.000)          │ ← NEW
│                                             │
│  [Langganan Sekarang]                       │
└─────────────────────────────────────────────┘
```

#### Wallet Page — subscription history

Di halaman Wallet (saat ini hanya show topup + addon consumes):

```
Tab "Transaksi":
  ├─ +Rp 100.000 — Topup via BCA (2026-06-20)
  ├─ -Rp 50.000  — Addon: Extra Store (2026-06-19)
  ├─ -Rp 100.000 — Langganan Pro (2026-06-18)      ← NEW
  └─ -Rp 100.000 — Langganan Pro (perpanjang)       ← NEW
```

---

### ✅ Acceptance Criteria (AC)

- [ ] AC-1: POST `/subscribe` dengan `pay_via_wallet=true` dan balance cukup → deduct wallet → activateSubscription → response `{status:'activated', method:'wallet'}`
- [ ] AC-2: POST `/subscribe` dengan `pay_via_wallet=true` dan balance kurang → response 402 + `{required_cents, balance_cents, topup_url}`
- [ ] AC-3: POST `/subscribe` tanpa `pay_via_wallet` (default false) → Xendit invoice seperti biasa
- [ ] AC-4: Referral discount tetap di-apply baik via wallet maupun Xendit
- [ ] AC-5: Wallet auto-renew: tenant dengan `auto_renew_via_wallet=true` → 3 hari sebelum expired → auto-deduct + extend 30 hari
- [ ] AC-6: Frontend checkout menampilkan wallet balance + opsi "Bayar dari Wallet"
- [ ] AC-7: Wallet page menampilkan transaksi subscription di history
- [ ] AC-8: `make check` pass

---

### 📁 Files to Change

**Backend:**
- `services/billing-service/main.go` — **PATCH** `handleSubscribe`: tambah param `pay_via_wallet`, logic wallet deduct + bypass Xendit
- `services/billing-service/main.go` — **PATCH** `handlePaymentWebhook`: tambah transaction_type 'subscription' untuk wallet payment
- `services/billing-service/main.go` — **NEW** `handleWalletSubscriptionAutoRenew`: cron job untuk auto-renew via wallet
- `shared/sdk/auth/quota.go` — tambah `DeductWalletBalanceForSubscription()` atau reuse existing (sudah cukup generic)
- `shared/migrations/NNNNNN_wallet_subscription.up.sql` — tambah `tenant_subscriptions.auto_renew_via_wallet BOOLEAN DEFAULT false`, perluas constraint transaction_type

**Frontend:**
- `frontend/umkm-web/src/components/Subscribe.vue` (atau modal activation) — wallet balance display + opsi pembayaran
- `frontend/umkm-web/src/components/Wallet.vue` — tambah tab subscription history
- `frontend/umkm-web/src/api.ts` — update `subscribe()` untuk kirim `pay_via_wallet` flag

---

### Notes:

- Wallet subscription adalah **opsi**, bukan kewajiban. Tenant bisa tetap pakai Xendit.
- Auto-renew flag (`auto_renew_via_wallet`) terpisah dari `auto_renew` addon — jangan campur.
- Referral discount priority: voucher → referral → final_price. Konsisten dengan F054 fix.
- Untuk MVP: wallet deduct full amount. Partial payment (wallet + Xendit) terlalu kompleks untuk sekarang— bisa jadi F059 jika diperlukan.
- Race condition: `DeductWalletBalance` sudah pakai `SELECT ... FOR UPDATE` di dalam transaksi (via `UPDATE ... WHERE balance_cents >= $1`). Aman untuk concurrent request.

---

## F059: Landing Page — Marketing & Onboarding

**Spec Status:** ✅ Approved
**Implementation:** ⏳ Pending

**Deskripsi:** Halaman publik (tanpa auth) yang menjelaskan WCH Platform, fitur, dan pricing. Calon tenant bisa melihat value proposition sebelum daftar. Route `/landing` (atau `/` untuk guest). User yang sudah login langsung redirect ke dashboard.

---

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

---

### 🖥️ UI Specs

#### Layout

```
┌─────────────────────────────────────────────────────┐
│  🏪 WCH Platform    [Fitur] [Harga] [Login] 🟢Daftar│ ← Navbar sticky
├─────────────────────────────────────────────────────┤
│                                                     │
│  Hero Section                                       │
│  ┌─────────────────────────────────────────────┐   │
│  │  Aplikasi UMKM All-in-One                   │   │
│  │  POS + Pembukuan + AI Chatbot dalam        │   │
│  │   satu platform                              │   │
│  │                                             │   │
│  │  [Mulai Gratis →] [Lihat Demo]              │   │
│  │                                             │   │
│  │  Dipakai oleh 100+ UMKM di Indonesia        │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  Features Grid (4-6 cards)                          │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐              │
│  │  POS │ │Buku  │ │  AI  │ │Lapor │              │
│  │ Kasir│ │Besar │ │Chat  │ │Keuan │              │
│  └──────┘ └──────┘ └──────┘ └──────┘              │
│                                                     │
│  Pricing Table                                      │
│  ┌──────┐ ┌──────┐ ┌──────┐                        │
│  │ Lite │ │ Pro  │ │Ultim │                        │
│  │ Gratis│ │Rp450k│ │Rp1jt │                        │
│  │ ...  │ │ ...  │ │ ...  │                        │
│  └──────┘ └──────┘ └──────┘                        │
│                                                     │
│  Footer                                             │
│  © 2026 WCH Platform                                │
└─────────────────────────────────────────────────────┘
```

#### Design Direction

- **Tone:** Professional, modern, trustworthy
- **Warna:** Dark theme (existing) untuk landing, dengan aksen hijau/teal untuk CTA
- **Font:** Pakai font existing system
- **Motion:** Subtle fade-in on scroll (CSS only, tanpa library)
- **Layout:** Asymmetric hero (text left, illustration/device mockup right)

#### Feature Cards

| Icon | Judul | Deskripsi |
|:-----|:------|:----------|
| 💰 | **Kasir POS** | Catat transaksi jual-beli dengan cepat. Dukung multi-pembayaran (tunai, QRIS, transfer). |
| 📒 | **Pembukuan Otomatis** | Double-entry accounting. Setiap transaksi POS langsung tercatat di jurnal. |
| 🤖 | **AI Customer Service** | Bot WhatsApp otomatis jawab pertanyaan pelanggan 24/7. Bisa di-training dengan FAQ toko. |
| 📊 | **Laporan Keuangan** | Laba rugi, arus kas, neraca — siap pakai untuk pajak dan pengajuan kredit. |
| 🏪 | **Multi-Toko** | Kelola beberapa cabang dalam satu akun. Cocok untuk franchise. |
| 🔗 | **Integrasi Marketplace** | (Coming soon) Hubungkan dengan GoFood, Grab, Shopee. |

#### Pricing Table

| | Lite | Pro | Ultimate |
|:--|:-----|:----|:---------|
| Harga | **Gratis** | **Rp 450K/bln** | **Rp 1.000K/bln** |
| POS | ✅ | ✅ | ✅ |
| Transaksi/bln | 100 | 10.000 | Unlimited |
| AI Chatbot | — | ✅ | ✅ |
| Laporan Keuangan | ✅ | ✅ | ✅ |
| Multi-User | 1 user | 5 user | Unlimited |
| AI Vision | — | — | ✅ |
| CTA | [Daftar Gratis] | [Mulai Trial] | [Hubungi Sales] |

#### Route & Redirect Logic

```
- GET / → LandingPage.vue (jika guest), redirect /dashboard (jika authed)
- GET /landing → LandingPage.vue (jika guest), redirect /dashboard (jika authed)
- Navbar "Login" → /login
- Navbar "Daftar" → /register
- CTA buttons:
  "Mulai Gratis" → /register
  "Lihat Demo" → scroll ke features section / modal video
```

### ✅ Acceptance Criteria (AC)

- [ ] AC-1: Guest buka `/` → lihat landing page (hero, features, pricing, footer)
- [ ] AC-2: User login buka `/` → redirect ke `/dashboard`
- [ ] AC-3: "Mulai Gratis" CTA → `/register`
- [ ] AC-4: Pricing table menampilkan 3 tier (Lite/Pro/Ultimate) — static atau dari backend
- [ ] AC-5: Responsive — mobile & desktop layout rapi
- [ ] AC-6: Dark/light mode konsisten dengan F056 theme
- [ ] AC-7: SEO meta tags (title, description, og:image)
- [ ] AC-8: `vue-tsc` clean, build pass

### 📁 Files to Change

- `frontend/umkm-web/src/components/LandingPage.vue` (NEW — ~300 baris)
- `frontend/umkm-web/src/router/index.ts` — route `/` conditional (guest → Landing, authed → Dashboard), route `/landing`
- `frontend/umkm-web/src/App.vue` — skip sidebar/layout untuk landing page (atau kondisional)

### Notes:

- **Pure frontend** — tidak perlu backend endpoint baru
- Pricing bisa static dulu. Dynamic dari backend bisa ditambahkan nanti jika diperlukan
- Landing page HARUS lightweight — no Chart.js, no heavy dependencies
- Gunakan CSS Variables existing (F056) untuk theme consistency

---

## F060: Sales Dashboard Chart — Visual Penjualan

**Spec Status:** ✅ Approved
**Implementation:** ⏳ Pending

**Deskripsi:** Ganti chart placeholder di DynamicDashboard.vue dengan chart penjualan real. Widget `daily_sales` dan `order_volume` menampilkan grafik garis/batang pendapatan harian, dengan period switcher (7 Hari / 30 Hari / 12 Bulan). Juga menambah widget "Top Products" real dan "Transaksi Terbaru" real.

---

### 📌 Background — State Saat Ini

```
现状 (Current):
  DynamicDashboard.vue (default dashboard):
    - Widget system with templates per business_type ✅
    - Chart widget (type: 'chart') renders PLACEHOLDER bars:
        <div v-for="i in 7" class="chart-bar"
             :style="{ height: (20 + Math.random() * 80) + '%' }">
      → Ini data acak, bukan real!
    - List widget (type: 'list') only shows empty state "Belum ada data"
    - Actions widget works (links to POS, Catalog, Journal)
    - Metric widgets work (income_summary, expense_summary dari income-statement API)

  Data Tersedia:
    - /api/umkm/reports/income-statement?from=&to= ✅ (aggregate)
    - /api/umkm/journal?from=&to=&limit= ✅ (list transactions)
    - Product catalog ✅
    - Chart.js library SUDAH terinstall ✅
    - Bar chart component SUDAH digunakan di Dashboard.vue (classic) ✅

  Masalah:
    - Widget chart pakai data random → gak berguna untuk user
    - Widget list (transactions) tidak fetching data real
    - Tidak ada endpoint khusus untuk sales-chart per-hari
    - Period switcher (hari/minggu/bulan/tahun) tidak ada
```

### 🎯 Tujuan

1. **Chart real:** Widget `daily_sales` dan `order_volume` menampilkan data penjualan real (bukan placeholder)
2. **Period switcher:** Tombol 7H / 30H / 12B — ganti rentang chart
3. **Top Products:** Widget best-selling products (dari transaksi POS atau journal)
4. **Recent Transactions:** Widget transaksi terbaru (dari journal entries)
5. **Backend endpoint:** `GET /api/umkm/reports/sales-chart?period=week|month|year`
6. **Classic Dashboard** juga diupdate dengan period switcher

---

### 📊 Backend: New Endpoint

#### `GET /api/umkm/reports/sales-chart?period=week|month|year`

**Response shape:**
```json
{
  "success": true,
  "data": {
    "period": "month",
    "labels": ["1 Juni", "2 Juni", ...],
    "revenue": [500000, 750000, 620000, ...],
    "expense": [200000, 300000, 250000, ...],
    "profit": [300000, 450000, 370000, ...]
  }
}
```

**SQL logic:**
```sql
-- Period = 'week' (7 hari terakhir, group by date)
SELECT DATE(created_at) as day,
       SUM(CASE WHEN type = 'credit' AND account_code ~ '^4' THEN amount ELSE 0 END) as revenue,
       SUM(CASE WHEN type = 'debit' AND account_code ~ '^5' THEN amount ELSE 0 END) as expense
FROM journal_entries
WHERE tenant_id = $1
  AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at)
ORDER BY day

-- Period = 'month' (30 hari)
-- Period = 'year' (12 bulan, GROUP BY month)
```

Atau, reuse data dari `income-statement` yang sudah ada — hitung per-hari dari data transaksi yang sudah di-aggregate.

#### `GET /api/umkm/reports/top-products?limit=5&from=&to=`

**Response:**
```json
{
  "success": true,
  "data": [
    { "name": "Nasi Goreng", "quantity": 42, "revenue_cents": 840000 },
    { "name": "Es Teh", "quantity": 38, "revenue_cents": 190000 }
  ]
}
```

**SQL logic:**
```sql
-- Dari journal_entries, filter debit ke account pendapatan (4xx)
-- Join dengan description / metadata untuk extract product name
-- Atau dari transaction_items jika ada
```

Jika tabel `transaction_items` tidak ada, gunakan pattern matching dari `journal_entries.description` (approximate).

#### `GET /api/umkm/reports/recent-transactions?limit=5`

**Response:**
```json
{
  "success": true,
  "data": [
    { "id": "...", "date": "2026-06-22", "description": "Penjualan Tunai", "amount_cents": 50000, "type": "income" }
  ]
}
```

**SQL:**
```sql
SELECT id, created_at, description, amount_cents, entry_type
FROM journal_entries
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2
```

### 🖥️ Frontend Changes

#### DynamicDashboard.vue — 3 perubahan

**1. Chart widget (type: 'chart') — render Chart.js real:**
```
<template>
  <div class="widget-chart">
    <div class="chart-header">
      <span class="widget-title">{{ widget.title }}</span>
      <div class="period-switcher" v-if="widget.id === 'daily_sales' || widget.id === 'order_volume'">
        <button :class="{ active: period === 'week' }" @click="setPeriod('week')">7H</button>
        <button :class="{ active: period === 'month' }" @click="setPeriod('month')">30H</button>
        <button :class="{ active: period === 'year' }" @click="setPeriod('year')">12B</button>
      </div>
    </div>
    <div style="height: 200px; width: 100%;">
      <Line v-if="chartReady" :data="chartData" :options="chartOptions" />
      <div v-else class="chart-loading">Memuat data...</div>
    </div>
  </div>
</template>
```

**2. List widget (type: 'list', id: 'recent_transactions') — fetch real data:**
```
GET /api/umkm/reports/recent-transactions?limit=5
→ render: description | amount | relative time
```

**3. List widget (id: 'best_selling', 'top_products') — fetch real data:**
```
GET /api/umkm/reports/top-products?limit=5
→ render: product name | qty sold | total revenue
```

**4. Metric widget enhancement:**
- Tambah loading state (skeleton)
- Tambah tooltip "vs bulan lalu" pada perubahan persen

#### Dashboard.vue (Classic) — period switcher

Classic Dashboard sudah punya Bar chart `handleCashFlow` data. Tambah period switcher:
- Tombol "Minggu Ini" / "Bulan Ini" / "Tahun Ini"
- Refetch chart data saat period berubah

### ✅ Acceptance Criteria (AC)

- [ ] AC-1: `GET /api/umkm/reports/sales-chart?period=week` → return 7 data points (per-hari)
- [ ] AC-2: `GET /api/umkm/reports/sales-chart?period=month` → return 30 data points
- [ ] AC-3: `GET /api/umkm/reports/sales-chart?period=year` → return 12 data points (per-bulan)
- [ ] AC-4: Dashboard widget `daily_sales` menampilkan Chart.js line/bar chart real
- [ ] AC-5: Period switcher (7H/30H/12B) berfungsi — ganti data chart
- [ ] AC-6: Widget `recent_transactions` menampilkan 5 transaksi terakhir real
- [ ] AC-7: Widget `top_products` menampilkan produk terlaris real
- [ ] AC-8: Loading state (skeleton/spinner) selama fetch
- [ ] AC-9: Empty state jika belum ada transaksi
- [ ] AC-10: `go build`, `go vet`, `go test`, `vue-tsc` clean

### 📁 Files to Change

**Backend:**
- `apps/umkm/accounting/main.go` — **NEW** handler `handleSalesChart` (GET /reports/sales-chart)
- `apps/umkm/accounting/main.go` — **NEW** handler `handleTopProducts` (GET /reports/top-products)
- `apps/umkm/accounting/main.go` — **NEW** handler `handleRecentTransactions` (GET /reports/recent-transactions)
- `apps/umkm/accounting/main.go` — routes untuk 3 endpoint baru

**Frontend:**
- `frontend/umkm-web/src/components/DynamicDashboard.vue` — update chart widget (Chart.js real data), update list widgets (recent transactions, top products), add period switcher + loading state + empty state
- `frontend/umkm-web/src/components/Dashboard.vue` — tambah period switcher pada classic dashboard
- `frontend/umkm-web/src/api.ts` — methods `api.getSalesChart()`, `api.getTopProducts()`, `api.getRecentTransactions()`

### Notes:

- `vue-chartjs` + `chart.js` sudah terinstall ✅ — tidak perlu npm install baru
- Endpoint `sales-chart` bisa reuse aggregasi dari `income-statement` tetapi di-breakdown per-day
- Untuk produk terlaris: jika tabel `transaction_items` tidak ada, gunakan journal entries description matching sebagai aproximasi
- Period switcher state disimpan di `ref()` lokal, tidak perlu localStorage
- Dark/light mode: Chart.js label colors harus adjust — gunakan CSS variable atau computed property yang read theme

---

## F056: Theme Management (Dark/Light/System)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Fitur perpindahan tema (Dark, Light, System Default) di UMKM frontend menggunakan CSS Variables di `:root` dan class `.theme-light`.

**Spec:**
1. Tambah CSS variables khusus untuk `.theme-light` di `frontend/umkm-web/src/assets/main.css`.
2. Buat state management/composable `useTheme()` untuk menghandle transisi state: `dark`, `light`, `system`.
3. Default `system` → deteksi OS `prefers-color-scheme`.
4. Simpan preferensi di `localStorage` (`theme-preference`).
5. Toggle di `AppSidebar.vue` atau `Settings.vue`.

**Acceptance Criteria:**
- [x] AC-1: Theme toggle ganti warna tanpa error.
- [x] AC-2: Local storage simpan `theme-preference`.
- [x] AC-3: Auto detect system mode kalau belum diset.

## F055: Force Password Change (Reset Default + Wajib Ganti)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Admin/tenant owner bisa mereset password user ke random default (8 karakter, alfanumerik, no ambiguous chars) berdasarkan username + nomor HP terdaftar. Setelah reset, user **wajib** mengubah password pada login berikutnya (flag `must_change_password=true`). Flow ini menggantikan reset-password via email untuk use case internal/tenant recovery.

**Spec:**
- **Endpoint:** `POST /auth/reset-password-default`
- **Input:** `{ "username": "...", "phoneNumber": "+62..." }`
- **Validasi:**
  - Username dan phoneNumber wajib diisi
  - Username harus terdaftar di tabel `users`
  - Nomor HP harus cocok dengan `users.phone_number`
  - Untuk keamanan: response generik — "Jika username terdaftar, password akan direset" (tidak reveal apakah username/phone valid atau tidak)
- **Backend logic:**
  1. Cari user by `username` → dapatkan `id` dan `phone_number`
  2. Bandingkan `phone_number` dengan input (constant-time safe via DB comparison)
  3. Generate random 8-char password (charset: `ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789`)
  4. Hash dengan bcrypt cost=12
  5. Update `users.password_hash` + set `must_change_password = true`
  6. Return success generik
- **Frontend flow:**
  1. `ForgotPassword.vue` — form username + nomor HP → POST ke `/auth/reset-password-default`
  2. Success → redirect ke `/login` dengan message "Password sudah direset, silakan login dan ubah password"
  3. User login → backend cek `must_change_password=true` → redirect ke `/force-password-change`
  4. `ForcePasswordChange.vue` — minta password lama (default) + password baru → POST `/auth/force-change-password`
  5. Success → clear localStorage tokens → redirect ke `/login`

**Previous Flow (email-based, DEPRECATED):**
- `POST /auth/forgot-password` dengan email → kirim token ke email → `POST /auth/reset-password` dengan token
- Flow ini TIDAK dihapus untuk backward compat, tapi `ForgotPassword.vue` sekarang mengarah ke flow F055

**Acceptance Criteria (AC):**
- [x] AC-1: `POST /auth/reset-password-default` dengan username + phone valid → password direset, `must_change_password=true`
- [x] AC-2: Username tidak terdaftar → response generik "Jika username terdaftar..." (tidak error 404)
- [x] AC-3: Phone number tidak cocok → response generik yang sama (tidak reveal info)
- [x] AC-4: Password default: 8 karakter, charset unambiguous, bcrypt cost=12
- [x] AC-5: Setelah reset, user login → redirect otomatis ke `/force-password-change`
- [x] AC-6: Force change: password lama (default) + baru + konfirmasi → update password, clear `must_change_password`
- [x] AC-7: Force change tanpa auth → 401
- [x] AC-8: Password baru < 8 char → reject 400
- [x] AC-9: Password baru ≠ konfirmasi → reject 400
- [x] AC-10: Route `/force-password-change` protected `requiresAuth: true`, redirect otomatis saat `must_change_password=true`
- [x] AC-11: `go build`, `go vet`, `go test` clean
- [x] AC-12: `vue-tsc` clean, frontend build pass

**Files Changed:**
- `services/auth-service/main.go` — `handleResetPasswordDefault`, `handleForceChangePassword`, `ResetPasswordDefaultRequest`
- `shared/migrations/000076_force_password_change.up.sql` — kolom `must_change_password`
- `frontend/umkm-web/src/components/ForgotPassword.vue` — form username + phone (bukan email)
- `frontend/umkm-web/src/components/ForcePasswordChange.vue` — UI force change password
- `frontend/umkm-web/src/router/index.ts` — route + guard must_change_password
- `frontend/umkm-web/src/api.ts` — `resetPasswordDefault()`, `forceChangePassword()`

---

## F049: Container Overhaul & Infrastructure Optimization
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Standarisasi penamaan container (`wch-` prefix), pengurangan duplikasi replika yang tidak perlu di environment dev, serta penambahan koneksi pool database (pgBouncer) untuk mencegah exhaustion koneksi pada arsitektur monorepo shared.

**Acceptance Criteria (AC):**
- [x] Semua container menggunakan `wch-` prefix secara eksplisit di `docker-compose.yml`.
- [x] Duplikasi replika container `wa-gateway` dan `umkm-chatbot` diturunkan menjadi 1.
- [x] Container pgbouncer ditambahkan di port 6432.
- [x] Environment variable koneksi PostgreSQL dari seluruh service diubah dari host `postgres:5432` menjadi `pgbouncer:6432`.

## F051: AI Quota Per-Modalitas (Text/Vision/Image)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Membedakan quota rate-limit untuk penggunaan AI berdasarkan modalitasnya. Saat ini seluruh request ke `ai-gateway` menghabiskan 1 pool quota yang sama, menyebabkan risiko perebutan resource antara chatbot (text) dan fitur OCR Campaign (vision).

**Tujuan:**
- Mencegah starvation quota dari layanan yang murah (text chatbot) terhadap layanan yang mahal (vision OCR, image generation).
- Menerapkan pembatasan `plan_features` secara lebih granular: `ai_text`, `ai_vision`, `ai_image`.

**Acceptance Criteria (AC):**
- [x] Database migration untuk menambahkan feature keys baru di `plan_features`.
- [x] Implementasi routing key di `ai-gateway` middleware sesuai dengan modalitas endpoint.
- [x] Redis keys quota dihitung per modalitas: `quota_counter:{tenant}:{period}:{feature}` (feature = `ai_text` / `ai_vision` / `ai_audio_stt` / `ai_audio_tts` / `image_gen` / `ai_image`).

**Files:**
- `shared/migrations/000072_ai_image_modality.up.sql` — seed `ai_image` plan_feature key + per-tier numeric limits
- `services/ai-gateway/main.go` — per-modality quota routing (`ai_text`, `ai_vision`, `ai_audio_*`, `image_gen`)
- `services/ai-gateway/image.go` — increments `ai_image` counter on image generation
- `shared/sdk/auth/quota_counter.go` — `ai_image` → `MaxImageGen` mapping
- `services/ai-gateway/f050_modality_test.go` — modality routing key assertions
- `shared/sdk/auth/quota_counter_test.go` — `ai_image` limit coverage

## F050: WCH E2E MCP Server (UI Testing & Browser Automation)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Server MCP kustom untuk otomatisasi UI browser, enabling Hermes untuk melakukan testing end-to-end (E2E) dan pengecekan UI secara langsung di environment dev (localhost).

**Tujuan:**
- Memberikan Hermes kemampuan untuk melihat/berinteraksi dengan UI.
- Mempercepat testing alur pendaftaran, konfigurasi chatbot, dan dashboard.

**Acceptance Criteria (AC):**
- [x] AC-1: Server MCP berjalan (`node infra/mcp/wch-e2e-server.js`) dan terintegrasi ke Hermes CLI.
- [x] AC-2: Implementasi tools: `e2e_navigate`, `e2e_click`, `e2e_fill`, `e2e_screenshot`, `e2e_expect_selector`.
- [x] AC-3: Flow testing: Login → Navigate `/chatbot-config` → Fill settings → Verify (contoh di README.md).

**Files:**
- `infra/mcp/wch-e2e-server.js` — Core MCP Server logic (stdio transport, single-page Playwright session).
- `infra/mcp/package.json` — Playwright + MCP SDK dependencies, `start` & `test` scripts.
- `infra/mcp/test-server.js` — Unit test (tool registry validation tanpa launch browser).
- `infra/mcp/README.md` — Quick start + AC-3 example flow.

## F033: Campaign Logistics Tracking

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem anti-bocor logistik kampanye (kaos, sembako, baliho) dari gudang pusat hingga ke rumah warga/TPS, dipantau via WhatsApp Bot dengan validasi lokasi.

**Tujuan:**
- Memastikan dana kampanye yang dibakar untuk barang fisik benar-benar sampai ke target.
- Deteksi dini jika ada koordinator wilayah yang menahan logistik.

**Acceptance Criteria (AC):**
- [ ] AC-1: API mencatat distribusi logistik (Item, Jumlah, Penerima, Lokasi).
- [ ] AC-2: Relawan yang menerima logistik wajib setor foto *selfie* + *share location* via WA Bot, sebelum status logistik berubah jadi "Diterima".
- [ ] AC-3: Dashboard Logistik menampilkan peringatan jika distribusi terhenti di satu titik lebih dari 2 hari.

## F034: Cost-per-Vote (Campaign Accounting)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Integrasi *Accounting Engine* (yang sudah ada di UMKM) ke modul *Campaign* untuk melacak setiap Rupiah yang keluar dan membaginya dengan jumlah dukungan valid.

**Tujuan:**
- Menghitung *Cost-per-Vote* di setiap desa/kecamatan secara real-time.
- Mencegah pengeluaran kampanye di daerah yang sudah over-target (hijau).

**Spec:**
- **Database:** Tabel `campaign_expenses` (tenant_id, campaign_id, expense_category, amount, target_region_type, target_region_id, description) — migration `000074`
- **API Endpoint (`POST /campaign/expenses`):** Catat pengeluaran kampanye dengan region targeting opsional. Auto-sync ke UMKM Accounting engine.
- **Cost-per-Vote Calculation:** `total_expense / total_valid_endorsements` — dihitung real-time di `GET /campaign/finance`
- **Alert System:** `checkAndAlertCPV()` — jika CPV > Rp 200.000, tulis ke tabel `notifications` dengan type `cpv_alert`

**Acceptance Criteria (AC):**
- [x] AC-1: Aplikasi Campaign dapat memanggil API Accounting internal untuk mencatat pengeluaran kampanye.
- [x] AC-2: Perhitungan *Cost-per-Vote* = (Total Pengeluaran Daerah X) / (Total Endorsement Valid Daerah X).
- [x] AC-3: Jika Cost-per-Vote di suatu desa melampaui batas wajar (misal Rp 200.000/suara), sistem mengirimkan alert notifikasi.

**Files:**
- `shared/migrations/000074_f034_campaign_expenses.up.sql` — tabel `campaign_expenses`
- `apps/campaign/api/handlers/finance.go` — `HandleCampaignFinance` (GET CPV + POST expense), `checkAndAlertCPV()`, `syncExpenseToAccounting()`

## F035: Auto-Scan KTP (AI OCR Vision)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Relawan kirim foto KTP via WA. N8N kirim ke AI Gateway `/v1/vision` -> Ekstrak NIK, Nama, Alamat jadi JSON -> Masuk otomatis ke tabel `citizens`.
**Target:** Menghilangkan salah ketik NIK (Typo) oleh relawan.

## F036: Dashboard Sentimen Isu Harian (AI NLP)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Chat relawan dari lapangan diproses AI untuk mengekstrak kata kunci keluhan warga. Diagregrasi ke tabel `village_issues` (Desa, Isu, Sentimen -1 s/d +1).
**Target:** Bahan pidato spesifik per-desa untuk Kandidat.

## F037: Wargame & Simulasi Kemenangan (Predictive AI)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** UI Slider di Dashboard. Kalkulasi algoritma menggabungkan data `campaign_expenses` (uang dibakar) dengan rasio konversi `endorsements`. 
**Target:** Memprediksi probabilitas menang vs Cost-per-vote jika anggaran digeser ke daerah lain.

## F038: Peta Kerawanan & Pelaporan Pelanggaran
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Tabel `fraud_reports`. Relawan kirim "Share Loc" + Foto pelanggaran (Spanduk dirusak / serangan fajar lawan). Tampil sebagai titik MERAH di Heatmap UI.
**Target:** Bukti hukum siap lapor Bawaslu.

## F039: Pemilih Siluman & Anomali Detektor
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Job otomatis yang mem-flag `endorsements`. Syarat siluman: Usia > 100 thn, 1 relawan setor 500 KTP dalam 1 jam (indikasi bot/joki), kode wilayah NIK tidak cocok dengan TPS.
**Target:** Cleansing data agar kandidat tidak tertipu "Data Sampah" timses.

## F040: WA Blast Bertarget (Micro-targeting)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Filter query di Frontend (misal: "Wanita, Desa A, Pekerjaan Petani") -> Lempar payload ke N8N / WA Gateway untuk *bulk send*.
**Target:** Efisiensi kuota WA, pesan kampanye super personal.

**Acceptance Criteria:**
- [x] AC-1: `POST /blast/target` dengan filter `village_id`, `gender` (L/P), `age_range` (e.g. "18-25", "60+").
- [x] AC-2: Exclude anomaly-flagged endorsements (`is_anomaly = TRUE`).
- [x] AC-3: Return filtered phone list + `target_count`.

**Files:**
- `apps/campaign/api/handlers/blast.go` — `HandleBlastTarget` + `parseAgeRange`.
- `apps/campaign/api/handlers/blast_test.go` — Unit test for age range parser.

## F041: Gamification & Leaderboard Relawan
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Query agregat `COUNT(endorsements) GROUP BY recruiter_id`. Tampil di UI. Bot WA otomatis kirim ranking ke relawan tiap minggu.
**Target:** Memacu kompetisi antar relawan lapangan.

## F042: WA Bot FAQ Panduan Kampanye (RAG)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Dokumen visi-misi paslon di-vectorize (pgvector `embeddings`). Jika warga/relawan tanya via WA, AI Gateway cari jawaban berbasis dokumen (RAG).
**Target:** Relawan lapangan selalu punya contekan cerdas.

**Acceptance Criteria:**
- [x] AC-1: `POST /bot/faq` dengan `question` → embed via AI Gateway `/v1/embeddings` → cosine similarity search di `vector_embeddings` (top-K=3).
- [x] AC-2: pgvector HNSW index untuk fast similarity.
- [x] AC-3: Fallback ke ILIKE keyword search kalau AI Gateway unavailable.
- [x] AC-4: Return `sources` (content + similarity) untuk transparansi.

**Files:**
- `shared/migrations/000071_campaign_rag_documents.{up,down}.sql` — `campaign_documents` table.
- `apps/campaign/api/handlers/faq.go` — `HandleBotFAQ` + `embedQuestion` + `vectorSearch` + `keywordSearch` + `synthesizeFallbackAnswer`.

## F043: Multi-Level Election & Sainte-Laguë Simulator
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Upgrade Campaign Engine agar mendukung pemilihan Legislatif (Pileg DPR/DPRD) dan DPD dengan kalkulasi perolehan kursi Sainte-Laguë yang realistik dan penanganan multi-dapil dalam satu dashboard.
**Target:** Menghitung probabilitas perolehan kursi real-time berdasarkan sisa suara & divisor Sainte-Laguë.

**Acceptance Criteria:**
- [x] AC-1: Sainte-Laguë divisor sequence (1, 3, 5, 7, ...) untuk seat allocation.
- [x] AC-2: Parliamentary threshold 4% per UU Pileg — parties di bawah threshold excluded.
- [x] AC-3: Multi-dapil dashboard: `GET /wargame/sainte-lague` tanpa `dapil_id` → list all tenant dapils.
- [x] AC-4: Single dapil detail: `GET /wargame/sainte-lague?dapil_id=...` → seat allocations + party standings (vote_share %, above_threshold bool).
- [x] AC-5: Final standings sorted by seats DESC, lalu votes DESC.

**Files:**
- `apps/campaign/api/handlers/pileg.go` — `HandleSainteLague` + `simulateAllDapils` + `simulateSingleDapil`.

## F044: Campaign Modular License & Payment System
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Monetisasi fitur Campaign via kombinasi Self-Service Payment Gateway (Xendit) untuk pembelian instan dan Manual License Key (Superadmin-generated) untuk transaksi B2B custom pricing.

**Acceptance Criteria:**
- [x] AC-1: `POST /billing/checkout` — generate mock Xendit invoice untuk `wargame_token` atau `intelligence_pack`.
- [x] AC-2: `POST /billing/webhook` — Xendit callback, idempotent (race-safe via `SELECT FOR UPDATE`), credit tokens / addons to campaign.
- [x] AC-3: Affiliate commission on PAID (referral config-driven rate, default 10%).
- [x] AC-4: `POST /superadmin/licenses/generate` — manual B2B license key.
- [x] AC-5: `GET /superadmin/licenses?used=&limit=` — list all licenses with usage status.
- [x] AC-6: `POST /licenses/redeem` — tenant burns license, atomic via tx.
- [x] AC-7: `GET /licenses/active?campaign_id=...` — return election_type, max_voters, wargame_tokens, active_addons per campaign.

**Files:**
- `apps/campaign/api/handlers/billing.go` — `HandleBillingCheckout` + `HandleBillingWebhook`.
- `apps/campaign/api/handlers/license.go` — `HandleSuperadminGenerateLicense` + `HandleRedeemLicense` + `HandleListLicenses` + `HandleTenantActiveAddons`.
- `apps/campaign/api/main.go` — routes registered.

## F045: UMKM Healthcare Clinic Queue System
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Modul reservasi antrian buat klinik UMKM. Sistem memberikan nomor antrian otomatis dan melakukan reminder via N8N WA Gateway.
**Fitur:** Backend (settings, book, cancel, queue, call) + N8N Workflows (booking_bot, reminder).

**Acceptance Criteria (AC):**
- [x] AC-1: Reservasi antrian (Book) dengan slot dinamis & validasi tipe antrian (Sequential/Timeslot).
- [x] AC-2: Pembatalan antrian via WA Bot.
- [x] AC-3: Notifikasi otomatis 1 jam sebelum jadwal via N8N WA Gateway.
- [x] AC-4: Call nomor antrian (Atomic increment counter).

**Files:**
- `apps/umkm/accounting/main.go` — handlers: `handleClinicBook`, `handleClinicCancel`, `handleClinicQueue`, `handleClinicCall`.
- `apps/umkm/accounting/clinic_middleware.go` — `requireClinicType` check.
- `apps/umkm/accounting/clinic_test.go` — Unit tests.


## F001: Webhook Subscription

## F027: Core Business Flow Fixes & Optimizations

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Perbaikan logika bisnis utama hasil audit keamanan dan flow transaksi untuk mencegah kerugian perusahaan dan tenant.

### Sub-Tasks:
1. **Accounting Hard-Delete Fix**:
   - `DELETE /api/umkm/accounts/{id}` harus memblokir penghapusan jika akun masih memiliki `journal_entries` atau balance `!= 0`. Kembalikan HTTP 400.
2. **Chatbot Instant Escalation**:
   - Deteksi fallback response di `apps/umkm/chatbot/main.go`. Jika AI mengeluarkan pesan fallback, langsung *trigger* escalation ke *owner* secara instan, bypass `AutoEscalateAfterMinutes`.
3. **AI Quota Race-Condition Fix**:
   - Modifikasi `QuotaMiddlewareFeature` di `shared/sdk/auth/quota_mw.go`. Lakukan check/increment kuota **SEBELUM** meneruskan ke handler API. Jika kuota habis, kembalikan `402 Payment Required` tanpa memanggil handler (vendor API).
4. **Billing Proration (Sisa Hari)**:
   - Modifikasi `activateSubscription` di `services/billing-service/main.go`. Hitung sisa hari dari subscription sebelumnya yang masih aktif. Berikan kompensasi (tambahan waktu atau konversi) pada subscription yang baru, atau setidaknya tidak menghilangkan masa aktif paket lama jika tidak prorata (contoh: masa aktif paket baru = hari ini + 30 + sisa hari paket lama yang sepadan/di-scale down, atau cukup tambahkan secara kasar).

**Acceptance Criteria (AC):**
- [x] AC-1: Accounting hard-delete block jika akun punya journal entries atau balance != 0
- [x] AC-2: Chatbot instant escalation on fallback via goroutine (bypass AutoEscalateAfterMinutes)
- [x] AC-3: AI Quota check/increment BEFORE handler via QuotaMiddlewareFeature middleware
- [x] AC-4: Billing proration — remaining days dari subscription lama ditambahkan ke voucher baru

---

## F026: N8N Notification Webhooks & Workflows

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Integrasi `services/notification-service` dengan *N8N Workflow Automation* untuk pengiriman pesan asinkron (WA, Email, Telegram). Membuka jalan untuk scheduled reports (UMKM) dan real-time alerts.

### Goals
1. Bikin standard REST API webhook endpoint di `notification-service` agar bisa dipanggil N8N.
2. Standardisasi format JSON payload untuk N8N ke WCH Platform.
3. Template Engine basic: Parsing variabel (`{{customer_name}}`, `{{total}}`) dari payload N8N.
4. Export N8N JSON workflow templates (contoh: Monthly Financial Report, Low Stock Alert) ke direktori `infra/n8n/workflows`.

### Rules & Constraints
- `notification-service` HANYA menerima HTTP request; eksekusi scheduling (cron) murni dari N8N.
- Security: Gunakan Header `X-Webhook-Secret` atau JWT token (superadmin/system level) pada N8N webhook call.
- Jangan hardcode provider (WA/Email) API. Hubungkan endpoint `/notify/whatsapp` ke `services/wa-gateway` (bukan external vendor Fonnte, since we have our own gateway).

### Impacted Files
- `services/notification-service/main.go`
- `services/notification-service/templates.go` (NEW)
- `shared/sdk/webhook/auth.go` (NEW/Update - Validate incoming N8N webhook signatures)
- `infra/n8n/workflows/umkm_low_stock_alert.json` (NEW)
- `infra/n8n/workflows/umkm_monthly_report.json` (NEW)

### Deployment
- Membutuhkan instance n8n berjalan di Docker network yang sama.

---

## F022: Excel/Google Sheet Import & Export

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** UMKM bisa export data ke Excel/CSV (untuk backup, akuntan, laporan pajak) dan import data dari spreadsheet ke aplikasi. Mendukung **3 entity**: journal_entries (transaksi), products (katalog), dan contacts (pelanggan/forwarders). Format file: `.xlsx` (Excel) dan `.csv` (Google Sheet export friendly).

**Spec:**

### Backend (apps/umkm/accounting)

**Export endpoints** (returns file blob with proper Content-Disposition):

| Method | Path | Format | Entity |
|:-------|:-----|:-------|:-------|
| `GET /export/journal?from=YYYY-MM-DD&to=YYYY-MM-DD&format=xlsx|csv` | download | Journal entries (header + lines) |
| `GET /export/products?format=xlsx|csv` | download | Product catalog |
| `GET /export/contacts?format=xlsx|csv` | download | Customer/forwarder contacts |

**Import endpoints** (multipart/form-data, field name `file`):

| Method | Path | Format | Entity |
|:-------|:-----|:-------|:-------|
| `POST /import/products` | xlsx/csv | Upsert products by SKU |
| `POST /import/contacts` | xlsx/csv | Upsert contacts by phone |
| `POST /import/journal` | xlsx/csv | Create journal entries (validate balanced) |

**Response shape import:**
```json
{
  "success": true,
  "data": {
    "imported": 42,
    "skipped": 3,
    "errors": [
      { "row": 5, "error": "SKU kosong" },
      { "row": 12, "error": "Harga tidak valid" }
    ]
  }
}
```

**CSV column spec (header row required):**

**products** (`name, sku, category, price_cents, stock, description, image_url`)
- `price_cents` integer (sen). Comma-as-thousand-separator NOT supported; user harus convert.
- `stock` integer. Default 0.
- `image_url` optional.

**contacts** (`name, phone, email, role, notes`)
- `phone` wajib, unique per tenant
- `role` ∈ {`customer`, `forwarder`, `supplier`}. Default `customer`.
- `email` optional.

**journal** (`date, description, reference, debit_account_code, credit_account_code, amount_cents`)
- Single-line entry per row (debit & credit on same row)
- For multi-line entry: split into multiple rows with same `reference` (UUID auto-generated per batch, same `reference` = same entry)
- `amount_cents` integer
- Validate balanced per `reference` (sum debit == sum credit)

**XLSX support:**
- 1 sheet per file
- Header row in row 1
- Date cells as Excel date (auto-parse)
- Money cells as number

**Limits:**
- Max 5000 rows per import
- Max file size 10 MB
- File extension whitelist: `.xlsx`, `.csv`

### Frontend

**UI entry points:**
- Sidebar menu baru: `Operasi` group → "Impor / Ekspor Data" (icon: 📥) → route `/data-transfer`
- Atau dari `ProductCatalog.vue` → tombol "Export" + "Import" (inline, per entity)

**Page `DataTransfer.vue`:**
- 3 tab: Jurnal, Produk, Kontak
- Tiap tab:
  - Tombol "Download Template" (CSV + XLSX)
  - Tombol "Export Data" (filter tanggal untuk jurnal)
  - Drop zone untuk upload file + tombol "Impor"
  - Preview hasil import (table) sebelum confirm
  - Toast: imported/skipped/errors count

**Inline di ProductCatalog.vue:**
- Tombol "📥 Import" → file picker → preview → confirm
- Tombol "📤 Export" → dropdown xlsx/csv

### Acceptance Criteria (AC):
- [x] AC-1: GET `/export/products?format=xlsx` return file .xlsx valid
- [x] AC-2: GET `/export/products?format=csv` return file .csv valid
- [x] AC-3: GET `/export/journal?from&to` return file dengan multiple baris per entry
- [x] AC-4: GET `/export/contacts` return file customer + forwarder
- [x] AC-5: POST `/import/products` (xlsx/csv) → upsert by SKU, response include imported/skipped/errors
- [x] AC-6: POST `/import/contacts` (xlsx/csv) → upsert by phone
- [x] AC-7: POST `/import/journal` (xlsx/csv) → create entries, validate balanced per reference
- [x] AC-8: Frontend `DataTransfer.vue` page dengan 3 tab + download template
- [x] AC-9: Inline Import/Export di `ProductCatalog.vue`
- [x] AC-10: Validasi 5000 row max, 10MB max, ext whitelist
- [x] AC-11: `go build`, `go vet`, `go test`, `vue-tsc` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — 6 handlers (3 export, 3 import) + helper parseCSV/parseXLSX
- `shared/sdk/xlsx/` — package baru: `reader.go` (read xlsx), `writer.go` (write xlsx), `csv.go` (CSV helpers)
- `frontend/umkm-web/src/api.ts` — methods `api.exportProducts/Contacts/Journal`, `api.importProducts/Contacts/Journal`, `api.downloadTemplate`
- `frontend/umkm-web/src/components/DataTransfer.vue` — page baru
- `frontend/umkm-web/src/components/ProductCatalog.vue` — inline Import/Export
- `frontend/umkm-web/src/router/index.ts` — route `/data-transfer`
- `frontend/umkm-web/src/config/menu.ts` — menu "Impor / Ekspor Data"

### Notes:
- XLSX pakai library `github.com/jung-kurt/gofpdf` untuk write PDF, dan `github.com/xuri/excelize/v2` untuk read/write xlsx (F021 reuse dependency).
- Import = upsert (SKU/phone sebagai natural key), bukan replace. User bisa re-import untuk update.
- Untuk journal import, file CSV/XLSX diasumsikan valid — backend validasi balance + account existence.
- Bukti audit: setiap import catat ke `import_logs` (insert via mini-migration F022b, atau reuse `subscription_tickets` table sebagai quick win).

---

## F023: FAQ Bot AI — Edit & Generate

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Tenant bisa mengelola FAQ Bot AI di halaman Settings. Setiap item FAQ bisa ditambah manual, di-generate otomatis oleh AI, diedit secara inline, dan dihapus.

**Spec:**

### Backend (`apps/umkm/accounting/main.go`)

| Method | Path | Deskripsi |
|:-------|:-----|:----------|
| `GET` | `/api/umkm/faqs` | List semua FAQ tenant |
| `POST` | `/api/umkm/faqs` | Tambah FAQ baru (`{question, answer}`) |
| `PUT` | `/api/umkm/faqs` | Edit FAQ existing (`{id, question, answer}`) |
| `DELETE` | `/api/umkm/faqs?id=...` | Hapus FAQ |
| `POST` | `/api/umkm/faqs/generate` | AI generate 3 FAQ otomatis via LLM |

### Frontend (`frontend/umkm-web/src/components/Settings.vue`)

- **FAQ list:** Tampilkan semua FAQ dengan Q/A + tombol Edit & Hapus per item
- **Inline edit:** Klik Edit → input fields muncul menggantikan display text → Save / Cancel
- **AI generate:** Tombol "✨ Generate Otomatis" → panggil AI Gateway → FAQ langsung muncul di list
- **Manual add:** Input question + answer di bagian bawah → Add

**Acceptance Criteria:**
- [x] AC-1: FAQ bisa ditambah manual
- [x] AC-2: AI generate FAQ otomatis berdasarkan nama toko
- [x] AC-3: FAQ hasil generate bisa diedit inline
- [x] AC-4: FAQ bisa dihapus
- [x] AC-5: PUT `/api/umkm/faqs` menerima `{id, question, answer}` untuk edit

**Files:**
- `apps/umkm/accounting/main.go` — `handleFaqs` (PUT handler)
- `frontend/umkm-web/src/components/Settings.vue` — inline edit mode
- `frontend/umkm-web/src/api.ts` — `api.put('/api/umkm/faqs', ...)`

---

## 🐛 Bug Fixes (2026-06-14 Session)

| # | Bug | Root Cause | Fix | File |
|:--|:----|:-----------|:----|:-----|
| 1 | `GET /transactions` → 500 | `journal_entries.metadata` column missing | Migration 000033: add `metadata JSONB` | `shared/migrations/000033_*` |
| 2 | `GET /settings` → 500 | Query SELECT `wa_provider` — column not in `tenants` table | Remove all `wa_provider` references from query, variable, response | `apps/umkm/accounting/main.go` |
| 3 | Frontend `/settings` → 401 | Nginx drops `Authorization` and `X-Tenant-ID` headers | Add `proxy_set_header` for both headers in nginx.conf | `frontend/umkm-web/nginx.conf` |
| 4 | `POST /api/wa/status` → 404 | API Gateway `StripPrefix("/api/wa")` strips path before proxy to wa-gateway | Remove `http.StripPrefix` from wa-gateway proxy | `services/api-gateway/main.go` |
| 5 | 403 "Fitur Chatbot memerlukan paket Lite" for lite/superadmin tenants | `GetTenantPlan()` reads Redis `tenant:plan:{id}` — never populated by login. Fallback ke tier tanpa akses → `HasChatbot: false` | Add `"superadmin"` to `Plans` map + populate Redis cache on login | `shared/sdk/auth/quota.go`, `services/auth-service/main.go` |
| 6 | `ERR_CONNECTION_REFUSED` port 8202 | WA Gateway service not running | Start wa-gateway service | `services/wa-gateway` |
| 7 | Port docs mismatch (8212 vs 8202) | CLAUDE.md port registry had wrong WA Gateway port | Update port registry in CLAUDE.md | `CLAUDE.md` |
| 8 | `bin/` binaries tracked in git | Binaries committed before `.gitignore` rule added | `git rm --cached` binaries, `.gitignore` already correct | `.gitignore` |

---

## F021: Cash Flow PDF Export

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Selesaikan Fase 5 PRD — generate PDF Laporan Arus Kas (Cash Flow Statement) untuk periode yang dipilih. Data sudah tersedia via endpoint JSON `GET /reports/cash-flow`; fitur ini tinggal membungkus output jadi file PDF yang siap download/cetak/share ke akuntan atau bank untuk pengajuan kredit. PDF patuh pada layout SAK-EMKM: header (identitas toko + periode), 3 section aktivitas (Operasional, Investasi, Pendanaan), summary net cash flow, footer (tanggal generate + signature line).

**Spec:**

### Backend (apps/umkm/accounting)

Endpoint baru:
- `GET /reports/cash-flow/pdf?from=YYYY-MM-DD&to=YYYY-MM-DD` — return `application/pdf` blob.

**PDF Layout (A4 portrait, margins 2cm):**

```
┌────────────────────────────────────────────────────────┐
│  [NAMA TOKO]                                           │
│  Laporan Arus Kas                                      │
│  Periode: 1 Januari 2026 – 31 Januari 2026            │
│  Dicetak: 14 Juni 2026 01:44 WITA                      │
├────────────────────────────────────────────────────────┤
│  I. ARUS KAS DARI AKTIVITAS OPERASIONAL                │
│    Kas Masuk:                                          │
│      Penjualan Tunai        Rp    5.000.000            │
│      Piutang Tertagih       Rp    2.000.000            │
│      Pendapatan Lain         Rp      500.000           │
│    Total Kas Masuk          Rp    7.500.000            │
│    Kas Keluar:                                          │
│      Beban Gaji              Rp   2.000.000            │
│      Beban Listrik           Rp     300.000            │
│      Beban Bahan Baku        Rp   1.500.000            │
│    Total Kas Keluar          Rp   3.800.000            │
│    Arus Kas Operasional     Rp   3.700.000            │
├────────────────────────────────────────────────────────┤
│  II. ARUS KAS DARI AKTIVITAS INVESTASI                │
│    Pembelian Aset           Rp (1.000.000)             │
│    Arus Kas Investasi       Rp (1.000.000)             │
├────────────────────────────────────────────────────────┤
│  III. ARUS KAS DARI AKTIVITAS PENDANAAN               │
│    Setor Modal               Rp 5.000.000              │
│    Arus Kas Pendanaan        Rp 5.000.000              │
├────────────────────────────────────────────────────────┤
│  KENAIKAN/(PENURUNAN) BERSIH KAS   Rp 7.700.000        │
│  Kas Awal Periode              Rp   X.XXX.XXX          │
│  Kas Akhir Periode             Rp X.XXX.XXX + net     │
├────────────────────────────────────────────────────────┤
│  Halaman 1 dari 1                                      │
│  Generated by WCH Platform • core_project              │
└────────────────────────────────────────────────────────┘
```

**Activity classification logic (berdasarkan account_type + account_code):**

| Category | Rule |
|:---------|:-----|
| Operating Inflow | debit ke cash account (100/101) DAN line counterpart = revenue/piutang (400/120) |
| Operating Outflow | credit ke cash account (100/101) DAN line counterpart = expense/beban (500-599) atau persediaan (130) |
| Investing | counterpart account = fixed asset (150-199) |
| Financing | counterpart account = modal (300), hutang (200-299), prive (310) |

Aturan disederhanakan: kalau counterpart account code di range tertentu → masuk kategori tsb. Sisanya → operating.

**Currency formatting:** IDR dengan format `Rp 1.234.567` (tanpa desimal, pakai titik sebagai thousand separator, tanpa sen — UMKM style).

**Library:** `github.com/jung-kurt/gofpdf` v1.16.2 (UTF-8 ready, lightweight, no native deps).

**Query enhancement:** Extend `handleCashFlow` untuk return per-line breakdown by counterpart account, lalu handler PDF build sectioned report.

### Frontend

**Journal.vue** (Laporan Keuangan section):
- Tambah tombol "📄 Download PDF" di samping tombol "Filter" untuk tab Arus Kas
- Klik → set `window.location = API_BASE + '/api/umkm/reports/cash-flow/pdf?from=...&to=...'`
- Loading state saat generate (PDF bisa 1-2 detik untuk data besar)

### Acceptance Criteria (AC):
- [x] AC-1: GET `/reports/cash-flow/pdf?from&to` return PDF binary (Content-Type: application/pdf)
- [x] AC-2: PDF berisi header (nama toko, periode, tanggal cetak)
- [x] AC-3: PDF punya 3 section (Operasional, Investasi, Pendanaan) + net cash flow + kas awal/akhir
- [x] AC-4: Currency di-format `Rp X.XXX.XXX`
- [x] AC-5: Query extend: return per-counterpart breakdown
- [x] AC-6: Frontend Journal.vue tab Arus Kas punya tombol Download PDF
- [x] AC-7: `go build`, `go vet`, `go test`, `vue-tsc` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — handler `handleCashFlowPDF` + extend `handleCashFlow` (return per-line counterpart data)
- `frontend/umkm-web/src/components/Journal.vue` — tombol Download PDF di tab Arus Kas

### Notes:
- PDF generation synchronous & on-the-fly (tidak ada background job). Untuk data <500 lines latency <500ms. Jika nanti jadi lambat, bisa di-caching.
- `gofpdf` dipakai juga di F022 (Excel tidak, pakai `xuri/excelize/v2`).
- Library `gofpdf` sudah di-pull di F021.
- Indonesian UMKM style: pakai kata "Beban", "Kas Masuk", "Arus Kas", bukan "Expense", "Cash Inflow", "Cash Flow".

---

## F020: AI CS Setup Wizard (Per-Tenant Chatbot Config UI)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Wizard UI untuk owner UMKM setup AI Customer Service mereka sendiri — tanpa coding. Owner bisa atur kepribadian bot, bahasa, jam operasional, kapan bot eskalasi ke admin, dan kalimat sapa/pesan di luar jam. Data disimpan di tabel `tenant_chatbot_configs` (sudah ada di migration 000029) dan di-load oleh chatbot service saat melayani customer. Skenario: Owner daftar → onboarding selesai → masuk wizard ini → 3 langkah simpel → AI CS langsung aktif dengan kepribadian sesuai toko.

**Spec:**

### Backend (apps/umkm/accounting)

Endpoint baru di `apps/umkm/accounting/main.go` (sesuai FEATURE_MAP → lokasi UMKM Accounting):

| Method | Path | Deskripsi |
|:-------|:-----|:----------|
| `GET /api/umkm/chatbot/config` | Ambil konfigurasi chatbot tenant. Auto-create default row jika belum ada (idempotent). |
| `PUT /api/umkm/chatbot/config` | Update konfigurasi. Partial update — hanya field yang dikirim yang di-update. |
| `POST /api/umkm/chatbot/config/test` | Kirim pesan test pakai konfigurasi saat ini (preview — panggil AI Gateway dengan system_prompt yang sudah di-render). Body: `{ "message": "..." }`. Return: `{ "reply": "...", "would_escalate": bool }`. |

**Validation rules:**
- `language` ∈ {`id`, `en`}
- `tone` ∈ {`friendly`, `formal`, `casual`, `professional`}
- `temperature` ∈ [0.0, 1.0]
- `max_tokens` ∈ [64, 4096]
- `max_context_messages` ∈ [1, 50]
- `rag_top_k` ∈ [1, 20]
- `rag_similarity_threshold` ∈ [0.0, 1.0]
- `business_hours_start` < `business_hours_end` (jika sama → reject)
- `business_days` ⊆ {0,1,2,3,4,5,6}
- `escalation_keywords` non-empty jika `escalation_enabled = true`
- `channels_enabled` non-empty (minimal 1 channel)

**Response shape (GET):**
```json
{
  "success": true,
  "data": {
    "llm_provider": "minimax",
    "llm_model": "MiniMax-M2.7",
    "temperature": 0.7,
    "max_tokens": 1024,
    "tone": "friendly",
    "language": "id",
    "max_context_messages": 10,
    "welcome_message": "Halo! Ada yang bisa saya bantu?",
    "fallback_message": "Maaf, saya belum bisa menjawab...",
    "outside_hours_message": "Terima kasih telah menghubungi...",
    "business_hours_start": "08:00",
    "business_hours_end": "22:00",
    "business_days": [1,2,3,4,5,6],
    "escalation_enabled": true,
    "escalation_keywords": ["bicara cs","hubungi admin","operator"],
    "auto_escalate_after_minutes": 5,
    "rag_enabled": true,
    "rag_top_k": 5,
    "rag_similarity_threshold": 0.7,
    "channels_enabled": ["whatsapp"],
    "is_active": true
  }
}
```

### Chatbot integration (apps/umkm/chatbot)

Update `buildSystemPrompt()` di `apps/umkm/chatbot/main.go`:
- Tambah HTTP call ke `accountingURL + "/api/umkm/chatbot/config"` (header `X-Tenant-ID`).
- Cache hasil di Redis dengan key `chatbot:config:{tenant_id}` TTL 5 menit — supaya tidak hit DB tiap chat.
- Jika config `is_active = false` → return `outside_hours_message` regardless jam.
- Honor `business_hours_start/end` + `business_days` — di luar jam → return `outside_hours_message` tanpa panggil LLM (hemat cost).
- Honor `language` → tambahkan instruksi bahasa di system prompt.
- Honor `tone` → tambahkan instruksi nada bicara.
- Honor `max_context_messages` → batasi context window yang dikirim.
- Honor `escalation_keywords` → case-insensitive substring match.
- `system_prompt` custom (jika di-set owner) → pakai itu sebagai base, override default template.

Cache invalidation: saat `PUT /config` dipanggil, chatbot auto-evict cache key (POST notification atau ev langsung via shared Redis key).

### Frontend (frontend/umkm-web)

Component baru: `src/components/ChatbotConfig.vue` (~400 baris).

**Struktur 3-step wizard dengan progress indicator:**

1. **Step 1 — Identitas Bot** (Nama, Bahasa, Tone)
   - Field: Bot name (input text, default: toko), Bahasa (radio: Indonesia/English), Tone (select: friendly/formal/casual/professional)
   - Preview panel kanan: "Bot kamu akan bicara dalam [Bahasa] dengan nada [Tone]"

2. **Step 2 — Jam Operasional & Auto-Escalation**
   - Field: Jam buka-tutup (time pickers), Hari operasional (checkbox 7 hari), Toggle escalation, Escalation keywords (tag input, default suggestions)
   - Preview: "Bot aktif Senin-Sabtu 08:00-22:00. Di luar jam, customer dapat pesan: ..."

3. **Step 3 — Kalimat & Channel**
   - Textarea: Welcome message, Fallback message, Outside hours message (3 textarea)
   - Channel toggles: WhatsApp (default ON, terkunci), Telegram (jika bot token configured), Web chat
   - Tombol "Test Bot" → modal dengan chat preview

**Navigation:**
- Tombol "Lanjut" di step 1-2, "Simpan & Aktifkan" di step 3
- Tombol "Kembali" di step 2-3
- Klik step indicator boleh loncat ke step yang sudah dikunjungi
- Progress disimpan per-step (kalau user keluar, draft tersimpan di sessionStorage)

**Entry points:**
- Setelah onboarding modal activation sukses (F015 flow) → redirect ke `/chatbot-config?first_run=1` (banner "Selamat, lengkapi setup CS AI Anda")
- Sidebar menu: Operasional → "AI CS" (icon: 🤖)
- Settings → bagian "Customer Service AI" → link "Setup/Edit"

**UX detail:**
- Toast success/error pakai pola yang sudah ada
- Loading state pakai skeleton atau spinner
- Empty state untuk fresh tenant: ilustration + CTA "Mulai Setup"
- Responsive: 1 kolom di mobile, 2 kolom (form + preview) di desktop

### Acceptance Criteria (AC):
- [x] AC-1: `GET /api/umkm/chatbot/config` return default config untuk tenant baru (auto-create)
- [x] AC-2: `PUT /api/umkm/chatbot/config` update partial fields, validasi semua constraints
- [x] AC-3: `POST /api/umkm/chatbot/config/test` panggil AI Gateway dengan system_prompt yang sudah di-render
- [x] AC-4: Chatbot `buildSystemPrompt()` baca config dari DB via accounting service, cache 5 menit
- [x] AC-5: Chatbot honor `language`, `tone`, `business_hours_*`, `escalation_keywords`, `max_context_messages`
- [x] AC-6: Di luar jam operasional → return `outside_hours_message` (skip LLM call, hemat cost)
- [x] AC-7: `is_active = false` → chatbot return `outside_hours_message` regardless jam
- [x] AC-8: Frontend `ChatbotConfig.vue` 3-step wizard, progress indicator, form validation
- [x] AC-9: Frontend panggil API real, toast feedback, simpan draft di sessionStorage
- [x] AC-10: Sidebar & Settings entry point berfungsi, banner first_run setelah onboarding
- [x] AC-11: `go build ./...`, `go vet`, `go test ./...`, `vue-tsc --noEmit` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — handler `handleChatbotConfig` (GET/PUT/POST test), SQL helpers
- `apps/umkm/chatbot/main.go` — update `buildSystemPrompt()`, Redis cache integration, business hours + escalation logic
- `frontend/umkm-web/src/components/ChatbotConfig.vue` — wizard component baru
- `frontend/umkm-web/src/api.ts` — method `api.getChatbotConfig`, `api.updateChatbotConfig`, `api.testChatbotConfig`
- `frontend/umkm-web/src/router/index.ts` — route `/chatbot-config`
- `frontend/umkm-web/src/components/AppSidebar.vue` — menu "AI CS"
- `frontend/umkm-web/src/components/Settings.vue` — link "Setup/Edit" CS AI
- `frontend/umkm-web/src/components/Onboarding.vue` — redirect ke `/chatbot-config?first_run=1` setelah activation

### Notes:
- Tabel `tenant_chatbot_configs` sudah ada lengkap dari migration 000029 (F007). Tidak perlu migration baru.
- Backend taruh di `apps/umkm/accounting` (bukan di `chatbot`) karena: (a) accounting sudah jadi hub konfigurasi tenant, (b) chatbot jadi cukup fokus ke runtime, (c) pengurangan coupling — chatbot bisa di-rebuild tanpa ganggu config storage.
- Cache 5 menit dipilih untuk keseimbangan: konfigurasi baru bisa sampai di chatbot max 5 menit (acceptable untuk setup yang jarang berubah), tapi hemat DB call.
- Untuk 'eskalasi' yang sudah ada (mark `[FORWARD_TO_ADMIN]`), logic keyword disatukan — `escalation_keywords` config menggantikan/extend keyword hardcoded.
- Tier 2 — impact langsung ke goal "UMKM bisa bikin CS AI otomatis".

---

## F019: Onboarding Sync via `/me` Endpoint (Fix Lite Tier)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sediakan endpoint `GET /me` di auth-service untuk sinkronisasi status user & tenant (`onboarding_completed`, `plan`, `role`, `is_frozen`) dari backend, dan refactor router guard frontend untuk refetch saat localStorage kosong (mis. login di device baru / cache dibersihkan). Fix redirect loop ke `/onboarding` yang sudah lama dicatat di CLAUDE.md. Sekaligus refactor hardcoded WA Gateway URL di chatbot ke config.

**Spec:**
- Endpoint baru `GET /auth/me` di auth-service (alias ringkas dari `/auth/profile` GET). Return JSON berisi `user_id`, `username`, `email`, `phone_number`, `role`, `telegram_chat_id`, `tenant_id`, `plan`, `business_type`, `is_frozen`, `onboarding_completed`.
- Route di api-gateway: `/api/me` → auth-service (dengan auth middleware + tenant rate limit).
- Field `onboarding_completed` juga ditambahkan ke response `GET /auth/profile` agar FE bisa sinkronkan via endpoint yang sudah ada.
- Frontend `router/index.ts`:
  - Tambah helper `fetchAndSyncMe()` yang cache 30s per `(tenant_id, user_id)`.
  - `beforeEach` guard: jika `token` ada tapi `onboarding_completed` missing di localStorage → panggil `fetchAndSyncMe()` untuk populate flag dari BE, baru tentukan redirect.
  - Sync `onboarding_completed`, `plan`, `role`, `subscription_status` ke localStorage/sessionStorage setelah fetch berhasil.
- Refactor `apps/umkm/chatbot/main.go`:
  - Tambah `var WAGatewayURL` + helper `waSendURL()`.
  - Ganti 3 call site hardcoded `http://wa-gateway:8202/api/wa/send` jadi `waSendURL()`.
  - Resolve order: `WA_GATEWAY_URL` env → `cfg.WhatsApp.GatewayURL` → production default.

**Acceptance Criteria (AC):**
- [x] AC-1: `GET /api/me` dengan token valid → return JSON berisi semua field yang disyaratkan
- [x] AC-2: `GET /api/me` tanpa token → 401
- [x] AC-3: `GET /api/profile` sekarang juga return `onboarding_completed`
- [x] AC-4: Reload halaman di device baru dengan localStorage kosong → FE panggil `/me` otomatis, populate flag, tidak redirect loop
- [x] AC-5: Cache `/me` 30s per (tenant_id, user_id) — tidak spam backend
- [x] AC-6: `go build ./...` & `go vet ./...` clean
- [x] AC-7: `go test ./...` all packages green
- [x] AC-8: `vue-tsc --noEmit` clean
- [x] AC-9: Chatbot `waSendURL()` honour `WA_GATEWAY_URL` env + `cfg.WhatsApp.GatewayURL`
- [x] AC-10: 0 hardcoded `wa-gateway:8202` di call site chatbot (sisanya cuma default fallback)

**Files Changed:**
- `services/auth-service/main.go` — `handleMe()` handler baru, field `onboarding_completed` di `handleProfile` GET response, route `/me`
- `services/api-gateway/main.go` — route `/api/me` → auth-service
- `frontend/umkm-web/src/api.ts` — method `api.me()`
- `frontend/umkm-web/src/router/index.ts` — `fetchAndSyncMe()` helper + updated `beforeEach` guard
- `apps/umkm/chatbot/main.go` — `WAGatewayURL` var, `waSendURL()` helper, 3 call site refactored

**Notes:**
- Lite Tier fix — menyentuh 2 service + 1 app + 2 frontend file, semua test pass.
- Branch: `fix/tier1-onboarding-loop`
- Cache 30s dipilih untuk keseimbangan antara freshness dan hemat backend call. Bisa di-tune via env nanti.
- Sinkronisasi hanya terjadi jika flag missing — happy path (user sudah onboarded + localStorage ada) tidak menambah request.

---

## F018: Telegram Auth (Register & Login via Telegram Bot)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User bisa daftar dan login via Telegram Bot. OTP dikirim melalui Telegram (bukan hanya WhatsApp), memanfaatkan bot Telegram yang sama dengan notification-service. Reuse Redis OTP key yang sudah ada — verify OTP endpoint yang sama tetap berfungsi untuk WA maupun Telegram.

**Spec:**
- User memulai chat dengan bot Telegram WCH → bot reply dengan Chat ID
- Frontend UMKM mengirimkan `telegramChatId` bersama data pendaftaran ke `POST /auth/telegram/register`
- Auth-service generate OTP dan kirim via `sendMessage` Telegram Bot API
- OTP disimpan di Redis dengan key yang sama (`otp:{phone}`) — verifikasi via `POST /auth/verify-otp` tetap berfungsi
- Untuk login: `POST /auth/telegram/login` — verifikasi nomor WA terdaftar, kirim OTP via Telegram, update `telegram_chat_id` di DB
- Webhook bot: `POST /auth/telegram/webhook` — handle command `/start` untuk mengembalikan Chat ID
- Reuse 1-hour OTP reuse window dari F017 — baik WA maupun Telegram

**Acceptance Criteria (AC):**
- [x] AC-1: POST `/auth/telegram/register` dengan `telegramChatId` + data → OTP terkirim ke Telegram
- [x] AC-2: POST `/auth/telegram/login` dengan `telegramChatId` + `phoneNumber` → OTP login terkirim ke Telegram
- [x] AC-3: POST `/auth/verify-otp` tetap berfungsi untuk verifikasi OTP (dari WA maupun Telegram)
- [x] AC-4: POST `/auth/verify-phone-login` tetap berfungsi untuk verifikasi login (dari WA maupun Telegram)
- [x] AC-5: Webhook `/auth/telegram/webhook` handle /start command dan return Chat ID
- [x] AC-6: `telegram_chat_id` tersimpan di tabel users saat registrasi & login via Telegram
- [x] AC-7: OTP 1-hour reuse window (F017) berfungsi untuk Telegram juga

**Files Changed:**
- `services/auth-service/main.go` — Telegram request types, `handleTelegramRegister()`, `handleTelegramLogin()`, `handleTelegramWebhook()`, `sendTelegramOTP()`, updated `handleVerifyOTP()` to parse map-based reg data
- `shared/sdk/config/config.go` — Telegram Bot config struct + loading
- `shared/migrations/000031_telegram_auth.up.sql` — `telegram_chat_id` column + index
- `.env.example` — `TELEGRAM_BOT_TOKEN` documentation

**Notes:**
- Bot token dibaca dari `TELEGRAM_BOT_TOKEN` env — shared dengan notification-service (bisa pakai bot yang sama)
- Webhook URL: `POST https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<domain>/auth/telegram/webhook`
- Chat ID Telegram user berbeda dengan WA number — mapping disimpan di `users.telegram_chat_id`
- Tidak perlu perubahan di API Gateway — `/auth/telegram/*` otomatis di-proxy oleh existing `/auth/` prefix

---

## F015: Onboarding Activation Flow

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User baru yang baru daftar bisa lanjut ke step 1 & 2 onboarding tanpa gate. Aktivasi (beli paket / masukkan voucher) baru diminta via modal dialog setelah step 2 selesai. Sistem auto-generate kode voucher sebagai bukti langganan. Superadmin bisa generate voucher dalam jumlah dengan masa aktif day-duration.

**Spec:**

### Onboarding Page (/onboarding)

- Step 1 (Pilih Jenis Usaha) — **tanpa gate**, user boleh pilih atau skip
- Step 2 (Detail Usaha: nama, alamat, nomor WA) — **tanpa gate**, boleh lanjut tanpa harus aktifkan
- Setelah step 2 selesai (klik "Lanjut"), muncul **Modal Activation**:
  - **Opsi A: Beli Paket** — pilih paket (Lite/Pro/Ultimate) → generate Xendit invoice → status subscription = `pending`
  - **Opsi B: Masukkan Kode Voucher** — input kode → validasi → langsung aktivasi jika valid

### Subscription Status Lifecycle

```
Tenant dibuat (register OTP)     → plan=inactive, is_frozen=true
User sampai modal activation
    ├─ Beli Paket                → status=pending, expires_at=now+pending_timeout
    │                              Xendit callback CONFIRMED → activateSubscription()
    │                              Pending > 24 jam tiddk dibayar → hapus tenant + user
    └─ Masukkan Voucher          → validate → activateSubscription() + generate system voucher
```

### Voucher Generation Flow (Superadmin)

```
Superadmin buka SuperAdminDashboard.vue
    └─ Klik "Generate Voucher" → modal input (plan, validity_days, quantity, program_name, max_uses)
           │
           ▼
    POST /api/superadmin/billing/vouchers/generate
           │  billing-service/main.go: handleAdminGenerateVouchers()
           │  1. Upsert voucher_programs (plan_id, program_name, bonus_months=0)
           │  2. INSERT N × voucher_codes (code, program_id, validity_days, is_redeemed=false)
           │  3. Return { codes: [{code, days}] }
           ▼
    Response → modal tampilkan semua kode + Download CSV + Copy
```

### Voucher Redemption Flow (User)

```
User masukkan kode voucher di modal activation
    │
    ▼
POST /auth/vouchers/redeem { code, phone }
    │  billing-service/main.go: handleRedeemVoucher()
    │  1. SELECT voucher_codes + voucher_programs (JOIN) WHERE code = $1 AND is_redeemed = false
    │  2. Cek tenant: SELECT tenants WHERE phone = $2
    │  3. Upsert voucher_subscriptions (akumulasi hari per plan, atau voucher baru jika plan berbeda)
    │  4. activateSubscription():
    │     ├─ Upsert tenant_subscriptions (remaining_days, current_plan_expires_at)
    │     ├─ Update tenants (plan, is_frozen=false, plan_priority)
    │     └─ INSERT subscription_tickets (tipe=voucher)
    │  5. UPDATE voucher_codes SET is_redeemed=true, used_by=tenant_id, used_at=NOW()
    │  6. Generate system voucher: INSERT voucher_codes (WCH-{id}-{ts}, system_generated)
    │  7. Kirim WA notification dengan kode voucher sistem
    ▼
Response → subscription aktif, masa aktif {validity_days} hari
```

### Day-Duration Logic (activateSubscription detail)

```
Tenant redeem voucher dengan validity_days = 30:

  Cek: apakah tenant sudah punya voucher_subscriptions dengan plan_id yang sama?
  │
  ├─ SUDAH ADA (same plan):
  │     UPDATE voucher_subscriptions SET remaining_days = remaining_days + 30
  │     UPDATE tenant_subscriptions SET remaining_days = remaining_days + 30,
  │                                    current_plan_expires_at = NOW() + 30 days
  │
  └─ BELUM ADA (new plan):
         INSERT voucher_subscriptions (tenant_id, plan_id, remaining_days=30, ...)
         INSERT tenant_subscriptions (tenant_id, plan_id, remaining_days=30, ...)
         UPDATE tenants SET plan=plan_name, is_frozen=false, plan_priority=X

  Priority plan: Pro > Business > Lite
  → Plan tertinggi menentukan menu/fitur yang bisa diakses user
```

### Auto-Generate System Voucher (setelah aktivasi via Xendit)

- Saat payment confirmed via Xendit webhook → sistem generate `voucher_codes` entry untuk tenant tersebut
- Format kode: `WCH-{short_tenant_id}-{timestamp}` (contoh: `WCH-a1b2-1750000000`)
- Jenis: `system_generated`, `is_used=true`, `plan_id` sesuai paket yang dibeli
- Kode ini dikirim via WhatsApp notification ke user sebagai "bukti langganan"

### Day-Duration Voucher System (bukan tanggal fixed)

- Kolom `validity_days` (INT) — jumlah hari aktif (bukan `valid_until` date)
- Kolom `remaining_days` — hari tersisa, dihitung saat dibaca
- Saat aktivasi voucher baru:
  - Jika tenant sudah punya voucher aktif dengan **plan yang sama** → akumulasi: `remaining_days += new_validity_days`
  - Jika plan **berbeda** → buat voucher baru secara terpisah (bukan overwrite)
- Priority plan: **Pro > Business > Lite** — sistem baca voucher dengan plan tertinggi sebagai plan aktif

### Auto-Delete Pending Tenant

- Worker `subscription-worker` atau cron di `billing-service` cek tenant dengan `status=pending` dan `created_at < now - 24 jam`
- Hapus row `tenants` + `users` terkait dari DB (CASCADE)
- Log penghapusan ke `subscription_tickets` dengan status `expired`

### Superadmin Voucher Management

- `POST /admin/vouchers/generate` — generate N voucher codes sekaligus
  - Body: `{ plan_id, validity_days, quantity, program_name, max_uses }`
  - Generate N kode acak, simpan ke `voucher_codes`
  - Response: `{ plan_id, validity_days, count, codes: [{code, days}] }`
- `GET /admin/vouchers` — list semua voucher (filter: used/unused, plan_id, program)
  - Response: `{ total, used, unused, codes: [{id, code, program_name, is_redeemed, used_by, used_at, created_at, target_plan}] }`
- `DELETE /admin/vouchers?id=<voucher_id>` — hapus voucher yang belum terpakai
  - Hanya bisa menghapus voucher dengan `is_redeemed = false` (redeemed → 400)
  - Response sukses: `{ status: 200, message: "Voucher deleted successfully" }`
- `GET /admin/tenants/{id}/vouchers` — list voucher aktif per tenant (untuk melihat masa aktif)
- **UI:** Card "Voucher Billing" di `SuperAdminDashboard.vue` memiliki:
  - Tombol "Lihat Daftar" di header card → buka modal daftar voucher (tabel filterable + tombol **🗑 Hapus** untuk unused vouchers)
  - Tombol "Generate Voucher" → buka modal generate (input: program name, paket, jumlah, masa aktif) → tampilkan semua kode yang di-generate + Download CSV + Copy per kode

### WhatsApp Notification (Activation)

- Pesan template saat aktivasi:
  ```
  🎉 Langganan WCH Platform berhasil diaktifkan!

  📋 Paket: {plan_name}
  ⏱️  Masa Aktif: {validity_days} hari
  🔑 Kode Voucher: {system_generated_voucher_code}

  Simpan kode ini sebagai bukti langganan Anda.
  ```

**Acceptance Criteria:**
- [x] AC-1: User baru daftar → sampai step 2 onboarding → tidak diblokir, modal activation muncul
- [x] AC-2: Pilih "Beli Paket" → invoice Xendit dibuat, status subscription = `pending`
- [x] AC-3: Bayar Xendit → webhook confirmed → tenant aktif, kode voucher sistem di-generate, dikirim via WA
- [x] AC-4: Pending > 24 jam tidak dibayar → tenant + user dihapus otomatis
- [x] AC-5: Pilih "Masukkan Voucher" → valid → langsung aktivasi + kode voucher sistem dikirim via WA
- [x] AC-6: Redeem voucher plan sama → hari aktif diakumulasi
- [x] AC-7: Redeem voucher plan berbeda → buat voucher baru, priority tetap plan tertinggi
- [x] AC-8: Superadmin bisa generate N voucher codes sekaligus via API
- [x] AC-9: Superadmin bisa lihat voucher aktif per tenant
- [x] AC-10: Superadmin bisa hapus voucher yang belum terpakai via button Hapus di daftar voucher

**Files yang perlu diubah:**
- `frontend/umkm-web/src/components/Onboarding.vue` — hapus gate di step 1 & 2, tambah modal activation
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Generate Voucher modal + Voucher List modal (UI layer)
- `frontend/umkm-web/src/superadminApi.ts` — `listVouchers()` + `generateVouchers()` API methods
- `services/billing-service/main.go` — `pending` subscription status, auto-delete expired, generate system voucher, day-duration logic, `handleAdminGenerateVouchers`, `handleAdminListVouchers`
- `services/auth-service/main.go` — sync `is_frozen` dan plan cache saat activate
- `shared/migrations/` — add `validity_days` / `remaining_days` columns, `pending_timeout` di `tenant_subscriptions`
- `services/subscription-worker/main.go` — cron job auto-delete expired pending tenants
- `services/wa-gateway/` — WhatsApp notification template untuk activation
- `apps/umkm/accounting/main.go` — quota middleware baca priority plan (Pro > Lite)

**Notes:**
- Billing-service adalah source of truth untuk subscription state
- Auth-service baca dari Redis cache, di-sync saat `activateSubscription()` dipanggil
- Pending timeout default: 24 jam (bisa di-config via env `SUBSCRIPTION_PENDING_TIMEOUT_HOURS`)
- Superadmin generate voucher: server-side via `POST /admin/vouchers/generate` di billing-service
- Superadmin UI voucher: `SuperAdminDashboard.vue` memiliki modal Generate (tampilkan semua kode + CSV download) dan modal List (tabel filterable semua voucher)

---

## F016: Hybrid WhatsApp (Cloud API + whatsmeow)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Pisahkan jalur pengiriman WhatsApp: Meta Cloud API (official) untuk pesan transaksional kritis, whatsmeow (unofficial) untuk chatbot dengan rate limiter ketat. Mengurangi risiko banned nomor WA pengguna.

**Spec:**
- WhatsApp Cloud API service baru (`services/wa-cloud-api`, port 8210) wrap Meta Graph API v22.0
- Per-tenant credentials di tabel `wa_cloud_api_credentials` (phone_number_id, access_token, verify_token)
- Message routing via header `X-Message-Type` di wa-gateway:
  - `otp` → Cloud API (auth-service OTP login/register)
  - `invoice` → Cloud API (billing-service payment notifications)
  - `subscription` → Cloud API (accounting revenue digest)
  - `system` → Cloud API (notification-service system alerts)
  - _(tanpa header)_ → whatsmeow (chatbot conversational)
- Fallback: Cloud API gagal → otomatis whatsmeow (logged as WARN)
- Rate limiter whatsmeow: token bucket 5 msg/menit/tenant (mencegah spam ban)
- Reconnect backoff whatsmeow: exponential (30s → 60s → 240s → 10m), max 1 reconnect/5 menit
- Webhook Meta di `/webhooks/wa-cloud/` untuk status callback & incoming messages
- Superadmin CRUD credentials via `/admin/credentials`

**Acceptance Criteria (AC):**
- [x] AC-1: OTP terkirim via Cloud API saat auth-service kirim dengan `X-Message-Type: otp`
- [x] AC-2: Invoice/payment notification terkirim via Cloud API
- [x] AC-3: Chatbot tetap bisa kirim/terima via whatsmeow (tanpa header khusus)
- [x] AC-4: Rate limiter memblokir pesan whatsmeow ke-6+ dalam 1 menit (HTTP 429)
- [x] AC-5: Cloud API gagal → otomatis fallback ke whatsmeow (logged warning)
- [x] AC-6: Webhook Meta diterima di `/webhooks/wa-cloud/`
- [x] AC-7: Superadmin bisa CRUD credential via `POST/GET /admin/credentials`

**Files yang diubah:**
- `services/wa-cloud-api/main.go` — Service baru wrap Meta Graph API
- `services/wa-cloud-api/migrations.go` — Auto-migration runner
- `services/wa-gateway/main.go` — Message router + rate limiter + reconnect backoff
- `shared/sdk/config/config.go` — WhatsApp Cloud API config fields
- `shared/migrations/000075_wa_cloud_api_credentials.up.sql` — New credential table
- `services/api-gateway/main.go` — Webhook route `/webhooks/wa-cloud/` + health check
- `services/auth-service/main.go` — `X-Message-Type: otp` + `X-Source: auth-service`
- `services/billing-service/main.go` — `X-Message-Type: invoice` + `X-Source: billing-service`
- `services/notification-service/main.go` — `X-Message-Type: system` + `X-Source: notification-service`
- `apps/umkm/accounting/main.go` — `X-Message-Type: subscription` + `X-Source: umkm-accounting`
- `Dockerfile` + `docker-compose.yml` + `Makefile` + `.env.example` — Infrastructure

**Notes:**
- WhatsApp Cloud API pricing ~$0.005-0.08/message tergantung tipe. Lebih mahal dari whatsmeow (tanpa biaya tambahan) tapi zero ban risk.
- Perlu Meta Business App + phone_number_id + permanent access token per tenant
- whatsmeow tetap dipakai untuk chatbot karena conversational messages via Cloud API akan mahal
- Nomor whatsmeow sebaiknya nomor "tumbal" khusus, bukan nomor bisnis utama
- Lihat `CLAUDE.md` section "📱 Hybrid WhatsApp Architecture" untuk detail arsitektur

---

## F017: OTP 1-Hour Reuse Window

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** OTP (registrasi & login via WhatsApp) berlaku selama 1 jam penuh. Selama masa aktif, sistem tidak mengirim ulang OTP — mengurangi volume pesan WA keluar dan menurunkan risiko banned oleh WhatsApp.

**Spec:**
- OTP disimpan di Redis dengan TTL 1 jam (sebelumnya 5 menit)
- Saat user minta OTP baru: cek dulu apakah OTP yang masih aktif ada di Redis
  - Jika ada → return success dengan pesan "OTP sudah dikirim. Masih berlaku 1 jam." (TIDAK kirim ulang)
  - Jika tidak ada → generate OTP baru, simpan ke Redis, kirim via WA Gateway
- OTP TIDAK dihapus setelah verifikasi berhasil — tetap berlaku selama 1 jam penuh
- Redis TTL menangani auto-expiry otomatis setelah 1 jam
- Mencakup 3 endpoint OTP:
  - `POST /register` → OTP registrasi (`otp:{phone}`)
  - `POST /phone-login` → OTP login (`phone-login-otp:{phone}`)
  - `POST /forgot-password` → tidak terpengaruh (email-based, bukan WA)

**Acceptance Criteria (AC):**
- [x] AC-1: Request OTP pertama → OTP baru dikirim via WA, TTL Redis 1 jam
- [x] AC-2: Request OTP kedua dalam 1 jam → tidak kirim ulang, return "OTP sudah dikirim"
- [x] AC-3: Verifikasi OTP sukses → OTP tetap bisa dipakai ulang selama 1 jam
- [x] AC-4: Setelah 1 jam → OTP expired otomatis, request baru generate OTP baru
- [x] AC-5: Login OTP dan Register OTP punya key Redis terpisah (tidak konflik)

**Files Changed:**
- `services/auth-service/main.go` — `handleRegister()`, `handleVerifyOTP()`, `handlePhoneLogin()`, `handleVerifyPhoneLogin()`

**Notes:**
- Mengurangi jumlah pesan WA keluar drastis saat user berkali-kali minta OTP
- Kombinasi dengan F016 (Hybrid WA) + rate limiter memperkuat mitigasi ban
- Risiko keamanan rendah: OTP tetap 6-digit, brute-force dalam 1 jam tidak feasible
- Test OTP `000000` tetap berfungsi di development

---

## F014: Flexible LLM Model System

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem LLM yang flexible dan dynamic dengan capability-based routing, mendukung multiple providers dan per-use-case model selection.

**Spec:**
- **Model Registry**: Konfigurasi model dari environment variables dengan capability tags
- **Capability Routing**: Otomatis pilih model berdasarkan `use_case`:
  - `product` — untuk mengambil data product (murah, fast model)
  - `faq` — untuk menjawab FAQ (murah, fast model)
  - `general` — untuk tugas umum (default, full model)
- **Multi-Provider Support**: MiniMax (primary), Gemini (fallback), OpenAI (optional)
- **Fallback Chain**: Automatic fallback ke provider lain jika primary gagal
- **Per-Model Metrics**: Track usage per model (requests, tokens, cost)
- **Prometheus Endpoint**: `/metrics` untuk monitoring
- **API Endpoint**: `/v1/models` untuk list available models

**Environment Variables:**
```bash
# Single model (default)
MINIMAX_MODELS=MiniMax-M2.7
MINIMAX_CAPABILITIES=general,product,faq

# Multiple models (semicolon-separated)
MINIMAX_MODELS=MiniMax-M2.7;MiniMax-M2.7-Fast
MINIMAX_CAPABILITIES=general,product,faq;general
MINIMAX_COST_PER_1M_IN=0.30;0.10
MINIMAX_FALLBACKS=gemini:gemini-1.5-flash
```

**API Usage:**
```json
// Chat request dengan use_case routing
POST /v1/chat
{
  "message": "Apa harga produk X?",
  "use_case": "product"  // → auto-route ke model dengan capability "product"
}

// Override specific model
POST /v1/chat
{
  "message": "Explain code...",
  "provider": "openai",
  "model": "gpt-4o"
}

// List available models
GET /v1/models
```

**Acceptance Criteria:**
- [x] AC-1: Model registry loaded dari environment variables
- [x] AC-2: `use_case` field mengarahkan ke model yang sesuai capability
- [x] AC-3: Fallback chain berfungsi (MiniMax → Gemini → mock)
- [x] AC-4: Per-model metrics trackable via `/metrics`
- [x] AC-5: `/v1/models` endpoint return semua available models

**Files:**
- `shared/sdk/config/config.go` — LLMModel / LLMConfig structs + loadLLMModels()
- `services/ai-gateway/main.go` — capability-based routing + metrics
- `.env.example` — updated dengan flexible model config

**Notes:**
- Fokus saat ini: MiniMax sebagai primary model
- OpenAI/Gemini sebagai fallback/optional
- Per-tenant model override bisa ditambahkan di future (via DB config)

---

## F012: Sidebar Navigation UI

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Ganti horizontal header menu dengan sidebar kiri yang grouped dan collapsible.

**Spec:**
- Sidebar kiri dengan grouped menu items
- Groups: Operasi (Dashboard, Kasir, Katalog), Keuangan (Jurnal), Sistem (Automasi, Pengaturan, Super Admin)
- Collapsible groups — klik header untuk expand/collapse
- Active route highlighting
- User profile di bottom sidebar
- Responsive: sidebar di desktop, drawer di mobile
- Data-driven menu config (bukan hardcoded HTML)

**Acceptance Criteria:**
- [x] AC-1: Sidebar menampilkan grouped menu items
- [x] AC-2: Groups bisa collapse/expand
- [x] AC-3: Active route di-highlight
- [x] AC-4: User profile terlihat di sidebar
- [x] AC-5: Mobile: hamburger → drawer sidebar
- [x] AC-6: Smooth transition animations

**Files:**
- `frontend/umkm-web/src/components/AppSidebar.vue` — sidebar component baru
- `frontend/umkm-web/src/config/menu.ts` — menu configuration
- `frontend/umkm-web/src/App.vue` — use sidebar
- `frontend/umkm-web/src/style.css` — global sidebar styles

**Notes:** Icon menggunakan emoji untuk simplicity (bisa upgrade ke lucide-icons nanti).

---

## F013: N8N Integration via Super Admin

**Spec Status:** ❌ Removed
**Implementation:** —

**Deskripsi:** Integrate N8N ke Super Admin dashboard sebagai monitoring hub, bukan custom UI.

**Spec:**
- Super Admin dashboard → link ke N8N UI (new tab)
- N8N status indicator (connected/running/error)
- Recent executions widget (fetch from N8N API)
- Quick action: "Buka Workflow Editor"

**Acceptance Criteria:**
- [x] AC-1: N8N status visible di Super Admin
- [x] AC-2: Direct link to N8N editor
- [x] AC-3: Recent executions shown

**Files:**
- `services/billing-service/main.go` — N8N status & executions endpoints
- `frontend/superadmin-web/src/views/Dashboard.vue` — Direct link ke N8N editor
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Direct link ke N8N editor

**Notes:** N8N UI tetap digunakan untuk workflow editing. Super Admin hanya sebagai hub + monitoring.

---

**REMOVED (2026-06-12):** F013 dihapus karena:
- Tidak perlu dedicated `/n8n` page — N8N editor langsung diakses via `http://localhost:5678`
- Fitur sudah terpenuhi cukup dengan link di Dashboard.vue (direct ke N8N editor)

---

## F001: Multi-Store Quota Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** 1 owner bisa buat banyak toko dengan quota per plan.

**Spec:**
- 1 owner = banyak `stores` dengan `business_type` berbeda (restoran + cafe, dll)
- Quota di-enforce via `plan_features.feature_key='max_stores'`
- Default per tier: Lite=1, Pro=1, Ultimate=5
- Superadmin bisa ubah quota via billing-service tanpa migration

**Acceptance Criteria:**
- [x] AC-1: GET `/api/umkm/stores` return quota info (`max_stores`, `can_add`)
- [x] AC-2: POST `/api/umkm/stores` check quota sebelum create
- [x] AC-3: Superadmin bisa CRUD plan-features via `/admin/plan-features`
- [x] AC-4: Header `X-User-Role: superadmin` injected by api-gateway

**Files:**
- `apps/umkm/accounting/main.go` — stores CRUD + quota check
- `services/billing-service/main.go` — superadmin plan-features CRUD

**Notes:** Quota dibaca langsung dari `plan_features` table, tidak di-cache.

---

## F002: Voucher Link Subscription Model

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Subscription = link-based voucher (primary) + Xendit (hybrid B2B).

**Spec:**
- Superadmin generate bulk voucher links via `/admin/voucher-links/generate`
- User klik link → redeem → subscription extend/created
- Hold = read-only + banner, user masih bisa login

**Voucher Lifecycle:**
```
[Superadmin] POST /admin/voucher-links/generate
    { program_id, count, valid_days, base_url }
    → Returns: { links: [{token, url}, ...] }

[User] Klik link → POST /voucher/redeem-link { token, tenant_id }
    1. Verify JWT signature
    2. Lookup voucher_links by SHA-256(token)
    3. Validate: is_active, not redeemed, not expired
    4. Check max_uses_per_tenant
    5. Mark link redeemed
    6. Extend or create subscription
```

**Acceptance Criteria:**
- [x] AC-1: Superadmin generate voucher links (bulk)
- [x] AC-2: User redeem via signed token link
- [x] AC-3: Subscription extend/created on redeem
- [x] AC-4: Tenant un-frozen on successful redeem

**Files:**
- `services/billing-service/main.go` — voucher generation + redemption
- `shared/migrations/000028_voucher_subscription.up.sql` — schema

**Notes:** Voucher token di-hash SHA-256 sebelum save ke DB.

---

## F003: Subscription Hold Worker

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done


**Spec:**
- Cek `tenant_subscriptions` setiap `EXPIRATION_CHECK_INTERVAL` (default 1 jam)
- Batch update: `status='frozen'`, `tenants.is_frozen=true`
- Liveness check: GET `/healthz`

**Acceptance Criteria:**
- [x] AC-1: Worker running dengan interval configurable
- [x] AC-2: Expired subscriptions frozen automatically
- [x] AC-3: `is_frozen` denormalized flag updated

**Files:**
- `docker-compose.yml` — worker service definition


---

## F004: Read-only Enforcement (Frozen Tenant)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Middleware block write operations saat tenant frozen.

**Spec:**
- Middleware `auth.RequireActiveSubscription`
- Block POST/PATCH/PUT/DELETE saat frozen
- GET tetap pass (user bisa lihat data)
- Set header `X-Subscription-Status: active|frozen`

**Acceptance Criteria:**
- [x] AC-1: Write operations blocked saat frozen
- [x] AC-2: Read operations tetap jalan
- [x] AC-3: Response include subscription status header

**Files:**
- `shared/sdk/auth/subscription_guard.go` — middleware
- `apps/umkm/accounting/main.go` — applied ke router

**Notes:** Banner message untuk UI frontend dari header `X-Subscription-Status`.

---

## F005: Superadmin Dashboard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Unified dashboard untuk superadmin (bukan per-product).

**Spec:**
- 1 unified dashboard di `frontend/superadmin-web/` (port 3401)
- Sections: Overview, Voucher Programs, Generate Links, Frozen Accounts
- Overview: tenant counts, voucher stats 30d, revenue (Xendit), subs by plan
- Frozen Accounts: list + kirim reminder WA

**Acceptance Criteria:**
- [x] AC-1: Overview dengan aggregated stats
- [x] AC-2: Voucher program CRUD
- [x] AC-3: Bulk generate + download CSV
- [x] AC-4: Frozen accounts list dengan WA reminder action

**Files:**
- `frontend/superadmin-web/` — Vue 3 frontend
- `services/billing-service/main.go` — dashboard APIs

**Notes:** API Gateway inject `X-User-Role: superadmin` dari JWT claim.

---

## F006: Multi-Tenant WA Session Pool

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Setiap tenant punya WA session sendiri untuk chatbot.

**Spec:**
- Tabel `wa_sessions` store session per tenant
- Status: `connected`, `qr_pending`, `disconnected`
- WA Gateway handle multi-device
- Session di-manage via N8N workflow

**Acceptance Criteria:**
- [x] AC-1: Tenant punya dedicated WA session
- [x] AC-2: Session status trackable
- [x] AC-3: QR code generation per tenant

**Files:**
- `services/wa-gateway/main.go` — WA session management
- `shared/migrations/000029_n8n_queue_mode.up.sql` — schema

**Notes:** Saat dev lokal, hanya satu WA device aktif.

---

## F007: Chatbot with RAG

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** AI chatbot dengan Retrieval-Augmented Generation.

**Spec:**
- FAQ dan Products di-index ke pgvector
- Chatbot retrieve relevant context sebelum LLM call
- Configurable per tenant (LLM, prompt, escalation settings)
- N8N workflow: Config → Session → RAG → LLM → Save

**Acceptance Criteria:**
- [x] AC-1: FAQ/Products indexed ke vector store
- [x] AC-2: Chatbot retrieve relevant context
- [x] AC-3: Per-tenant chatbot config
- [x] AC-4: Multi-channel session (WA, web, etc)

**Files:**
- `apps/umkm/chatbot/main.go` — chatbot API
- `services/ai-gateway/main.go` — embeddings endpoint
- `n8n/workflows/rag_indexer.json` — index workflow
- `n8n/workflows/universal_chatbot.json` — chatbot workflow

**Notes:** Embeddings via OpenAI/Anthropic melalui ai-gateway.

---

## F008: Escalation to Chatwoot

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Chatbot escalation ke human agent via Chatwoot.

**Spec:**
- Trigger escalation berdasarkan keyword atau fallback
- Buat conversation di Chatwoot
- Transfer context (conversation history, customer info)
- Log escalation history

**Acceptance Criteria:**
- [x] AC-1: Auto-escalation based on config
- [x] AC-2: Conversation created in Chatwoot
- [x] AC-3: Context transferred to agent
- [x] AC-4: Escalation history logged

**Files:**
- `n8n/workflows/escalation_handler.json` — escalation workflow
- `shared/migrations/000029_n8n_queue_mode.up.sql` — escalation_history table

**Notes:** Chatwoot running di port 3000 (docker-compose).

---

## F009: N8N Queue Mode Automation

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** N8N dengan Redis queue untuk horizontal scaling.

**Spec:**
- N8N Main: UI + Webhook Receiver + Workflow Editor
- N8N Worker: Execution Worker (scalable)
- Redis DB 2: Bull Queue untuk job distribution
- 8 workflows configured

**Workflows:**
| Workflow | Trigger | Purpose |
|:---------|:--------|:--------|
| `universal_chatbot.json` | Webhook | Multi-tenant chatbot |
| `rag_indexer.json` | Webhook | Index FAQ/Products |
| `escalation_handler.json` | Webhook | Escalation to Chatwoot |
| `master_automations.json` | Cron (1m) | Execute due automations |
| `daily_revenue_digest.json` | Cron | Revenue digest to Telegram |
| `campaign_voter_onboard.json` | Webhook | Voter onboarding |
| `voucher_wa_distribute.json` | Webhook | Voucher WA distribution |

**Acceptance Criteria:**
- [x] AC-1: N8N running dengan queue mode
- [x] AC-2: Redis queue configured
- [x] AC-3: All 8 workflows deployed

**Files:**
- `docker-compose.yml` — n8n-main, n8n-worker, redis config
- `n8n/workflows/*.json` — workflow definitions
- `infra/postgres/init.sql` — auto-create `wch_n8n` database
- `.env` / `.env.example` — `N8N_DB_*`, `N8N_ENCRYPTION_KEY` vars

**Notes:** Worker auto-configure dari shared database — scaling tinggal `docker-compose up -d --scale n8n-worker=N`. Persistence via dedicated `wch_n8n` database, backup: `pg_dump wch_n8n > n8n_backup.sql`.

---

## F010: Campaign Volunteer Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Manajemen volunteer untuk campaign.

**Spec:**
- CRUD volunteer dengan role (ketua, saksi, dll)
- Assign volunteer ke TPS/area
- Track volunteer activity
- Encrypted NIK storage

**Acceptance Criteria:**
- [x] AC-1: Volunteer CRUD
- [x] AC-2: Volunteer assignment to area
- [x] AC-3: NIK encrypted at rest
- [x] AC-4: Activity tracking

**Files:**
- `apps/campaign/api/handlers/volunteer.go`
- `apps/campaign/api/main.go`

**Notes:** NIK di-encrypt AES-256-GCM, key dari `cfg.EncryptionKey`.

---

## F011: Campaign Voter Onboarding

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Voter registration via webhook dari N8N.

**Spec:**
- N8N workflow trigger voter onboarding
- Voter data di-encrypt sebelum save
- Link voter ke TPS

**Acceptance Criteria:**
- [x] AC-1: Webhook endpoint untuk voter creation
- [x] AC-2: Voter data encrypted
- [x] AC-3: TPS assignment
- [x] AC-4: Bulk import support

**Files:**
- `apps/campaign/api/handlers/voter.go`
- `n8n/workflows/campaign_voter_onboard.json`

**Notes:** Bulk import via CSV dengan async processing.

---

## 📍 Lokasi Kode (Quick Reference)

### Mau Tambah Endpoint/API?

```
UMKM Accounting    ──→ apps/umkm/accounting/main.go (flat pattern)
UMKM Business      ──→ apps/umkm/business/main.go (flat pattern)
UMKM Chatbot       ──→ apps/umkm/chatbot/main.go (flat pattern)
UMKM Automation    ──→ apps/umkm/automation/main.go (worker)

Campaign API       ──→ apps/campaign/api/handlers/<nama>.go
                     + daftarkan di apps/campaign/api/main.go

Auth Service       ──→ services/auth-service/main.go
AI Gateway         ──→ services/ai-gateway/main.go
Billing Service    ──→ services/billing-service/main.go
WA Gateway         ──→ services/wa-gateway/main.go
Notification       ──→ services/notification-service/main.go
API Gateway        ──→ services/api-gateway/main.go
```

### Mau Tambah Tabel Database?

```bash
# Cek nomor terakhir:
ls shared/migrations/*.up.sql | tail -1

# Buat migration baru:
shared/migrations/NNNNNN_nama_fitur.up.sql
shared/migrations/NNNNNN_nama_fitur.down.sql
```

### Mau Tambah Config?

```
1. shared/sdk/config/config.go  ← Tambah field + LoadConfig()
2. .env.example                 ← Tambah dengan contoh nilai
3. docker-compose.yml           ← Tambah env var
```

### Mau Tambah UI Frontend?

```
UMKM      ──→ frontend/umkm-web/src/components/<Nama>.vue
Campaign  ──→ frontend/campaign-web/src/
Superadmin ──→ frontend/superadmin-web/src/
```

### Mau Tambah Service/Worker?

```
Wajib update:
☐ Makefile
☐ Dockerfile
☐ docker-compose.yml
☐ services/api-gateway/main.go (jika REST API)
☐ CLAUDE.md (Port Registry)
☐ .env.example
```

---

## 🔧 Cara Menambah Feature Baru

1. **Tambah SPEC entry** di section ini dengan format:
   ```
   ### F012: [Nama Feature]
   **Spec Status:** ⏳ Draft
   **Implementation:** ⏳ Pending
   ...
   ```

2. **User approve** — tambahkan comment atau ubah status ke "✅ Approved"

3. **AI implement** — setelah approved, AI coding berdasarkan SPEC

4. **Update implementation status** — ubah ke "✅ Done" setelah selesai

5. **Update Feature Registry table** di atas

---

## F024: Paid-Only Enforcement (Hardening)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done


**Spec:**

### Database (migration 000038)
- `ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'lite'`
- `ALTER TABLE usage_quotas ALTER COLUMN plan_tier SET DEFAULT 'lite'`

### Backend (`shared/sdk/auth/quota.go`)
- `GetTenantPlan()` → return `"inactive"` saat Redis nil/miss
- `GetPlan()` → fallback `Plans["inactive"]` (semua fitur off, locked)
- Test `TestGetPlan_InactiveFallback` (expect `"inactive"`, MaxTransactions 0)

### Backend (rate limiter)

### Backend (accounting)
- `apps/umkm/accounting/main.go` `getAutomationLimit(plan)` refactor: default = 0 (fail-safe), explicit case `"enterprise", "ultimate", "superadmin" → 999`

### Docs
- `docs/MIGRATION_REGISTRY.md`: tambah entry 2026-06-14, list 000033/000034/000038
- `CLAUDE.md` Fix #5 + Architecture Note updated

### Acceptance Criteria (AC):
- [x] AC-2: `go test ./shared/sdk/auth/...` pass (`TestGetPlan_InactiveFallback` green)
- [x] AC-3: `go vet ./apps/umkm/accounting/...` clean
- [x] AC-5: Docs (FEATURE_MAP, MIGRATION_REGISTRY, CLAUDE.md) reflect new state

### Files Changed:
- `shared/sdk/auth/quota.go`
- `shared/sdk/auth/auth_test.go`
- `services/api-gateway/main.go`
- `apps/umkm/accounting/main.go`
- `apps/umkm/accounting/main_test.go`
- `docs/FEATURE_MAP.md`
- `docs/MIGRATION_REGISTRY.md`
- `CLAUDE.md`

### Notes:
- ⚠️ **Migration conflict**: Claude agent membuat `shared/migrations/000034_billing_cycle.*.sql` dengan prefix yang sama dengan `000034_tenant_faqs_updated_at.*.sql` yang sudah ada. **Salah satu harus rename ke `000035_*` sebelum deploy** untuk menghindari collision saat migrate run.

---

## F025: Tier Restrictions Overhaul (Multimodal AI)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done — All phases complete. Phase 3 endpoints are mocked pending API key provisioning.

**Deskripsi:** Single source of truth untuk tier restrictions (sejajarkan Go `Plans` map dengan DB `plan_features`). Tambah enforcement per-modality (text/vision/audio/image-gen). Per-tier counter mechanism untuk quota tracking. Siapkan fondasi untuk AI multimodal (vision STT/TTS/image gen) di `ultimate` tier.

**Spec:** Lihat rancangan sebelumnya — quota counter table, per-tier AI capability matrix, middleware enforcement. Spesifikasi final menunggu approval owner.

### Commits (Phase 1 — Align source of truth to DB)
- `5ce4cee` feat(db): add numeric quota columns to plan_features
- `a5e7486` feat(db): seed numeric quotas for lite/pro/ultimate
- `037c7a7` feat(sdk): add PlanFeaturesRow struct + IsUnlimited helper
- `30c2c59` refactor(sdk): GetPlan returns PlanFeaturesRow (DB-driven stub)
- `c757cf8` test(sdk): migrate auth tests to PlanFeaturesRow fields + document CheckQuota TODO
- `6bee66f` fix(sdk): set PlanName="inactive" in GetPlanFeatures stub for symmetry
- `de16672` refactor(sdk): remove Plans map + fix umkm/business caller with tier allowlist

### Commits (Phase 2 — Quota counter mechanism)
- `fb4040d` feat(db): add quota_counters table for per-feature tracking
- `b6f081f` feat(sdk): add quota counter helpers (Redis atomic, DB persist stub)
- `60ab95e` feat(sdk): add QuotaMiddlewareFeature with 402 response
- `d1f6e38` feat(ai): wire quota middleware to text endpoints (chat, stream, embeddings)
- `4479fee` feat(chatbot): increment chatbot_messages counter per processed message
- `2c813cd` feat(worker): cron job to archive old quota_counters monthly
- `90e7d62` feat(notification): warn tenant at 80% quota usage (idempotent daily)
- `47cf332` feat(billing): superadmin endpoint to view tenant quota usage
- `ab1242c` feat(fe): display quota usage in Settings page

### Files Planned:
- `shared/migrations/000035_quota_counters.{up,down}.sql` (NEW)
- `shared/migrations/000036_multimodal_features.{up,down}.sql` (NEW)
- `shared/sdk/auth/quota.go` (extend `PlanTier` + add `MaxVisionRequests`, `MaxAudioMinutes`, `MaxImageGen`)
- `shared/sdk/auth/quota_mw.go` (NEW — middleware)
- `shared/sdk/auth/quota_counter.go` (NEW — atomic counter helpers)
- `services/ai-gateway/main.go` (add `/v1/vision`, `/v1/audio/*`, `/v1/image/generate`)
- `services/ai-gateway/handlers/{vision,audio,image}.go` (NEW)
- `apps/umkm/chatbot/main.go` (detect WA message type, route ke vision/STT)
- `services/wa-gateway/main.go` + `services/wa-cloud-api/main.go` (download media)
- `shared/sdk/mediaproxy/` (NEW — WhatsApp media download helper)
- `frontend/umkm-web/src/components/ChatbotConfig.vue` (multimodal toggle)

### Commits (Phase 3):
- `7d960c0` — feat(chatbot): add multimodal config toggles (vision, voice) to db and ui
- `b02bd1f` — feat(chatbot): handle image and voice notes by routing to AI gateway multimodal endpoints
- `65d5069` — feat(wa): forward image and audio messages to chatbot via local tmp proxy
- `fdd3968` — feat(ai): add multimodal endpoint stubs with quota wiring

### Notes:
- Vendor/model asumsi: MiniMax-M3-Vision, Whisper large-v3 (STT), ElevenLabs/edge-tts (TTS), MiniMax-Image-1 — perlu konfirmasi owner

---

*Lihat [CONTRIBUTING.md](../CONTRIBUTING.md) untuk panduan coding.*

## F028: N8N Media Delivery (PDF/Excel via WA)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Upgrade pipeline notifikasi WA agar N8N bisa mengirim file attachment (PDF/Excel) ke WhatsApp user.

**Spec:**
1. **notification-service (`N8NPayload`)**: Tambah field `media_url` & `media_name`, teruskan ke `wa-gateway`.
2. **wa-gateway (`/api/wa/send`)**:
   - Parse `media_url` & `media_name`.
   - Download file ke buffer memory.
   - Upload via `whatsmeow.MediaDocument`.
   - Kirim `waE2E.DocumentMessage` dengan caption teks.

**Acceptance Criteria:**
- [x] AC-1: `notification-service` teruskan parameter media.
- [x] AC-2: `wa-gateway` download file dari `media_url`.
- [x] AC-3: File terkirim ke WA user.
- [x] AC-4: Linter & Tests pass.
### F028-B: N8N Media Delivery (Telegram & Email Extension)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Melengkapi fitur pengiriman media (PDF/Excel) F028 agar juga berlaku ke channel Telegram dan Email (jika ada) saat dikirim dari N8N. 

**Spec:**
1. **notification-service**: Tambah handler untuk Telegram `/webhook/n8n/telegram`.
2. **notification-service**: Fungsi baru `sendTelegramMedia` yang menggunakan endpoint `sendDocument` dari Telegram API jika `media_url` dikirim.

**Acceptance Criteria:**
- [x] AC-1: Endpoint N8N telegram terpasang.
- [x] AC-2: `sendTelegramMedia` handle multipart upload.
- [x] AC-3: Linter pass.
### F029: Dynamic Multimodal Guardrails (Feature Toggles)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Implementasi guardrail dinamis berbasis kuota pada UMKM Chatbot.

**Spec:**
1. **`apps/umkm/chatbot/main.go`**:
   - Tarik plan tenant pakai `auth.GetPlan()`.
   - Cek `MaxAIAudioMinutes == 0`. Jika ya, tolak pesan audio.
   - Cek `MaxAIVision == 0`. Jika ya, tolak pesan gambar.
2. **Pesan Penolakan**:
   - Audio: "layanan pesan suara belum diaktifkan..."
   - Image: "layanan analisa gambar belum diaktifkan..."

**Acceptance Criteria:**
- [x] AC-1: Chatbot baca limit tier.
- [x] AC-2: Chatbot blokir jika 0.
- [x] AC-3: Linter clean.
### F030: Real DB Integration for Plan Features

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done (BE + FE — GET endpoint + PlanFeatures.vue fetch per plan)

**Deskripsi:** Menghapus STUB pada fungsi `GetPlanFeatures()` di SDK agar Chatbot (dan layanan lain) membaca batasan plan secara *real* dari database. Menyediakan endpoint untuk UI Superadmin agar form "Plan Matrix" bisa mengambil data *current limit* secara aktual.

**Spec:**
1. **SDK `GetPlanFeatures` (Backend)**:
   - Modifikasi `shared/sdk/auth/plan_features.go`. Ganti `STUB` dengan logika SQL read:
     1. Ambil `plan` milik `tenant_id` dari cache/DB.
     2. SELECT semua kolom `max_*` dari tabel `plan_features` sesuai `plan_id`.
     3. Terapkan Redis caching per `plan_id` (TTL 1 jam) agar Chatbot tidak spam DB saat mengecek guardrail.
2. **Billing Service (Backend)**:
   - Buat endpoint `GET /admin/plan-features-matrix` yang mengembalikan `SELECT * FROM plan_features` agar UI memiliki data numerik *initial state* saat meload form.
3. **Superadmin Web (Frontend)**:
   - Modifikasi `PlanFeatures.vue` dan `client.ts` untuk memanggil `GET /admin/plan-features-matrix` alih-alih `listPlans()`. Data hasil fetch dipetakan (mapped) ke `formStates`.

**Acceptance Criteria:**
- [x] AC-1: `GetPlanFeatures` mereturn data asli dari PostgreSQL tabel `plan_features` (via `saas_plans` numeric columns + `plan_features` key/value).
- [x] AC-2: Endpoint Matrix `GET` ada di `billing-service` dan terpanggil oleh `superadmin-web` (`handleAdminPlanFeaturesMatrix` GET handler).
- [x] AC-3: UI Plan Matrix menampilkan angka limit sesuai database saat dimuat ulang.
- [x] AC-4: `make check` pass.

### F030: Fix GetPlanFeatures Cache Invalidations and Real DB Reads

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** SDK `GetPlanFeatures` diubah dari yang tadinya mock menjadi fungsi utuh yang menarik tabel DB asli. Ketika fitur tier di-update oleh Superadmin, cache-nya divalidasi agar aktif instan.

**Spec:**
1. **`shared/sdk/auth/plan_features.go` (`GetPlanFeatures`)**:
   - Tarik limit fitur untuk tenant menggunakan query SQL asli.
   - Simpan data limit di Redis (`plan_features:<tier>`) selama 1 jam biar ngga beratkan DB.
2. **`services/billing-service/main.go` (`handleAdminPlanFeaturesMatrixUpdate`)**:
   - Menambahkan perintah penghapusan cache `cache.Client.Del("plan_features:"+planID)`.
3. **Penyatuan Dependensi**: Import `core_project/shared/sdk/cache` ke billing-service.

**Acceptance Criteria:**
- [x] AC-1: `GetPlanFeatures` melayani data DB sejati.
- [x] AC-2: Redis diclear pada update limit.
- [x] AC-3: API, Frontend, & Chatbot tersambung real-time.

---

## F031: Campaign Validation & Data Integrity (Anti-Double)

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

**Deskripsi:** Sistem pencegahan data ganda, validasi NIK (Nomor Induk Kependudukan), dan pencatatan dukungan silang oleh timses/paslon yang berbeda pada modul Campaign menggunakan arsitektur Citizen-Centric Normalized. Fitur ini memastikan DPT (Daftar Pemilih Tetap) dan data KTP relawan 100% akurat, clean, dan tidak bisa disabotase.

**Spec:**
1. **Pemisahan Data Warga (`citizens`)**:
   - Master data identitas warga disimpan terpisah di tabel `citizens` dengan NIK sebagai Primary Key unik.
   - Pendaftaran relasi dukungan masuk ke tabel `endorsements` yang mereferensikan `citizen_id` dan `tenant_id` (paslon).
2. **Cek Relawan Double (Intra-Campaign)**:
   - Relawan/timses A mendaftarkan NIK `1234567890123456`.
   - Jika timses B (masih satu kubu/paslon yang sama) mendaftarkan NIK yang sama, sistem **menerima data tersebut dan menambahkan relasi baru di `endorsements`**, namun memberikan status/flag `conflict_internal`.
   - UI Dashboard memunculkan warning konflik agar admin pusat bisa verifikasi timses mana yang berhak mengklaim warga tersebut.
3. **Cek Pemilih Silang (Inter-Campaign/Paslon Lain)**:
   - Jika Paslon Y mendaftarkan NIK `1234567890123456` yang sudah diklaim Paslon X (multi-tenant), sistem **tetap menerima** input tersebut di DB, namun menambahkan record di `endorsements` Paslon Y dengan status/flag `conflict_external` (Sengketa Lintas Paslon).
   - Dashboard memunculkan daftar NIK sengketa lintas paslon (indikasi swing voters atau data ganda).
4. **Dukungan Lintas Tingkatan (Cross-Level Endorsements)**:
   - Sistem dapat melacak apakah seorang warga mendukung paslon di tingkat yang berbeda (contoh: Mendukung Calon Gubernur A dan Calon Bupati B).
   - Dashboard paslon memiliki menu filter khusus: **"Pemilih Irisan"**. Memungkinkan kandidat melihat: *"Siapa saja pendukung saya yang juga mendukung kandidat X di tingkat provinsi?"*. Ini sangat berguna untuk strategi kampanye tandem (paket) antar paslon.
5. **Validasi Format NIK (KTP)**:
   - Verifikasi NIK harus 16 digit angka valid. Jika tidak valid, status endorsement diset `invalid_nik`.
6. **Rekonsiliasi DPT (Data KPU)**:
   - Tabel `dpt_records` memuat data DPT resmi KPU.
   - Saat warga didaftarkan ke `citizens`, lakukan pengecekan ke `dpt_records` secara otomatis. Jika NIK cocok, set `is_dpt_verified = true` dan `tps_id` sesuai DPT. Jika tidak cocok, set `is_dpt_verified = false` (kategori Unregistered/Non-DPT).

**Acceptance Criteria (AC):**
- [ ] AC-1: API pendaftaran (Webhook N8N/WA) menerima NIK ganda di tenant yang sama atau berbeda, menyimpannya di DB, dan menandainya dengan status conflict yang sesuai.
- [ ] AC-2: Format NIK yang tidak valid (bukan 16 digit) ditandai dengan status `invalid_nik`.
- [ ] AC-3: Proses pendaftaran mencocokkan NIK ke tabel `dpt_records` dan mengeset flag `is_dpt_verified` & `tps_id` secara aktual.
- [ ] AC-4: UI Dashboard menampilkan statistik rasio: Valid, Invalid, Terdaftar Paslon Lain, dan Non-DPT.

---

## F032: Modul Saksi & Real Count C1 (Hari H)

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

**Deskripsi:** Sistem pengawalan suara di TPS pada hari pemilihan (Hari H). Saksi TPS bertugas memvalidasi kehadiran, memotret form C1 Plano, dan mengirimkannya ke sistem via WhatsApp. Data diproses untuk menayangkan Real Count internal secara real-time untuk mendahului dan mengawal rekapitulasi resmi KPU.

**Spec:**
1. **Registrasi Saksi**:
   - Tambahkan *flag* atau relasi khusus (`is_saksi`, `tps_id_assigned`) pada relawan (`volunteers`) untuk menandai bahwa orang tersebut adalah Saksi Mandat untuk TPS tertentu.
2. **Absensi / Kehadiran Pagi (Fraud Prevention)**:
   - Jam 07:00 pagi, saksi wajib mengirim *Live Location* dan *Selfie* di TPS ke WA Bot.
   - Sistem mencocokkan koordinat *Live Location* dengan koordinat TPS (Geo-fencing).
   - Dashboard pusat memunculkan indikator warna (Hijau = Saksi Hadir, Merah = Saksi Bolos/TPS Kosong) sehingga tim reaksi cepat bisa dikirim.
3. **Setor C1 via WA Bot + AI Vision**:
   - Setelah penghitungan, saksi memotret form C1 Plano (kertas hasil akhir) dan mengetik angka suara manual (misal: "Suara Paslon 1: 150, Paslon 2: 80, Batal: 5").
   - Dikirim ke WA Bot. AI Gateway (Multimodal Vision) membaca foto C1 dan memverifikasi apakah angka yang diketik saksi *match* dengan angka tulisan tangan di kertas C1.
   - Jika *Match*: Masuk tabel `real_count_records` (Status: `Auto-Verified`).
   - Jika *Missmatch/Blur*: Masuk antrian (Status: `Needs Human Review`) untuk dicek admin pusat.
4. **Dashboard Real Count**:
   - Tayangan data real-time masuknya C1 (persentase data masuk, total suara per paslon).
   - Agregasi otomatis tingkat Desa -> Kecamatan -> Kabupaten.

**Acceptance Criteria (AC):**
- [ ] AC-1: Relawan bisa di-assign sebagai saksi ke TPS spesifik.
- [ ] AC-2: Endpoint API untuk menerima input suara (C1) dari Webhook N8N/WA.
- [ ] AC-3: Foto C1 tersimpan aman di object storage / lokal dengan referensi TPS ID.
- [ ] AC-4: UI Real Count ter-update otomatis seiring data C1 masuk.

---

## F034: Add-on Wallet & Meta API Connector

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem saldo (wallet) untuk tenant UMKM guna membayar fitur berbiaya tinggi (AI Multimodal & Meta API Blast). Harga kredit dikelola Superadmin. Fitur Broadcast/Blast massal diwajibkan menggunakan Meta Cloud API (memotong saldo Wallet) dan memblokir QR (Whatsmeow) untuk tipe pesan broadcast demi menghindari pemblokiran nomor UMKM.

**Spec:**
1. **Wallet Tables (`wallet_credits`, `wallet_transactions`)**:
   - `balance_cents` (BigInt) per tenant.
   - History transaksi (Top-up vs Konsumsi).
2. **Superadmin Pricing Config (`addon_prices`)**:
   - Kolom: `addon_key` (string), `price_cents` (int), `unit` (string: per_request, per_minute, per_session).
   - Keys: `ai_vision`, `ai_audio_stt`, `wa_blast_api`, `wa_session_meta`.
3. **Consumption Logic (Middleware/Interceptor)**:
   - **AI Text**: INCLUDED (limit harian ikut tier bulanan).
   - **AI Vision/Audio**: Potong saldo tiap request sukses.
   - **WA API Meta**: Potong saldo tiap sesi chat dibuka.
4. **Meta Connector Flow**:
   - User beli add-on "WhatsApp Meta API".
   - Sistem set `tenants.wa_provider = 'meta_cloud'`.
   - **Trigger**: Dashboard UMKM memunculkan modal "Koneksi Ulang Diperlukan". User harus input Meta Phone ID & Token. Whatsmeow (QR) otomatis diputus/dinonaktifkan untuk tenant tersebut.

**Acceptance Criteria (AC):**
- [x] AC-1: Superadmin bisa ubah harga addon (price_cents, unit, is_active, description) via PATCH `/admin/addon-prices/{key}`.
- [x] AC-2: `/v1/vision` (dan audio/image) deduct wallet via `ConsumeWalletAddon()` — insufficient balance → no deduction, endpoint returns mock.
- [x] AC-3: `addon_prices` table extended: unit, description, is_active kolom added + backfilled (migration 000067).
- [x] AC-4: AI Text (chatbot biasa) tetap jalan karena quota via `QuotaMiddlewareFeature`, bukan wallet.
- [x] AC-5: WA Gateway menolak broadcast jika tenant tidak setup Cloud API (sudah ada via F048 — lihat F048 spec).
- [x] AC-6: Wallet.vue UI — halaman depan untuk tenant lihat saldo + topup.

**Files Changed:**
- `shared/migrations/000067_wallet_addon_extend.up.sql` (NEW) — extend addon_prices (unit, description, is_active)
- `shared/sdk/auth/quota_mw.go` — `ConsumeWalletAddon()` + `addonPricePerUnit()`
- `services/ai-gateway/vision.go` — wallet deduction on ai_vision
- `services/ai-gateway/audio.go` — wallet deduction on ai_audio_stt + ai_audio_tts
- `services/ai-gateway/image.go` — wallet deduction on image_gen
- `services/billing-service/main.go` — extend GET+PATCH addon_prices (unit, is_active, description)
- `services/billing-service/main.go` — wallet topup via Xendit per-tenant (xendit_api_key dari DB)


---

## F035: Discount Vouchers (Percent & Fixed)

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

**Deskripsi:** Memberikan opsi kepada Superadmin untuk membuat voucher dengan tipe diskon uang (persentase / rupiah tetap), bukan hanya voucher akses tambahan (bonus_months).

**Spec:**
1. **API Endpoint (`POST /admin/vouchers/generate`)**:
   - Tambah parameter opsional di request body:
     - `voucher_type` (string): `bonus_months` (default), `discount_percent`, `discount_fixed`.
     - `discount_value` (int): Nominal diskon. Max 100 untuk persentase.
2. **Database Insertion**:
   - Modifikasi query `INSERT INTO voucher_programs` agar tidak melakukan hardcode parameter `voucher_type` dan `discount_value`.

**Acceptance Criteria (AC):**
- [ ] AC-1: Endpoint backend `/admin/vouchers/generate` dapat menerima `voucher_type` dan `discount_value`.
- [ ] AC-2: Voucher dengan diskon 20% tersimpan benar di `voucher_programs` (voucher_type = 'discount_percent', discount_value = 20).
- [ ] AC-3: Transaksi `POST /subscribe` menggunakan voucher diskon menghitung `finalPrice` secara akurat (sudah diimplementasi, tinggal trigger).


---

## F036: Lifetime Affiliate, External Agent & Public Leaderboard

**Spec Status:** ✅ Approved
**Implementation:** ⏳ Pending

**Deskripsi:** Sistem komisi *Lifetime Recurring* untuk Agen/Afiliator eksternal (tidak harus menjadi subscriber). Dilengkapi dengan papan peringkat (Leaderboard) publik untuk memicu kompetisi antar agen, serta portal pencairan dana (withdrawal) komisi tunai.

**Spec:**
1. **Database & Tracking**:
   - Tabel `affiliates` (user_id, referral_code unik, bank_info, cash_balance_cents, total_earnings_cents).
   - Tabel `affiliate_earnings` (affiliate_id, tenant_id, invoice_id, amount_cents, created_at).
   - Tabel `affiliate_withdrawals` (affiliate_id, amount_cents, status, admin_note).
   - Modifikasi tabel `tenants`: tambah kolom `referred_by_affiliate_id` (kunci *lifetime lock*).

2. **Skema Win-Win**:
   - **Klien (Tenant Baru):** Input kode agen (misal `AGEN-BUDI`) saat pertama kali langganan → dapat diskon 10% (One-time).
   - **Agen:** Saat *invoice* lunas (baik pertama kali maupun perpanjangan bulan ke-X), sistem mengecek `tenants.referred_by_affiliate_id`. Jika ada, Agen mendapat potongan komisi (misal 20% dari nilai *invoice*) selamanya.

3. **Public Leaderboard API**:
   - Endpoint `GET /api/public/affiliate-leaderboard`.
   - Tidak butuh *Auth* (bisa diakses publik/landing page).
   - Menampilkan TOP 10 Agen bulan ini & All-Time berdasarkan jumlah *closing* (tenant baru) dan *revenue* yang di-generate. Data di-masking (misal: "Budi S. - 150 Closing").

4. **Portal Agen (Frontend)**:
   - Dashboard agen: Link Referral, Saldo Tersedia, Riwayat Komisi, Tombol "Tarik Dana" (Withdraw).
   - Syarat withdraw: Saldo minimal Rp 100.000. Status masuk ke `pending` untuk diproses manual oleh Superadmin (transfer mBanking).

**Acceptance Criteria (AC):**
- [ ] AC-1: Input kode referral saat pendaftaran/langganan pertama mengunci `referred_by` selamanya di tabel `tenants`.
- [ ] AC-2: Perpanjangan otomatis (*renewal*) pada bulan kedua tetap memicu komisi ke Agen melalui mekanisme *payment webhook*.
- [ ] AC-3: Endpoint Public Leaderboard mereturn agregasi agen teratas tanpa membocorkan data sensitif.
- [ ] AC-4: Agen dapat melakukan Request Withdrawal, memotong saldo tunai sementara (`pending` state).

## F046: Hierarchical Coordinator Assignment

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem penunjukan koordinator kampanye berlapis (Gubernur → Kabupaten → Kecamatan → Desa → TPS) dengan validasi area scope otomatis dan tier access untuk melihat hierarki.

**Tujuan:**
- Memungkinkan kandidat membuat koordinator per level wilayah sesuai tingkat pemilihan
- Mencegah cross-area assignment (korcam kec A gak bisa nunjuk kordes kec B)
- Menyediakan API untuk premium kandidat melihat seluruh relawan di hierarki wilayahnya

**Spec:**
- **Mandatory NIK First**: Koordinator yang ditunjuk harus sudah terdaftar di `citizens` (via KTP scan atau manual entry)
- **Dynamic Hierarchy**: Setiap campaign punya level hirarki terbatas sesuai `campaign_type`:
  - Pilgub/Pilpres/Pileg Prov: 5 level (Prov → Kab → Kec → Desa → TPS)
  - Pilkada/Pileg Kab: 4 level (Kab → Kec → Desa → TPS)
- **Area Scope Validation**: Assignment hanya boleh dalam satu cabang wilayah yang sama
- **Cross-Election Allowed**: Satu NIK bisa jadi koordinator di 3 paslon berbeda sekaligus (no dedup)
- **Premium Tier**: Hanya kandidat yang punya fitur `premium_coordination_view` yang bisa lihat seluruh relawannya di dashboard
- **Unlimited Witnesses**: Satu TPS bisa punya 1-N saksi, tidak terbatas

**Acceptance Criteria (AC):**
- [ ] AC-1: Endpoint `POST /coordinator/assign` menerima NIK + level + wilayah_id, validasi area scope
- [ ] AC-2: Endpoint `GET /coordinator/list?level=kordes&region_id=xxx` mengembalikan daftar koordinator di wilayah tersebut
- [ ] AC-3: Endpoint `GET /coordinator/hierarchy` hanya tampil untuk premium tier, menampilkan semua relawan di bawahnya
- [ ] AC-4: Error "Area mismatch" jika korcam kec X mencoba assign kordes kec B

**Files yang perlu diubah:**
- `apps/campaign/api/handlers/coordinator.go` — handler baru untuk assignment & hierarchy
- `shared/migrations/000059_coordinator_hierarchy.up.sql` — tabel `campaign_coordinators`
- `apps/campaign/api/handlers/volunteer.go` — patch untuk validasi area scope

**Testing:**
- Unit test: `apps/campaign/api/handlers/coordinator_test.go` (8 test cases: tenant validation, JSON binding, missing fields, enum validation, query param parsing)

**Notes:**
- Koordinator di-link ke `user_id` di tabel `users`, bukan buat account baru
- Level enum: `korprov`, `korKab`, `korKec`, `korKades`, `saksi_tps`

## F047: Business Type-Based Module System (Klinik Focus)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Pilihan jenis usaha saat registrasi UMKM. Hanya klinik yang dapat akses "Antrean Klinik". Sidebar dinamis berdasarkan business type.

**Tujuan:**
- Menu sidebar UMKM menampilkan hanya modul yang relevan dengan jenis usaha
- Klinik memiliki modul khusus: Antrean, Rekam Medis, Jadwal Dokter, Notifikasi WA
- Mencegah modul klinik muncul di tenant yang bukan klinik

**Spec:**
- **Registrasi Flow:** Tambah dropdown "Jenis Usaha" di form pendaftaran
  - `clinic`, `restaurant`, `retail`, `workshop`, `general`
- **Database:** Kolom `business_type` (VARCHAR) + `clinic_doctors` (text array) di tabel `tenants`
- **Frontend Menu:** Render beda per business_type
  - Clinic: `/clinic/frontdesk`, `/clinic/medical-record`, `/clinic/schedule`, `/clinic/notifications`
  - Restaurant/Retail: POS, Katalog, Inventori
- **Medical Record:** Pasien bisa input keluhan + riwayat datang (text only, no PDF)
- **Doctor Schedule:** CRUD jadwal praktek dokter (hari, jam mulai, jam selesai)
- **WA Notification:** Auto-kirim reminder 1 jam sebelum jadwal (timezone WIB)

**Acceptance Criteria (AC):**
- [x] AC-1: Tenant baru bisa pilih business_type saat registrasi (Register.vue dropdown 8 opsi, default `umum`, FK ke `business_types.id`)
- [x] AC-2: Menu sidebar UMKM menyesuaikan business_type (AppSidebar filter by `businessTypes[]`)
- [x] AC-3: Endpoint `/clinic/*` gagal untuk tenant non-clinic (requireClinicType middleware → 403)
- [x] AC-4: Dokter bisa di-add di `/clinic/schedule` dengan nama + spesialisasi (ClinicFrontdesk tab "Jadwal Dokter", form lengkap)
- [x] AC-5: Rekam Medis CRUD (tab "Rekam Medis", form + list, POST/GET `/clinic/medical-records`)
- [x] AC-6: Migrasi `business_type` FK ke `business_types(id)` via migration 000061 (idempotent INSERT 'clinic' row)

**Files yang perlu diubah:**
- `shared/migrations/000061_business_type.up.sql` — INSERT `business_type = 'clinic'` ke business_types + add clinic_doctors + clinic_services
- `apps/umkm/accounting/clinic_middleware.go` — middleware `requireClinicType` cek business_type
- `apps/umkm/accounting/main.go` — wrap semua `/clinic/*` route dengan middleware
- `frontend/umkm-web/src/components/Register.vue` — tambah dropdown business_type (8 opsi)
- `frontend/umkm-web/src/api.ts` — registerWA/telegramRegister kirim `businessType`
- `frontend/umkm-web/src/config/menu.ts` — 3 menu klinik baru dengan `businessTypes: ['clinic']` filter
- `frontend/umkm-web/src/components/AppSidebar.vue` — filter items by businessTypes
- `frontend/umkm-web/src/App.vue` — fetch & pass `businessType` prop ke sidebar
- `frontend/umkm-web/src/components/ClinicFrontdesk.vue` — 3 tab (Antrean / Rekam Medis / Jadwal Dokter)
- `frontend/umkm-web/src/router/index.ts` — 3 route alias redirect ke `/clinic/frontdesk?tab=...`
- `services/auth-service/main.go` — struct RegisterRequest + INSERT ke tenants dengan business_type

**Testing:**
- Unit test: `apps/umkm/accounting/clinic_test.go` (10+ test cases: middleware, missing tenant, JSON binding, enum validation, time validation)
- Mock data sesuai real schema: `tenants.id` UUID, `business_types.id` VARCHAR(50), `patient_medical_records`, `clinic_doctor_schedules`

**Notes:**
- Clinics bisa punya multiple dokter (array text di DB)
- Fitur ini akan menjadi dasar pricing tier beda per business type
## F048: WA Provider Preferences (Auto, Cloud API, Whatsmeow) & Chatbot Activation Guard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done (v2 — Hybrid WA Setup wizard)

**Enhancement v2 — Hybrid WA Setup Wizard:**
- Backend: Validation endpoint (`/validate`) di wa-cloud-api untuk test credential ke Meta Graph API
- Backend: `handleWACloudAPICredential` auto-validasi credential setelah save
- Backend: Kolom `verification_status`, `verified_at`, `last_checked_at`, `check_error` di `wa_cloud_api_credentials` (migration 000070)
- Frontend: WASetup.vue — flow 2-step (Validate → Save) dengan real-time credential check
- API Gateway: Route `/api/wa/validate` → wa-cloud-api:8210/validate

**Files changed (v2):**
- `shared/migrations/000070_wa_credential_verification.{up,down}.sql`
- `services/wa-cloud-api/main.go` — `handleValidateCredential`
- `services/api-gateway/main.go` — `/api/wa/validate` route
- `apps/umkm/accounting/main.go` — enhanced `handleWACloudAPICredential`
- `frontend/umkm-web/src/api.ts` — `validateCloudAPICredential()`
- `frontend/umkm-web/src/components/WASetup.vue` — 2-step validate+save, status badges

**Acceptance Criteria (AC):**
- [x] AC-1: Migration 000063 terbuat dan diaplikasikan.
- [x] AC-2: `wa-gateway` membaca `wa_provider_preference` dan override routing.
- [x] AC-3: UI `ChatbotConfig.vue` menampilkan toggle WA Provider dan menyimpan ke DB.
- [x] AC-4: Backend endpoint `/api/chatbot/permissions` mengembalikan `has_wa_cloud_api`.
- [x] AC-5: Frontend lock Cloud API option jika `has_wa_cloud_api = false`.
- [x] AC-6: `auth-service` membaca `auth_wa_provider_preference` untuk routing OTP.
- [x] AC-7: Test integrasi: pesan chatbot bisa dipaksa ke cloud_api atau whatsmeow.
- [x] AC-8: Activasi chatbot → BE return error 400 kalau tidak ada WA connection valid.

**Testing:**
- Unit test: `services/wa-gateway/wa_gateway_test.go`
- Unit test: `apps/umkm/accounting/chatbot_config_test.go`
- Build: `go build ./...` ✅
- Vet: `go vet ./apps/umkm/accounting/ ./services/wa-cloud-api/ ./services/api-gateway/` ✅
- All tests pass ✅

---

## F052: Tier-First Feature System + Per-Tenant Addon Guard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Refactor sistem feature gating: (1) Fitur melekat di Tier — superadmin atur ON/OFF per tier di DB, tidak perlu code change untuk add/remove fitur; (2) Addon melekat di Tenant — tenant beli addon → tersimpan per-tenant → guard cek "tier support + addon active". Dua guard = `HasFeatureAccess(tenant, feature)` + `HasAddonAccess(tenant, addonKey)`.

---

### 📌 Background — State Saat Ini

```
现状 (Current):
  plan_features (DB) → feature_key/enabled per plan
    → PlanFeaturesRow (Go struct, hardcoded fields: HasPOS, HasChatbot, ...)
    → HasFeatureAccess() switch/case per feature name
    → hardcoded "Fitur X memerlukan paket Lite..."

问题 (Problems):
  1. Tambah fitur baru → migration (DB) + code change (Go struct + switch)
  2. Tidak ada Addon table → F034 (addon wallet) done; F052 bikin Addon guard foundation
  3. Guard tersebar (HasFeatureAccess, CheckQuota, RequireFeature, RequireClinicType, ...)
  4. Enum "lite/pro/ultimate" hardcoded di banyak tempat
```

---

### 🎯 Tujuan (Goals)

1. **Admin-flexible features**: Superadmin ubah ON/OFF fitur per tier langsung di DB — tanpa code change.
2. **Per-tenant addons**: Addon dibeli tenant → melekat di tenant itu → tidak tergantung tier.
3. **Single source of truth guard**: `CanUse(tenantID, feature)` → cek tier + addon.
4. **Scalable**: Tambah fitur baru cukup INSERT DB row, tidak perlu deploy Go code.

---

### 🏗️ Arsitektur Baru

```
┌──────────────────────────────────────────────────────────────┐
│  Guards (shared/sdk/auth/)                                  │
│                                                              │
│  CanUse(tenantID, "ai_vision")                              │
│    ├─ 1. TierFeatureEnabled?(tier, "ai_vision")             │
│    │     → SELECT is_enabled FROM plan_features              │
│    │       WHERE plan_id = $tier AND feature_key = "ai_vision"│
│    │                                                         │
│    └─ 2. TenantHasAddon?(tenantID, "ai_vision")             │
│          → SELECT 1 FROM tenant_addons                       │
│            WHERE tenant_id = $tid AND addon_key = "ai_vision"│
│            AND status = 'active' AND expires_at > NOW()      │
│                                                              │
│  Result: Tier ON? → allowed                                  │
│          Tier OFF but Addon active? → allowed                │
│          Tier OFF and no Addon? → denied                     │
└──────────────────────────────────────────────────────────────┘
```

---

### 📊 Data Model

#### Tabel `saas_plans` (existing)
```
id VARCHAR(20), name, price_monthly, is_active, sort_order
→ Lite / Pro / Ultimate + "inactive" sentinel
```

#### Tabel `plan_features` (existing, perlu refactor)
```
Kolom existing:
  id, plan_id, feature_key, feature_name, feature_value, is_enabled

Tambah kolom (migration 000065):
  min_tier VARCHAR(20)  -- tier minimum untuk akses (nullable)
  -- Contoh: ai_vision min_tier='ultimate' → hanya Ultimate bisa beli addon
  -- Jika NULL → semua tier bisa akses (default)

Index: UNIQUE(plan_id, feature_key)
```

#### Tabel `available_features` (NEW — registry fitur)
```
id              UUID PK
feature_key     VARCHAR(100) UNIQUE  -- "ai_vision", "wa_cloud_api", "extra_store"
feature_name    VARCHAR(255)
description     TEXT
category        VARCHAR(50)  -- 'ai', 'wa', 'storage', 'addon'
is_addon        BOOLEAN      -- TRUE = berbayar, FALSE = bundled per tier
default_enabled VARCHAR(20)[] -- tier list where enabled by default: ARRAY['pro','ultimate']
addon_price_cents BIGINT     -- harga per unit (jika is_addon=TRUE)
addon_unit      VARCHAR(20)  -- 'per_month', 'per_request', 'per_session'
created_at      TIMESTAMPTZ
```

#### Tabel `tenant_addons` (NEW — addon aktif per tenant)
```
id              UUID PK DEFAULT gen_random_uuid()
tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
addon_key       VARCHAR(100) NOT NULL REFERENCES available_features(feature_key)
status          VARCHAR(20)  -- 'active', 'expired', 'cancelled'
purchased_at    TIMESTAMPTZ
expires_at      TIMESTAMPTZ  -- NULL = unlimited
auto_renew      BOOLEAN DEFAULT true
purchase_price_cents BIGINT  -- harga saat dibeli
wallet_transaction_id UUID   -- REF ke wallet_transactions
created_at      TIMESTAMPTZ DEFAULT NOW()
UNIQUE(tenant_id, addon_key)
```

---

### 🔐 Guard Logic

#### 1. `CanUseFeature(tenantID, featureKey)` — core guard
```go
func CanUseFeature(ctx context.Context, tenantID, featureKey string) (bool, string) {
    tier := GetTenantPlan(ctx, tenantID)
    feat := GetFeatureDef(featureKey)  // from available_features cache

    // Addon-only check (feature tidak ada di plan_features)
    if feat != nil && feat.IsAddon {
        return CanUseAddon(ctx, tenantID, featureKey)
    }

    // Bundled feature check
    row, _ := GetPlanFeaturesRow(ctx, tier)
    // cek plan_features.is_enabled untuk featureKey
    enabled := row.IsFeatureEnabled(featureKey)
    if enabled {
        return true, ""
    }

    // Fallback: apakah ini addon yang di-upgrade dari tier?
    addonOK, _ := CanUseAddon(ctx, tenantID, featureKey)
    if addonOK {
        return true, ""
    }

    return false, fmt.Sprintf("Fitur %s tidak tersedia di paket %s.", feat.FeatureName, tier)
}
```

#### 2. `CanUseAddon(ctx, tenantID, addonKey)` — addon guard
```go
func CanUseAddon(ctx context.Context, tenantID, addonKey string) (bool, error) {
    feat := GetFeatureDef(addonKey)
    if feat == nil || !feat.IsAddon {
        return false, nil // bukan addon
    }

    // Cek tier minimum
    row, _ := GetPlanFeaturesRow(ctx, GetTenantPlan(ctx, tenantID))
    if feat.MinTier != "" && !row.TierAllowsFeature(addonKey) {
        return false, nil // tier tidak memenuhi min tier
    }

    // Cek tenant_addons
    var exists bool
    err := db.Pool.QueryRow(ctx,
        `SELECT EXISTS(
            SELECT 1 FROM tenant_addons
            WHERE tenant_id=$1 AND addon_key=$2
            AND status='active'
            AND (expires_at IS NULL OR expires_at > NOW())
        )`, tenantID, addonKey).Scan(&exists)
    return exists, err
}
```

#### 3. `RequireFeature(feature string)` middleware
```go
// Supercedes existing RequireFeature — delegates to CanUseFeature
func RequireFeature(feature string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID, ok := r.Context().Value(TenantIDKey).(string)
            if !ok || tenantID == "" {
                response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
                return
            }
            allowed, reason := CanUseFeature(r.Context(), tenantID, feature)
            if !allowed {
                w.Header().Set("X-Feature-Gate", "denied")
                w.Header().Set("X-Feature-Required", feature)
                response.Error(w, http.StatusForbidden, reason, nil)
                return
            }
            w.Header().Set("X-Feature-Gate", "allowed")
            next.ServeHTTP(w, r)
        })
    }
}
```

---

### 🛒 Addon Purchase Flow

```
1. Tenant buka menu "Add-ons" → list semua available_addons
   → GET /api/umkm/addons (accounting service)
   → Response: { key, name, description, price_cents, unit,
                  is_active: bool, expires_at?, has_addon: bool }

2. Tenant klik "Beli" → modal konfirmasi harga
   → POST /api/umkm/addons/purchase { addon_key }
   → BE: Cek wallet balance → DeductWalletBalance()
   → INSERT tenant_addons (status='active', expires_at=NOW()+1bulan, ...)
   → Return success

3. GuardCanUseFeature() → otomatis allow karena tenant_addons aktif
   → Tidak perlu restart, tidak perlu deploy
   → Cached 1 menit via Redis: addon_check:{tenant_id}:{addon_key}
```

---

### 📋 Seed Data (Migration 000065)

```sql
-- available_features registry
INSERT INTO available_features (feature_key, feature_name, category, is_addon, default_enabled, addon_price_cents, addon_unit) VALUES
    -- Bundled features (is_addon=FALSE, default_enabled sesuai tier)
    ('accounting',       'Double-Entry Accounting',      'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('pos',              'Point of Sale',                  'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('chatbot',          'AI Chatbot WhatsApp',           'ai',        false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('ai_text',          'AI Text (Chat)',                 'ai',        false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('inventory',        'Inventory Management',            'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('reports',         'Laporan Keuangan',               'core',      false, ARRAY['lite','pro','ultimate'], 0,        NULL),
    ('multi_user',       'Multi-User Access',             'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('advanced_reports', 'Laporan Keuangan Lanjutan',     'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('api_access',       'API Access',                    'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    ('custom_branding',  'Custom Branding',               'core',      false, ARRAY['ultimate'],            0,        NULL),
    ('priority_support', 'Priority Support',              'core',      false, ARRAY['pro','ultimate'],       0,        NULL),
    -- Addon features (is_addon=TRUE)
    ('ai_vision',        'AI Vision (Foto KTP/Produk)',   'ai',        true,  NULL,                           5000,      'per_request'),
    ('ai_audio',         'AI Audio (Voice Note)',          'ai',        true,  NULL,                           1000,      'per_minute'),
    ('wa_cloud_api',     'WA Cloud API Broadcast',        'wa',        true,  NULL,                           5000,      'per_session'),
    ('wa_blast',         'WA Blast Massal',               'wa',        true,  NULL,                           10000,     'per_request'),
    ('extra_store',      'Extra POS Store',               'storage',   true,  NULL,                           5000000,  'per_month'),
    ('extra_user',       'Extra User Seat',               'storage',   true,  NULL,                           1000000,  'per_month');
```

---

### 🗑️ Cleanup / Migration

#### Migration 000065: Plan Features Refactor
```sql
-- 1. Buat available_features
CREATE TABLE available_features (...);

-- 2. Tambah kolom min_tier ke plan_features
ALTER TABLE plan_features ADD COLUMN IF NOT EXISTS min_tier VARCHAR(20);

-- 3. Sync: INSERT missing features ke plan_features dari available_features
-- (bundled features = is_addon=FALSE)

-- 4. Indexes
CREATE UNIQUE INDEX idx_tenant_addons_lookup
  ON tenant_addons(tenant_id, addon_key)
  WHERE status = 'active';

CREATE INDEX idx_available_features_category
  ON available_features(category) WHERE is_addon = TRUE;
```

---

### 🔄 Phase Plan

| Phase | Scope | Effort |
|:------|:------|:-------|
| Phase 1 | DB schema + `CanUseFeature` SDK + `available_features` seed | Backend only |
| Phase 2 | Migrasi `HasFeatureAccess` → `CanUseFeature`, remove hardcoded switch | Backend refactor |
| Phase 3 | Addon purchase flow (wallet deduction + tenant_addons INSERT) | Backend + FE |
| Phase 4 | UI "Add-ons" page (list + buy + my addons) | Frontend only |
| Phase 5 | Superadmin: UI plan matrix editor (add/remove features per tier) | Frontend |

---

### ✅ Acceptance Criteria (AC)

- [x] AC-1: `CanUseFeature(ctx, tenant, "ai_vision")` → TRUE jika tenant_addons punya "ai_vision" aktif (addon guard)
- [x] AC-2: `CanUseFeature(ctx, tenant, "pos")` → TRUE jika tier=pro/ultimate/lite (bundled plan check)
- [x] AC-3: `CanUseAddon(ctx, tenant, "addon_key")` → TRUE jika tenant_addons row aktif dan belum expired
- [x] AC-4: `RequireFeature(feature)` middleware → delegates to `CanUseFeature` → 403 + reason jika denied
- [x] AC-5: Superadmin flag AC-7 ✅: plan_features.is_enabled edit via existing endpoint (F030 done)
- [x] AC-6: `make check` pass (lint + build + test)

**Note:** AC-7 (addon purchase flow: INSERT tenant_addons) adalah F053 scope.
AC-8 (GET /api/umkm/addons) adalah F053 scope.

---

### 📁 Files Changed (Phase 1)

**Backend:**
- `shared/migrations/000068_tier_addon_system.up.sql` (NEW) — available_features + tenant_addons + min_tier
- `shared/migrations/000068_tier_addon_system.down.sql` (NEW)
- `shared/sdk/auth/can_use.go` (NEW) — `CanUseFeature()` + `CanUseAddon()` + `GetFeatureDef()` + cache
- `shared/sdk/auth/quota.go` — `RequireFeature()` delegates to `CanUseFeature()`, `HasFeatureAccess()` deprecated

**Frontend:** F053 scope.
- `frontend/umkm-web/src/components/Addons.vue` (NEW — purchase UI)
- `frontend/umkm-web/src/api.ts` — `api.getAddons()`, `api.purchaseAddon(addonKey)`
- `frontend/umkm-web/src/router/index.ts` — route `/addons`
- `frontend/umkm-web/src/config/menu.ts` — menu "Add-ons" (if addon_count > 0)

---

### 💡 Saran untuk UMKM (Catatan Implementasi)

1. **Tier bundling tetap simpel**: Fitur "bundled" (accounting, POS, chatbot) tetap dilampirkan ke tier. Addon untuk yang mahal (AI Vision, WA Blast, Extra Store).
2. **Harga addons realistis untuk UMKM Indonesia**:
   - AI Vision: Rp 50/request (sensor foto KTP/product)
   - WA Cloud API: Rp 50/session (bukan per message)
   - Extra Store: Rp 50.000/bulan (tambahan toko POS)
3. **Superadmin UI tidak perlu kompleks**: Plan matrix editor cukup form edit existing `plan_features` — F030 sudah punya `GET /admin/plan-features-matrix`. Tinggal tambah `PUT` per-row.
4. **Wallet integration dulu**: F034 (addon wallet) sebaiknya jadi dependensi — tanpa wallet, addon purchase tidak bisa dilakukan.
5. **Graceful degradation**: Jika `available_features` belum ter-seed, fallback ke `PlanFeaturesRow` yang ada sekarang. Addon check return FALSE jika `tenant_addons` table belum ada.

---

## F053: Admin-Configurable Addon Pricing + Addon Purchase Flow

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Harga addon dikonfigurasi oleh superadmin via UI di `available_features` (BUKAN `addon_prices` — lihat F057 untuk konsolidasi). Tenant membeli addon → wallet deducted → `tenant_addons` row dibuat → fitur langsung aktif. Ini adalah kelanjutan dari F052 (Tier-First Feature System) dan F034 (Wallet).

---

### 📌 Background — State Saat Ini

```
现状 (Current):
  Primary: available_features table (F052/F057 — migration 000068):
    feature_key, feature_name, description, category, is_addon,
    addon_price_cents, addon_unit, default_enabled
    Seeded addons: ai_vision=500000/request, ai_audio=10000/month,
                   wa_blast=200000/request, extra_store=50000/month,
                   extra_user=10000/month

  Legacy: addon_prices table (F034 — migration 000055):
    ai_vision, ai_audio_stt, wa_blast_api, wa_session_meta
    → OVERLAP dengan available_features, akan dikonsolidasi di F057

  handlePurchaseAddon EXISTS (billing-service line 4171):
    POST /addons/purchase — wallet deducted → tenant_addons upsert
    ✅ Referral commission SUDAH
    ❌ Referral discount BELUM (downline bayar full price)

  handleAddonMarketplace EXISTS (billing-service line 4103):
    GET /addon-marketplace — return dari available_features (BUKAN addon_prices)

  tenant_addons table: SUDAH ADA (migration 000068)

  handleWalletTopup EXISTS: tenant bisa topup wallet via Xendit

  Missing:
  1. Tenant-facing Addons.vue — halaman "Toko Addon" untuk beli
  2. Auto-renew cron — addon expired tidak auto-perpanjang
  3. Referral discount — belum di-apply sebelum wallet deduct
```

---

### 🎯 Tujuan (Goals)

1. Superadmin atur harga addon via `available_features` (BUKAN `addon_prices` — F057 konsolidasi).
2. Tenant beli addon dari halaman "Toko Addon" (`Addons.vue`).
3. Pembelian: referral discount → wallet deducted → `tenant_addons` row aktif.
4. Addon punya expiry (bulanan), auto-renew via wallet.
5. Admin lihat siapa punya addon apa.

---

### 📊 Data Model

#### Tabel `tenant_addons` (SUDAH ADA di migration 000068)
```
id              UUID PK DEFAULT gen_random_uuid()
tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
addon_key       VARCHAR(100) NOT NULL
status          VARCHAR(20)  -- 'active', 'expired', 'cancelled'
purchased_at    TIMESTAMPTZ
expires_at      TIMESTAMPTZ  -- NULL = unlimited/per-use
auto_renew      BOOLEAN DEFAULT true
purchase_price_cents BIGINT  -- harga saat dibeli
wallet_transaction_id UUID
created_at      TIMESTAMPTZ DEFAULT NOW()
UNIQUE(tenant_id, addon_key)
INDEX idx_tenant_addons_lookup ON (tenant_id, addon_key) WHERE status='active'
INDEX idx_tenant_addons_expires ON (expires_at) WHERE status='active' AND expires_at IS NOT NULL
```

#### Sumber harga: `available_features` (BUKAN `addon_prices`)
```
available_features.addon_price_cents  — harga dalam sen
available_features.addon_unit         — 'per_month', 'per_request', 'per_session', 'per_minute'
```

⚠️ `addon_prices` table LEGACY — akan di-deprecate setelah konsolidasi F057.

---

### 🔄 Addon Purchase Flow (Final)

```
Tenant klik "Beli" di halaman Add-ons
         │
         ▼
GET /addon-marketplace
  → return: { addon_key, feature_name, price_cents, addon_unit,
              has_addon, addon_status (active/expired) }
         │
         ▼
POST /addons/purchase { addon_key }
         │
         ├─ 1. Cek addon exists di available_features ✅
         ├─ 2. Cek sudah punya active addon ✅ (409 Conflict)
         │
         ├─ 3. F054: Cek referral discount:
         │     SELECT referred_by_affiliate_id FROM tenants
         │     → hitung discount_amount = price * discount_percent / 100
         │     → final_price = price - discount_amount
         │     → INSERT invoice_referrals (untuk audit trail)
         │
         ├─ 4. Cek wallet balance:
         │     balance >= final_price? → deduct ✅
         │     balance < final_price?  → 402 Payment Required (topup dulu)
         │
         ├─ 5. F054: Hitung affiliate commission dari final_price:
         │     → INSERT affiliate_earnings (transaction_type='addon_purchase')
         │     → UPDATE affiliates.cash_balance_cents
         │
         ├─ 6. Upsert tenant_addons (status='active', expires_at=+1bulan) ✅
         │
         └─ 7. Invalidate addon cache ✅
```

### 🔄 Auto-Renew Cron (MISSING — perlu dibuat)

```
Cron job (di billing-service atau subscription-worker) — setiap jam:
         │
         ├─ SELECT FROM tenant_addons
         │   WHERE status='active'
         │   AND auto_renew = true
         │   AND expires_at < NOW() + 24 jam
         │
         ├─ Untuk setiap row:
         │   ├─ Cek wallet balance >= addon_price_cents
         │   ├─ Jika cukup:
         │   │   ├─ Deduct wallet
         │   │   ├─ UPDATE expires_at = expires_at + 1 month
         │   │   └─ INSERT wallet_transactions (type='addon_auto_renew')
         │   │
         │   └─ Jika tidak cukup:
         │       ├─ UPDATE status = 'expired'
         │       └─ Kirim notifikasi "Saldo tidak cukup untuk perpanjang [addon]"
         │
         └─ Invalidate cache per tenant yang berubah
```

---

### 🖥️ UI Specs

#### Halaman Add-ons (Tenant) — `frontend/umkm-web/src/components/Addons.vue` (NEW)

**Route:** `/addons`
**Menu:** muncul di sidebar jika `addon_count > 0` ATAU tier punya `is_addon=TRUE` features

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│  💰 Add-ons                                            │
│  Saldo Wallet: Rp 150.000   [Topup]                     │
├─────────────────────────────────────────────────────────────┤
│  [Tab: AI]  [Tab: WhatsApp]  [Tab: Storage]             │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────┐  ┌──────────────────────┐     │
│  │ 🤖 AI Vision         │  │ 🎙 AI Audio           │     │
│  │ Foto KTP, Produk     │  │ Voice note → teks    │     │
│  │ Rp 5.000/request    │  │ Rp 10.000/menit      │     │
│  │                      │  │                      │     │
│  │ ✅ Aktif (23 hari)  │  │ ❌ Tidak aktif       │     │
│  │ [Kelola]            │  │ [Beli Rp 10.000]     │     │
│  └──────────────────────┘  └──────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

#### Superadmin Addon Prices UI — extend `PlanFeatures.vue`

**Route:** `/admin/addon-prices` (existing, extend)
**Yang bisa diedit:**
- `price_cents` → input number (Rp)
- `unit` → select: per_request | per_minute | per_session | per_month
- `is_active` → toggle (disable dari marketplace)
- `description` → textarea

---

### ✅ Acceptance Criteria (AC)

- [x] AC-1: Superadmin buka `available_features` → list semua addon dengan harga + unit → bisa edit price_cents + unit + is_active → Save → DB updated (via handleAdminAvailableFeaturesCollection/PATCH ✅)
- [x] AC-2: Tenant GET `/addon-marketplace` → list all addon features from `available_features` with has_addon per tenant (✅)
- [x] AC-3: Tenant POST `/addons/purchase` → referral discount applied → wallet deducted → `tenant_addons` row upserted → expires_at = now+1mo (✅)
- [x] AC-4: Insufficient balance → HTTP 402 + wallet_url in response (✅)
- [x] AC-5: Already active addon → HTTP 409 Conflict (✅)
- [x] AC-6: `CanUseAddon` → false for expired rows (F052 ✅)
- [x] AC-7: Referral discount applied BEFORE wallet deduct (F054 fix)
- [ ] AC-8: Auto-renew cron (subscription-worker) → pending (low priority, can be separate PR)
- [x] AC-9: `make check` pass (✅)

**Note:** AC-8 (auto-renew cron) deferred. Manual renew via POST `/addons/purchase` sufficient for MVP.

---

### 📁 Files Changed

**Backend:**
- `shared/migrations/000067_wallet_addon_extend.up.sql` (F034)
- `shared/migrations/000068_tier_addon_system.up.sql` (F052)
- `services/billing-service/main.go` — new routes + 3 handlers: `handleAddonMarketplace`, `handlePurchaseAddon`, `handleMyAddons`
- `shared/sdk/auth/can_use.go` — `CanUseAddon()` + `InvalidateAddonCache()` (F052)
- `shared/sdk/auth/quota_mw.go` — `ConsumeWalletAddon()` (F034)

**Frontend:** Addons.vue + api.ts + menu (separate task).

---

## F054: Referral System: Discount Downline + Commission Upline

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Tenant yang daftar via kode referral mendapat potongan harga **seumur hidup untuk semua pembelian** (subscription renewal, addon purchase, campaign checkout). Upline (agen/affiliator) mendapat komisi setiap downlinenya melakukan pembayaran apa pun, juga seumur hidup. Semua configurable oleh superadmin via `referral_config`. Berlaku untuk **semua produk WCH (UMKM + Campaign)**.

---

### 📌 Background — State Saat Ini

```
现状 (Current):
  referral_config table EXISTS (migration 000057):
    discount_percent=10, commission_percent=10  (singleton row id=1)
    + Extended di migration 000069: min_purchase_cents, max_commission_cents, is_active, referral_link_base

  handleAdminReferralConfig EXISTS (billing-service line 4667):
    GET/PUT /admin/referral-config
    ✅ SUDAH pakai header X-User-Role (fixed sejak implementasi)

  handleAffiliateRedeemReferral EXISTS (billing-service line 4517):
    POST /affiliate/redeem-referral
    → sets tenants.referred_by_affiliate_id saat registrasi

  handleSubscribe (line 620-631):
    ✅ Referral discount sudah di-apply ke invoice amount
    ✅ invoice_referrals row dibuat (line 727-733)

  handlePaymentWebhook (line 1235-1287):
    ✅ Affiliate commission dari subscription renewal
    ✅ first_purchase_at di-update di affiliate_referrals

  handlePurchaseAddon (line 4247-4264):
    ✅ Affiliate commission dari addon purchase SUDAH
    ❌ Referral discount BELUM — downline bayar full price untuk addon

  HandleBillingWebhook (campaign billing.go line 147-177):
    ✅ Affiliate commission dari campaign checkout SUDAH
    ❌ Referral discount BELUM — downline bayar full price

  Referral link:
    ❌ Route /r/{code} di API gateway BELUM ada
    ❌ Register.vue pre-fill dari link referral BELUM
```

**⚠️ Voucher + Referral stacking bug:** di `handleSubscribe` (line 643), voucher `discount_percent` menimpa referral discount. Seharusnya stack: voucher dulu → hasilnya dikurangi referral discount.

---

### 🎯 Tujuan (Goals)

1. **Downline discount seumur hidup**: Setiap pembayaran (subscription renewal, addon purchase, campaign checkout) → cek referral → potong `discount_percent` dari total. Berlaku **selama tenant aktif** (bukan sekali).
2. **Upline commission seumur hidup**: Setiap pembayaran sukses dari downline → hitung `commission_percent` → INSERT `affiliate_earnings`. Berlaku untuk semua jenis transaksi.
3. **Voucher + Referral stacking**: Voucher diproses dulu, **lalu** referral discount dihitung dari harga setelah voucher (bukan override).
4. **Wallet subscription**: Subscriptions juga bisa dibayar via wallet (setelah topup) — referral discount di-apply sebelum wallet deduct.
5. **Admin-configurable**: Semua % dari `referral_config` — tidak perlu code change.
6. **Scope**: Berlaku untuk semua produk WCH (UMKM + Campaign).
7. **Affiliate dashboard**: Lihat komisi per-transaksi, total earnings, pending withdraw.
8. **Affiliate referral link**: `wch.id/r/AGEN-XXXXX` → redirect ke Register.vue dengan pre-fill.

---

### 🔄 Flow End-to-End

#### A. Registrasi dengan Referral

```
1. Affiliate bagi link: wch.id/r/AGEN-ABCD
   atau kode: AGEN-ABCD (input saat register)

2. User daftar → Register.vue POST /auth/register
   Body: { ..., referral_code: "AGEN-ABCD" }

3. Backend:
   a. Cek affiliate exists WHERE referral_code = $code
   b. INSERT tenants + users
   c. INSERT affiliate_referrals (tenant_id, affiliate_id, referred_at)
      → table baru: affiliate_referrals
   d. TIDAK langsung kasih discount di sini
      (discount applied saat PAYMENT, bukan saat registrasi)
```

#### B. Pembayaran Pertama (Subscription)

```
User pilih paket → handleSubscribe (POST /subscribe)
         │
         ├─ 1. Cek apakah tenant punya referred_by_affiliate_id
         │     SELECT referred_by_affiliate_id FROM tenants WHERE id=$tid
         │     → Jika NULL → skip referral logic
         │
         ├─ 2. Hitung discount:
         │     affiliate = SELECT * FROM affiliates WHERE id = referred_by_affiliate_id
         │     discount_pct = referral_config.discount_percent
         │     discount_amount = price * discount_pct / 100
         │     final_price = price - discount_amount
         │
         ├─ 3. Buat invoice Xendit dengan final_price (sudah ada logic ini!)
         │     → Invoice created with discounted amount
         │     → User bayar: Rp 90.000 (bukan Rp 100.000)
         │
         └─ 4. Simpan referral info di invoice record:
              INSERT INTO invoice_referrals (invoice_id, affiliate_id, discount_amount)
```

#### C. Pembayaran Berhasil (Xendit Webhook)

```
handlePaymentWebhook trigger
         │
         ├─ 1. Handle subscription activation (sudah ada)
         │
         ├─ 2. Hitung commission:
         │     affiliate = SELECT ... FROM affiliates a
         │       JOIN tenants t ON t.referred_by_affiliate_id = a.id
         │       WHERE t.id = tenant_id
         │     commission_pct = referral_config.commission_percent
         │     commission_amount = paid_amount * commission_pct / 100
         │
         ├─ 3. INSERT affiliate_earnings:
         │     INSERT affiliate_earnings
         │       (affiliate_id, tenant_id, invoice_id,
         │        amount_cents=commission_amount,
         │        commission_rate_percent=commission_pct,
         │        transaction_type='subscription_renewal',
         │        paid_at=NOW())
         │
         ├─ 4. UPDATE affiliates: cash_balance_cents += commission_amount
         │
         └─ 5. (BONUS) Notifikasi WA ke affiliate:
              "💰 Komisi baru +Rp {amount}
               Dari: {tenant_name}
               Total earning: Rp {cash_balance}"
```

#### D. Pembayaran Addon (dengan referral discount)

```
handlePurchaseAddon trigger
         │
         ├─ 1. Cek referral:
         │     affiliate_id = tenants.referred_by_affiliate_id
         │     → Jika NULL → skip referral, proceed ke step 4
         │
         ├─ 2. Hitung referral discount:
         │     discount_pct = referral_config.discount_percent
         │     discount_amount = purchase_price_cents * discount_pct / 100
         │     final_price = purchase_price_cents - discount_amount
         │
         ├─ 3. Catat referral discount ke invoice_referrals:
         │     INSERT invoice_referrals (invoice_id=tid, affiliate_id, discount_amount)
         │
         ├─ 4. Deduct final_price dari wallet (bukan full price)
         │
         ├─ 5. Hitung commission dari final_price (bukan dari full price):
         │     commission_amount = final_price * commission_percent / 100
         │
         ├─ 6. INSERT affiliate_earnings (transaction_type='addon_purchase')
         │
         └─ 7. UPDATE affiliates.cash_balance_cents += commission_amount
```

#### E. Pembayaran Campaign Checkout (dengan referral discount)

```
HandleBillingCheckout (campaign billing.go)
         │
         ├─ 1. Cek referral:
         │     affiliate_id = tenants.referred_by_affiliate_id
         │     → Jika NULL → skip
         │
         ├─ 2. Hitung discount:
         │     discount_pct = referral_config.discount_percent
         │     final_price = amount_cents - (amount_cents * discount_pct / 100)
         │
         ├─ 3. CREATE Xendit invoice dengan final_price (bukan mock!)
         │    → /webhooks/xendit/campaign → campaign API
         │
         ├─ 4. HandleBillingWebhook:
         │     Commission dari paid_amount (seperti subscription)
         │     UPDATE affiliates.cash_balance_cents
         │
         └─ 5. INSERT affiliate_earnings (transaction_type='campaign_purchase')
```

#### F. Voucher + Referral Stacking Rule

```
handleSubscribe (line 617-649)
         │
         ├─ 1. base_price = priceMonthly
         │
         ├─ 2. Voucher discount (jika ada):
         │     └─ discount_percent → price_after_voucher = base_price * (100-voucher)/100
         │     └─ discount_fixed  → price_after_voucher = max(0, base_price - voucher_value)
         │
         ├─ 3. Referral discount (jika ada referral):
         │     └─ discount_amount = price_after_voucher * referral_config.discount_percent / 100
         │     └─ final_price = max(0, price_after_voucher - discount_amount)
         │
         └─ 4. Final_price = yang dikirim ke Xendit invoice
             (sebelumnya bug: voucher override referral, sekarang stack: voucher → referral)
```

#### G. Subscription via Wallet — Lihat F058

Detail lengkap ada di **F058: Wallet Payment untuk Subscription & Topup**. Intinya:
- `handleSubscribe` dengan `pay_via_wallet=true` → deduct wallet → activateSubscription langsung (bypass Xendit)
- Referral discount tetap di-apply sebelum wallet deduct
- Auto-renew: cron akan auto-deduct wallet 3 hari sebelum expired

---

### 📊 Data Model — Extensions

#### Tabel `affiliate_referrals` (NEW — track who referred whom + when)
```
id              UUID PK DEFAULT gen_random_uuid()
affiliate_id    UUID NOT NULL REFERENCES affiliates(id)
tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
referred_at     TIMESTAMPTZ DEFAULT NOW()
first_purchase_at TIMESTAMPTZ  -- NULL sampai pertama kali bayar
UNIQUE(affiliate_id, tenant_id)
```

#### Tabel `invoice_referrals` (NEW — track referral discount applied per invoice)
```
id              UUID PK DEFAULT gen_random_uuid()
invoice_id      VARCHAR(100) NOT NULL
affiliate_id    UUID NOT NULL REFERENCES affiliates(id)
discount_amount BIGINT NOT NULL  -- dalam sen
applied_at      TIMESTAMPTZ DEFAULT NOW()
UNIQUE(invoice_id)
```

#### Tabel `referral_config` (EXISTING — extend)
```
Tambah kolom:
  min_purchase_cents    BIGINT DEFAULT 0   -- min purchase agar commission dihitung
  max_commission_cents  BIGINT DEFAULT 0   -- 0 = unlimited
  is_active             BOOLEAN DEFAULT true
  referral_link_base     VARCHAR(255) DEFAULT 'wch.id/r'  -- untuk generate link
```

---

### 🖥️ UI Specs

#### Affiliate Dashboard — extend `AffiliateDashboard.vue`

**Yang sudah ada:**
- Registrasi affiliate
- Tampilkan referral code
- Withdraw request

**Yang perlu ditambah:**
- Tab baru: "Komisi" → list `affiliate_earnings` per transaksi
  - Kolom: Tanggal, Downline, Tipe, Amount (Rp), Rate %
- Tab baru: "Statistik"
  - Total komisi | Pending withdraw | Available balance
  - Jumlah downline aktif | Total downline
- Tampilan referral link: `https://wch.id/r/AGEN-XXXXX`

#### Superadmin Referral Config — extend `ReferralConfig.vue`

```
┌─────────────────────────────────────────────────────────────┐
│  ⚙️ Pengaturan Referral                                   │
├─────────────────────────────────────────────────────────────┤
│  Potongan untuk Downline (%)     [10___] %                  │
│  Komisi untuk Upline (%)         [10___] %                  │
│                                                             │
│  Min. Pembelian untuk Komisi (Rp)  [0________] sen         │
│  Max. Komisi per Transaksi (Rp)   [0 = unlimited] sen      │
│                                                             │
│  [ ] Aktifkan Sistem Referral                              │
│                                                             │
│  ─────────────────────────────────────────────────────────  │
│  Preview:                                                   │
│  Downline beli paket Rp 100.000:                           │
│    → Dapat potongan: Rp 10.000                              │
│    → Upline dapat komisi: Rp 10.000                        │
│  ─────────────────────────────────────────────────────────  │
│                                                    [Save]   │
└─────────────────────────────────────────────────────────────┘
```

---

### ✅ Acceptance Criteria (AC)

- [ ] AC-1: Tenant daftar dengan kode referral → `affiliate_referrals` row created ✅ (sudah ada di auth-service)
- [ ] AC-2: Downline beli subscription → invoice amount DIDISKON `discount_percent` ✅ (sudah di handleSubscribe)
- [ ] AC-3: Downline beli addon → DIDISKON juga (discount di-apply sebelum wallet deduct)
- [ ] AC-4: Downline beli campaign checkout → DIDISKON juga
- [ ] AC-5: Pembayaran sukses (subscription/addon/campaign) → upline dapat commission ✅ (semua sudah)
- [ ] AC-6: Voucher + referral stacking: voucher dulu, referral dari hasil voucher (bukan override) — fix bug line 643 handleSubscribe
- [ ] AC-7: Subscription bisa bayar via wallet (bypass Xendit) jika balance cukup — **Lihat F058**
- [ ] AC-8: Affiliate lihat komisi per transaksi di dashboard ✅ (sudah ada handleAffiliateEarnings)
- [ ] AC-9: Superadmin ubah `discount_percent`/`commission_percent` → langsung生效 ✅ (handleAdminReferralConfig)
- [ ] AC-10: Referral link `https://wch.id/r/AGEN-XXXX` → redirect ke Register.vue dengan pre-fill
- [ ] AC-11: Campaign checkout real Xendit invoice (bukan mock)
- [ ] AC-12: Campaign webhook Xendit di-route oleh API gateway
- [ ] AC-13: `make check` pass

---

### 📁 Files to Change

**Backend:**
- `shared/migrations/000067_referral_system.up.sql` (EXISTING ✅) — affiliate_referrals + invoice_referrals + extend referral_config
- `services/billing-service/main.go` — **PATCH** `handleSubscribe` line 643: fix voucher override referral → stacking logic
- `services/billing-service/main.go` — **PATCH** `handlePurchaseAddon`: add referral discount BEFORE wallet deduct (lines 4216-4230)
- `services/billing-service/main.go` — **PATCH** `handlePaymentWebhook`: referral discount applied to campaign checkout
- `services/billing-service/main.go` — **NEW** `handleSubscribe`: opsi bayar via wallet (bypass Xendit) jika balance cukup
- `services/api-gateway/main.go` — route `/r/{code}` → redirect ke frontend register dengan referral code
- `apps/campaign/api/handlers/billing.go` — **PATCH** `HandleBillingCheckout`: real Xendit invoice (bukan mock)
- `apps/campaign/api/handlers/billing.go` — **PATCH** `HandleBillingWebhook`: apply referral discount

**Frontend:**
- `frontend/umkm-web/src/components/AffiliateDashboard.vue` — extend with Commission tab + Stats tab
- `frontend/umkm-web/src/components/Register.vue` — handle referral link pre-fill
- `frontend/umkm-web/src/components/Addons.vue` — halaman toko addon (tenant-facing, masih MISSING)
- `frontend/umkm-web/src/superadminApi.ts` — `api.getAffiliateReferrals()`, `api.getAffiliateEarnings()`
- `frontend/superadmin-web/src/views/ReferralConfig.vue` — extend form: min_purchase, max_commission, is_active, preview

**Infra:**
- `services/api-gateway/main.go` — route `/webhooks/xendit/campaign` → campaign API (still MISSING)
- `services/billing-service/main.go` — **NEW** Auto-renew cron untuk tenant_addons

**Sudah ada (tidak perlu perubahan):** ✅
- `handleAdminReferralConfig` (X-User-Role ✅)
- `handleAffiliateRedeemReferral` (auth-service ✅)
- `handleAffiliateReferrals` + `handleAffiliateEarnings` (billing-service ✅)
- `handlePurchaseAddon` commission (✅ sudah ada)
- `HandleBillingWebhook` commission (campaign ✅ sudah ada)
- Migration 000069 (✅ sudah jalan)

---

### 💡 Catatan Teknis Penting

1. **Discount diterapkan di invoice Xendit** — bukan di aplikasi. User bayar amount yang sudah didiskon. Ini memastikan compliance dengan payment gateway.
2. **Commission dihitung dari amount yangactually dibayar** (bukan original price) — ini lebih fair untuk affiliate.
3. **Race condition prevention**: Gunakan `SELECT ... FOR UPDATE` pada affiliates saat update cash_balance.
4. **Commission cap**: Jika `max_commission_cents > 0`, maka `commission = MIN(commission, max_commission_cents)`.
5. **Affiliate tanpa downline payment**: Jika affiliate belum punya downline yang pernah bayar, mereka tetap bisa withdraw dari `cash_balance = 0` → should be blocked.
6. **Grace period referral cookie**: Simpan referral_code di cookie 30 hari agar jika user browse lalu daftar nanti, affiliate tetap dapat komisi.

### F050: Staff Management UI

**Spec Status:** ⏳ Draft

**Description:**
Halaman/Dialog untuk menampilkan daftar karyawan/staff sebuah UMKM dan melakukan pengaturan seperti ganti username, nomor HP, dan reset password.

**Acceptance Criteria:**
- [ ] BE: Endpoint `GET /auth/staff` untuk mendapatkan daftar staff berdasarkan tenant_id.
- [x] BE: Endpoint `PUT /auth/staff/{id}` untuk update detail staff.
- [x] BE: Access Control - Hanya Owner dan Admin yang dapat Add/Edit/Delete Staff.
- [ ] FE: Tabel daftar staff di halaman Settings.
- [ ] FE: Modal edit staff untuk ubah data.

**Implementation:** ⏳ Pending
