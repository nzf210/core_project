# PR: Priority 1 + Priority 2 (F019, F020, F021, F022) — Onboarding sync, AI CS Wizard, Cash Flow PDF, Excel/Sheet I/O

## 📋 Summary

Bundle 4 fitur (1 quick win + 3 enhancements) yang semuanya sudah di-spec dan di-test:

- **F019** — Onboarding sync via `/me` endpoint (fixes redirect loop on new device)
- **F020** — AI CS Setup Wizard (per-tenant chatbot config UI, 3-step wizard)
- **F021** — Cash Flow PDF Export (A4 PDF, 3-activity SAK-EMKM layout)
- **F022** — Excel/CSV Import & Export (products, contacts, journal entries)

## 🔧 Changes

### Backend (Go)
- `services/auth-service/main.go` — new `GET /me` endpoint + `onboarding_completed` field in `GET /profile`
- `services/api-gateway/main.go` — proxy `/api/me` → auth-service
- `apps/umkm/accounting/main.go` — massive additions:
  - `GET/PUT /chatbot/config` + `POST /chatbot/config/test` (F020)
  - `AIGatewayURL` var (F020)
  - `GET /reports/cash-flow/pdf` (F021)
  - 6 endpoints untuk Excel/CSV I/O (F022)
  - Extended `/reports/cash-flow` return per-activity breakdown (F021)
  - Helpers: `loadChatbotConfigByTenant`, `validateChatbotConfig`, `renderSystemPromptFromConfig`, `formatIDR`, `parseUploadedFile`, `indexHeaders`, `writeFileResponse`
- `apps/umkm/chatbot/main.go` — runtime integration:
  - `loadChatbotConfig` (Redis cache 5 min)
  - `isWithinBusinessHours` (skip LLM when out-of-hours, cost saving)
  - `containsEscalationKeyword`
  - `buildSystemPrompt` accepts `*chatConfigCache` for language/tone/custom-prompt
  - `waSendURL()` helper (refactor from F019)
  - `WAGatewayURL` var (F019)
- `shared/sdk/config/config.go` — already has `WhatsApp.GatewayURL` (reused F019/F020)
- `go.mod` — added `github.com/jung-kurt/gofpdf` v1.16.2 + `github.com/xuri/excelize/v2` v2.10.1

### Frontend (Vue 3)
- `src/api.ts` — methods:
  - `me()` (F019)
  - `getChatbotConfig`, `updateChatbotConfig`, `testChatbotConfig` (F020)
  - `cashFlowPDFUrl`, `exportFile`, `importFile`, `templateURL` (F021, F022)
- `src/router/index.ts` — `fetchAndSyncMe` + 30s cache; route `/chatbot-config` (F020) + `/data-transfer` (F022)
- `src/config/menu.ts` — new menu items: "AI Customer Service" + "Impor / Ekspor"
- `src/components/ChatbotConfig.vue` — 🆕 3-step wizard with stepper, preview, draft autosave, test modal
- `src/components/DataTransfer.vue` — 🆕 3-tab page (Products/Contacts/Journal) with template download, import/export, result panel
- `src/components/AppSidebar.vue` — (unchanged, uses menu config)
- `src/components/Onboarding.vue` — after first-time activation, redirect to wizard with `?first_run=1`
- `src/components/Settings.vue` — new "Customer Service AI" deep-link section
- `src/components/Journal.vue` — "📄 PDF Arus Kas" button for F021
- `src/components/ProductCatalog.vue` — inline Export XLSX + Import (CSV/XLSX) wired to F022

### Docs
- `docs/FEATURE_MAP.md` — F019, F020, F021, F022 specs (all ✅ Approved + Done)
- `audit_report.md` — initial project audit (scope of Priority 1 + 2)

## 📊 Stats

```
 apps/umkm/accounting/main.go          | +941 -13
 apps/umkm/chatbot/main.go             | +188 -3
 docs/FEATURE_MAP.md                   | +366 -0
 frontend/umkm-web/src/api.ts          | +59 -0
 frontend/umkm-web/src/router/index.ts | +54 -1
 frontend/umkm-web/src/components/ChatbotConfig.vue | 🆕 535
 frontend/umkm-web/src/components/DataTransfer.vue | 🆕 200
 ... (and 6 more files)
```

4 commits, total +2600/-50 lines.

## ✅ Test Results

- `go build ./...` ✓
- `go vet ./...` ✓
- `go test ./...` ✓ (17 packages green)
- `vue-tsc --noEmit` ✓

## 🧪 Manual test plan (after deploy)

1. **F019**: Login di device baru / clear localStorage → tidak loop ke `/onboarding`
2. **F020**: 
   - Daftar akun baru → setelah aktivasi, dilempar ke `/chatbot-config?first_run=1`
   - Isi 3 step wizard → simpan
   - Chat via WhatsApp → bot bales sesuai konfigurasi (bahasa, tone, jam)
3. **F021**: Buka Journal → klik "📄 PDF Arus Kas" → PDF terdownload, layout A4 sesuai spec
4. **F022**:
   - Buka `/data-transfer` → tab Produk
   - Download template CSV → edit → upload
   - Lihat hasil imported/skipped/errors
   - Export XLSX → buka di Excel/Google Sheet → data sesuai

## 🔗 Related
- Issue/docs: `docs/FEATURE_MAP.md` → F019, F020, F021, F022
- Audit: `audit_report.md` (Priority 1 + 2 scope)

## ⚠️ Migration notes
- **0 SQL migrations required** — semua schema sudah ada (sebelumnya di F006/F007/migration 000029)
- **2 new Go deps**: `jung-kurt/gofpdf` (PDF) + `xuri/excelize/v2` (XLSX). Both pure Go, no CGO.

## 📸 Screenshots (TODO: attach after deploy)
- ChatbotConfig wizard (3 steps)
- DataTransfer page
- Cash Flow PDF (A4)
- Excel export sample
