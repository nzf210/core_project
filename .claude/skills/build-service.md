# build-service

Build dan verify Go services sebelum commit atau deploy.

## When to invoke

- User mengetik `/build-service`
- Sebelum commit perubahan Go code
- Sebelum deploy ke staging/production

## What this does

Menjalankan build check untuk memastikan semua Go services compile tanpa error:

```bash
# Build all services
go build ./...

# Verify no errors
if [ $? -eq 0 ]; then
    echo "✅ Build successful - all services compile"
else
    echo "❌ Build failed - fix errors before commit"
    exit 1
fi
```

## Execution

1. Run `go build ./...` untuk compile semua packages
2. Report hasil build (success/fail)
3. Jika fail, tampilkan error messages
4. Block commit jika build gagal (kecuali `--no-verify`)

## Integration

Auto-invoke sebelum:
- Git commit (via pre-commit hook)
- Docker build
- Deploy workflow
