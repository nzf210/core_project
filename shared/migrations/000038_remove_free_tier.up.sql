-- 000038_remove_free_tier.up.sql
-- Hapus tier "free" dari semua tier registry.
-- Semua tenant "free" di-migrate ke "lite" (default akses minimum).
-- Kolom `plan` di `tenants` dan `plan_tier` di `usage_quotas` di-set DEFAULT 'lite'.

-- 1. Migrate data existing
UPDATE tenants SET plan = 'lite' WHERE plan = 'free';
UPDATE usage_quotas SET plan_tier = 'lite' WHERE plan_tier = 'free';

-- 2. Update DEFAULT untuk tenant baru
ALTER TABLE tenants ALTER COLUMN plan SET DEFAULT 'lite';
ALTER TABLE usage_quotas ALTER COLUMN plan_tier SET DEFAULT 'lite';

-- 3. Pastikan constraint check tidak reject 'lite' (no-op defensive)
-- (Tidak ada CHECK constraint aktif di schema saat ini, hanya validasi aplikasi)
