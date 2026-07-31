-- infra/postgres/init.sql
-- Auto-run on first postgres start via /docker-entrypoint-initdb.d/
-- Creates wch_n8n_db database for N8N persistence (terpisah dari platform).
-- Idempotent: aman dijalankan berkali-kali, tidak error jika database sudah ada.

DROP FUNCTION IF EXISTS create_database_if_not_exists(text, text);

CREATE FUNCTION create_database_if_not_exists(db_name text, db_owner text)
RETURNS void AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = db_name) THEN
        EXECUTE format('CREATE DATABASE %I OWNER %I', db_name, db_owner);
        RAISE NOTICE 'Database % created.', db_name;
    ELSE
        RAISE NOTICE 'Database % already exists, skipping.', db_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

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

-- Step 2: Create database (via helper function, outside transaction block)
-- CREATE DATABASE cannot run inside DO $$ transaction, so use a function
SELECT create_database_if_not_exists('wch_n8n_db', 'wch_n8n');
