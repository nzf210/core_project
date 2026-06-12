-- infra/postgres/init.sql
-- Auto-run on first postgres start via /docker-entrypoint-initdb.d/
-- Creates wch_n8n database for N8N persistence (terpisah dari platform).
-- Idempotent: aman dijalankan berkali-kali, tidak error jika database sudah ada.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'wch_n8n') THEN
        CREATE USER wch_n8n WITH ENCRYPTED PASSWORD 'M_4k4zz45@n8nsaasumkm';
        RAISE NOTICE 'Role wch_n8n created.';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'wch_n8n_db') THEN
        CREATE DATABASE wch_n8n_db OWNER wch_n8n;
        RAISE NOTICE 'Database wch_n8n_db created for N8N persistence.';
    ELSE
        RAISE NOTICE 'Database wch_n8n_db already exists, skipping.';
    END IF;
END $$;
