# Rencana Perbaikan Kualitas Kode WCH Platform
**Tanggal:** 2026-07-31  
**Status:** Draft — Menunggu Approval

---

## 🎯 Executive Summary

Berdasarkan audit proyek, ditemukan beberapa area yang perlu perbaikan:

| Area | Status Saat Ini | Target |
|:-----|:---------------|:-------|
| **Test Coverage** | 41 file test / 230 file Go (17.8%) | Minimal 75% coverage |
| **File Size** | 5 file backend melebihi 450 baris | 100% compliance dengan SonarQube |
| **Test Coverage per Service** | Banyak service tanpa test (0-1 file) | Minimal 2-3 test file per service |
| **Frontend Testing** | Tidak ada test Vue | Minimal 50% component coverage |
| **Security** | Beberapa endpoint tanpa rate limiting | 100% critical endpoint protected |
| **Documentation** | Beberapa fitur tanpa detailed spec | 100% approved spec |

**Estimasi Effort:** 3-4 minggu (bertahap)  
**Priority:** HIGH (Technical Debt & SonarQube Compliance)

---

## 📊 Temuan Audit Detail

### 1. File Melebihi Batas Ukuran (⚠️ CRITICAL)

**WAJIB DIPECAH — Melanggar standar SonarQube 450 baris:**

```
./services/wa-gateway/event_handler.go: 815 lines (⚠️ EXCEEDS 450 line limit)
./services/auth-service/superadmin_handlers.go: 477 lines
./services/auth-service/telegram_staff_handlers.go: 459 lines
./services/auth-service/auth_handlers.go: 748 lines
./services/auth-service/login_handlers.go: 479 lines
```

**Dampak:**
- Melanggar konvensi CLAUDE.md (max 450 baris)
- Sulit maintenance & review
- SonarQube Code Smell tinggi

**Solusi:**
- Pecah berdasarkan resource/domain
- Extract helper functions ke file terpisah
- Pisahkan validation logic

---

### 2. Test Coverage Rendah (⚠️ HIGH PRIORITY)

**Services tanpa test sama sekali:**

```
services/subscription-worker: Go files=1, Test files=0 ❌
```

**Services dengan test minimal (1 file test):**

```
services/api-gateway: Go files=4, Test files=1
services/billing-service: Go files=27, Test files=1 ⚠️ (ratio sangat rendah)
services/notification-service: Go files=5, Test files=1
services/wa-cloud-api: Go files=5, Test files=1
services/wa-gateway: Go files=10, Test files=1 ⚠️ (ratio rendah)
```

**Services dengan test coverage OK:**

```
services/auth-service: Go files=16, Test files=2 ⚠️ (masih bisa ditingkatkan)
services/ai-gateway: Go files=9, Test files=2 ✅
apps/campaign: Go files=35, Test files=3 ⚠️
apps/umkm: Go files=53, Test files=8 ⚠️
```

**Perbandingan:**
- Total Go files: 230
- Total test files: 41
- Ratio: **17.8%** (Target: minimal 50% file, 75% coverage)

---

### 3. Frontend Testing

**Status:** TIDAK ADA test frontend sama sekali ❌

**Frontend structure:**
```
frontend/umkm-web/       - Vue 3 (Port 3201-3203)
frontend/campaign-web/   - Vue 3 (Port 3301)
frontend/superadmin-web/ - Vue 3 (Port 3401)
```

**Dampak:**
- UI bug tidak terdeteksi sebelum production
- Regression risk tinggi saat refactor
- Tidak ada confidence saat deploy

---

### 4. Security & Performance

**Issues yang perlu dicek:**

1. **Rate Limiting:**
   - Apakah semua public endpoint punya rate limiting?
   - Validation di UMKM Accounting (flat pattern) — banyak handler dalam 1 file

2. **Input Validation:**
   - Perlu review semua endpoint untuk SQL injection prevention
   - JSON binding validation
   - File upload validation (size, type)

3. **Sensitive Data:**
   - Audit logging untuk perubahan sensitive (billing, wallet)
   - Encryption di-rest vs in-transit

4. **API Gateway:**
   - Timeout configuration
   - Circuit breaker pattern
   - Health check endpoints

---

### 5. Feature Implementation Status

**Dari FEATURE_MAP.md:**
```
Total features: 45+ (F001-F067)
Status breakdown:
  - ✅ Done: 40 features
  - 🔨 In Progress: 0
  - ⏳ Draft/Pending: 5
  - ❌ Rejected/Cancelled: 1
```

**Features tanpa detailed spec:**
- Beberapa fitur hanya punya inline spec di FEATURE_MAP.md
- Perlu cek mana yang butuh detailed spec di `docs/specs/`

---

## 🛠️ Rencana Perbaikan

### Phase 1: File Size Compliance (Week 1) — CRITICAL

**Priority: P0**

**Goal:** Semua file backend < 450 baris

