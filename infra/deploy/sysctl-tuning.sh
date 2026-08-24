#!/bin/bash
# OS-level kernel tuning untuk 100K concurrent connections
# Jalankan sekali di production/staging server sebagai root
# Usage: sudo bash sysctl-tuning.sh

set -e

echo "Applying kernel tuning for 100K concurrent connections..."

# Apply sysctl settings
sysctl -w net.core.somaxconn=65535
sysctl -w net.ipv4.tcp_max_syn_backlog=65535
sysctl -w net.ipv4.ip_local_port_range="1024 65535"
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_fin_timeout=15
sysctl -w net.core.netdev_max_backlog=65535
sysctl -w fs.file-max=1000000

# Persist across reboots
cat > /etc/sysctl.d/99-wch-highload.conf << 'EOF'
# WCH Platform — high-concurrency kernel tuning
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
net.core.netdev_max_backlog = 65535
fs.file-max = 1000000
EOF

# Raise file descriptor limits
cat > /etc/security/limits.d/99-wch-highload.conf << 'EOF'
# WCH Platform — file descriptor limits
* soft nofile 1000000
* hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
EOF

echo "Done. Re-login or run 'ulimit -n 1000000' in current shell to apply fd limits."
echo "Kernel params are active immediately (no reboot needed)."
