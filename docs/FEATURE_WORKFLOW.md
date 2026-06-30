# Feature Development Workflow

> **Best practices untuk menambah feature baru di WCH Platform.**

---

## 📝 Phase 1: Planning & Specification

### 1.1 Identify Need

**Trigger:**
- User request / feedback
- Business requirement
- Technical debt / refactoring
- Competitor analysis

**Questions to answer:**
- ❓ Apa problem yang diselesaikan?
- ❓ Siapa user yang terpengaruh? (Superadmin / Owner / Staff / End-user)
- ❓ Apakah ini MVP atau full feature?
- ❓ Urgency: Critical / High / Medium / Low?

### 1.2 Write Specification

**Small Feature (<50 lines code, <5 AC):**

→ Tulis spec **inline** di `docs/FEATURE_MAP.md` dalam section `---` delimiter

```markdown
---

## F0XX: Feature Name

**Objectives:**
- [Goal 1]
- [Goal 2]

**AC:**
- [ ] AC-1: User dapat melakukan X
- [ ] AC-2: System validasi Y

**Implementation:**
- Files: `apps/umkm/accounting/main.go`
- Endpoint: `POST /api/resource`

---
```

**Complex Feature (>50 lines, >5 AC, multiple files):**

→ Buat file terpisah di `docs/specs/F0XX_feature_name.md`

```bash
# 1. Copy template
cp docs/SPEC_TEMPLATE.md docs/specs/F0XX_feature_name.md

# 2. Fill in sections
# - Objectives (WHY)
# - Acceptance Criteria (WHAT)
# - Technical Spec (HOW)
# - Testing Strategy
```

→ Tambah entry di `docs/FEATURE_MAP.md` dengan link:

```markdown
| F0XX | Feature Name | ⏳ Draft | ⏳ Pending | 2026-07-01 |
```

Lalu di bawah tabel:

```markdown
---

**Spec:** [→ docs/specs/F0XX_feature_name.md](specs/F0XX_feature_name.md)

---
```

### 1.3 Review & Clarify

**Self-review checklist:**
- [ ] Objectives jelas dan measurable?
- [ ] AC comprehensive (cover happy path + edge cases)?
- [ ] Technical approach feasible?
- [ ] Dependencies identified?
- [ ] UI mockup (jika frontend involved)?

**AI Review:**
- AI akan baca spec dan tanya clarification jika:
  - AC ambiguous ("user-friendly" → define metric)
  - Technical approach kurang detail
  - Missing error handling
  - No rollback plan

**Team Review (Optional):**
- Share spec di Slack / GitHub Discussion
- Get feedback dari engineer lain
- Update spec berdasarkan input

### 1.4 Approval

Setelah spec **final dan tidak ada ambiguitas**, ubah status:

```markdown
**Status:** ✅ Approved
```

Di `FEATURE_MAP.md` tabel:

```markdown
| F0XX | Feature Name | ✅ Approved | ⏳ Pending | 2026-07-01 |
```

⚠️ **WAJIB:** AI **tidak boleh** implement sebelum status `✅ Approved`.

---

## 🛠️ Phase 2: Implementation

### 2.1 Read Spec Thoroughly

**AI akan:**
1. Baca `FEATURE_MAP.md` untuk cek status
2. Jika ada link `→ docs/specs/`, baca file spec lengkap
3. Identifikasi files yang perlu diubah
4. Plan implementation order (DB → Backend → Frontend)

### 2.2 Development Order

**Standard flow:**

```
1. Database Migration
   → shared/migrations/0000XX_feature.up.sql
   → Run locally: go run shared/sdk/migrate/migrate.go

2. Backend Implementation
   → Handler functions
   → Business logic
   → API endpoints
   → Error handling

3. Backend Testing
   → Unit tests (*_test.go)
   → make check

4. Frontend Implementation
   → API client methods (api.ts)
   → Components (*.vue)
   → Routes (router.ts)
   → State management (jika perlu)

5. Integration Testing
   → Manual test via UI
   → Verify all AC

6. Documentation Update
   → README.md (jika workflow berubah)
   → CLAUDE.md (jika pattern baru)
```

### 2.3 Code Quality Standards

**Backend (Go):**
- [ ] Handler max 450 lines (ponytail: split jika mendekati)
- [ ] No `panic()` di handler
- [ ] Parameterized queries (anti SQL injection)
- [ ] Structured logging (`slog.Info/Error`)
- [ ] Error messages user-friendly

**Frontend (Vue):**
- [ ] Component max 500 lines (ponytail: extract reusable)
- [ ] Composition API (not Options API)
- [ ] Tailwind CSS (no inline styles)
- [ ] Props validation
- [ ] Loading states + error handling

