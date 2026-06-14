# F025: Tier Restrictions Overhaul + AI Multimodal — Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Sejajarkan tier restrictions (single source of truth), tambah quota enforcement + counter middleware, extend AI Gateway dengan vision/STT/TTS/image-gen endpoints.

**Architecture:**
- **Phase 1 (Align):** DB `plan_features` table jadi single source of truth. Rewrite `Plans` map di Go jadi DB-driven + cache.
- **Phase 2 (Counter):** `quota_counters` table + atomic increment middleware di API Gateway. 402 Payment Required saat exceeded.
- **Phase 3 (Multimodal):** AI Gateway extend dengan `/v1/vision`, `/v1/audio/transcribe`, `/v1/audio/speak`, `/v1/image/generate`. Chatbot deteksi message type WA → route ke endpoint sesuai.

**Tech Stack:** Go 1.25, PostgreSQL 16 (pgx/v5), Redis 7 (go-redis/v9), whatsmeow (WA), openai-go SDK, Vue 3 FE.

---

## Open Decisions (Pending Owner)

Vendor untuk multimodal:
- **Vision:** MiniMax-M3-Vision (asumsi — perlu konfirmasi API capability)
- **STT:** Whisper large-v3 (openai-whisper) atau self-host via whisper.cpp
- **TTS:** Edge TTS (free, Microsoft) atau ElevenLabs (paid, quality)
- **Image Gen:** MiniMax-Image-1 (asumsi) atau DALL-E 3 / Stable Diffusion

**Default asumsi plan ini:** MiniMax-M3 untuk semua (text/vision/image), Whisper self-hosted untuk STT, Edge TTS untuk TTS. Owner bisa override per task.

Single source of truth: **Option B** (DB `plan_features` table). `Plans` map di Go dihapus, di-replace dengan function `GetPlanFeatures(tenantID) → FeatureFlags` yang baca DB dengan Redis cache.

---

# Phase 1: Align Source of Truth (~6 tasks)

**Objective:** Hapus `Plans` map hardcoded di Go. Baca dari DB `plan_features` via `GetPlanFeatures(tenantID)`. Tambah numeric quota columns.

### Task 1.1: Tambah columns numerik ke `plan_features`

**Files:**
- Create: `shared/migrations/000039_plan_features_numeric.up.sql`
- Create: `shared/migrations/000039_plan_features_numeric.down.sql`

**Step 1: Write migration**

```sql
-- 000039_plan_features_numeric.up.sql
-- Add numeric quota columns to plan_features for runtime enforcement
-- (Previously: feature_value was free-form VARCHAR(50), now structured)

ALTER TABLE plan_features
    ADD COLUMN IF NOT EXISTS max_users INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_transactions INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_text INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_vision INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_ai_audio_minutes INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_image_gen INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_products INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_customers INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_storage_mb INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS api_rate_limit_per_min INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS data_retention_months INT DEFAULT 0;
```

**Step 2: Write down**

```sql
-- 000039_plan_features_numeric.down.sql
ALTER TABLE plan_features
    DROP COLUMN IF EXISTS max_users,
    DROP COLUMN IF EXISTS max_transactions,
    DROP COLUMN IF EXISTS max_ai_text,
    DROP COLUMN IF EXISTS max_ai_vision,
    DROP COLUMN IF EXISTS max_ai_audio_minutes,
    DROP COLUMN IF EXISTS max_image_gen,
    DROP COLUMN IF EXISTS max_products,
    DROP COLUMN IF EXISTS max_customers,
    DROP COLUMN IF EXISTS max_storage_mb,
    DROP COLUMN IF EXISTS api_rate_limit_per_min,
    DROP COLUMN IF EXISTS data_retention_months;
```

**Step 3: Verify**
```bash
ls shared/migrations/000039_*
```

**Step 4: Commit**
```bash
cd ~/Desktop/dev/core_project
git add shared/migrations/000039_*
git commit -m "feat(db): add numeric quota columns to plan_features"
```

---

### Task 1.2: Seed numeric values untuk lite/pro/ultimate

**Files:**
- Create: `shared/migrations/000040_seed_plan_features_numeric.up.sql`
- Create: `shared/migrations/000040_seed_plan_features_numeric.down.sql`

**Step 1: Write seed**

