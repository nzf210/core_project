# F058: Superadmin Impersonate + Grafana Monitoring

**Date:** 2026-06-22  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F005](../FEATURE_MAP.md) (Superadmin Dashboard), [F067](../FEATURE_MAP.md) (Grafana Production Monitoring)

---

## 🎯 Objectives

Superadmin dapat troubleshoot tenant issues dengan login langsung sebagai tenant owner dan mengakses monitoring infrastructure.

**Tujuan eksplisit:**
1. Impersonate tenant owner untuk troubleshooting tanpa perlu password tenant (audit trail lengkap)
2. Akses Grafana monitoring dashboard via navbar link untuk observability real-time
3. Simplifikasi support workflow — reduce back-and-forth saat debugging tenant-specific issues

**Problem yang diselesaikan:**
- Support team harus minta password tenant atau reset password untuk troubleshoot → privacy concern + friction
- Grafana dashboard tersebar di bookmarks/docs → butuh unified entry point dari superadmin UI
- No audit trail saat superadmin access tenant account → compliance risk

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Impersonate API Endpoint**
  - *Verification:* `POST /api/superadmin/tenants/{tenant_id}/impersonate` generate JWT token dengan `role: owner` + `impersonated_by` field
  - *Example:* Response: `{ access_token: "...", tenant: { id, business_name, owner_name, plan } }`, JWT expiry 12 jam

- [x] **AC-2: Frontend Impersonate Button**
  - *Verification:* Button "🔓 Login Sebagai" di Tenant Management table (TenantManagement.vue)
  - *Example:* Click → confirmation dialog → POST impersonate → open `app.example.com?impersonate_token=<token>` di tab baru

- [x] **AC-3: Authorization Guard**
  - *Verification:* Hanya superadmin yang bisa call impersonate endpoint (JWT role check)
  - *Example:* Tenant owner coba akses → 403 Forbidden

- [x] **AC-4: Audit Trail**
  - *Verification:* Impersonate token memiliki claim `impersonated_by: <superadmin_id>` untuk tracking
  - *Example:* `/auth/validate` dengan impersonate token → response include `impersonated_by` field

