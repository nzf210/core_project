# new-feature

Workflow untuk menambah fitur baru sesuai Spec-First Workflow WCH Platform.

## When to invoke

- User mengetik `/new-feature`
- Sebelum implement fitur baru
- Untuk memastikan fitur sudah di-approve di `docs/FEATURE_MAP.md`

## What this does

Enforce Spec-First Workflow:

1. **Cek FEATURE_MAP.md** - Apakah fitur sudah terdaftar?
2. **Baca detailed spec** - Jika ada link `→ docs/specs/F<ID>_*.md`
3. **Verify status** - Sudah ✅ Approved?
4. **Block jika belum approved** - Jangan implement fitur yang masih ⏳ Draft atau ❌ Rejected

## Spec-First Workflow (MANDATORY)

```
USER menulis SPEC → AI review & clarify → USER approve
     ↓                    ↓                      ↓
FEATURE_MAP.md      AI tanya clarifications   USER comment/approve
                         ↓                      ↓
                 AI wait for approval    AI implement dari SPEC
                                              ↓
                                      USER review diff
                                              ↓
                                      JALANKAN TESTING
```

## Execution Steps

### Step 1: Check Feature Registry

```bash
# Cek apakah fitur sudah ada di registry
grep -i "feature-name" docs/FEATURE_MAP.md
```

### Step 2: Verify Status

Cek kolom `Status (Spec)` di FEATURE_MAP.md:
- ✅ Approved → OK untuk implement
- ⏳ Draft → STOP, tanya USER untuk clarification
- ❌ Rejected → STOP, jangan implement

### Step 3: Read Detailed Spec (if exists)

Jika ada link `→ docs/specs/F<ID>_*.md`, baca file spec lengkap untuk:
- Acceptance Criteria (AC)
- Technical Details
- API Contracts
- Database Schema
- Test Cases

### Step 4: Implement

Implement sesuai spec dengan:
- Follow coding conventions di CLAUDE.md
- Add unit tests
- Update `Implementation` status di FEATURE_MAP.md setelah selesai

## Example

```bash
# User request: "Implement F048 WA Provider Preference"

# Step 1: Check registry
grep "F048" docs/FEATURE_MAP.md
# → F048 | WA Provider Preference | ✅ Approved | ✅ Done

# Step 2: Read spec
cat docs/specs/F048_wa_provider_preferences.md

# Step 3: Implement sesuai AC
# - Add enum column
# - Add UI dropdown
# - Implement routing logic
# - Write tests

# Step 4: Update status
# Edit FEATURE_MAP.md: Implementation → ✅ Done
```

## Integration

Auto-invoke saat:
- User request fitur baru tanpa menyebut feature ID
- Before implement any business logic changes
- Saat ambiguous requirements detected
