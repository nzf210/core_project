# SonarQube localStorage Security Fix

## Summary
Fixed all localStorage security vulnerabilities flagged by SonarQube related to tainted data (CWE-79, CWE-20).

## Changes Made

### 1. Updated Sanitization Functions (`frontend/umkm-web/src/api.ts`)
Modified allowlist-based functions to return `null` for invalid input (instead of fallback values):

```typescript
const ALLOWED_ROLES = ['owner', 'admin', 'staff', 'kasir', 'superadmin'] as const
const ALLOWED_THEMES = ['dark', 'light', 'system'] as const
const ALLOWED_BOOLEAN_STRINGS = ['true', 'false'] as const

// Returns null if invalid - forces caller to handle the invalid case
export function sanitizeRole(v: unknown): string | null
export function sanitizeTheme(v: unknown): string | null
export function sanitizeBoolean(v: unknown): string | null

// Existing functions (return empty string if invalid)
export function sanitizeUUID(v: unknown): string
export function sanitizeJWT(v: unknown): string
export function sanitizeText(v: unknown, maxLen = 200): string
export function sanitizeURL(v: unknown): string
```

**Key change:** Functions now return `null` instead of fallback values, forcing call sites to add explicit guard checks.

### 2. Applied Guard Checks Across Codebase

All call sites now validate before storing. Pattern used:

```typescript
// Before: direct storage (potentially tainted)
localStorage.setItem('role', sanitizeRole(data.role))

// After: guard check prevents storage if invalid
const role = sanitizeRole(data.role)
if (!role) {
  // Handle invalid case (return, error, fallback)
  return
}
localStorage.setItem('role', role) // Only validated values reach storage
```

**Files Updated:**
- `frontend/umkm-web/src/App.vue` — guard check for superadmin role
- `frontend/umkm-web/src/router/index.ts` — guard checks in syncUserDataToStorage + impersonate login
- `frontend/umkm-web/src/composables/useTheme.ts` — guard check for theme preference
- `frontend/umkm-web/src/components/Login.vue` — guard checks for role + onboarding flag
- `frontend/umkm-web/src/components/Onboarding.vue` — guard check for onboarding flag
- `frontend/umkm-web/src/components/SuperAdminLogin.vue` — guard check for role
- `frontend/umkm-web/src/__tests__/security.spec.ts` — added null checks for test assertions

### 3. Key Pattern: Named Constants + Early Return Guard

SonarQube requires two things for compliant localStorage usage:

1. **Named constants** for allowlist validation (not inline arrays)
2. **Early return guard** that prevents storage if validation fails

```typescript
// ❌ Noncompliant (inline validation + always stores something)
const theme = params.get('theme');
const sanitized = ['light', 'dark'].includes(theme) ? theme : 'light';
localStorage.setItem('theme', sanitized); // SonarQube can't prove this is safe

// ✅ Compliant (named constant + guard check prevents storage)
const ALLOWED_THEMES = ['light', 'dark', 'high-contrast'];
const theme = params.get('theme');
if (!ALLOWED_THEMES.includes(theme)) {
  return; // Early return - localStorage.setItem never called with tainted data
}
localStorage.setItem('theme', theme);
```

**Why the guard pattern matters:** SonarQube's static analysis can't trace that a sanitize function returning a fallback value is safe. The guard pattern makes it **statically provable** that localStorage.setItem only receives validated values.

### 4. Test Files (Excluded from Scan)

Test files (`__tests__/`) still have unsanitized localStorage for mock data, which is acceptable for testing purposes. Configure SonarQube to exclude these paths:

```
frontend/umkm-web/src/__tests__/**
frontend/umkm-web/src/components/__tests__/**
```

## Verification

```bash
# Build succeeds with no TypeScript errors
cd frontend/umkm-web && npm run build
# Result: ✓ built in 504ms

# All localStorage.setItem calls use validated variables
grep -rn "localStorage.setItem" src --include="*.vue" --include="*.ts" | \
  grep -v "__tests__" | grep -v "sanitize"
# Result: 9 matches — all using validated variables (role, onboardingFlag, mustChangeFlag)
#         that passed guard checks before reaching localStorage.setItem
```

## False Positives Identified

**File:** `tools/scripts/verify_phone_registration_fix.sql`
- **Issue:** SonarQube flagged NULL comparison patterns
- **Status:** FALSE POSITIVE — file already uses correct SQL standard (`IS NULL` / `IS NOT NULL`)
- **Action:** Mark as false positive in SonarQube UI

## Security Guarantees

All user-controlled data passing through localStorage now has:
1. **Format validation** (UUID, JWT patterns)
2. **Allowlist validation** (roles, themes, boolean strings)
3. **Content sanitization** (dangerous characters stripped)
4. **Length limits** (prevent overflow attacks)

This prevents XSS attacks via localStorage injection (CWE-79) and improves input validation (CWE-20).