```sql
-- 000040_seed_plan_features_numeric.up.sql
-- Single source of truth for all tier quotas (post F024)

-- LITE (Rp 150rb/bln)
UPDATE plan_features SET
    max_users = 3, max_transactions = 1000, max_ai_text = 250,
    max_ai_vision = 0, max_ai_audio_minutes = 0, max_image_gen = 0,
    max_products = 100, max_customers = 500, max_storage_mb = 1000,
    api_rate_limit_per_min = 60, data_retention_months = 12
WHERE plan_id = 'lite';

-- PRO (Rp 450rb/bln)
UPDATE plan_features SET
    max_users = 10, max_transactions = 10000, max_ai_text = 5000,
    max_ai_vision = 50, max_ai_audio_minutes = 0, max_image_gen = 0,
    max_products = 1000, max_customers = 5000, max_storage_mb = 10000,
    api_rate_limit_per_min = 300, data_retention_months = 36
WHERE plan_id = 'pro';

-- ULTIMATE (Rp 1.5jt/bln)
UPDATE plan_features SET
    max_users = -1, max_transactions = -1, max_ai_text = -1,
    max_ai_vision = 500, max_ai_audio_minutes = 60, max_image_gen = 30,
    max_products = -1, max_customers = -1, max_storage_mb = -1,
    api_rate_limit_per_min = 1000, data_retention_months = 60
WHERE plan_id = 'ultimate';
```

**Step 2: Write down** — inverse, set all to 0.

**Step 3: Commit**
```bash
git add shared/migrations/000040_*
git commit -m "feat(db): seed numeric quotas for lite/pro/ultimate"
```

---

### Task 1.3: Tambah `PlanFeaturesRow` struct + loader di shared SDK

**Files:**
- Create: `shared/sdk/auth/plan_features.go`
- Create: `shared/sdk/auth/plan_features_test.go`

**Step 1: Write failing test**

```go
// shared/sdk/auth/plan_features_test.go
package auth

import (
    "context"
    "testing"
)

func TestGetPlanFeatures_DBRead(t *testing.T) {
    // Skip if no DB
    if db == nil {
        t.Skip("DB not initialized")
    }
    // Setup test tenant
    ctx := context.Background()
    plan, err := GetPlanFeatures(ctx, "test-tenant-id")
    if err != nil {
        t.Fatalf("GetPlanFeatures: %v", err)
    }
    if plan.Tier == "" {
        t.Error("expected tier, got empty")
    }
}

func TestPlanFeatures_UnlimitedSentinel(t *testing.T) {
    p := PlanFeaturesRow{MaxUsers: -1}
    if !p.IsUnlimited("max_users") {
        t.Error("-1 should mean unlimited")
    }
    p = PlanFeaturesRow{MaxUsers: 10}
    if p.IsUnlimited("max_users") {
        t.Error("10 should not be unlimited")
    }
}
```

**Step 2: Run test** — expected FAIL (function not defined)

**Step 3: Implement**

```go
// shared/sdk/auth/plan_features.go
package auth

import (
    "context"
    "time"

    "core_project/shared/sdk/cache"
)

type PlanFeaturesRow struct {
    Tier                    string `json:"tier"`
    PlanName                string `json:"plan_name"`
    MaxUsers                int    `json:"max_users"`
    MaxTransactions         int    `json:"max_transactions"`
    MaxAIText               int    `json:"max_ai_text"`
    MaxAIVision             int    `json:"max_ai_vision"`
    MaxAIAudioMinutes       int    `json:"max_ai_audio_minutes"`
    MaxImageGen             int    `json:"max_image_gen"`
    MaxProducts             int    `json:"max_products"`
    MaxCustomers            int    `json:"max_customers"`
    MaxStorageMB            int    `json:"max_storage_mb"`
    APIRateLimitPerMin      int    `json:"api_rate_limit_per_min"`
    DataRetentionMonths     int    `json:"data_retention_months"`
    HasAccounting           bool   `json:"has_accounting"`
    HasPOS                  bool   `json:"has_pos"`
    HasChatbot              bool   `json:"has_chatbot"`
    HasAI                   bool   `json:"has_ai"`
    HasInventory            bool   `json:"has_inventory"`
    HasReports              bool   `json:"has_reports"`
    HasMultiUser            bool   `json:"has_multi_user"`
    HasAPIAccess            bool   `json:"has_api_access"`
    HasAdvancedReport       bool   `json:"has_advanced_report"`
    HasCustomBranding       bool   `json:"has_custom_branding"`
    HasPrioritySupport      bool   `json:"has_priority_support"`
}

func (p PlanFeaturesRow) IsUnlimited(field string) bool {
    switch field {
    case "max_users":
        return p.MaxUsers == -1
    case "max_transactions":
        return p.MaxTransactions == -1
    case "max_ai_text":
        return p.MaxAIText == -1
    case "max_ai_vision":
        return p.MaxAIVision == -1
    case "max_ai_audio_minutes":
        return p.MaxAIAudioMinutes == -1
    case "max_image_gen":
        return p.MaxImageGen == -1
    case "max_products":
        return p.MaxProducts == -1
    case "max_customers":
        return p.MaxCustomers == -1
    case "max_storage_mb":
        return p.MaxStorageMB == -1
    }
    return false
}

// GetPlanFeatures reads from DB with Redis cache (5 min TTL)
func GetPlanFeatures(ctx context.Context, tenantID string) (PlanFeaturesRow, error) {
    cacheKey := "plan_features:" + tenantID
    if cache.Client != nil {
        if val, err := cache.Client.Get(ctx, cacheKey).Result(); err == nil && val != "" {
            // Parse JSON, return
            var row PlanFeaturesRow
            if err := jsonUnmarshal(val, &row); err == nil {
                return row, nil
            }
        }
    }
    // DB read
    row, err := queryPlanFeaturesFromDB(ctx, tenantID)
    if err != nil {
        return PlanFeaturesRow{Tier: "inactive"}, err
    }
    if cache.Client != nil {
        if val, err := jsonMarshal(row); err == nil {
            cache.Client.Set(ctx, cacheKey, val, 5*time.Minute)
        }
    }
    return row, nil
}

// Implementations below require DB connection (set in init)
var (
    db interface {
        QueryRow(ctx context.Context, sql string, args ...interface{}) Row
    }
)

type Row interface {
    Scan(dest ...interface{}) error
}

func queryPlanFeaturesFromDB(ctx context.Context, tenantID string) (PlanFeaturesRow, error) {
    // Stub: actual implementation in main app context
    return PlanFeaturesRow{Tier: "inactive"}, nil
}

// jsonMarshal/JsonUnmarshal helpers (use stdlib in actual impl)
func jsonMarshal(v interface{}) (string, error) { return "", nil }
func jsonUnmarshal(s string, v interface{}) error { return nil }
```

