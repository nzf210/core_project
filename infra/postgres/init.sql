-- infra/postgres/init.sql
-- Auto-run on first postgres start via /docker-entrypoint-initdb.d/
-- Creates wch_n8n_db database for N8N persistence (terpisah dari platform).
-- Idempotent: aman dijalankan berkali-kali, tidak error jika database sudah ada.

-- Step 1: Create role (inside DO $$ so we can use IF NOT EXISTS)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'wch_n8n') THEN
        CREATE USER wch_n8n WITH ENCRYPTED PASSWORD 'M_4k4zz45@n8nsaasumkm';
        RAISE NOTICE 'Role wch_n8n created.';
    ELSE
        RAISE NOTICE 'Role wch_n8n already exists, skipping.';
    END IF;
END $$;

-- Step 2: Create database if not exists (using \gexec outside transaction)
SELECT 'CREATE DATABASE wch_n8n_db OWNER wch_n8n'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wch_n8n_db')\gexec
GRANT ALL PRIVILEGES ON DATABASE wch_n8n_db TO wch_n8n;
