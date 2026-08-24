#!/bin/sh
set -e

# Generate userlist.txt from environment variables at runtime
# Plaintext password required for PgBouncer to perform SCRAM handshake with PostgreSQL
cat > /etc/pgbouncer/userlist.txt << EOF
"${DB_USER:-postgres}" "${DB_PASSWORD:-postgres}"
EOF

chmod 600 /etc/pgbouncer/userlist.txt

# Generate pgbouncer.ini with dynamic DB_NAME
cat > /etc/pgbouncer/pgbouncer.ini << EOF
[databases]
${DB_NAME:-wch_core} = host=postgres port=5432 dbname=${DB_NAME:-wch_core}
${N8N_DB_NAME:-wch_n8n_db} = host=postgres port=5432 dbname=${N8N_DB_NAME:-wch_n8n_db}

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432

auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt

pool_mode = transaction

max_client_conn = 10000
default_pool_size = 100
min_pool_size = 10
reserve_pool_size = 50
reserve_pool_timeout = 3

server_reset_query = DISCARD ALL
server_check_delay = 30
server_check_query = SELECT 1
server_lifetime = 3600
server_idle_timeout = 30

server_round_robin = 0

log_connections = 1
log_disconnections = 1
log_pooler_errors = 1
verbose = 0

stats_period = 60

admin_users = ${DB_USER:-postgres}
EOF

exec pgbouncer /etc/pgbouncer/pgbouncer.ini