**Note:** Implementasi di atas adalah skeleton. Real implementation pass DB connection dari caller. Lihat existing pattern di `quota.go` yang baca Redis.

**Step 4: Run test** — expected PASS (or skip if DB nil)

**Step 5: Commit**
```bash
git add shared/sdk/auth/plan_features.go shared/sdk/auth/plan_features_test.go
git commit -m "feat(sdk): add PlanFeaturesRow + GetPlanFeatures with Redis cache"
```

---

### Task 1.4: Update `auth.GetPlan()` return `PlanFeaturesRow` bukan `PlanTier`

**Files:**
- Modify: `shared/sdk/auth/quota.go:64-71`
- Modify: `shared/sdk/auth/auth_test.go` (semua test yang pakai `GetPlan()`)

**Step 1: Modify function**

```go
// shared/sdk/auth/quota.go
func GetPlan(tenantID string) PlanFeaturesRow {
    row, err := GetPlanFeatures(context.Background(), tenantID)
    if err != nil {
        return PlanFeaturesRow{Tier: "inactive"}
    }
    return row
}
```

**Step 2: Update tests** — ganti semua `plan.MaxUsers` jadi `plan.MaxUsers` (sama, type aman). Update `plan.Tier != "inactive"` checks.

**Step 3: Run tests**
```bash
go test -count=1 -short ./shared/sdk/auth/...
```
Expected: PASS (cached unless files changed)

**Step 4: Commit**
```bash
git add shared/sdk/auth/quota.go shared/sdk/auth/auth_test.go
git commit -m "refactor(sdk): GetPlan returns PlanFeaturesRow (DB-driven)"
```

---

### Task 1.5: Hapus `Plans` map hardcoded

**Files:**
- Modify: `shared/sdk/auth/quota.go:37-45`

**Step 1: Delete Plans map** (lines 37-45)

**Step 2: Verify no usages** — search `Plans[`
```bash
grep -rn "Plans\[" --include="*.go" .
```
Expected: only in quota.go (the map definition) and possibly tests. Update tests.

**Step 3: Commit**
```bash
git add shared/sdk/auth/quota.go
git commit -m "refactor(sdk): remove hardcoded Plans map (DB-driven)"
```

---

### Task 1.6: Update semua reference `plan.MaxTransactions` dll ke `PlanFeaturesRow`

**Files:**
- Modify: `shared/sdk/auth/quota.go` (CheckQuota, QuotaMiddleware, RequireFeature)
- Modify: all handlers yang baca `auth.GetPlan(tenantID)` (grep dulu)

**Step 1: Grep usages**
```bash
grep -rn "auth\.GetPlan\|auth\.Plans\[" --include="*.go" .
```

**Step 2: Update each** — ganti `plan.MaxTransactions` → `plan.MaxTransactions` (sama), `plan.Features.HasPOS` → `plan.HasPOS` (flatten).

**Step 3: Run go vet + tests**
```bash
go vet ./shared/...
go test -count=1 -short ./shared/...
```

