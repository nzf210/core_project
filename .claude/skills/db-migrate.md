# db-migrate

Database migration runner untuk WCH Platform.

## When to invoke

- User mengetik `/db-migrate`
- Setelah menambah migration file baru di `shared/migrations/`
- Sebelum deploy ke staging/production
- Saat troubleshoot database schema issues

## What this does

Run database migrations menggunakan auto-migration system:

```bash
# Check migration status
make migrate-status

# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down
```

## Migration System

WCH Platform menggunakan **auto-migration** yang berjalan otomatis saat service startup:

- Location: `shared/migrations/000001_*.up.sql` (sequential numbering)
- Tracker table: `schema_migrations` (mencatat versi yang sudah dijalankan)
- Format: `NNNNNN_description.up.sql` / `.down.sql`

## Creating New Migration

```bash
# Generate migration files
make migrate-new NAME=add_invoice_table

# Edit generated files:
# shared/migrations/000029_add_invoice_table.up.sql
# shared/migrations/000029_add_invoice_table.down.sql
```

## Migration Best Practices

1. **Idempotent** - migration harus bisa dijalankan ulang tanpa error
2. **Transactional** - setiap migration wrapped dalam transaction
3. **Reversible** - `.down.sql` harus bisa rollback `.up.sql`
4. **Test locally** - jalankan di local DB sebelum commit

## Services dengan Auto-Migration

- `services/auth-service`
- `services/billing-service`
- `apps/umkm/accounting`
- `apps/umkm/chatbot`
- `apps/campaign/api`

Migration runner: `shared/sdk/migrate`
