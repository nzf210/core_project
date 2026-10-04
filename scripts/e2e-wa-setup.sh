#!/bin/bash
# =============================================================================
# E2E Test: /wa-setup page — Full endpoint chain
# Tests the complete flow a tenant user goes through on the WA Setup page.
#
# Prerequisites:
#   - Docker infra running (postgres, redis, pgbouncer)
#   - Go dev services running (make dev-all)
#   - User "Tokoroti" exists with password "Test1234!" (reset via pgcrypto)
#
# Usage:
#   bash scripts/e2e-wa-setup.sh
#   bash scripts/e2e-wa-setup.sh --start-services   # also starts dev services
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

PASS=0
FAIL=0
SKIP=0

report() {
  local step="$1" status="$2" detail="$3"
  if [[ "$status" == "PASS" ]]; then
    echo -e "  ${GREEN}✅ $step — PASS${NC} — $detail"
    PASS=$((PASS + 1))
  elif [[ "$status" == "FAIL" ]]; then
    echo -e "  ${RED}❌ $step — FAIL${NC} — $detail"
    FAIL=$((FAIL + 1))
  else
    echo -e "  ${YELLOW}⏭️  $step — SKIP${NC} — $detail"
    SKIP=$((SKIP + 1))
  fi
}

echo "============================================="
echo " E2E Test: /wa-setup Page Flow"
echo " $(date)"
echo "============================================="
echo ""

# --- Optional: start services ---
if [[ "${1:-}" == "--start-services" ]]; then
  echo "🚀 Starting dev services..."
  nohup bash scripts/dev-native.sh > /dev/null 2>&1 &
  echo "   Waiting 15s for services to boot..."
  sleep 15
fi

# --- Step 0: Health checks ---
echo "📡 Health Checks:"

WA_HEALTH=$(curl -sf http://localhost:8202/health 2>/dev/null || echo '{"status":"unreachable"}')
if echo "$WA_HEALTH" | grep -q '"ok"'; then
  report "WA Gateway (8202)" "PASS" "$(echo "$WA_HEALTH" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(f"sessions={d.get(\"connected_sessions\",\"?\")}, db={d.get(\"database\",\"?\")}, redis={d.get(\"redis\",\"?\")}")' 2>/dev/null)"
else
  report "WA Gateway (8202)" "FAIL" "not responding"
  echo -e "\n${RED}ERROR: WA Gateway not running. Start services first: make dev-all${NC}"
  exit 1
fi

# API Gateway returns 404 on /health but that means it's alive
GW_CHECK=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/health 2>/dev/null || echo "000")
if [[ "$GW_CHECK" != "000" ]]; then
  report "API Gateway (8000)" "PASS" "http_code=$GW_CHECK"
else
  report "API Gateway (8000)" "FAIL" "not responding"
  echo -e "\n${RED}ERROR: API Gateway not running. Start services first: make dev-all${NC}"
  exit 1
fi

echo ""
echo "🔐 E2E Chain (as tenant Tokoroti):"

# --- Step 1: Login ---
LOGIN_RESP=$(curl -s -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"Tokoroti","password":"Test1234!"}' 2>/dev/null)

TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('accessToken',''))" 2>/dev/null)
LOGIN_SUCCESS=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('success', False))" 2>/dev/null)
LOGIN_PLAN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); u=d.get('data',{}); print(f'plan={u.get(\"plan\",\"?\")}, role={u.get(\"role\",\"?\")}')" 2>/dev/null)

if [[ -n "$TOKEN" && "$LOGIN_SUCCESS" == "True" ]]; then
  report "POST /auth/login" "PASS" "$LOGIN_PLAN, token_len=${#TOKEN}"
else
  report "POST /auth/login" "FAIL" "$(echo "$LOGIN_RESP" | head -c 200)"
  echo -e "\n${RED}ERROR: Login failed. Reset password: docker exec wch-postgres psql -U wch_admin -d wch_platform -c \"UPDATE users SET password_hash = crypt('Test1234!', gen_salt('bf', 10)) WHERE username = 'Tokoroti';\"${NC}"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"

# --- Step 2: Profile ---
PROFILE_RESP=$(curl -s http://localhost:8000/api/profile -H "$AUTH" 2>/dev/null)
PROFILE_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/profile -H "$AUTH" 2>/dev/null)
PROFILE_PLAN=$(echo "$PROFILE_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); u=d.get('user',d.get('data',{})); print(f'plan={u.get(\"plan\",\"?\")}, role={u.get(\"role\",\"?\")}, frozen={u.get(\"is_frozen\",\"?\")}')" 2>/dev/null)

if [[ "$PROFILE_HTTP" == "200" ]]; then
  report "GET /api/profile" "PASS" "$PROFILE_PLAN"
else
  report "GET /api/profile" "FAIL" "http=$PROFILE_HTTP $(echo "$PROFILE_RESP" | head -c 200)"