**Step 4: Commit**
```bash
git add -A
git commit -m "refactor(sdk): migrate all auth.GetPlan() callers to PlanFeaturesRow"
```

---

# Phase 2: Quota Counter + Middleware (~10 tasks)

**Objective:** Track per-tenant per-modality usage. Atomic increment. 402 response saat exceeded.

### Task 2.1: Migration `quota_counters` table

**Files:**
- Create: `shared/migrations/000041_quota_counters.up.sql`
- Create: `shared/migrations/000041_quota_counters.down.sql`

**Step 1: Write**

```sql
-- 000041_quota_counters.up.sql
-- Per-tenant per-feature counter with period key (monthly)

CREATE TABLE IF NOT EXISTS quota_counters (
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_yyyymm CHAR(6) NOT NULL,           -- '202606'
    feature_key  VARCHAR(50) NOT NULL,        -- 'ai_text', 'ai_vision', 'ai_audio_stt', 'ai_audio_tts', 'image_gen', 'ocr_scans', 'chatbot_messages'
    count        BIGINT NOT NULL DEFAULT 0,
    reset_at     TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, period_yyyymm, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_quota_counters_period ON quota_counters(period_yyyymm);
CREATE INDEX IF NOT EXISTS idx_quota_counters_reset ON quota_counters(reset_at);
```

**Step 2: Down**

```sql
-- 000041_quota_counters.down.sql
DROP TABLE IF EXISTS quota_counters;
```

**Step 3: Commit**
```bash
git add shared/migrations/000041_*
git commit -m "feat(db): add quota_counters table for per-feature tracking"
```

---

### Task 2.2: Atomic counter helpers di shared SDK

**Files:**
- Create: `shared/sdk/auth/quota_counter.go`
- Create: `shared/sdk/auth/quota_counter_test.go`

**Step 1: Write test**

```go
// shared/sdk/auth/quota_counter_test.go
package auth

import (
    "context"
    "testing"
)

func TestIncrementQuota_NoDB(t *testing.T) {
    // Should not panic
    IncrementQuota(context.Background(), "t1", "ai_text", 1)
}

func TestCheckQuota_NoDB(t *testing.T) {
    ok, used, limit := CheckQuotaCounter(context.Background(), "t1", "ai_text")
    if !ok {
        t.Error("expected ok=true when no DB")
    }
    if used != 0 || limit != -1 {
        t.Errorf("expected used=0, limit=-1, got used=%d, limit=%d", used, limit)
    }
}
```

**Step 2: Implement**

```go
// shared/sdk/auth/quota_counter.go
package auth

import (
    "context"
    "fmt"
    "time"

    "core_project/shared/sdk/cache"
)

const counterKeyPrefix = "quota_counter:"

func currentPeriod() string {
    return time.Now().UTC().Format("200601")
}

func currentPeriodEnd() time.Time {
    now := time.Now().UTC()
    return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

// IncrementQuota increments the counter for a tenant+feature in current period.
// Returns (currentCount, limit, error). If no DB wired, returns (0, -1, nil).
func IncrementQuota(ctx context.Context, tenantID, feature string, delta int) (int64, int, error) {
    period := currentPeriod()
    redisKey := fmt.Sprintf("%s%s:%s:%s", counterKeyPrefix, tenantID, period, feature)

    // Redis fast path
    var count int64
    if cache.Client != nil {
        newCount, err := cache.Client.IncrBy(ctx, redisKey, int64(delta)).Result()
        if err == nil {
            cache.Client.ExpireAt(ctx, redisKey, currentPeriodEnd().Add(48*time.Hour))
            count = newCount
        }
    }

    // DB write (async best-effort if available)
    if err := persistQuotaCounter(ctx, tenantID, period, feature, count); err != nil {
        return count, -1, err
    }

    // Get limit from plan features
    plan, _ := GetPlanFeatures(ctx, tenantID)
    limit := getFeatureLimit(plan, feature)
    return count, limit, nil
}

// CheckQuotaCounter returns (ok, used, limit)
func CheckQuotaCounter(ctx context.Context, tenantID, feature string) (bool, int64, int) {
    period := currentPeriod()
    redisKey := fmt.Sprintf("%s%s:%s:%s", counterKeyPrefix, tenantID, period, feature)

    var used int64
    if cache.Client != nil {
        if val, err := cache.Client.Get(ctx, redisKey).Int64(); err == nil {
            used = val
        }
    }
    plan, _ := GetPlanFeatures(ctx, tenantID)
    limit := getFeatureLimit(plan, feature)
    if limit == -1 {
        return true, used, limit
    }
    return used < int64(limit), used, limit
}

func getFeatureLimit(p PlanFeaturesRow, feature string) int {
    switch feature {
    case "ai_text":
        return p.MaxAIText
    case "ai_vision":
        return p.MaxAIVision
    case "ai_audio_stt":
        return p.MaxAIAudioMinutes
    case "ai_audio_tts":
        return p.MaxAIAudioMinutes
    case "image_gen":
        return p.MaxImageGen
    case "ocr_scans":
        if p.Tier == "ultimate" { return -1 }
        if p.Tier == "pro" { return 500 }
        if p.Tier == "lite" { return 50 }
        return 0
    case "chatbot_messages":
        if p.Tier == "ultimate" { return -1 }
        if p.Tier == "pro" { return 1000 }
        if p.Tier == "lite" { return 250 }
        return 0
    }
    return 0
}

// persistQuotaCounter writes to DB (stub; real impl needs DB connection)
func persistQuotaCounter(ctx context.Context, tenantID, period, feature string, count int64) error {
    // Real impl: INSERT ... ON CONFLICT (tenant_id, period_yyyymm, feature_key) DO UPDATE SET count = $3
    return nil
}
```

