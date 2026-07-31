# F020: AI CS Setup Wizard (Per-Tenant Chatbot Config UI)

**Date:** 2026-06-13  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F007](../FEATURE_MAP.md) (Chatbot with RAG), [F015](../FEATURE_MAP.md) (Onboarding Modal Activation)

---

## 🎯 Objectives

UMKM owner dapat setup AI Customer Service sendiri via wizard UI tanpa coding — atur kepribadian bot, jam operasional, escalation keywords, dan pesan otomatis.

**Tujuan eksplisit:**
1. Owner UMKM dapat configure chatbot personality (language, tone, LLM parameters) via self-service UI
2. Owner dapat atur business hours + escalation keywords untuk auto-route ke admin saat bot tidak bisa jawab
3. Chatbot service load config dari DB (cached) → apply config saat melayani customer → hemat cost LLM di luar jam operasional

**Problem yang diselesaikan:**
- Setup chatbot butuh developer → barrier adoption tinggi untuk UMKM non-teknis
- Config chatbot hardcoded di backend → setiap perubahan butuh redeploy
- Tidak ada business hours control → LLM dipanggil 24/7 walau toko tutup → waste API cost

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: GET Chatbot Config (Idempotent Auto-Create)**
  - *Verification:* `GET /api/umkm/chatbot/config` return default config untuk tenant baru (auto-create row jika belum ada)
  - *Example:* Tenant baru call endpoint → INSERT default config → return config with `is_active: false`

- [x] **AC-2: PUT Chatbot Config (Partial Update)**
  - *Verification:* `PUT /api/umkm/chatbot/config` update partial fields, validasi constraints (language, tone, temperature, business_hours, escalation_keywords)
  - *Example:* Request `{ "tone": "friendly", "temperature": 0.8 }` → update 2 fields saja, tidak override field lain

- [x] **AC-3: POST Test Chatbot (Preview)**
  - *Verification:* `POST /api/umkm/chatbot/config/test` panggil AI Gateway dengan system_prompt yang sudah di-render dari config
  - *Example:* Request `{ "message": "halo" }` → response `{ "reply": "Halo! Ada yang bisa saya bantu?", "would_escalate": false }`

- [x] **AC-4: Chatbot Load Config from DB (Cached)**
  - *Verification:* `buildSystemPrompt()` di chatbot service baca config via HTTP call ke accounting service, cache Redis 5 menit
  - *Example:* Redis key `chatbot:config:{tenant_id}` TTL 300s → reduce DB hit dari setiap chat message

- [x] **AC-5: Honor Config in Chatbot Runtime**
  - *Verification:* Chatbot honor `language`, `tone`, `business_hours_start/end`, `business_days`, `escalation_keywords`, `max_context_messages`
  - *Example:* Config `language: id, tone: friendly` → system prompt: "Kamu adalah asisten ramah yang bicara Bahasa Indonesia..."

- [x] **AC-6: Outside Business Hours → Skip LLM Call**
  - *Verification:* Di luar jam operasional → return `outside_hours_message` tanpa panggil AI Gateway (hemat cost)
  - *Example:* Business hours 08:00-22:00, message masuk jam 23:00 → instant reply "Terima kasih telah menghubungi..." (no LLM)

- [x] **AC-7: is_active=false → Always Outside Hours Message**
  - *Verification:* Jika `is_active = false` → chatbot return `outside_hours_message` regardless jam
  - *Example:* Owner pause chatbot → all incoming messages get canned response

- [x] **AC-8: Frontend 3-Step Wizard**
  - *Verification:* `ChatbotConfig.vue` wizard dengan 3 steps: Identitas Bot, Jam & Escalation, Kalimat & Channel, dengan progress indicator
  - *Example:* Step 1 (Bahasa, Tone) → Step 2 (Business hours, Keywords) → Step 3 (Messages, Channels) → Save & Activate

- [x] **AC-9: Frontend API Integration + Draft Persistence**
  - *Verification:* Frontend call real API (GET/PUT), toast feedback, simpan draft di sessionStorage (restore saat reload)
  - *Example:* User edit Step 1 → refresh page → draft restored dari sessionStorage

- [x] **AC-10: Entry Points (Sidebar, Settings, First Run)**
  - *Verification:* Sidebar menu "AI CS", Settings link "Setup/Edit", redirect `/chatbot-config?first_run=1` setelah onboarding
  - *Example:* New tenant complete onboarding → auto-redirect ke wizard dengan banner "Lengkapi setup CS AI Anda"

