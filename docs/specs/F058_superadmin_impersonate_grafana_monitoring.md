# F058: Superadmin Impersonate + Grafana Monitoring


## 🔄 Spec-First Workflow

```
USER menulis SPEC      →       AI review & clarify      →       USER approve
     ↓                         ↓                                  ↓
 FEATURE_MAP.md         AI tanya clarifications           USER comment/approve
                              ↓                                  ↓
                      AI wait for approval          AI implement dari SPEC
                                                            ↓
                                                    USER review diff\n                                                            ↓\n                                                    JALANKAN TESTING
```

### Aturan untuk AI:
1. Baca FEATURE_MAP.md sebelum coding
2. Kalau ada feature baru/wubah, tanya USER dulu:
   - "Ada SPEC untuk fitur ini?" → kalau belum, buat draft SPEC
   - "SPEC ini sudah diapprove?" → kalau belum, jangan implement
3. Ambiguitas? → Tanya clarification dulu
4. **Setiap feature baru WAJIB punya plan dulu**:
   - Buat file plan di `docs/plans/<YYYY-MM-DD>-<feature-name>.md`
   - Plan harus bite-sized, copy-pasteable, dan siap dieksekusi oleh subagent
   - Jangan coding sebelum plan selesai di-review/approve
5. Implementasi selesai? → Update kolom `Implementation` di tabel
6. **Testing Wajib** — Setiap kali ada *perubahan*, *tambah fungsi*, atau *hapus fungsi*, JALANKAN TEST sebelum menyelesaikan task:
   - `make check` (untuk menjalankan linter, build, dan semua test)
   - Atau `go test ./apps/umkm/... -v` (untuk test spesifik)
7. Setelah selesai → Update kembali `FEATURE_MAP.md` (status + testing result)


## 📊 Feature Registry