**Step 3: Run tests + commit**
```bash
go test -count=1 -short ./shared/sdk/auth/...
git add shared/sdk/auth/quota_counter.go shared/sdk/auth/quota_counter_test.go
git commit -m "feat(sdk): add IncrementQuota + CheckQuotaCounter with Redis fast path"
```

---

### Task 2.3: Quota enforcement middleware

**Files:**
- Create: `shared/sdk/auth/quota_mw.go`
- Create: `shared/sdk/auth/quota_mw_test.go`

**Step 1: Test**

```go
// shared/sdk/auth/quota_mw_test.go
package auth

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestQuotaMiddleware_PassesWhenUnderLimit(t *testing.T) {
    called := false
    h := QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
    }))
    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, "t1"))
    h.ServeHTTP(httptest.NewRecorder(), req)
    if !called {
        t.Error("handler should be called when under limit")
    }
}
```

**Step 2: Implement**

```go
// shared/sdk/auth/quota_mw.go
package auth

import (
    "context"
    "net/http"
    "encoding/json"

    "core_project/shared/sdk/response"
)

func QuotaMiddlewareFeature(feature string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID, ok := r.Context().Value(TenantIDKey).(string)
            if !ok || tenantID == "" {
                response.Error(w, http.StatusUnauthorized, "Tenant context missing", nil)
                return
            }
            ok, used, limit := CheckQuotaCounter(r.Context(), tenantID, feature)
            if !ok {
                w.Header().Set("X-Quota-Feature", feature)
                w.Header().Set("X-Quota-Used", itoa(used))
                w.Header().Set("X-Quota-Limit", itoa(limit))
                response.Error(w, http.StatusPaymentRequired, "Quota exceeded for feature: "+feature+". Upgrade your plan.", map[string]interface{}{
                    "feature": feature,
                    "used":    used,
                    "limit":   limit,
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func itoa(n int64) string {
    return jsonNumber(n)
}

func jsonNumber(n int64) string {
    b, _ := json.Marshal(n)
    return string(b)
}
```

**Step 3: Tests + commit**
```bash
go test -count=1 -short ./shared/sdk/auth/...
git add shared/sdk/auth/quota_mw.go shared/sdk/auth/quota_mw_test.go
git commit -m "feat(sdk): add QuotaMiddlewareFeature with 402 response"
```

---

### Task 2.4: Wire middleware to AI Gateway endpoints

**Files:**
- Modify: `services/ai-gateway/main.go:110-116`

**Step 1: Wrap handlers**

```go
// services/ai-gateway/main.go
mux.HandleFunc("/v1/chat", auth.QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(handleChat)))
mux.HandleFunc("/v1/chat/stream", auth.QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(handleChatStream)))
mux.HandleFunc("/v1/embeddings", auth.QuotaMiddlewareFeature("ai_text")(http.HandlerFunc(handleEmbeddings)))
// Vision/Audio/Image added in Phase 3
```

**Step 2: Increment counter after successful LLM call**

Inside `handleChat` (around line 280 where `writeJSON` happens):
```go
// After successful response
auth.IncrementQuota(r.Context(), tenantID, "ai_text", 1)
```

**Step 3: Commit**
```bash
git add services/ai-gateway/main.go
git commit -m "feat(ai): wire quota middleware to text endpoints"
```

---

### Task 2.5: Wire middleware to chatbot (text only first)

**Files:**
- Modify: `apps/umkm/chatbot/main.go` (main handler)

**Step 1: After successful chat processing, increment**
```go
// At end of processChatJob or equivalent
auth.IncrementQuota(ctx, job.TenantID, "chatbot_messages", 1)
```