**Tasks:**

1. **wa-gateway/event_handler.go (815 lines):**
   ```
   Pecah menjadi:
   - event_handler.go (routing & dispatch logic, <450 lines)
   - message_processor.go (message processing logic)
   - media_handler.go (media upload/download)
   - session_manager.go (WA session management)
   ```

2. **auth-service/auth_handlers.go (748 lines):**
   ```
   Pecah menjadi:
   - auth_handlers.go (routing & basic auth, <450 lines)
   - registration_handlers.go (register logic)
   - otp_handlers.go (OTP generation & verification)
   - token_handlers.go (JWT refresh & validation)
   ```

3. **auth-service/login_handlers.go (479 lines):**
   ```
   Pecah menjadi:
   - login_handlers.go (login routing, <450 lines)
   - phone_login_handlers.go (phone/WA login)
   - telegram_login_handlers.go (telegram auth)
   ```

4. **auth-service/superadmin_handlers.go (477 lines):**
   ```
   Pecah menjadi:
   - superadmin_handlers.go (routing & basic CRUD, <450 lines)
   - tenant_management_handlers.go (tenant ops)
   - impersonate_handlers.go (F058 impersonate)
   ```

5. **auth-service/telegram_staff_handlers.go (459 lines):**
   ```
   Pecah menjadi:
   - telegram_staff_handlers.go (routing, <450 lines)
   - telegram_validators.go (validation logic)
   ```

**Testing:**
- `go build ./...` — ensure no compilation errors
- `make check` — ensure all existing tests pass
- Manual smoke test semua affected endpoints

---

### Phase 2: Test Coverage — Critical Services (Week 2)

**Priority: P1**

**Goal:** Services tanpa test atau coverage < 30% punya minimal coverage

**Tasks:**

1. **subscription-worker (PRIORITY 1):**
   ```
   Target: 60% coverage
   Test focus:
   - Hold/freeze tenant logic
   - Expiry detection
   - Notification trigger
   - Edge cases (grace period, manual hold)
   ```

2. **billing-service (PRIORITY 2):**
   ```
   Target: 50% coverage (27 Go files → minimal 10-12 test files)
   Test focus:
   - Xendit webhook validation
   - Voucher redemption logic
   - Wallet deduction (ConsumeWalletAddon)
   - Referral discount calculation
   - Subscription activation/renewal
   ```

3. **wa-gateway (PRIORITY 3):**
   ```
   Target: 50% coverage
   Test focus:
   - Provider routing (Cloud API vs whatsmeow)
   - Rate limiting (token bucket)
   - Message type detection (broadcast vs transactional)
   - Fallback logic
   ```

4. **api-gateway:**
   ```
   Target: 40% coverage
   Test focus:
   - Routing logic
   - Header injection (X-Tenant-ID, X-User-Role)
   - JWT validation
   - Rate limiting middleware
   ```

**Test Pattern:**
```go
// Unit tests (pure functions)
func TestValidateXenditWebhook(t *testing.T) { ... }
func TestCalculateReferralDiscount(t *testing.T) { ... }

// Integration-style tests (with httptest)
func TestHandleSubscribe_WithVoucher(t *testing.T) { ... }
func TestWAGateway_ProviderRouting(t *testing.T) { ... }
```

---

### Phase 3: Frontend Testing Foundation (Week 2-3)

**Priority: P2**

**Goal:** Setup testing infrastructure untuk Vue apps

**Tasks:**

1. **Setup Vitest untuk ketiga frontend:**
   ```bash
   # umkm-web, campaign-web, superadmin-web
   npm install -D vitest @vue/test-utils happy-dom
   ```

2. **Buat test untuk critical components (umkm-web):**
   ```
   Priority components:
   - Login.vue (auth flow)
   - Settings.vue (FAQ, WA config)
   - WASetup.vue (2-step validation)
   - Wallet.vue (topup, balance)
   - Addons.vue (marketplace)
   - Dashboard.vue (overview cards)
   ```

3. **API mocking:**
   ```javascript
   // Mock api.ts calls
   vi.mock('@/api', () => ({
     login: vi.fn(),
     getSettings: vi.fn(),
     // ...
   }))
   ```

4. **Test pattern:**
   ```javascript
   // Component mount test
   test('WASetup shows validation error', async () => {
     const wrapper = mount(WASetup)
     await wrapper.find('button').trigger('click')
     expect(wrapper.text()).toContain('Access token diperlukan')
   })
   ```

**Target:**
- umkm-web: 10 component tests (critical paths)
- superadmin-web: 5 component tests
- campaign-web: 5 component tests

---

### Phase 4: Security Audit & Hardening (Week 3)

**Priority: P1**

**Goal:** Semua critical endpoint protected & validated

**Tasks:**

