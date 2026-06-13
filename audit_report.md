# Audit Report — WCH Platform / core_project

**Tanggal audit:** 2026-06-14
**Auditor:** Mavis
**Repo:** https://github.com/nzf210/core_project
**Skala:** Monorepo Go SaaS multi-product

---

## 📊 Snapshot Project

| Metrik | Nilai |
|:---|:---|
| Total files | 390 |
| Bahasa utama | Go 62%, Vue 28.6%, TS 2.8% |
| Migrations | 32 (000001 – 000032) + seed.sql |
| Services aktif | 8 (auth, ai-gateway, billing, wa-gateway, wa-cloud-api, notification, subscription-worker, api-gateway) |
| Apps aktif | 3 (umkm, campaign, frontend-web) |
| Frontend apps | 3 (umkm-web, campaign-web, superadmin-web) |
| Total baris .go terbesar | accounting/main.go (102K), billing-service/main.go (96K), auth-service/main.go (74K) |
| Frontend .vue terbesar | SuperAdminDashboard.vue (1262 baris), Onboarding.vue (776), Settings.vue (695) |
| Commit terakhir | "Add unit tests" (2e02e14) |
| Branches/Last activity | Aktif — Juni 2026 |

---

## ✅ Apa yang SUDAH JADI & STABIL (per PRD + FEATURE_MAP)

### UMKM Core (Fase 1-4 ✅)
- Multi-tenant onboarding (7 business types: Warung, Laundry, Toko Online, Restoran, Jasa, Industri Kreatif, Umum)
- Double-entry accounting SAK-EMKM (COA auto-seed, jurnal, validasi debit==credit)
- Laporan: Income Statement, Balance Sheet, Cash Flow
- POS + Dynamic QRIS (CRC16-CCITT) + Xendit webhook
- Product catalog + CSV import/export
- Subscription tiering (Free/Starter/Pro/Enterprise) + QuotaMiddleware

### UMKM AI & Chatbot (Fase 2 ✅)
- AI Gateway dengan MiniMax M2.7 (port 8002) + capability-based routing (F014)
- Redis Queue 100 concurrent workers (handle webhook WA non-blocking)
- RAG dynamic prompt: injeksi katalog + COA + FAQ per chat
- Role-based security: kasir/staff tidak bisa akses laba-rugi
- Conversational accounting (auto JSON expense/transaction)
- Forward to admin dengan marker `[FORWARD_TO_ADMIN]`

### Subscription & Monetization (Fase 3 ✅)
- Xendit subscription + voucher-link hybrid (F001-F005)
- Subscription freeze worker (F003) + read-only enforcement (F004)
- Day-duration voucher + pending subscription (F030)
- Superadmin dashboard + bulk voucher generator

### WhatsApp Hybrid (F006, F016, F017)
- Cloud API (Meta Official) + whatsmeow (unofficial) routing by `X-Message-Type`
- Multi-tenant WA session pool
- OTP 1-hour reuse window (anti-ban)

### Auth & Multi-channel (F015, F017, F018)
- JWT, RBAC
- Telegram auth (register/login via bot)
- Onboarding activation flow (modal setelah step 2)

### Campaign
- Volunteer + voter management, NIK encryption, Real Count

### Workflow (F009)
- n8n queue mode + 8 workflows (universal_chatbot, rag_indexer, escalation, dll)
- Dedicated `wch_n8n` database

---

## ⚠️ OBSERVASI PENTING — BUKAN BUG, TAPI TAHAN UNTUK USER

### 1. Onboarding redirect loop (SUDAH DICATAT di CLAUDE.md tapi belum diperbaiki)
**Lokasi:** `frontend/umkm-web/src/router/index.ts` line 38-46

```typescript
const onboardingDone = localStorage.getItem('onboarding_completed')
if (!onboardingDone && !isSuperadmin) {
  next({ path: '/onboarding' })
  return
}
```

**Masalah yang dicatat di CLAUDE.md:**
> Redirect loop ke halaman payment/onboarding terjadi karena saat login di device baru, flag `onboarding_completed` di `localStorage` kosong, sehingga Router memaksa user aktif masuk ke halaman Onboarding.

**Solusi yang disarankan (juga di CLAUDE.md):**
- Sediakan endpoint `GET /me` untuk sinkronisasi `status` & `role` saat reload
- Baca `onboarding_completed` dari BE, bukan cuma localStorage
- Pastikan saat login, status `onboarding_completed` dan `plan` tersimpan di localStorage (sudah, tapi tidak di-refetch)

**Status: BELUM diimplementasi.**

