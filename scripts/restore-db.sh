#!/bin/bash
set -e

echo "=== 1. Menghentikan container sementara (volume kosong wch-stg_pgdata) ==="
docker stop wch-postgres wch-pgbouncer wch-redis wch-n8n-main 2>/dev/null || true
docker rm wch-postgres wch-pgbouncer wch-redis wch-n8n-main 2>/dev/null || true

echo "=== 2. Menjalankan container dengan volume data asli (core_project_pgdata) ==="
cd /home/syahril/dev/core_project
COMPOSE_PROJECT_NAME=core_project docker compose up -d postgres pgbouncer redis n8n-main

echo "=== 3. Menunggu PostgreSQL & PgBouncer siap ==="
until docker exec wch-postgres pg_isready -U wch_admin -d wch_platform 2>/dev/null; do
  echo "Menunggu PostgreSQL ready..."
  sleep 2
done

echo "=== 4. Verifikasi Data Tenant di Database Asli ==="
docker exec wch-postgres psql -U wch_admin -d wch_platform -c "SELECT id, name, plan, business_name FROM tenants;"

echo "=== 5. Restart Microservices Native (Go / Air) ==="
./scripts/dev-native.sh --stop || true
sleep 2
./scripts/dev-native.sh

echo "=== SELESAI! Database asli aktif dan semua service sudah terhubung ==="