**Step 2: Commit**
```bash
git add apps/umkm/chatbot/main.go
git commit -m "feat(chatbot): increment chatbot_messages counter per job"
```

---

### Task 2.6: Reset counter cron worker

**Files:**
- Modify: `services/subscription-worker/main.go` (atau add ke existing scheduler)

**Step 1: Tambah cron job** — `0 0 1 * *` (every 1st of month at 00:00) → loop counter lama, archive atau delete.

**Step 2: Implement**
```go
// At month roll: delete counters older than 60 days
result, err := DB.Exec(ctx, "DELETE FROM quota_counters WHERE reset_at < NOW() - INTERVAL '60 days'")
```

**Step 3: Commit**
```bash
git add services/subscription-worker/main.go
git commit -m "feat(worker): cron job to archive old quota_counters"
```

---

### Task 2.7: Soft-warn notification at 80% usage

**Files:**
- Modify: `services/notification-service/main.go` (add new function)
- Or: create `services/notification-service/quota_warn.go`

**Step 1: Function `CheckAndNotifyQuota(ctx, tenantID, feature)`**

**Step 2: Call after `IncrementQuota` returns count >= 0.8 * limit

**Step 3: Commit**
```bash
git add services/notification-service/
git commit -m "feat(notification): warn tenant at 80% quota usage"
```

---

### Task 2.8: Superadmin quota dashboard endpoint

**Files:**
- Modify: `services/billing-service/main.go` (add `GET /admin/quota/:tenant_id`)

**Step 1: Handler** — return current counters + plan limits for given tenant.

**Step 2: Test + commit**
```bash
git add services/billing-service/main.go
git commit -m "feat(billing): superadmin endpoint to view tenant quota usage"
```

---

### Task 2.9: FE update — quota usage display

**Files:**
- Modify: `frontend/umkm-web/src/components/Settings.vue` (add quota section)
- Modify: `frontend/umkm-web/src/api.ts` (add `getQuotaUsage()`)

**Step 1: Add usage bar component showing X/Y used with progress bar**

**Step 2: Test FE build**
```bash
cd frontend/umkm-web && npm run build
```

**Step 3: Commit**
```bash
git add frontend/umkm-web/
git commit -m "feat(fe): display quota usage in Settings page"
```

---

### Task 2.10: Update FEATURE_MAP F025 + integration test

**Files:**
- Modify: `docs/FEATURE_MAP.md` (F025 status: ⏳ Planning → 🔨 In Progress → ✅ Done)
- Create: `tests/integration/quota_test.go`

**Step 1: Update F025 status to ✅ Done**

**Step 2: Integration test** — spin up test tenant, hit `/v1/chat` 5x, verify 6th returns 402.

**Step 3: Commit**
```bash
git add docs/FEATURE_MAP.md tests/
git commit -m "test: add quota enforcement integration test"
```

---

# Phase 3: AI Multimodal (~15 tasks)

**Objective:** Tambah vision/STT/TTS/image-gen endpoints di AI Gateway. Update chatbot untuk handle WA image/audio messages.

### Task 3.1: Add `/v1/vision` endpoint

**Files:**
- Create: `services/ai-gateway/handlers/vision.go`
- Modify: `services/ai-gateway/main.go` (register route)

**Step 1: Handler signature**

```go
// services/ai-gateway/handlers/vision.go
type VisionRequest struct {
    TenantID  string `json:"tenant_id"`
    ImageURL  string `json:"image_url"`  // HTTPS URL or base64 data URI
    Prompt    string `json:"prompt"`
    Model     string `json:"model"`      // default "MiniMax-M3-Vision"
}

func handleVision(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
        return
    }
    var req VisionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }
    // Quota check (already in middleware if wrapped)
    // Call MiniMax-M3-Vision API
    response, err := callVisionAPI(req)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
        return
    }
    auth.IncrementQuota(r.Context(), req.TenantID, "ai_vision", 1)
    writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: response})
}
```

**Step 2: Stub `callVisionAPI`** — return mock for now, real impl nanti.

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/vision.go services/ai-gateway/main.go
git commit -m "feat(ai): add /v1/vision endpoint stub"
```

---

### Task 3.2: Add `/v1/audio/transcribe` endpoint (STT)

**Files:**
- Create: `services/ai-gateway/handlers/audio.go`
- Modify: `services/ai-gateway/main.go`

**Step 1: Handler**

```go
type TranscribeRequest struct {
    TenantID  string `json:"tenant_id"`
    AudioURL  string `json:"audio_url"`  // HTTPS URL to audio file
    Language  string `json:"language"`   // "id", "en" - default "id"
}