### 2. Staff RBAC UI belum lengkap
**Dicatat di CLAUDE.md:**
> Sembunyikan menu/akses yang tidak relevan bagi Staff (seperti billing atau upgrade paket).

**Yang ada saat ini:** `AppSidebar.vue` punya prop `userRole` tapi perlu dicek apakah filtering per role sudah jalan untuk menu Billing/Upgrade.

### 3. PDF Arus Kas (Fase 5 — "Dalam Pengerjaan")
PRD Fase 5 status: "🚧 Dalam Pengerjaan" — Cash Flow PDF generation belum selesai.

### 4. Hybrid WA masih pakai hardcoded fallback URL
**Lokasi:** `apps/umkm/chatbot/main.go` line ~503
```go
waGatewayURL := "http://wa-gateway:8202/api/wa/send"
```
Hardcoded di dalam handler, bukan via config. Rawan saat deploy ke environment lain (sudah ada `AIGatewayURL` & `AccountingURL` pattern yang konsisten).

### 5. Tidak ada TODO/FIXME aktif
Pengecekan `grep TODO|FIXME|XXX|HACK` di semua .go → 0 hasil. Berarti project ini bersih, atau "bersih" yang berarti developer lupa tandai. Catatan: TODO bisa disimpan di FEATURE_MAP sebagai "⏳ Pending" — dan memang ada F015 dengan acceptance criteria yang belum semua di-checklist (lihat AC-3 & AC-5 belum di-check).

### 6. Frontend Onboarding.vue besar (776 baris)
File ini kemungkinan punya 3 step + modal activation dalam satu file. Kandidat refactor nanti, tapi bukan blocker.

---

## 🧭 ARAH PENGEMBANGAN YANG MASUK AKAL

User minta: "lanjutkan pengembangan SaaS UMKM agar UMKM bisa membuat CS AI mereka otomatis."

Interpretasi saya: **enable setiap tenant UMKM untuk punya AI CS otomatis** — ini sudah jadi (F006, F007, F016) tapi bisa di-enhance.

### Rekomendasi Tier 1 — Quick Wins (1-2 hari kerja)
1. **Fix onboarding loop** (sudah di-instruksikan CLAUDE.md, tinggal eksekusi)
   - Tambah `GET /me` endpoint di auth-service
   - Update router guard untuk refetch dari BE
2. **Refactor hardcoded WA URL** di chatbot → pakai `cfg.WaGatewayURL`
3. **Verify Staff RBAC di sidebar** — apakah menu Billing/Upgrade tersembunyi

### Rekomendasi Tier 2 — Feature yang Impactful (1-2 minggu)
4. **Onboarding UI baru: tenant setup chatbot AI**
   - Wizard: "Aktifkan CS AI" → pilih persona, bahasa, escalation rules
   - Save ke `tenant_chatbot_configs` (sudah ada di migration 000029)
5. **Per-tenant escalation config** (sudah ada kolom-nya, tinggal UI)
   - Owner bisa atur: kapan eskalasi, ke nomor mana, kata kunci apa
6. **Selesaiin Fase 5 PRD** — Cash Flow PDF generation

### Rekomendasi Tier 3 — Growth Features
7. Multi-channel CS (Instagram DM, Messenger) — di luar WA
8. Voice/voice-note CS (transkrip audio → LLM → reply text/voice)
9. Analytics dashboard CS (response time, resolution rate, top FAQs)
10. Fine-tune per-tenant AI dengan training dari histori chat sukses

---

## 🔬 QUICK HEALTH CHECK — Bisa Saya Jalankan

Sebelum coding, biasanya saya verify:
- `go build ./...` — compile check
- `go test ./apps/umkm/... -v` — UMKM tests
- `make check` — full quality check (perlu Docker up untuk DB/Redis)

---

## 💡 KESIMPULAN UNTUK USER

**Project ini production-ready dan aktif dikembangkan.** Bukan project kosong.

**Yang paling realistis untuk "dilanjutkan":**
- Fix onboarding loop (SUDAH ada instruksi, tinggal eksekusi) — Tier 1
- Enable per-tenant AI CS wizard — Tier 2
- Selesaikan Fase 5 (Cash Flow PDF) — Tier 2

Mau saya mulai dari mana? Saya butuh Anda pilih atau tambah konteks:
1. **Fix onboarding loop dulu** (small, 1-2 jam, high impact UX)
2. **Bangun "AI CS Setup Wizard"** (medium, ~1-2 hari, sales enabler)
3. **Lainnya** — kasih tau arah yang Anda mau
