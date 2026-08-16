# Deploy SSH Timeout Fix Guide

**Error:** `dial tcp ***:3209: i/o timeout` when GitHub Actions tries to connect to VPS

**Status:** Network connectivity issue between GitHub Actions runners and VPS port 3209

---

## Root Cause

GitHub Actions cannot reach your VPS on port 3209. Possible reasons:

1. VPS firewall blocking port 3209 from GitHub's IP ranges
2. SSH daemon not listening on port 3209
3. VPS IP address changed but GitHub secrets not updated
4. ISP/cloud provider blocking non-standard SSH ports

---

## Fix Steps

### Step 1: Verify VPS SSH Configuration

SSH into your VPS manually and check:

```bash
# Check if SSH is listening on port 3209
sudo ss -tlnp | grep :3209

# Check SSH daemon config
sudo grep "^Port" /etc/ssh/sshd_config

# If port is wrong, edit sshd_config
sudo nano /etc/ssh/sshd_config
# Add or change: Port 3209

# Restart SSH daemon
sudo systemctl restart sshd
```

### Step 2: Check VPS Firewall

```bash
# Ubuntu/Debian (ufw)
sudo ufw status
sudo ufw allow 3209/tcp
sudo ufw reload

# CentOS/RHEL (firewalld)
sudo firewall-cmd --list-all
sudo firewall-cmd --permanent --add-port=3209/tcp
sudo firewall-cmd --reload

# Raw iptables
sudo iptables -L -n | grep 3209
sudo iptables -A INPUT -p tcp --dport 3209 -j ACCEPT
sudo iptables-save
```

### Step 3: Check Cloud Provider Security Groups

If your VPS is on a cloud provider (AWS, GCP, DigitalOcean, etc.):

1. Log into cloud console
2. Find your VPS's security group / firewall rules
3. Add inbound rule: **TCP port 3209 from 0.0.0.0/0** (or GitHub's IP ranges)

**GitHub Actions IP ranges:**
- Download from: https://api.github.com/meta
- Look for `actions` field

### Step 4: Test Connection from GitHub Actions

Add this temporary test job to `.github/workflows/deploy.yml`:

```yaml
  test-ssh-connection:
    name: Test SSH Connection
    runs-on: ubuntu-latest
    steps:
      - name: Test SSH connectivity
        run: |
          # Test if port is open
          timeout 10 bash -c "cat < /dev/null > /dev/tcp/${{ secrets.VPS_HOST }}/3209" && echo "Port 3209 is open" || echo "Port 3209 is CLOSED or FILTERED"
          
          # Try SSH with verbose output
          ssh -v -p 3209 -o ConnectTimeout=10 -o StrictHostKeyChecking=no ${{ secrets.VPS_USERNAME }}@${{ secrets.VPS_HOST }} "echo 'SSH connection successful'" || echo "SSH failed"
```

### Step 5: Verify GitHub Secrets

Check your GitHub repository secrets at:
`https://github.com/YOUR_ORG/core_project/settings/secrets/actions`

Required secrets:
- `VPS_HOST` - VPS IP address or hostname
- `VPS_USERNAME` - SSH username (usually `deploy` or `root`)
- `VPS_SSH_KEY` - Private SSH key (full key including `-----BEGIN` and `-----END` lines)

**Test VPS_HOST is correct:**
```bash
# From your local machine
ping $(echo "YOUR_VPS_IP")
ssh -p 3209 deploy@YOUR_VPS_IP
```

### Step 6: Alternative - Use Standard SSH Port 22

If port 3209 is being blocked by network policies, switch to port 22:

**Option A: Change workflow to use port 22**

Edit `.github/workflows/deploy.yml` line 79:
```yaml
# Change from:
echo "vps_ssh_port=${{ vars.VPS_SSH_PORT || '3209' }}" >> $GITHUB_OUTPUT

# To:
echo "vps_ssh_port=${{ vars.VPS_SSH_PORT || '22' }}" >> $GITHUB_OUTPUT
```

**Option B: Set GitHub repository variable**

1. Go to: `https://github.com/YOUR_ORG/core_project/settings/variables/actions`
2. Create variable: `VPS_SSH_PORT` = `22`
3. Update VPS SSH config to listen on port 22

---

## Quick Diagnostic Commands

Run these on your VPS to diagnose:

```bash
# 1. Is SSH running?
sudo systemctl status sshd

# 2. What ports is SSH listening on?
sudo ss -tlnp | grep sshd

# 3. Can localhost connect?
nc -zv 127.0.0.1 3209

# 4. Firewall status
sudo ufw status verbose  # Ubuntu
sudo firewall-cmd --list-all  # CentOS

# 5. Check if port is publicly accessible (from external machine)
nc -zv YOUR_VPS_IP 3209
```

---

## Recommended Solution

**For staging environment**, the safest fix:

1. **Use standard port 22** for SSH (less likely to be blocked)
2. **Update GitHub variable** `VPS_SSH_PORT` to `22`
3. **Keep VPS firewall rules** allowing GitHub Actions IPs only (security)

**For production**, consider:
- Using GitHub's self-hosted runner on the VPS network (no SSH needed)
- Using Cloudflare Tunnel (zero-trust access, no open ports)
- Using Tailscale subnet router (secure access without firewall rules)

---

## After Fix - Test Deployment

Once connectivity is restored:

```bash
# Tag to trigger deploy
git tag stg-be-v1.0.0-test
git push origin stg-be-v1.0.0-test

# Watch GitHub Actions
# https://github.com/YOUR_ORG/core_project/actions
```

---

## Related Files

- Workflow: `.github/workflows/deploy.yml`
- VPS setup script: `scripts/staging-setup.sh`
- Port registry: `docs/PORT_REGISTRY.md`

---

**Next Step:** Check VPS firewall and SSH config first (Step 1-2), then verify GitHub secrets (Step 5).