func handleTranscribe(w http.ResponseWriter, r *http.Request) {
    // Decode, validate, call Whisper, return text
    // IncrementQuota(tenantID, "ai_audio_stt", durationMinutes)
}
```

**Step 2: Stub Whisper client call**

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/audio.go services/ai-gateway/main.go
git commit -m "feat(ai): add /v1/audio/transcribe endpoint stub"
```

---

### Task 3.3: Add `/v1/audio/speak` endpoint (TTS)

**Files:**
- Modify: `services/ai-gateway/handlers/audio.go`

**Step 1: Handler**

```go
type SpeakRequest struct {
    TenantID  string `json:"tenant_id"`
    Text      string `json:"text"`
    Voice     string `json:"voice"`      // "id-ID-ArdiNeural" etc
    Format    string `json:"format"`     // "mp3", "ogg"
}

func handleSpeak(w http.ResponseWriter, r *http.Request) {
    // Call Edge TTS or ElevenLabs, return audio bytes
    // IncrementQuota(tenantID, "ai_audio_tts", charCount/1000)
}
```

**Step 2: Commit**
```bash
git add services/ai-gateway/handlers/audio.go
git commit -m "feat(ai): add /v1/audio/speak endpoint stub"
```

---

### Task 3.4: Add `/v1/image/generate` endpoint

**Files:**
- Create: `services/ai-gateway/handlers/image.go`
- Modify: `services/ai-gateway/main.go`

**Step 1: Handler**

```go
type GenerateImageRequest struct {
    TenantID  string `json:"tenant_id"`
    Prompt    string `json:"prompt"`
    Size      string `json:"size"`      // "1024x1024", "1024x1792", "1792x1024"
    N         int    `json:"n"`          // 1-4
}

func handleGenerateImage(w http.ResponseWriter, r *http.Request) {
    // Call MiniMax-Image-1 or DALL-E, return image URL
    // IncrementQuota(tenantID, "image_gen", 1)
}
```

**Step 2: Commit**
```bash
git add services/ai-gateway/handlers/image.go services/ai-gateway/main.go
git commit -m "feat(ai): add /v1/image/generate endpoint stub"
```

---

### Task 3.5: Wire quota middleware to all new endpoints

**Files:**
- Modify: `services/ai-gateway/main.go`

**Step 1: Wrap with QuotaMiddlewareFeature**
```go
mux.Handle("/v1/vision", auth.QuotaMiddlewareFeature("ai_vision")(http.HandlerFunc(handleVision)))
mux.Handle("/v1/audio/transcribe", auth.QuotaMiddlewareFeature("ai_audio_stt")(http.HandlerFunc(handleTranscribe)))
mux.Handle("/v1/audio/speak", auth.QuotaMiddlewareFeature("ai_audio_tts")(http.HandlerFunc(handleSpeak)))
mux.Handle("/v1/image/generate", auth.QuotaMiddlewareFeature("image_gen")(http.HandlerFunc(handleGenerateImage)))
```

**Step 2: Commit**
```bash
git add services/ai-gateway/main.go
git commit -m "feat(ai): wire quota middleware to multimodal endpoints"
```

---

### Task 3.6: Real MiniMax-M3-Vision integration

**Files:**
- Modify: `services/ai-gateway/handlers/vision.go`
- Modify: `shared/sdk/config/config.go` (add VisionAPIKey, VisionBaseURL)

**Step 1: Replace stub `callVisionAPI` with real openai-go client call to vision model**

**Step 2: Test with sample image**

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/vision.go shared/sdk/config/config.go
git commit -m "feat(ai): integrate MiniMax-M3-Vision API"
```

---

### Task 3.7: Real Whisper STT integration

**Files:**
- Modify: `services/ai-gateway/handlers/audio.go`

**Step 1: Use openai-go audio transcription endpoint OR self-host whisper.cpp**

**Step 2: Test with sample .ogg file**

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/audio.go
git commit -m "feat(ai): integrate Whisper STT"
```

---

### Task 3.8: Real Edge TTS integration

**Files:**
- Modify: `services/ai-gateway/handlers/audio.go`

**Step 1: Use github.com/rany2/edge-tts or call Microsoft Edge TTS endpoint**

