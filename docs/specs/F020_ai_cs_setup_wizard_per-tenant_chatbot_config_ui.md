# F020: AI CS Setup Wizard (Per-Tenant Chatbot Config UI)


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
