# code-review

Automated code review untuk correctness, security, dan simplification.

## When to invoke

- User mengetik `/code-review`
- Sebelum merge PR
- Setelah implement fitur kompleks
- Saat refactoring code

## What this does

Review code changes untuk:
1. **Correctness bugs** - Logic errors, edge cases tidak tertangani
2. **Security issues** - SQL injection, path traversal, XSS
3. **Simplification** - Code duplication, unnecessary complexity
4. **Best practices** - Error handling, resource leaks

## Execution Steps

### Step 1: Get changed files

```bash
# Current branch vs main
git diff main...HEAD --name-only

# Or staged files
git diff --cached --name-only
```

### Step 2: Review per file type

**Go files:**
- SQL injection risk (string concatenation in queries)
- Path traversal (user input in file paths)
- Error handling (unchecked errors)
- Resource leaks (unclosed connections)
- Complexity (functions >50 lines)

**Vue/TypeScript files:**
- XSS risk (v-html with user input)
- API call error handling
- Component complexity
- Props validation

**Bash scripts:**
- Command injection (unquoted variables)
- Error handling (missing `set -e`)
- Safe file operations

### Step 3: Report findings

```
=== Code Review: services/auth-service/main.go ===

❌ BLOCKER: SQL injection risk at line 245
   query := "SELECT * FROM users WHERE id = '" + userID + "'"
   Fix: Use parameterized query with $1 placeholder

⚠️  MAJOR: Unchecked error at line 312
   result, err := db.Query(...)
   Fix: Add if err != nil check

✅ Security: All user inputs properly validated
✅ Error handling: 95% coverage
```

## Integration

- Auto-invoke before PR creation
- Block merge jika ada BLOCKER findings
- Generate review comments on PR