**Database:**
- [ ] Every table has: `id`, `tenant_id`, `created_at`, `updated_at`
- [ ] Index on `tenant_id`
- [ ] Foreign keys with `ON DELETE CASCADE`
- [ ] Migration reversible (`.down.sql`)

### 2.4 Testing

**Wajib sebelum commit:**

```bash
# Backend
make check                      # lint + build + test
go test ./apps/umkm/... -v     # specific package

# Frontend
cd frontend/umkm-web
npm run lint                    # ESLint
npm run build                   # Vite build check
```

**Manual verification:**
- [ ] All AC dari spec terpenuhi
- [ ] Happy path works
- [ ] Error cases handled gracefully
- [ ] Mobile responsive (jika UI)
- [ ] No console errors

---

## ✅ Phase 3: Completion

### 3.1 Update Implementation Status

**Di `FEATURE_MAP.md`:**

```markdown
| F0XX | Feature Name | ✅ Approved | ✅ Done | 2026-07-01 |
```

**Di spec file (jika ada):**

```markdown
**Implementation:** ✅ Done
```

### 3.2 Commit Convention

```bash
git add <files>
git commit -m "Implement F0XX: Feature Name

- Add endpoint POST /api/resource
- Add UI component FeatureComponent.vue
- Migration 0000XX adds feature_table
- All AC verified

Closes #123

Co-Authored-By: Claude <noreply@anthropic.com>"
```

**Commit message structure:**
- **Line 1:** `Implement F0XX: Feature Name` (max 72 chars)
- **Line 2:** Blank
- **Line 3+:** Bullet points (what changed)
- **Footer:** Closes #issue, Co-Authored-By

### 3.3 Deployment Checklist

**Development:**
```bash
make start-all              # Verify local
curl http://localhost:8201/api/resource  # Smoke test
```

**Production:**
```bash
# 1. Merge to main
git push origin main

# 2. SSH to server
ssh production-server

# 3. Pull & migrate
git pull
make migrate-up

# 4. Restart services
docker compose restart umkm-accounting

# 5. Smoke test
curl https://api.example.com/healthz
```

---

## 🔄 Iteration & Refinement

### When to Iterate

**Feature complete but:**
- User feedback → UX improvement needed
- Edge case discovered → add handling
- Performance issue → optimization

**Process:**
1. Create new AC in spec (e.g., AC-7, AC-8)
2. Update status `⏳ Draft` → discuss
3. After agreement → `✅ Approved`
4. Implement iteration
5. Update status `✅ Done` (again)

### When to Reject

**Spec rejected if:**
- Business priority shifted
- Technical unfeasible
- Duplicate with existing feature
- Out of product scope

**Mark as:**
```markdown
**Status:** ❌ Rejected
**Reason:** [1-2 kalimat alasan]
```

---

## 📊 Metrics & Success

### Feature Success Metrics

**Measure after 1 week production:**
- Adoption rate (% tenant yang pakai)
- Error rate (sentry/logs)
- Performance (response time)
- User feedback (support tickets)

**Dashboard:**
- Grafana: feature-specific metrics
- Sentry: error tracking
- PostHog (future): event analytics

---

## 💡 Tips & Best Practices

### Writing Good Specs

✅ **DO:**
- Start with "User can..."
- Include examples & mockups
- Define edge cases explicitly
- Specify error messages

❌ **DON'T:**
- Vague AC ("intuitive UI")
- Assume implementation details
- Skip rollback plan
- Forget about mobile

### Working with AI

✅ **DO:**
- Provide complete spec before asking implement
- Clarify ambiguous requirements
- Review AI code before merge
- Ask "why" if approach unclear

❌ **DON'T:**
- Say "implement X" tanpa spec
- Assume AI knows business context
- Skip testing "karena AI sudah test"
- Commit without review

### Common Pitfalls

1. **Spec Drift** — Implementation berbeda dari spec
   - Fix: Sync spec dengan actual implementation

2. **Feature Creep** — Scope bertambah terus
   - Fix: Lock AC, tulis enhancement di "Future" section

3. **Testing Debt** — Deploy tanpa test coverage
   - Fix: Block merge jika `make check` fail

4. **Documentation Lag** — Code update, docs tidak
   - Fix: Update docs dalam commit yang sama

---

## 🔗 Quick Links

- [Spec Template](SPEC_TEMPLATE.md)
- [Feature Registry](FEATURE_MAP.md)
- [Detailed Specs](specs/)
- [Contributing Guide](../CONTRIBUTING.md)
- [Developer Guide](DEVELOPER_GUIDE.md)