- [x] **AC-5: Grafana Monitoring Link**
  - *Verification:* Navbar link "📊 Monitoring (Grafana)" di SuperAdminDashboard (App.vue)
  - *Example:* Click → `target="_blank"` ke `VITE_GRAFANA_URL` (default: http://localhost:3001)

- [x] **AC-6: Token Expiry**
  - *Verification:* Impersonate token expire setelah 12 jam
  - *Example:* Token generated 10:00 → valid sampai 22:00 → setelah itu 401 Unauthorized

- [x] **AC-7: Tenant Validation**
  - *Verification:* Impersonate endpoint validasi tenant exists di DB sebelum generate token
  - *Example:* POST dengan `tenant_id` yang tidak ada → 404 Not Found

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│      Superadmin Dashboard (TenantManagement.vue)    │
│  Button "🔓 Login Sebagai" → confirmation dialog    │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│  POST /api/superadmin/tenants/{id}/impersonate      │
│         (Bearer: superadmin JWT token)              │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│            API Gateway :8000 (Proxy)                │
│  /api/superadmin/tenants/* → auth-service:8001      │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│         Auth Service :8001 (Impersonate Handler)    │
│  1. Validate superadmin JWT                         │
│  2. Query tenant + owner dari DB                    │
│  3. Generate JWT: role=owner, impersonated_by=...   │
│  4. Return impersonate token                        │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│  Frontend: sessionStorage.setItem('superadmin_token'│
│  Open new tab: app.example.com?impersonate_token=..│
└─────────────────────────────────────────────────────┘
         ↓ (future enhancement)
┌─────────────────────────────────────────────────────┐
│  UMKM Web: router guard detect query param          │
│  → validate token → set localStorage → dashboard    │
└─────────────────────────────────────────────────────┘
```

### Database Schema

**No migration needed** — uses existing `tenants` and `users` tables.

```sql
-- Query untuk impersonate endpoint
SELECT u.id AS user_id, u.name AS owner_name, u.role,
       t.id AS tenant_id, t.business_name, t.plan
FROM users u
JOIN tenants t ON u.tenant_id = t.id
WHERE t.id = $1 AND u.role = 'owner'
LIMIT 1;
```

### API Endpoints

#### `POST /api/superadmin/tenants/:tenant_id/impersonate`

**Headers:**
```
Authorization: Bearer <superadmin_jwt_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "tenant": {
      "id": "tenant-uuid",
      "business_name": "Warung Bu Siti",
      "owner_name": "Siti Rahman",
      "plan": "lite"
    }
  }
}
```

**JWT Claims (impersonate token):**
```json
{
  "user_id": "owner-user-uuid",
  "tenant_id": "tenant-uuid",
  "role": "owner",
  "impersonated_by": "superadmin-user-uuid",
  "exp": 1719936000
}
```

**Error Cases:**
- `401 Unauthorized` — Missing/invalid superadmin JWT token
- `403 Forbidden` — User role bukan `superadmin`
- `404 Not Found` — Tenant tidak ditemukan atau tidak memiliki owner user
- `500 Internal Server Error` — DB error atau JWT signing error

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (auth-service):**
```go
// impersonate_test.go
func TestHandleImpersonate_ValidSuperadmin(t *testing.T) {
    // Mock DB query return owner user
    // Generate impersonate token
    // Verify JWT claims include impersonated_by
}

func TestHandleImpersonate_TenantNotFound(t *testing.T) {
    // Mock DB query return no rows
    // Expect 404 Not Found
}

func TestHandleImpersonate_NonSuperadminForbidden(t *testing.T) {
    // JWT role = owner (not superadmin)
    // Expect 403 Forbidden
}
```

**Frontend (superadmin-web):**
```typescript
// TenantManagement.spec.ts
describe('Impersonate Button', () => {
  it('shows confirmation dialog on click', async () => {
    // Click "🔓 Login Sebagai" → verify dialog open
  })

  it('calls impersonate API and opens new tab', async () => {
    // Mock API response → verify window.open() called
  })

  it('stores superadmin token before impersonate', async () => {
    // Verify sessionStorage.setItem('superadmin_token') called
  })
})
```

### Integration Tests

```bash
# 1. Login superadmin
SUPERADMIN_TOKEN=$(curl -X POST http://localhost:8000/api/auth/superadmin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.access_token')

# 2. Impersonate tenant
IMPERSONATE_TOKEN=$(curl -X POST http://localhost:8000/api/superadmin/tenants/{tenant_id}/impersonate \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" | jq -r '.data.access_token')

# 3. Validate impersonate token
curl -X POST http://localhost:8000/api/auth/validate \
  -H "Authorization: Bearer $IMPERSONATE_TOKEN" | jq
# → should show role: owner, impersonated_by: <superadmin_id>

# 4. Use impersonate token to access tenant resources
curl -X GET http://localhost:8000/api/umkm/dashboard \
  -H "Authorization: Bearer $IMPERSONATE_TOKEN" \
  -H "X-Tenant-ID: {tenant_id}"
# → should return 200 OK with tenant dashboard data

# 5. Non-superadmin cannot impersonate
OWNER_TOKEN=$(curl -X POST http://localhost:8000/api/auth/login ...)
curl -X POST http://localhost:8000/api/superadmin/tenants/{tenant_id}/impersonate \
  -H "Authorization: Bearer $OWNER_TOKEN"
# → 403 Forbidden
```

### Manual Testing Checklist

- [ ] Login sebagai superadmin → buka Tenant Management
- [ ] Click "🔓 Login Sebagai" → confirmation dialog muncul
- [ ] Confirm → API call success → new tab open dengan `?impersonate_token=...`
- [ ] (Future) New tab auto-login → dashboard tenant terbuka
- [ ] Verify `impersonated_by` claim di JWT token (decode via jwt.io)
- [ ] Click "📊 Monitoring (Grafana)" → new tab open ke Grafana dashboard
- [ ] Token expire setelah 12 jam → 401 Unauthorized
- [ ] Non-superadmin coba akses impersonate endpoint → 403 Forbidden

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Impersonate token generated", 
  "superadmin_id", superadminID,
  "tenant_id", tenantID,
  "owner_id", ownerID,
  "token_expiry", expiryTime)

slog.Warn("Impersonate attempt by non-superadmin", 
  "user_id", userID,
  "role", role,
  "tenant_id", tenantID)
```

**Metrics to track:**
- Impersonate token generation count per day (detect abuse)
- Impersonate token validation success/failure rate
- Average time superadmin stay impersonated (session duration)

**Alerts:**
- Impersonate token generation > 50/day → investigate potential abuse
- Impersonate from unexpected IP → security alert

**Audit Log (Future Enhancement):**
- Dedicated `impersonate_logs` table: `superadmin_id`, `tenant_id`, `started_at`, `ended_at`, `ip_address`
- Dashboard di Superadmin UI untuk track impersonate history

---

## 🚀 Rollout Plan

### Phase 1: Backend + API (Done ✅)
- Deploy `services/auth-service/impersonate.go` dengan handler + JWT generation
- Deploy api-gateway proxy route `/api/superadmin/tenants/*`
- Test: cURL impersonate endpoint → verify JWT claims

### Phase 2: Superadmin UI (Done ✅)
- Deploy superadmin-web dengan impersonate button di Tenant Management
- Deploy Grafana navbar link di App.vue
- Test: Click button → API call → new tab open

### Phase 3: Umkm-web Auto-Login (Future)
- Umkm-web router guard detect `?impersonate_token` query param
- Validate token via `/auth/validate` → set localStorage
- Redirect ke dashboard tanpa manual login

### Phase 4: Audit & Monitoring (Future)
- Add impersonate_logs table migration
- Add Grafana dashboard untuk impersonate activity
- Add alert rule untuk suspicious impersonate patterns

### Rollback
- **API rollback:** Revert auth-service + api-gateway → impersonate endpoint 404
- **Frontend rollback:** Revert superadmin-web → button hilang, no impact ke existing functionality
- **Emergency:** Disable impersonate via feature flag: `if !cfg.Features.AllowImpersonate { return 403 }`

---

## 🔮 Future Enhancements (Out of Scope)

- **Grafana Level 2 Integration:** Embed Grafana via `<iframe>` dengan `GF_SECURITY_ALLOW_EMBEDDING=true` + `GF_AUTH_PROXY_ENABLED=true` untuk seamless auth
- **Grafana Level 3 Integration:** Backend `/api/superadmin/metrics` pull dari Grafana API → render Chart.js di Vue (no external redirect)
- **Restore Flow:** Button "Kembali ke Superadmin" di umkm-web navbar saat `impersonated_by` detected → restore `superadmin_token` dari sessionStorage → redirect superadmin-web
- **Impersonate Time Limit:** Auto-expire impersonate session setelah 1 jam activity idle (bukan 12 jam fixed)
- **Permission Restriction:** Impersonate token tidak bisa akses sensitive endpoints (billing, password change) → reduced privilege mode
- **Multi-Factor Confirmation:** Require 2FA/OTP sebelum allow impersonate untuk high-value tenants

---

## 📚 References

- [F005: Superadmin Dashboard](../FEATURE_MAP.md) — Base superadmin UI feature
- [F067: Grafana Production Monitoring](../FEATURE_MAP.md) — Monitoring infrastructure setup
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725) — JWT security considerations
- [Auth Service Implementation](../../services/auth-service/impersonate.go) — Impersonate handler code

---

## 📝 Notes & Decisions

**2026-06-22:** Decision: JWT expiry 12 jam (bukan 24 jam) untuk balance antara UX (tidak logout tengah troubleshoot) dan security (minimize blast radius jika token leaked).  
**2026-06-22:** `impersonated_by` claim mandatory — wajib untuk audit trail dan compliance. Jika di masa depan butuh disable impersonate, bisa query DB untuk revoke semua token dengan `impersonated_by` claim.  
**2026-06-22:** Grafana Level 1 integration (external link) untuk MVP — Level 2/3 defer ke future karena butuh Grafana config change + backend proxy logic yang kompleks.  
**2026-06-22:** Umkm-web auto-login (AC-8) defer ke Phase 3 — butuh router guard logic + query param detection. Saat ini superadmin manual paste token di login form (acceptable untuk MVP).