**Step 2: Test with sample text**

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/audio.go
git commit -m "feat(ai): integrate Edge TTS"
```

---

### Task 3.9: Real MiniMax-Image-1 integration

**Files:**
- Modify: `services/ai-gateway/handlers/image.go`

**Step 1: Call image generation API**

**Step 2: Test with sample prompt**

**Step 3: Commit**
```bash
git add services/ai-gateway/handlers/image.go
git commit -m "feat(ai): integrate image generation API"
```

---

### Task 3.10: WA media download helper

**Files:**
- Create: `shared/sdk/mediaproxy/whatsapp.go`

**Step 1: Function `DownloadWhatsAppMedia(ctx, jid, messageID) ([]byte, error)` — use whatsmeow client**

**Step 2: Test stub**

**Step 3: Commit**
```bash
git add shared/sdk/mediaproxy/
git commit -m "feat(sdk): WhatsApp media download helper"
```

---

### Task 3.11: WA-gateway forward image messages to chatbot

**Files:**
- Modify: `services/wa-gateway/main.go` (handle image message event)
- Modify: `services/wa-cloud-api/main.go` (same)

**Step 1: Detect `*events.Message` with `ImageMessage` → forward to chatbot with media URL**

**Step 2: Commit**
```bash
git add services/wa-gateway/main.go services/wa-cloud-api/main.go
git commit -m "feat(wa): forward image messages to chatbot with media URL"
```

---

### Task 3.12: WA-gateway forward audio messages to chatbot

**Files:**
- Modify: `services/wa-gateway/main.go`
- Modify: `services/wa-cloud-api/main.go`

**Step 1: Same pattern as 3.11 but for `AudioMessage`**

**Step 2: Commit**
```bash
git add services/wa-gateway/main.go services/wa-cloud-api/main.go
git commit -m "feat(wa): forward audio messages to chatbot"
```

---

### Task 3.13: Chatbot vision handler (image→text→response)

**Files:**
- Modify: `apps/umkm/chatbot/main.go`

**Step 1: New code path for image messages**:
```go
if msg.Type == "image" {
    // 1. Download media via mediaproxy
    // 2. Upload to /v1/vision with prompt "Describe this image and identify any products"
    // 3. Process vision result as text through existing chatbot pipeline
    // 4. Reply text (or image generation if user asks for product photo)
}
```

**Step 2: Commit**
```bash
git add apps/umkm/chatbot/main.go
git commit -m "feat(chatbot): handle image messages via vision API"
```

---

### Task 3.14: Chatbot audio handler (voice→text→response)

**Files:**
- Modify: `apps/umkm/chatbot/main.go`

**Step 1: New code path for audio messages**:
```go
if msg.Type == "audio" {
    // 1. Download media
    // 2. Call /v1/audio/transcribe
    // 3. Use transcript as message in existing pipeline
    // 4. Optionally reply with TTS if user prefers voice
}
```

**Step 2: Commit**
```bash
git add apps/umkm/chatbot/main.go
git commit -m "feat(chatbot): handle voice notes via STT"
```

---

### Task 3.15: FE toggle multimodal di ChatbotConfig

**Files:**
- Modify: `frontend/umkm-web/src/components/ChatbotConfig.vue`
- Modify: `frontend/umkm-web/src/api.ts`

**Step 1: Add tab "AI Modality"**:
- Toggle "Enable vision (image messages)"
- Toggle "Enable voice reply (TTS)"
- Voice selection dropdown
- Show current usage / limit

**Step 2: Test FE build**
```bash
cd frontend/umkm-web && npm run build
```

**Step 3: Commit**
```bash
git add frontend/umkm-web/
git commit -m "feat(fe): ChatbotConfig multimodal toggles + usage display"
```

---

### Task 3.16: Update FEATURE_MAP F025 to ✅ Done

**Files:**
- Modify: `docs/FEATURE_MAP.md`

**Step 1: Update status to ✅ Done + add completion date**

**Step 2: Final test run**
```bash
go vet ./... && go test -count=1 -short ./shared/...
```

**Step 3: Commit**
```bash
git add docs/FEATURE_MAP.md
git commit -m "docs: mark F025 as Done"
```

---

## Summary

**Total: 31 tasks** across 3 phases
- Phase 1 (Align): 6 tasks, ~1-2 hours
- Phase 2 (Counter): 10 tasks, ~4-6 hours
- Phase 3 (Multimodal): 15 tasks, ~1-2 weeks (vendor integration dependent)

**Critical Path:** Phase 1 → Phase 2 → Phase 3

**Vendor Dependencies (Phase 3):**
- MiniMax-M3-Vision API access + key
- Whisper API access (or self-host whisper.cpp)
- Edge TTS endpoint access
- MiniMax-Image-1 API access

**Risk:** Phase 3 vendor integration may slip if API access not granted. Mitigate with stub implementations in Phase 1-2 (work without vendor).

**Verification (full F025):**
```bash
go vet ./... && go test -count=1 -short ./shared/...
make check
curl -X POST http://localhost:8002/v1/vision -d '{"tenant_id":"t1","image_url":"https://...","prompt":"describe"}'
curl -X POST http://localhost:8002/v1/audio/transcribe -d '{"tenant_id":"t1","audio_url":"https://..."}'
```

---

*Last updated: 2026-06-14 — Initial F025 plan*