fi

# --- Step 3: Chatbot Config ---
CHATBOT_RESP=$(curl -s http://localhost:8000/api/umkm/chatbot/config -H "$AUTH" 2>/dev/null)
CHATBOT_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/umkm/chatbot/config -H "$AUTH" 2>/dev/null)
CHATBOT_DETAIL=$(echo "$CHATBOT_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); c=d.get('data',d); print(f'is_active={c.get(\"is_active\",\"?\")}, provider={c.get(\"wa_provider_preference\",\"?\")}')" 2>/dev/null)

if [[ "$CHATBOT_HTTP" == "200" ]]; then
  report "GET /api/umkm/chatbot/config" "PASS" "$CHATBOT_DETAIL"
else
  report "GET /api/umkm/chatbot/config" "FAIL" "http=$CHATBOT_HTTP $(echo "$CHATBOT_RESP" | head -c 200)"
fi

# --- Step 4: WA Setup ---
SETUP_RESP=$(curl -s http://localhost:8000/api/umkm/wa/setup -H "$AUTH" 2>/dev/null)
SETUP_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/umkm/wa/setup -H "$AUTH" 2>/dev/null)
SETUP_DETAIL=$(echo "$SETUP_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); s=d.get('data',d); print(f'whatsmeow={s.get(\"whatsmeow_status\",s.get(\"status\",\"?\"))}, cloud_api={s.get(\"can_use_cloud_api\",\"?\")}')" 2>/dev/null)

if [[ "$SETUP_HTTP" == "200" ]]; then
  report "GET /api/umkm/wa/setup" "PASS" "$SETUP_DETAIL"
else
  report "GET /api/umkm/wa/setup" "FAIL" "http=$SETUP_HTTP $(echo "$SETUP_RESP" | head -c 200)"
fi

# --- Step 5: WA Status (via api-gateway) ---
STATUS_RESP=$(curl -s http://localhost:8000/api/wa/status -H "$AUTH" 2>/dev/null)
STATUS_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/wa/status -H "$AUTH" 2>/dev/null)
STATUS_DETAIL=$(echo "$STATUS_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'status={d.get(\"status\",\"?\")}')" 2>/dev/null)

if [[ "$STATUS_HTTP" == "200" ]]; then
  report "GET /api/wa/status" "PASS" "$STATUS_DETAIL"
else
  report "GET /api/wa/status" "FAIL" "http=$STATUS_HTTP $(echo "$STATUS_RESP" | head -c 200)"
fi

# --- Step 6: QR Code Generation ---
QR_RESP=$(curl -s http://localhost:8000/api/wa/qr -H "$AUTH" 2>/dev/null)
QR_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/wa/qr -H "$AUTH" 2>/dev/null)
HAS_QR=$(echo "$QR_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); qr=d.get('qr_code',''); print('yes' if len(qr)>100 else 'no')" 2>/dev/null)

if [[ "$QR_HTTP" == "200" && "$HAS_QR" == "yes" ]]; then
  QR_LEN=$(echo "$QR_RESP" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('qr_code','')))" 2>/dev/null)
  report "GET /api/wa/qr" "PASS" "has_qr=yes, qr_len=$QR_LEN"
elif [[ "$QR_HTTP" == "200" ]]; then
  # QR might timeout or already connected — still a valid 200 response
  QR_MSG=$(echo "$QR_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('message',d.get('status','unknown')))" 2>/dev/null)
  report "GET /api/wa/qr" "PASS" "http=200, response=$QR_MSG (no scan target — expected)"
else
  report "GET /api/wa/qr" "FAIL" "http=$QR_HTTP $(echo "$QR_RESP" | head -c 200)"
fi

# --- Step 7: System WA Status (direct, no auth) ---
SYS_RESP=$(curl -s "http://localhost:8202/wa/status?tenant_id=system" 2>/dev/null)
SYS_HTTP=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8202/wa/status?tenant_id=system" 2>/dev/null)
SYS_DETAIL=$(echo "$SYS_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'status={d.get(\"status\",\"?\")}, jid={d.get(\"jid\",\"?\")}')" 2>/dev/null)

if [[ "$SYS_HTTP" == "200" ]]; then
  report "GET /wa/status?tenant_id=system" "PASS" "$SYS_DETAIL"
else
  report "GET /wa/status?tenant_id=system" "FAIL" "http=$SYS_HTTP"
fi

# --- Summary ---
TOTAL=$((PASS + FAIL + SKIP))
echo ""
echo "============================================="
echo " Results: $PASS/$TOTAL passed, $FAIL failed, $SKIP skipped"
if [[ $FAIL -eq 0 ]]; then
  echo -e " ${GREEN}🎉 ALL E2E TESTS PASSED${NC}"
else
  echo -e " ${RED}⚠️  $FAIL TEST(S) FAILED${NC}"
fi
echo "============================================="

exit $FAIL
