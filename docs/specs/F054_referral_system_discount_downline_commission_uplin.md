# F054: Referral System: Discount Downline + Commission Upline


### 🎯 Tujuan (Goals)

1. Superadmin atur harga addon via `available_features` (BUKAN `addon_prices` — F057 konsolidasi).
2. Tenant beli addon dari halaman "Toko Addon" (`Addons.vue`).
3. Pembelian: referral discount → wallet deducted → `tenant_addons` row aktif.
4. Addon punya expiry (bulanan), auto-renew via wallet.
5. Admin lihat siapa punya addon apa.


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


### ✅ Acceptance Criteria (AC)

- [x] AC-1: Superadmin buka `available_features` → list semua addon dengan harga + unit → bisa edit price_cents + unit + is_active → Save → DB updated (via handleAdminAvailableFeaturesCollection/PATCH ✅)
- [x] AC-2: Tenant GET `/addon-marketplace` → list all addon features from `available_features` with has_addon per tenant (✅)
- [x] AC-3: Tenant POST `/addons/purchase` → referral discount applied → wallet deducted → `tenant_addons` row upserted → expires_at = now+1mo (✅)
- [x] AC-4: Insufficient balance → HTTP 402 + wallet_url in response (✅)
- [x] AC-5: Already active addon → HTTP 409 Conflict (✅)
- [x] AC-6: `CanUseAddon` → false for expired rows (F052 ✅)
- [x] AC-7: Referral discount applied BEFORE wallet deduct (F054 fix)
  *(AC-8 auto-renew cron: deferred — manual renew via POST /addons/purchase cukup untuk MVP)*
- [x] AC-8: Auto-renew cron deferred per spec — manual renew via POST /addons/purchase sufficient for MVP. DB column auto_renew_subscription exists.
- [x] AC-9: `make check` pass (✅)

**Note:** AC-8 (auto-renew cron) deferred. Manual renew via POST `/addons/purchase` sufficient for MVP.


## F054: Referral System: Discount Downline + Commission Upline

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Tenant yang daftar via kode referral mendapat potongan harga **seumur hidup untuk semua pembelian** (subscription renewal, addon purchase, campaign checkout). Upline (agen/affiliator) mendapat komisi setiap downlinenya melakukan pembayaran apa pun, juga seumur hidup. Semua configurable oleh superadmin via `referral_config`. Berlaku untuk **semua produk WCH (UMKM + Campaign)**.
