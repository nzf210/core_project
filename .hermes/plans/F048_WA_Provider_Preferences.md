## F048: WA Provider Preferences (Auto, Cloud API, Whatsmeow)

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

**Deskripsi:** Memberikan fleksibilitas bagi tenant untuk memilih provider WhatsApp untuk layanan Chatbot/CS mereka. Opsi default adalah `auto` (hybrid routing), tapi tenant dengan addon `wa_session_meta` bisa memaksa (force) ke `cloud_api`, atau tenant biasa bisa memaksa ke `whatsmeow` murni.

**Spec:**
1. Database: Tambah enum `wa_provider_enum` (auto, whatsmeow, cloud_api) via migration 000063.
2. `tenant_chatbot_configs` ditambahkan kolom `wa_provider_preference`.
3. `tenants` ditambahkan kolom `auth_wa_provider_preference`.
4. Backend `wa-gateway`: Routing pesan `isTransactional` di-override jika tenant menyetel preferensi eksplisit.
5. Backend `auth-service`: Memilih provider WA berdasarkan preferensi tenant saat kirim OTP (fallback ke whatsmeow jika kosong).
6. Frontend UMKM (`ChatbotConfig.vue`): Tambah toggle "WA Provider" di UI (dropdown/radio). Cloud API dilock jika tidak ada add-on.

**Acceptance Criteria (AC):**
- [x] AC-1: Migration 000063 terbuat dan teraplikasi.
- [x] AC-2: `wa-gateway` membaca `wa_provider_preference` dan override routing (force whatsmeow → skip cloud, force cloud_api → no fallback).
- [ ] AC-3: UI `ChatbotConfig.vue` menampilkan toggle WA Provider dan menyimpan ke DB.
- [ ] AC-4: `wa_session_meta` addon mengeksekusi lock/unlock opsi Cloud API.
- [ ] AC-5: `auth-service` membaca `auth_wa_provider_preference` untuk routing OTP.
- [ ] AC-6: Test integrasi: pesan chatbot bisa dipaksa ke cloud_api atau whatsmeow.

**Files yang perlu diubah:**
- `shared/migrations/000063_wa_provider_preferences.up.sql` — Migration enum + kolom.
- `services/wa-gateway/main.go` — Dynamic routing dengan preferensi.
- `apps/umkm/accounting/main.go` — Handler ChatbotConfig GET/PUT WA provider.
- `frontend/umkm-web/src/components/ChatbotConfig.vue` — UI toggle provider.
- `frontend/umkm-web/nginx.conf` — Affiliate routes fix.
- `frontend/umkm-web/src/components/ClinicFrontdesk.vue` — UI cleanup.

**Notes:**
- Backend wa-gateway sudah diimplementasi: `getTenantWAProviderPreference` + override routing.
- Backend handler ChatbotConfig (UMKM Accounting) belum diimplementasi.
- Frontend ChatbotConfig.vue (UI toggle) belum diimplementasi.
- Affiliate endpoint `/affiliate/*` routing di nginx sudah difix (sebelumnya error JSON parsing).
- ClinicFrontdesk.vue styling sudah diupgrade ke glass-card + btn class standar.