- [x] **AC-11: Build & Test Pass**
  - *Verification:* `go build ./...`, `go vet`, `go test ./...`, `vue-tsc --noEmit` clean
  - *Example:* CI/CD green check

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│      Frontend (ChatbotConfig.vue 3-Step Wizard)     │
│  Step 1: Bahasa, Tone, Temperature                  │
│  Step 2: Business Hours, Escalation Keywords        │
│  Step 3: Messages (Welcome, Fallback, Outside Hrs)  │
│  → PUT /api/umkm/chatbot/config                     │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Backend (apps/umkm/accounting)                 │
│  1. GET /chatbot/config → SELECT * FROM             │
│     tenant_chatbot_configs WHERE tenant_id = ...    │
│     → If not exist, INSERT default config           │
│                                                      │
│  2. PUT /chatbot/config → Validate constraints      │
│     → UPDATE tenant_chatbot_configs SET ...         │
│     → Invalidate Redis cache chatbot:config:{tid}   │
│                                                      │
│  3. POST /chatbot/config/test → Build system_prompt │
│     → Call AI Gateway → Return reply preview        │
└─────────────────────────────────────────────────────┘
         ↓ (chatbot runtime)
┌─────────────────────────────────────────────────────┐
│      Chatbot Service (apps/umkm/chatbot)            │
│  buildSystemPrompt():                                │
│    1. Check Redis: chatbot:config:{tenant_id}       │
│    2. If cache MISS → HTTP GET accounting:8201/...  │
│    3. Cache result (TTL 5 min)                       │
│    4. Check is_active, business_hours, business_days │
│    5. If outside hours → return outside_hours_msg    │
│    6. Else → build LLM prompt from config            │
└─────────────────────────────────────────────────────┘
```

### Database Schema

**No migration needed** — `tenant_chatbot_configs` table already exists from migration 000029 (F007).

**Table:** `tenant_chatbot_configs`
```sql
CREATE TABLE tenant_chatbot_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- LLM Config
    llm_provider VARCHAR(50) DEFAULT 'minimax',
    llm_model VARCHAR(100) DEFAULT 'MiniMax-M2.7',
    temperature FLOAT DEFAULT 0.7,
    max_tokens INT DEFAULT 1024,
    tone VARCHAR(50) DEFAULT 'friendly',
    language VARCHAR(10) DEFAULT 'id',
    max_context_messages INT DEFAULT 10,
    
    -- Messages
    welcome_message TEXT DEFAULT 'Halo! Ada yang bisa saya bantu?',
    fallback_message TEXT DEFAULT 'Maaf, saya belum bisa menjawab pertanyaan itu.',
    outside_hours_message TEXT DEFAULT 'Terima kasih telah menghubungi kami. Kami sedang tutup saat ini.',
    
    -- Business Hours
    business_hours_start TIME DEFAULT '08:00',
    business_hours_end TIME DEFAULT '22:00',
    business_days INT[] DEFAULT ARRAY[1,2,3,4,5,6],  -- 0=Sun, 6=Sat
    
    -- Escalation
    escalation_enabled BOOLEAN DEFAULT true,
    escalation_keywords TEXT[] DEFAULT ARRAY['bicara cs', 'hubungi admin', 'operator'],
    auto_escalate_after_minutes INT DEFAULT 5,
    
    -- RAG
    rag_enabled BOOLEAN DEFAULT true,
    rag_top_k INT DEFAULT 5,
    rag_similarity_threshold FLOAT DEFAULT 0.7,
    
    -- Channels
    channels_enabled TEXT[] DEFAULT ARRAY['whatsapp'],
    is_active BOOLEAN DEFAULT false,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### API Endpoints

#### `GET /api/umkm/chatbot/config`

