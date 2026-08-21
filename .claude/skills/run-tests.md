# run-tests

Run test suite untuk verifikasi code changes.

## When to invoke

- User mengetik `/run-tests`
- Sebelum commit code changes
- Setelah refactoring
- Saat fix bugs

## What this does

Menjalankan Go test suite dengan coverage dan verification:

```bash
# Run all tests in short mode (skip DB-dependent tests)
go test ./... -short -v

# Full test suite dengan DB integration
go test ./... -v

# Dengan coverage report
go test ./... -cover -coverprofile=coverage.out
```

## Execution Steps

1. Detect changed packages (`git diff --name-only | grep "\.go$"`)
2. Run tests untuk packages yang berubah
3. Report hasil (pass/fail) dengan coverage
4. Block commit jika ada test failures

## Test Patterns

**Unit tests** (pure functions, no DB):
- Fast, no external dependencies
- Run di `-short` mode

**Integration tests** (DB, Redis, external services):
- Skip di `-short` mode dengan check:
  ```go
  if testing.Short() {
      t.Skip("skipping integration test in short mode")
  }
  ```

## Integration

Auto-invoke:
- Pre-commit hook (run tests di changed packages)
- CI/CD workflow
- Before deploy
