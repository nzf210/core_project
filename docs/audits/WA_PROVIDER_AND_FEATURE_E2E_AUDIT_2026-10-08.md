# Audit Transisi WhatsApp dan Bukti End-to-End Fitur

**Tanggal:** 2026-10-08
**Cakupan:** fitur aktif UMKM, Campaign, shared services; alur Whatsmeow ↔ Meta Cloud API
**Metode:** tinjauan registry dan alur kode, Go tests dalam mode short, unit/component tests frontend, build, vet
**Batas keselamatan:** tidak menggunakan database production, kredensial/provider Meta live, atau mengirim pesan sungguhan.

## Ringkasan

- Registry mencatat **72 fitur berstatus Implemented: Done** (F001–F012, F014–F027, F029–F074). Status ini adalah klaim implementasi, bukan bukti semua acceptance criteria telah diuji end-to-end.
- **0/72 fitur dapat dinyatakan verified E2E dalam audit ini.** Semua 72 tetap *no E2E evidence*; ini bukan klaim semua fitur gagal. Test E2E yang ada berpotensi menarget service localhost dan mengubah data / memicu pengiriman OTP, sehingga tidak dijalankan tanpa harness terisolasi.
- Transisi provider mempunyai mode yang eksplisit dan masuk akal (`auto`, `whatsmeow`, `cloud_api`), tetapi dispatch Meta sukses/gagal, fallback, dan efeknya pada alur OTP belum dibuktikan oleh test integrasi.
- Ada risiko operasional OTP yang nyata pada implementasi: registrasi memulai pengiriman secara asynchronous lalu dapat menyelesaikan respons tanpa hasil pengiriman. Selain itu, jalur OTP ke Meta diterjemahkan menjadi pesan teks biasa, bukan template autentikasi.
- Hasil test memperlihatkan blocker kualitas yang perlu dibereskan: **UMKM frontend 26 gagal dari 60**, dan **Campaign frontend build gagal** pada tipe `role` di `TaskBoard.vue`. Backend Go short suite, vet, dan build lulus.

## Kesimpulan perpindahan Whatsmeow ↔ Meta Cloud API

| Jalur | Perilaku yang terlihat dari kode | Status bukti |
|---|---|---|
| `auto` + pesan transaksional | Mencoba Cloud API; bila gagal, melanjutkan ke jalur Whatsmeow. | Implementasi ditemukan; belum ada test dispatch end-to-end. |
| `auto` + percakapan biasa | Memilih Whatsmeow. | Implementasi ditemukan; belum ada test dispatch end-to-end. |
| `whatsmeow` | Melewati Cloud API dan memakai sesi Whatsmeow tenant. | Implementasi ditemukan; belum ada test dispatch end-to-end. |
| `cloud_api` | Memaksa Cloud API; kegagalan menghasilkan HTTP 502 dan tidak fallback. | Implementasi ditemukan; belum ada test dispatch end-to-end. |
| Fallback aktual ke Whatsmeow | Respons sukses dari jalur fallback tidak menyatakan provider yang akhirnya mengirim; metrik/log ada. | Belum diuji lintas dua provider. |
| Peralihan akun/sesi | Provider preference mengubah routing pesan; Cloud credentials dan sesi Whatsmeow adalah koneksi terpisah. Ini bukan pemindahan sesi atau riwayat secara otomatis. | Tidak ada bukti alur cutover E2E. |

### Temuan penting

