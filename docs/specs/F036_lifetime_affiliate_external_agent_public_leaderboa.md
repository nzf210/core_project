# F036: Lifetime Affiliate, External Agent & Public Leaderboard


## F036: Lifetime Affiliate, External Agent & Public Leaderboard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

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
- [x] AC-1: Referral code locking via `referred_by_affiliate_id` di `tenants` — lifetime lock on first subscribe ✅
- [x] AC-2: Renewal komisi via payment webhook — F054 referral commission pada `handlePaymentWebhook` ✅
- [x] AC-3: Public leaderboard — `HandleCampaignAffiliateLeaderboard` (affiliate.go line 63) return masked data tanpa auth ✅
- [x] AC-4: Withdrawal request via `handleAffiliateWithdrawRequest` di billing-service — `pending` state ✅

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
- [x] AC-1: Endpoint `POST /coordinator/assign` menerima NIK + level + wilayah_id, validasi area scope ✅
- [x] AC-2: Endpoint `GET /coordinator/list?level=kordes&region_id=xxx` mengembalikan daftar koordinator di wilayah tersebut ✅
- [x] AC-3: Endpoint `GET /coordinator/hierarchy` hanya tampil untuk premium tier ✅ (HandleCoordinatorHierarchy + checkPlanFeature implemented in coordinator.go)
- [x] AC-4: Error "Area mismatch" jika korcam kec X mencoba assign kordes kec B ✅

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