| ID | Feature | Spec Status | Implementation | Last Updated |
|:---|:--------|:------------|:---------------|:-------------|
| F001 | Multi-Store Quota | ✅ Approved | ✅ Done | 2026-06-12 |
| F002 | Voucher Link Subscription | ✅ Approved | ✅ Done | 2026-06-12 |
| F003 | Subscription Hold Worker | ✅ Approved | ✅ Done | 2026-06-12 |
| F004 | Read-only Enforcement (Frozen) | ✅ Approved | ✅ Done | 2026-06-12 |
| F005 | Superadmin Dashboard | ✅ Approved | ✅ Done | 2026-06-12 |
| F006 | Multi-Tenant WA Session Pool | ✅ Approved | ✅ Done | 2026-06-01 |
| F007 | Chatbot with RAG | ✅ Approved | ✅ Done | 2026-06-01 |
| F008 | Escalation to Chatwoot | ✅ Approved | ✅ Done | 2026-06-01 |
| F009 | N8N Queue Mode Automation | ✅ Approved | ✅ Done | 2026-06-01 |
| F010 | Campaign Volunteer Management | ✅ Approved | ✅ Done | 2026-06-12 |
| F011 | Campaign Voter Onboarding | ✅ Approved | ✅ Done | 2026-06-12 |
| F012 | Sidebar Navigation UI | ✅ Approved | ✅ Done | 2026-06-12 |
| F013 | N8N Integration via Super Admin | ❌ Removed | — | — |
| F014 | Flexible LLM Model System | ✅ Approved | ✅ Done | 2026-06-12 |
| F015 | Onboarding Activation Flow | ✅ Approved | ✅ Done | 2026-06-13 (UI: 2026-06-14) |
| F016 | Hybrid WhatsApp (Cloud API + whatsmeow) | ✅ Approved | ✅ Done | 2026-06-13 |
| F017 | OTP 1-Hour Reuse Window | ✅ Approved | ✅ Done | 2026-06-13 |
| F018 | Telegram Auth (Register & Login) | ✅ Approved | ✅ Done | 2026-06-13 |
| F019 | Onboarding Sync via /me (Fix Lite Tier) | ✅ Approved | ✅ Done | 2026-06-14 |
| F020 | AI CS Setup Wizard (Per-Tenant Config UI) | ✅ Approved | ✅ Done | 2026-06-14 |
| F021 | Cash Flow PDF Export | ✅ Approved | ✅ Done | 2026-06-14 |
| F022 | Excel/Google Sheet Import & Export | ✅ Approved | ✅ Done | 2026-06-14 |
| F023 | FAQ Bot AI — Edit & Generate | ✅ Approved | ✅ Done | 2026-06-14 |
| F024 | Paid-Only Enforcement (Hardening) | ✅ Approved | ✅ Done | 2026-06-14 |
| F025 | Tier Restrictions Overhaul + AI Multimodal | ✅ Approved | ✅ Done | 2026-06-22 |
| F026 | N8N Notification Webhooks & Workflows | ✅ Approved | ✅ Done | 2026-06-14 |
| F027 | Core Business Flow Fixes & Optimizations | ✅ Approved | ✅ Done | 2026-06-14 |
| F029 | Dynamic Multimodal Guardrails | ✅ Approved | ✅ Done | 2026-06-14 |
| F030 | GetPlanFeatures DB Integration | ✅ Approved | ✅ Done | 2026-06-14 |
| F031 | Campaign Anti-Double Validation | ✅ Approved | ✅ Done | 2026-06-17 |
| F032 | Modul Saksi & Real Count C1 | ✅ Approved | ✅ Done | 2026-06-17 |
| F033 | Campaign Logistics Tracking | ✅ Approved | ✅ Done | 2026-06-17 |
| F034 | Add-on Wallet & Meta API Connector | ✅ Approved | ✅ Done | 2026-06-20 |
| F035 | Discount Vouchers (Percent & Fixed) | ✅ Approved | ✅ Done | 2026-06-20 |
| F036 | Lifetime Affiliate, External Agent & Public Leaderboard | ✅ Approved | ✅ Done | 2026-06-20 |
| F037 | Dashboard Sentimen Isu Harian (AI NLP) | ✅ Approved | ✅ Done | 2026-06-17 |
| F038 | Wargame & Simulasi Kemenangan | ✅ Approved | ✅ Done | 2026-06-17 |
| F039 | Peta Kerawanan & Pelaporan Pelanggaran | ✅ Approved | ✅ Done | 2026-06-17 |
| F040 | WA Bot FAQ Panduan Kampanye (RAG) | ✅ Approved | ✅ Done | 2026-06-17 |
| F041 | Gamification & Leaderboard Relawan | ✅ Approved | ✅ Done | 2026-06-17 |
| F042 | Auto-Scan KTP (AI OCR Vision) | ✅ Approved | ✅ Done | 2026-06-17 |
| F043 | Multi-Level Election & Sainte-Laguë Simulator | ✅ Approved | ✅ Done | 2026-06-20 |
| F044 | Campaign Modular License & Payment System | ✅ Approved | ✅ Done | 2026-06-20 |
| F045 | UMKM Healthcare Clinic Queue System | ✅ Approved | ✅ Done | 2026-06-17 |
| F046 | Hierarchical Coordinator Assignment | ✅ Approved | ✅ Done | 2026-06-20 |
| F047 | Hardening Migration (F024 cleanup) | ✅ Approved | ✅ Done | 2026-06-17 |
| F048 | WA Provider Preferences & Activation Guard | ✅ Approved | ✅ Done (v2) | 2026-06-20 |
| F049 | Container Overhaul & Infrastructure Optimization | ✅ Approved | ✅ Done | 2026-06-17 |
| F050 | WCH E2E MCP Server (UI Testing & Browser Automation) | ✅ Approved | ✅ Done | 2026-06-20 |
| F051 | AI Quota Per-Modalitas (Text/Vision/Image) | ✅ Approved | ✅ Done | 2026-06-20 |
| F052 | Tier-First Feature System + Per-Tenant Addon Guard | ✅ Approved | ✅ Done | 2026-06-20 |
| F053 | Admin-Configurable Addon Pricing + Addon Purchase Flow | ✅ Approved | ✅ Done | 2026-06-22 |
| F054 | Referral System: Discount Downline + Commission Upline | ✅ Approved | ✅ Done | 2026-06-22 |
| F055 | Password Reset via Chat (WA + Telegram) v2 | ✅ Approved | ✅ Done | 2026-06-30 |
| F056 | Theme Management (Dark/Light/System) | ✅ Approved | ✅ Done | 2026-06-21 |
| F057 | Superadmin Feature Matrix + Addon Tier Gating | ✅ Approved | ✅ Done | 2026-06-22 |
| F058 | Superadmin Impersonate + Grafana Monitoring | ✅ Approved | ✅ Done | 2026-06-22 |
| F059 | Wallet Payment untuk Subscription & Topup | ✅ Approved | ✅ Done | 2026-06-21 |
| F060 | Landing Page — Marketing & Onboarding | ✅ Approved | ✅ Done | 2026-06-21 |
| F061 | Sales Dashboard Chart — Visual Penjualan | ✅ Approved | ✅ Done | 2026-06-21 |
| F062 | Staff Management UI (Settings.vue) | ✅ Approved | ✅ Done | 2026-06-22 |
| F063 | WA Keyword Registration (REG/OTP/VERIF) + WA Center | ✅ Approved | ✅ Done | 2026-06-27 |
| F064 | Platform WA Provider Detection & OTP Routing | ✅ Approved | ✅ Done | 2026-06-28 |
| F065 | Landing Page Content Management — Superadmin JSON Editor | ✅ Approved | ✅ Done | 2026-06-29 |
| F066 | Dynamic Feature Gating — Zero-Hardcode Feature Toggle System | ✅ Approved | ✅ Done | 2026-06-30 |
| F067 | Grafana Production-Ready Monitoring — Prometheus + 8 Dashboards | ✅ Approved | ✅ Done | 2026-07-01 |