1. **OTP bisa gagal tanpa kegagalan yang terlihat oleh pemanggil registrasi — risiko tinggi.** `handleRegister` menjalankan `sendWAGatewayOTP` dalam goroutine. Fungsi pengirim mencatat kegagalan status HTTP ke log, tetapi tidak mengembalikan hasil kepada handler. Karena itu respons registrasi tidak memastikan OTP berhasil terkirim.
2. **OTP Cloud API dikirim sebagai teks biasa — risiko tinggi, bergantung pada jendela percakapan dan template yang disetujui.** Gateway meneruskan `X-Message-Type: otp`; `wa-cloud-api` hanya membangun payload template jika `type == "template"` dan nama template disertakan. Tipe `otp` yang diteruskan akhirnya menjadi payload `text`. Pengiriman autentikasi Meta perlu diverifikasi menggunakan template autentikasi yang sesuai untuk kondisi penerima sebenarnya; ini tidak diuji terhadap Meta live.
3. **Fallback `auto` bukan jaminan delivery.** Fallback hanya berpindah ke Whatsmeow; Whatsmeow tetap memerlukan sesi tenant yang valid/terhubung dan terkena pembatasan rate. Sementara `cloud_api` sengaja tidak fallback. Persyaratan provider, sesi aktif, dan kondisi template harus dipenuhi sebelum pergantian dianggap mulus.
4. **Test provider yang ada hanya memeriksa helper/rules.** Ada test untuk preferensi header dan klasifikasi transaksional, tetapi tidak ditemukan test yang memanggil `handleCloudAPIRouting` untuk membuktikan dispatch sukses, fallback aktual, forced Cloud API 502, atau respons dari provider kedua. Test `isTransactional` yang ditemukan tidak mengikutsertakan kasus `broadcast`.

## Hasil verifikasi yang dijalankan

| Pemeriksaan | Hasil | Arti/batasan |
|---|---|---|
| `go test ./... -short -count=1` | Lulus semua paket | Test yang bergantung pada service/integrasi dilewati oleh mode short; bukan bukti E2E. |
| `go vet ./...` | Lulus | Pemeriksaan statis Go. |
| `go build ./...` | Lulus | Semua paket Go berhasil dikompilasi. |
| UMKM frontend Vitest, hanya `src` | **34 lulus, 26 gagal** (60 total) | Kegagalan tersebar pada `api.spec.ts` (3), `security.spec.ts` (5), `Dashboard.spec.ts` (6), `Login.spec.ts` (5), dan `Settings.spec.ts` (7). Termasuk mismatch ekspektasi format/tanggal, sanitasi/redirect, assertion UI, serta mock/request API yang tidak terpenuhi. Kegagalan test tidak dengan sendirinya membuktikan bug produksi, tetapi suite belum layak sebagai bukti fitur lulus. |
| UMKM frontend build/type-check | Lulus | Ada peringatan ukuran bundle >500 kB. |
| Campaign frontend Vitest, hanya `src` | **32 lulus** | Ada warning `window.alert` tidak diimplementasikan oleh jsdom. |
| Campaign frontend build | **Gagal** | `TaskBoard.vue`: tipe user yang digunakan tidak memiliki properti `role` (dilaporkan pada baris sekitar 26). |
| Superadmin frontend Vitest/build | **17 test lulus; build lulus** | Menunjukkan cakupan yang lulus di suite tersebut, bukan validasi seluruh fitur platform. |
| Browser/E2E terhadap layanan | Tidak dijalankan | Harness yang tersedia dapat memakai kembali Vite/service localhost; beberapa skenario dapat memanggil autentikasi, mengubah data, atau meminta OTP. Belum terbukti terisolasi sesuai batas keselamatan. |

## Inventaris registry fitur

Semua item di bawah berstatus `✅ Done` pada registry, tetapi **tidak ada yang memperoleh status E2E verified dari audit ini**. “No E2E evidence” berarti tidak tersedia bukti E2E aman dan lengkap, bukan berarti fitur dipastikan rusak.

