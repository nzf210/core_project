#!/bin/sh
# Import semua workflow di /workflows/*.json ke n8n.
# Tandai .workflow_imported supaya idempotent (tidak re-import setiap restart).

set -e

sleep 5

IMPORT_DIR="/workflows"
MARKER="/home/node/.n8n/.workflow_imported"

if [ ! -f "$MARKER" ]; then
  echo "Importing n8n workflows from $IMPORT_DIR ..."
  for wf in "$IMPORT_DIR"/*.json; do
    [ -e "$wf" ] || continue
    name=$(basename "$wf")
    echo "  → $name"
    n8n import:workflow --input="$wf" || echo "    failed: $name"
  done
  touch "$MARKER"
  echo "All workflows imported."
else
  echo "Workflows already imported, skipping. Delete $MARKER to force re-import."
fi

exec n8n start
