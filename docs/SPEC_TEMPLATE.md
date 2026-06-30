# F0XX: [Feature Name]

**Date:** YYYY-MM-DD  
**Status:** ⏳ Draft | ✅ Approved | ❌ Rejected  
**Implementation:** ⏳ Pending | 🔄 In Progress | ✅ Done  
**Related:** [F001](F001_multi_store_quota.md), [F016](F016_hybrid_whatsapp.md)

---

## 🎯 Objectives

<!-- Apa yang ingin dicapai dengan feature ini? Tulis 2-3 kalimat yang jelas. -->

**Tujuan eksplisit:**
1. [Objective 1 — measurable dan spesifik]
2. [Objective 2]
3. [Objective 3]

**Problem yang diselesaikan:**
- [Pain point user atau bottleneck teknis saat ini]
- [Kenapa feature ini penting sekarang]

---

## 📋 Acceptance Criteria (AC)

<!-- Kondisi sukses yang measurable. Jika semua AC terpenuhi, feature dianggap selesai. -->

- [ ] **AC-1:** [Deskripsi kriteria — harus observable/testable]
  - *Verification:* [Cara memverifikasi — endpoint, UI, test case]
  - *Example:* [Contoh input/output atau screenshot]

- [ ] **AC-2:** [Kriteria kedua]
  - *Verification:* [Cara verifikasi]
  - *Example:* [Contoh]

- [ ] **AC-3:** [Kriteria ketiga]
  - *Verification:* [Cara verifikasi]
  - *Example:* [Contoh]

<!-- Tambahkan AC sampai feature coverage lengkap (typical: 5-10 AC) -->

---

## 🛠️ Technical Specification

### Architecture Overview

<!-- High-level diagram atau deskripsi komponen yang terlibat -->

```
[Component A] → [Component B] → [Database]
      ↓               ↓
  [Service C]    [External API]
```

### Database Schema

<!-- Jika ada perubahan DB, tulis CREATE TABLE atau ALTER TABLE -->

```sql
-- Migration: 0000XX_feature_name.up.sql
CREATE TABLE feature_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    field_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_feature_table_tenant_id ON feature_table(tenant_id);
```

### API Endpoints

<!-- REST API yang akan di-add/modify -->

#### `POST /api/resource`

**Request:**
```json
{
  "field": "value",
  "nested": {
    "key": "value"
  }
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "field": "value"
  }
}
```

**Error Cases:**
- `400 Bad Request` — Invalid input
- `401 Unauthorized` — Missing/invalid token
- `403 Forbidden` — Feature not enabled for tenant plan
- `404 Not Found` — Resource tidak ditemukan

#### `GET /api/resource/:id`

**Response (200 OK):**
```json
{
  "success": true,
  "data": { /* resource object */ }
}
```

### Files Modified/Created

<!-- Daftar file yang akan diubah atau dibuat -->

**Backend:**
- `apps/umkm/accounting/main.go` — Add handler `handleFeature()`
- `shared/migrations/0000XX_feature_name.up.sql` — DB schema
- `apps/umkm/accounting/feature_test.go` — Unit tests (NEW)

**Frontend:**
- `frontend/umkm-web/src/components/FeatureComponent.vue` (NEW)
- `frontend/umkm-web/src/api.ts` — Add API methods
- `frontend/umkm-web/src/router.ts` — Add route `/feature`

**Infrastructure:**
- `docker-compose.yml` — (jika perlu tambah service)
- `.env.example` — (jika ada ENV var baru)

### Dependencies

<!-- Feature lain atau library yang diperlukan -->

- **Internal:** [F016 Hybrid WhatsApp](F016_hybrid_whatsapp.md) — untuk notifikasi
- **External:** `github.com/lib/package` v1.2.3 — untuk X functionality

### Configuration

<!-- ENV vars atau config baru -->

```bash
# .env.example
FEATURE_ENABLED=true
FEATURE_API_KEY=your_key_here
FEATURE_TIMEOUT=30s
```

---

## 🎨 UI/UX Specification

<!-- Mockup, wireframe, atau deskripsi UI jika applicable -->

### Screen: Feature Page

**Layout:**
```
┌─────────────────────────────────────┐
│  Header: Feature Title              │
├─────────────────────────────────────┤
│  [ Input Field ]                    │
│  [ Button: Submit ]                 │
├─────────────────────────────────────┤
│  Results Table:                     │
│  | Col 1 | Col 2 | Actions |        │
│  |-------|-------|---------|        │
│  | Data  | Data  | [Edit]  |        │
└─────────────────────────────────────┘
```

**Interactions:**
- User klik "Submit" → POST /api/resource → Table refresh
- User klik "Edit" → Open modal → PUT /api/resource/:id

**Validation:**
- Field X wajib diisi (required)
- Field Y format email
- Submit button disabled jika form invalid

---

## 🧪 Testing Strategy

### Unit Tests

```go
// apps/umkm/accounting/feature_test.go
func TestHandleFeature_Success(t *testing.T) {
    // Arrange: mock request
    req := httptest.NewRequest("POST", "/api/resource", bytes.NewReader(payload))
    
    // Act: call handler
    rec := httptest.NewRecorder()
    handleFeature(rec, req)
    
    // Assert: check response
    assert.Equal(t, 200, rec.Code)
}
```

### Integration Tests

```bash
# Test flow end-to-end
curl -X POST http://localhost:8201/api/resource \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"field": "value"}'

# Expected: 200 OK + data in response
```

### Manual Testing Checklist

- [ ] Create new resource via UI
- [ ] Edit existing resource
- [ ] Delete resource (soft delete jika applicable)
- [ ] Pagination works (jika list besar)
- [ ] Permission check (non-owner cannot edit)
- [ ] Mobile responsive

---

## 🚀 Deployment Notes

### Migration Steps (Production)

```bash
# 1. Run migration
make migrate-up

# 2. Restart affected services
docker compose restart umkm-accounting

# 3. Verify
curl http://api-gateway:8000/healthz
```

### Rollback Plan

```bash
# If something goes wrong:
make migrate-down  # Rollback DB schema
git revert <commit-hash>  # Rollback code
docker compose restart umkm-accounting
```

### Feature Flags (Optional)

```sql
-- Enable feature per-tenant
UPDATE tenant_features SET is_enabled = true 
WHERE tenant_id = '...' AND feature_key = 'feature_name';
```

---

## 🔮 Future Enhancements (Out of Scope)

<!-- Ideas untuk iterasi berikutnya, tapi bukan bagian dari MVP ini -->

- **Enhancement 1:** [Deskripsi — kenapa belum masuk scope]
- **Enhancement 2:** [Deskripsi]
- **Integration:** [Service/API eksternal yang bisa diintegrasikan nanti]

---

## 📚 References

<!-- Link ke docs eksternal, RFC, atau design doc lain -->

- [External API Docs](https://example.com/docs)
- [Design Discussion (GitHub Issue #123)](https://github.com/org/repo/issues/123)
- [Similar Implementation (Open Source)](https://github.com/other/project)

---

## 📝 Notes & Decisions

<!-- Catatan penting selama development atau design decisions yang perlu documented -->

**2026-07-01:** Decided to use X instead of Y because [reason]  
**2026-07-05:** User feedback: [insight] → adjust AC-3 to include [change]