| ID | Fitur | Hasil audit E2E |
|---|---|---|
| F001 | Multi-Store Quota | No E2E evidence |
| F002 | Voucher Link Subscription | No E2E evidence |
| F003 | Subscription Hold Worker | No E2E evidence |
| F004 | Read-only Enforcement (Frozen) | No E2E evidence |
| F005 | Superadmin Dashboard | No E2E evidence |
| F006 | Multi-Tenant WA Session Pool | No E2E evidence |
| F007 | Chatbot with RAG | No E2E evidence |
| F008 | Escalation to Chatwoot | No E2E evidence |
| F009 | N8N Queue Mode Automation | No E2E evidence |
| F010 | Campaign Volunteer Management | No E2E evidence |
| F011 | Campaign Voter Onboarding | No E2E evidence |
| F012 | Sidebar Navigation UI | No E2E evidence |
| F014 | Flexible LLM Model System | No E2E evidence |
| F015 | Onboarding Activation Flow | No E2E evidence |
| F016 | Hybrid WhatsApp (Cloud API + whatsmeow) | No E2E evidence |
| F017 | OTP 1-Hour Reuse Window | No E2E evidence |
| F018 | Telegram Auth (Register & Login) | No E2E evidence |
| F019 | Onboarding Sync via /me (Fix Lite Tier) | No E2E evidence |
| F020 | AI CS Setup Wizard (Per-Tenant Config UI) | No E2E evidence |
| F021 | Cash Flow PDF Export | No E2E evidence |
| F022 | Excel/Google Sheet Import & Export | No E2E evidence |
| F023 | FAQ Bot AI — Edit & Generate | No E2E evidence |
| F024 | Paid-Only Enforcement (Hardening) | No E2E evidence |
| F025 | Tier Restrictions Overhaul + AI Multimodal | No E2E evidence |
| F026 | N8N Notification Webhooks & Workflows | No E2E evidence |
| F027 | Core Business Flow Fixes & Optimizations | No E2E evidence |
| F029 | Dynamic Multimodal Guardrails | No E2E evidence |
| F030 | GetPlanFeatures DB Integration | No E2E evidence |
| F031 | Campaign Anti-Double Validation | No E2E evidence |
| F032 | Modul Saksi & Real Count C1 | No E2E evidence |
| F033 | Campaign Logistics Tracking | No E2E evidence |
| F034 | Add-on Wallet & Meta API Connector | No E2E evidence |
| F035 | Discount Vouchers (Percent & Fixed) | No E2E evidence |
| F036 | Lifetime Affiliate, External Agent & Public Leaderboard | No E2E evidence |
| F037 | Dashboard Sentimen Isu Harian (AI NLP) | No E2E evidence |
| F038 | Wargame & Simulasi Kemenangan | No E2E evidence |
| F039 | Peta Kerawanan & Pelaporan Pelanggaran | No E2E evidence |
| F040 | WA Bot FAQ Panduan Kampanye (RAG) | No E2E evidence |
| F041 | Gamification & Leaderboard Relawan | No E2E evidence |
| F042 | Auto-Scan KTP (AI OCR Vision) | No E2E evidence |
| F043 | Multi-Level Election & Sainte-Laguë Simulator | No E2E evidence |
| F044 | Campaign Modular License & Payment System | No E2E evidence |
| F045 | UMKM Healthcare Clinic Queue System | No E2E evidence |
| F046 | Hierarchical Coordinator Assignment | No E2E evidence |
| F047 | Hardening Migration (F024 cleanup) | No E2E evidence |
| F048 | WA Provider Preferences & Activation Guard | No E2E evidence |
| F049 | Container Overhaul & Infrastructure Optimization | No E2E evidence |
| F050 | WCH E2E MCP Server (UI Testing & Browser Automation) | No E2E evidence |
| F051 | AI Quota Per-Modalitas (Text/Vision/Image) | No E2E evidence |
| F052 | Tier-First Feature System + Per-Tenant Addon Guard | No E2E evidence |
| F053 | Admin-Configurable Addon Pricing + Addon Purchase Flow | No E2E evidence |
| F054 | Referral System: Discount Downline + Commission Upline | No E2E evidence |
| F055 | Password Reset via Chat (WA + Telegram) v2 | No E2E evidence |
| F056 | Theme Management (Dark/Light/System) | No E2E evidence |
| F057 | Superadmin Feature Matrix + Addon Tier Gating | No E2E evidence |
| F058 | Superadmin Impersonate + Grafana Monitoring | No E2E evidence |
| F059 | Wallet Payment untuk Subscription & Topup | No E2E evidence |
| F060 | Landing Page — Marketing & Onboarding | No E2E evidence |
| F061 | Sales Dashboard Chart — Visual Penjualan | No E2E evidence |
| F062 | Staff Management UI (Settings.vue) | No E2E evidence |
| F063 | WA Keyword Registration (REG/OTP/VERIF) + WA Center | No E2E evidence |
| F064 | Platform WA Provider Detection & OTP Routing | No E2E evidence |
| F065 | Landing Page Content Management — Superadmin JSON Editor | No E2E evidence |
| F066 | Dynamic Feature Gating — Zero-Hardcode Feature Toggle System | No E2E evidence |
| F067 | Grafana Production-Ready Monitoring — Prometheus + 8 Dashboards | No E2E evidence |
| F068 | Standardisasi Format Rupiah — formatRupiah() & formatRupiahShort() | No E2E evidence |
| F069 | Redis-Backed WA Registration Session Persistence | No E2E evidence |
| F070 | Smart Dynamic QRIS (0% Fee) & Struk Digital WhatsApp POS | No E2E evidence |
| F071 | Modular Business Workflow — Laundry Order & Wash Tracking | No E2E evidence |
| F072 | Modular Business Workflow — Restaurant KOT & Table Management | No E2E evidence |
| F073 | Modular Business Workflow — Bengkel & Servis SPK Tracking | No E2E evidence |
| F074 | Dynamic Business Type Chatbot Skills & Context Enrichment | No E2E evidence |

