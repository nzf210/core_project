#!/bin/sh

# Tunggu database siap (jika perlu)
sleep 5

# Cek apakah workflow sudah pernah di-import
if [ ! -f /home/node/.n8n/.workflow_imported ]; then
  echo "Meng-import Master Automations Workflow..."
  n8n import:workflow --input=/workflows/master_automations.json
  
  if [ $? -eq 0 ]; then
    echo "Import berhasil."
    touch /home/node/.n8n/.workflow_imported
  else
    echo "Gagal meng-import workflow."
  fi
else
  echo "Workflow sudah pernah di-import, melewati langkah ini."
fi

# Mulai n8n
exec n8n start