**Response (200 OK):**
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
    "is_active": false
  }
}
```

**Idempotent Behavior:**
- If config not exists → INSERT default row → return default config
- If config exists → return existing config

**Error Cases:**
- `401 Unauthorized` — Missing/invalid JWT token
- `500 Internal Server Error` — DB error

#### `PUT /api/umkm/chatbot/config`

**Request:**
```json
{
  "tone": "formal",
  "temperature": 0.5,
  "business_hours_start": "09:00",
  "business_hours_end": "18:00",
  "escalation_keywords": ["hubungi manusia", "bicara owner"],
  "is_active": true
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Chatbot config updated successfully"
}
```

**Validation Rules:**
- `language` ∈ {`id`, `en`}
- `tone` ∈ {`friendly`, `formal`, `casual`, `professional`}
- `temperature` ∈ [0.0, 1.0]
- `max_tokens` ∈ [64, 4096]
- `max_context_messages` ∈ [1, 50]
- `rag_top_k` ∈ [1, 20]
- `rag_similarity_threshold` ∈ [0.0, 1.0]
- `business_hours_start` < `business_hours_end`
- `business_days` ⊆ {0,1,2,3,4,5,6}
- `escalation_keywords` non-empty if `escalation_enabled = true`
- `channels_enabled` non-empty (minimal 1 channel)

**Error Cases:**
- `400 Bad Request` — Validation error (e.g., invalid tone, business_hours_start >= business_hours_end)
- `401 Unauthorized` — Missing/invalid JWT token

#### `POST /api/umkm/chatbot/config/test`

**Request:**
```json
{
  "message": "Halo, apakah masih buka?"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "reply": "Halo! Ya, kami masih buka sampai jam 22:00 malam ini. Ada yang bisa saya bantu?",
    "would_escalate": false
  }
}
```

**Backend Logic:**
1. Load current tenant config
2. Build system_prompt from config (same logic as chatbot runtime)
3. Call AI Gateway `/v1/chat/completions` with system_prompt + user message
4. Check if reply contains escalation keywords → `would_escalate: true`

**Error Cases:**
- `400 Bad Request` — Missing `message` field
- `401 Unauthorized` — Missing/invalid JWT token
- `502 Bad Gateway` — AI Gateway error

### Redis Keys

| Key | Value | TTL | Usage |
|:----|:------|:----|:------|
| `chatbot:config:{tenant_id}` | JSON (config object) | 300s (5 min) | Chatbot runtime config cache |

**Cache Invalidation:**
- On `PUT /chatbot/config` → DELETE `chatbot:config:{tenant_id}`
- Chatbot service auto-refetch on cache MISS

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (apps/umkm/accounting):**
```go
// chatbot_config_test.go
func TestHandleChatbotConfig_AutoCreate(t *testing.T) {
    // Mock DB: config not exists
    // GET /chatbot/config
    // Expect: INSERT default config, return 200 OK
}

func TestHandleChatbotConfig_PartialUpdate(t *testing.T) {
    // PUT with {"tone": "formal", "temperature": 0.5}
    // Expect: UPDATE only those 2 fields, other fields unchanged
}

func TestHandleChatbotConfig_ValidationError(t *testing.T) {
    // PUT with {"temperature": 1.5}  // out of range
    // Expect: 400 Bad Request
}

func TestHandleChatbotTest_Success(t *testing.T) {
    // POST /chatbot/config/test {"message": "halo"}
    // Mock AI Gateway response
    // Expect: 200 OK with reply
}
```

**Chatbot Service (apps/umkm/chatbot):**
```go
// chatbot_config_runtime_test.go
func TestBuildSystemPrompt_LoadFromCache(t *testing.T) {
    // Mock Redis cache HIT
    // Expect: no HTTP call to accounting service
}

func TestBuildSystemPrompt_OutsideBusinessHours(t *testing.T) {
    // Config: business_hours 08:00-22:00, current time 23:00
    // Expect: return outside_hours_message, no LLM call
}

func TestBuildSystemPrompt_EscalationKeyword(t *testing.T) {
    // Message contains "hubungi admin"
    // Expect: would_escalate = true
}
```

### Integration Tests

```bash
# 1. GET config (auto-create for new tenant)
curl -X GET http://localhost:8201/api/umkm/chatbot/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
# → 200 OK, default config

# 2. PUT config (partial update)
curl -X PUT http://localhost:8201/api/umkm/chatbot/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "tone": "formal",
    "temperature": 0.5,
    "business_hours_start": "09:00",
    "business_hours_end": "18:00",
    "is_active": true
  }'
# → 200 OK

# 3. Test chatbot (preview)
curl -X POST http://localhost:8201/api/umkm/chatbot/config/test \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"message":"Halo, masih buka?"}'
# → 200 OK with reply + would_escalate flag

# 4. Verify Redis cache
redis-cli GET "chatbot:config:$TENANT_ID"
# → JSON config cached

