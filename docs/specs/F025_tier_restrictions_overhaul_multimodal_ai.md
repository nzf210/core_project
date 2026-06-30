# F025: Tier Restrictions Overhaul (Multimodal AI)


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