**Di luar hitungan 72:** F013 ditandai Removed; F028 tidak terdaftar; F075 berstatus In Review / Not Started dan tidak dianggap selesai atau diimplementasikan.

## Prioritas agar klaim E2E dapat diverifikasi

1. Tambahkan harness terisolasi: database disposable, WA gateway fake/sink, Meta API stub, dan tanpa akses kredensial atau nomor live.
2. Uji matriks provider secara nyata melalui handler HTTP: `auto` transactional Cloud success/failure → fallback; percakapan → Whatsmeow; `whatsmeow` bypass; `cloud_api` failure → 502 tanpa fallback; verifikasi provider final dan body respons.
3. Untuk OTP, kirim status delivery kembali ke alur registrasi (atau catat job/status yang dapat dipoll); uji timeout, 4xx/5xx, retry dan keputusan fallback. Jangan mengartikan HTTP 200 registrasi sebagai bukti OTP terkirim.
4. Tambahkan pemetaan OTP ke template autentikasi Meta yang disetujui dan test payload-nya terhadap stub Meta. Verifikasi konfigurasi/template per tenant tanpa mengirim pesan sungguhan.
5. Benahi 26 kegagalan UMKM frontend dan error build Campaign. Setelah itu, jalankan suite E2E per alur fitur dalam lingkungan lokal yang dibatasi jaringan dan datanya.
6. Tambahkan test E2E yang ditelusurkan ke acceptance criteria setiap fitur prioritas tinggi; baru setelah seluruh acceptance criteria diuji, ubah status audit masing-masing dari `No E2E evidence`.

## Referensi kode dan registry

- Registry status fitur: `docs/FEATURE_MAP.md` (F001–F075).
- Routing provider: `services/wa-gateway/cloud_routing.go`, `services/wa-gateway/send_handlers.go`.
- Test preferensi/klasifikasi routing: `services/wa-gateway/wa_gateway_test.go`.
- Registrasi dan pengiriman OTP asinkron: `services/auth-service/registration_handlers.go`, `services/auth-service/otp_utils.go`.
- Konversi tipe pesan Meta: `services/wa-cloud-api/handlers.go`.
- Catatan test run terdahulu (bukan hasil audit terkini): `docs/TESTING_INVESTIGATION_REPORT.md`.