1. **Rate Limiting Audit:**
   ```
   Cek semua public endpoints:
   - POST /auth/register
   - POST /auth/login
   - POST /auth/verify-otp
   - POST /api/wa/send (sudah ada token bucket, verify)
   - POST /subscribe (billing)
   - POST /addons/purchase (billing)
   ```

2. **Input Validation Audit:**
   ```
   Priority check:
   - SQL injection (parameterized queries — should be OK)
   - XSS (JSON response escaping)
   - File upload validation (media_url download — check SSRF)
   - NIK validation (16 digit, numeric only)
   ```

3. **Sensitive Endpoint Audit:**
   ```
   Endpoints yang perlu extra logging:
   - Wallet topup/deduction
   - Subscription activation
   - Voucher redemption
   - Superadmin impersonate
   - WA Cloud API credential save
   ```

4. **Add security tests:**
   ```go
   func TestRateLimiting_BlocksAfterThreshold(t *testing.T) { ... }
   func TestSQLInjection_ParameterizedQuery(t *testing.T) { ... }
   func TestXSS_JSONEscaping(t *testing.T) { ... }
   ```

---

### Phase 5: Documentation & Spec Completion (Week 4)

**Priority: P2**

**Goal:** Semua feature punya detailed spec (jika perlu)

**Tasks:**

1. **Review FEATURE_MAP.md:**
   - Identifikasi feature kompleks tanpa detailed spec
   - Buat detailed spec di `docs/specs/F0XX_*.md`

2. **Update README per service:**
   ```
   Setiap service perlu README.md dengan:
   - Purpose & responsibility
   - API endpoints (jika HTTP service)
   - Environment variables
   - Dependencies
   - How to test locally
   ```

3. **API Documentation:**
   ```
   Buat OpenAPI/Swagger spec untuk:
   - API Gateway (routing overview)
   - UMKM Accounting API
   - Campaign API
   - Billing Service (external endpoints)
   ```

4. **Runbook:**
   ```
   Buat runbook untuk:
   - Deployment (sudah ada DEPLOYMENT.md, review)
   - Troubleshooting common issues
   - Rollback procedure
   - Database migration failure recovery
   ```

---

## ✅ Acceptance Criteria

### Phase 1: File Size Compliance
- [ ] Semua file Go < 450 baris
- [ ] `go build ./...` pass
- [ ] `make check` pass (no regression)
- [ ] Manual smoke test endpoints affected

### Phase 2: Test Coverage
- [ ] subscription-worker: minimal 1 test file, 60% coverage
- [ ] billing-service: minimal 10 test files, 50% coverage
- [ ] wa-gateway: minimal 3 test files, 50% coverage
- [ ] api-gateway: minimal 2 test files, 40% coverage
- [ ] `go test ./... -cover` → overall coverage > 50%

### Phase 3: Frontend Testing
- [ ] Vitest setup di 3 frontend apps
- [ ] umkm-web: 10 component tests
- [ ] superadmin-web: 5 component tests
- [ ] campaign-web: 5 component tests
- [ ] `npm test` pass di semua frontend

### Phase 4: Security
- [ ] Rate limiting di semua public endpoints
- [ ] Input validation audit selesai
- [ ] Security test cases pass
- [ ] Sensitive endpoint logging aktif

### Phase 5: Documentation
- [ ] README.md di setiap service
- [ ] OpenAPI spec untuk 3 main APIs
- [ ] Runbook deployment & troubleshooting

---

## 🚀 Quick Wins (Can Start Immediately)

**Priority yang bisa dikerjakan parallel:**

1. **File Size Compliance (Phase 1)** — CRITICAL, blocking SonarQube
2. **subscription-worker test** — Small scope, high impact
3. **Frontend Vitest setup** — Independent dari backend work

**Low-hanging fruits:**
- Add README.md ke services tanpa doc (1-2 jam per service)
- Add rate limiting ke endpoint yang belum ada (via middleware, 30 menit per endpoint)
- Fix obvious security issues (e.g., missing input validation)

---

## 📝 Notes

**Dependencies:**
- Phase 1 tidak blocking Phase 2-5 (bisa parallel)
- Phase 3 (Frontend testing) fully independent
- Phase 4 (Security) butuh Phase 2 selesai untuk test framework

**Risks:**
- File refactoring (Phase 1) bisa introduce regression → WAJIB test sebelum merge
- Test coverage target 75% mungkin terlalu tinggi untuk MVP → bisa adjust ke 50-60%
- Frontend testing bisa expand scope jika ditemukan banyak bug

**Follow-up:**
- Setelah Phase 1-5 selesai, consider CI/CD integration:
  - SonarQube scan otomatis di PR
  - Coverage report di GitHub Actions
  - Lighthouse CI untuk frontend performance

---

## 🤝 Approval Required

**USER:** Apakah rencana ini sudah OK? Ada yang perlu diubah prioritasnya?

Saya siap mulai Phase 1 (File Size Compliance) jika sudah approved.