# 5. Chatbot runtime (outside business hours)
# Send WA message at 23:00 (outside 09:00-18:00)
# → Instant reply with outside_hours_message (no LLM call)
```

### Manual Testing Checklist

- [ ] Fresh tenant GET config → auto-create default config
- [ ] PUT config with valid values → update success, toast feedback
- [ ] PUT config with invalid temperature (> 1.0) → 400 error, toast error message
- [ ] POST test with message "halo" → preview reply shown
- [ ] POST test with message "hubungi admin" → would_escalate = true
- [ ] Chatbot receive message at 08:30 (inside hours) → LLM reply
- [ ] Chatbot receive message at 23:00 (outside hours) → outside_hours_message (no LLM)
- [ ] Set is_active = false → chatbot always return outside_hours_message
- [ ] Wizard Step 1 → fill Bahasa, Tone → preview panel update
- [ ] Wizard Step 2 → set business hours 09:00-18:00 → preview update
- [ ] Wizard Step 3 → custom welcome message → Save & Activate → redirect dashboard
- [ ] Sidebar "AI CS" menu → navigate to `/chatbot-config`
- [ ] Settings "Setup/Edit" link → navigate to `/chatbot-config`
- [ ] First run flow: complete onboarding → auto-redirect `/chatbot-config?first_run=1` with banner

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Chatbot config loaded", 
  "tenant_id", tenantID,
  "source", "cache",  // or "db"
  "is_active", config.IsActive)

slog.Info("Outside business hours", 
  "tenant_id", tenantID,
  "current_time", currentTime,
  "business_hours", fmt.Sprintf("%s-%s", config.Start, config.End))
```

**Metrics to track:**
- Config cache hit rate (target: >90%)
- Outside hours message count vs LLM call count (cost savings)
- Escalation keyword match rate
- Average wizard completion time (detect UX friction)

**Alerts:**
- Cache hit rate < 70% → investigate Redis connectivity or TTL misconfiguration
- LLM call count spike outside business hours → business_hours config not honored

---

## 🚀 Rollout Plan

### Phase 1: Backend Handlers (Done ✅)
- Deploy `apps/umkm/accounting` dengan 3 endpoints (GET/PUT/POST)
- Test via cURL → verify idempotent auto-create, partial update, validation

### Phase 2: Chatbot Integration (Done ✅)
- Update `buildSystemPrompt()` di chatbot service → load config dari accounting API
- Redis cache integration (5 min TTL)
- Business hours check → skip LLM if outside hours
- Escalation keyword matching

### Phase 3: Frontend Wizard (Done ✅)
- Deploy umkm-web dengan `ChatbotConfig.vue` (3-step wizard)
- Entry points: Sidebar, Settings, First run redirect
- Test: end-to-end flow from onboarding → wizard → save → chatbot runtime

### Phase 4: Analytics Dashboard (Future)
- Track wizard completion funnel (Step 1 → Step 2 → Step 3 → Save)
- Track cost savings (LLM calls avoided via business hours)
- A/B test default tone (friendly vs formal) → measure escalation rate

### Rollback
- **Phase 1 rollback:** Remove handlers dari accounting routing → 404
- **Phase 2 rollback:** Revert chatbot `buildSystemPrompt()` → use hardcoded default config
- **Emergency:** Set `is_active = false` untuk all tenants via DB migration → disable all chatbots

---

## 🔮 Future Enhancements (Out of Scope)

- **Multi-Language System Prompt:** Support EN, ID, CN system prompts (auto-switch based on `language` config)
- **A/B Test Config:** Tenant bisa run A/B test untuk 2 different tones/temperatures → measure customer satisfaction
- **Advanced Escalation:** Escalate based on sentiment analysis (negative sentiment → auto-escalate) bukan hanya keyword matching
- **Chatbot Analytics Dashboard:** Tenant dashboard dengan metrics: response time, escalation rate, customer satisfaction score
- **Voice & Multimodal Config:** Extend wizard untuk configure vision/audio AI (F025 integration)

---

## 📚 References

- [F007: Chatbot with RAG](../FEATURE_MAP.md) — Base chatbot system + RAG integration
- [F015: Onboarding Modal Activation](../FEATURE_MAP.md) — First run flow → redirect to wizard
- [MiniMax API Docs](https://www.minimaxi.com/document/guides/chat-model/V2) — LLM provider integration
- [Redis Caching Best Practices](https://redis.io/docs/manual/client-side-caching/) — Cache invalidation pattern

---

## 📝 Notes & Decisions

**2026-06-13:** Decision: Backend di `apps/umkm/accounting` (bukan `chatbot`) karena accounting sudah jadi hub konfigurasi tenant. Chatbot fokus ke runtime execution, pengurangan coupling.  
**2026-06-13:** Cache TTL 5 menit → balance antara freshness (config baru apply max 5 menit) dan performance (reduce DB hit). Config jarang berubah (setup once, edit occasionally).  
**2026-06-13:** Business hours check di chatbot runtime → skip LLM call di luar jam → significant cost savings (50-70% less API calls for typical UMKM with 08:00-22:00 hours).  
**2026-06-13:** `escalation_keywords` config replace hardcoded `[FORWARD_TO_ADMIN]` marker → tenant customize keywords sesuai bahasa/kultur mereka.  
**2026-06-13:** Default `is_active = false` untuk new tenant → force user complete wizard before chatbot goes live (prevent misconfigured bot serving customers).
