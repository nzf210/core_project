#!/bin/bash
set -e

echo "Stopping temporary containers..."
docker stop wch-n8n-main wch-pgbouncer wch-redis wch-postgres 2>/dev/null || true
docker rm wch-n8n-main wch-pgbouncer wch-redis wch-postgres wch-stg-n8n-worker-1 wch-stg-n8n-worker-2 wch-stg-n8n-worker-3 2>/dev/null || true

echo "Starting original infrastructure containers..."
docker start wch-stg-postgres wch-stg-redis wch-stg-pgbouncer wch-stg-n8n-main core_project-n8n-worker-1 core_project-n8n-worker-2 core_project-n8n-worker-3

echo "Infrastructure containers started."