## F058: Superadmin Impersonate + Grafana Monitoring

**Spec Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Last Updated:** 2026-06-22

### 🎯 Objectives

Superadmin dapat:
1. Login sebagai tenant owner untuk troubleshooting tanpa password tenant (impersonate)
2. Akses Grafana monitoring dashboard via external link dari navbar

### 📝 Spec

#### AC-1: Impersonate API Endpoint
- [x] `POST /api/superadmin/tenants/{tenant_id}/impersonate` — generate JWT token `role: owner` + `impersonated_by` audit trail
- [x] Query owner tenant dari DB, validasi tenant exists
- [x] JWT expiry 12 jam
- [x] Response: `{ access_token, tenant: { id, business_name, owner_name, plan } }`

#### AC-2: Frontend Button "Login Sebagai"
- [x] Button "🔓 Login Sebagai" di Tenant Management table (TenantManagement.vue)
- [x] Click → POST impersonate → open `app.example.com?impersonate_token=<token>` di tab baru
- [x] Confirmation dialog sebelum impersonate

#### AC-3: Grafana Monitoring Link
- [x] Navbar link "📊 Monitoring (Grafana)" di SuperAdminDashboard navbar (App.vue)
- [x] `target="_blank"` ke `VITE_GRAFANA_URL` (default: http://localhost:3001)
- [x] Level 1 integration — external link only, no auth sharing

### 🛠️ Implementation Details

**Backend Files:**
- `services/auth-service/impersonate.go` (NEW) — `handleImpersonate()` handler
- `services/auth-service/main.go` — register route `/superadmin/tenants/` + jwtMiddleware
- `services/api-gateway/main.go` — proxy `/api/superadmin/tenants/` ke auth-service:8001

**Frontend Files:**
- `frontend/superadmin-web/src/views/TenantManagement.vue` — impersonate button + `impersonateTenant()` method
- `frontend/superadmin-web/src/App.vue` — Grafana navbar link + `VITE_GRAFANA_URL` data property
- `frontend/superadmin-web/.env` — `VITE_GRAFANA_URL=http://localhost:3001`

**Config:**
- `.env.example` — documented `GRAFANA_URL=http://localhost:3001`

### 🔄 Flow

```
Superadmin click "🔓 Login Sebagai" → Confirm dialog
    ↓
POST /api/superadmin/tenants/{id}/impersonate (Bearer: superadmin JWT)
    ↓
auth-service: query owner → generate JWT (role: owner, impersonated_by: superadmin_id)
    ↓
Frontend: sessionStorage.setItem('superadmin_token', ...) → open tab baru
    ↓
app.example.com?impersonate_token=<token> → umkm-web auto-login (future)
```

### 🚧 Future Enhancements

**Level 2 Grafana Integration:**
- Embed via `<iframe>` (butuh `GF_SECURITY_ALLOW_EMBEDDING=true`)

**Level 3 Grafana Integration:**
- Backend `/api/superadmin/metrics` pull dari Grafana API → render Chart.js di Vue

**Restore Flow:**
- Button "Kembali ke Superadmin" di umkm-web navbar saat `impersonated_by` detected
- Click → restore `superadmin_token` dari sessionStorage → redirect superadmin-web

**Umkm-web Auto-Login:**
- Detect `?impersonate_token` query param di router guard
- Validate token via `/auth/validate` → set localStorage → redirect dashboard

### ✅ Testing

**Manual Test:**
```bash
# 1. Login superadmin
curl -X POST http://localhost:8010/api/auth/superadmin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"superadmin","password":"Admin123!"}'
# → save access_token

# 2. Impersonate tenant
curl -X POST http://localhost:8010/api/superadmin/tenants/{tenant_id}/impersonate \
  -H 'Authorization: Bearer <superadmin_token>'
# → returns impersonate token

# 3. Validate impersonate token
curl -X POST http://localhost:8010/api/auth/validate \
  -H 'Authorization: Bearer <impersonate_token>'
# → should return role: owner, impersonated_by: <superadmin_id>
```

**Frontend Test:**
1. Login superadmin → go to Tenant Management
2. Click "🔓 Login Sebagai" → confirm dialog → tab baru terbuka
3. Check navbar → "📊 Monitoring (Grafana)" link muncul
4. Click link → Grafana dashboard terbuka di tab baru
