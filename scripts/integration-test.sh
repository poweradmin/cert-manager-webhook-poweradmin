#!/usr/bin/env bash
#
# Integration test for cert-manager-webhook-poweradmin against a local PowerAdmin instance.
#
# Prerequisites:
#   - PowerAdmin running locally (e.g., via devcontainer)
#   - A test zone must exist in PowerAdmin
#   - A valid API key
#
# Usage:
#   ./scripts/integration-test.sh [POWERADMIN_URL] [API_KEY] [ZONE_NAME]
#
# Defaults (matching PowerAdmin devcontainer test data):
#   POWERADMIN_URL=http://localhost:8080
#   API_KEY=test-api-key-for-automated-testing-12345
#   ZONE_NAME=admin-zone.example.com

set -euo pipefail

POWERADMIN_URL="${1:-http://localhost:8080}"
API_KEY="${2:-test-api-key-for-automated-testing-12345}"
ZONE_NAME="${3:-admin-zone.example.com}"
API_VERSION="${API_VERSION:-v2}"

RECORD_NAME="_acme-test.${ZONE_NAME}"
RECORD_VALUE="integration-test-$(date +%s)"
# V1 API requires TXT content to be enclosed in quotes; V2 accepts bare values
if [ "$API_VERSION" = "v1" ]; then
  RECORD_CONTENT="\"\\\"${RECORD_VALUE}\\\"\""
else
  RECORD_CONTENT="\"${RECORD_VALUE}\""
fi

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== cert-manager-webhook-poweradmin integration test ==="
echo "PowerAdmin URL: ${POWERADMIN_URL}"
echo "API Version:    ${API_VERSION}"
echo "Zone:           ${ZONE_NAME}"
echo ""

# 1. Test API connectivity
echo "--- Test API connectivity ---"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "X-API-Key: ${API_KEY}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones")

if [ "$HTTP_CODE" = "200" ]; then
  pass "API reachable (HTTP ${HTTP_CODE})"
else
  fail "API returned HTTP ${HTTP_CODE}"
  echo "Cannot continue without API access."
  exit 1
fi

# 2. Find zone ID
echo "--- Find zone ---"
if [ "$API_VERSION" = "v2" ]; then
  ZONE_ID=$(curl -s \
    -H "X-API-Key: ${API_KEY}" \
    "${POWERADMIN_URL}/api/${API_VERSION}/zones" | \
    python3 -c "import sys,json; resp=json.load(sys.stdin); zones=resp['data']['zones']; print(next((z['id'] for z in zones if z['name']=='${ZONE_NAME}'), ''))")
else
  ZONE_ID=$(curl -s \
    -H "X-API-Key: ${API_KEY}" \
    "${POWERADMIN_URL}/api/${API_VERSION}/zones" | \
    python3 -c "import sys,json; resp=json.load(sys.stdin); zones=resp['data']; print(next((z['id'] for z in zones if z['name']=='${ZONE_NAME}'), ''))")
fi

if [ -n "$ZONE_ID" ]; then
  pass "Found zone '${ZONE_NAME}' with ID ${ZONE_ID}"
else
  fail "Zone '${ZONE_NAME}' not found"
  exit 1
fi

# 3. Create TXT record (simulates Present)
echo "--- Create TXT record (Present) ---"
CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"${RECORD_NAME}\",\"type\":\"TXT\",\"content\":${RECORD_CONTENT},\"ttl\":120,\"priority\":0,\"disabled\":0}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records")

CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
if [ "$CREATE_CODE" = "201" ] || [ "$CREATE_CODE" = "200" ]; then
  pass "Created TXT record (HTTP ${CREATE_CODE})"
else
  fail "Failed to create TXT record (HTTP ${CREATE_CODE})"
  echo "$CREATE_RESPONSE" | head -1
fi

# 4. Verify record exists
echo "--- Verify TXT record exists ---"
RECORDS=$(curl -s \
  -H "X-API-Key: ${API_KEY}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records?type=TXT")

RECORD_ID=$(echo "$RECORDS" | python3 -c "
import sys, json
resp = json.load(sys.stdin)
records = resp['data']
for r in records:
    if r['name'] == '${RECORD_NAME}' and r['content'] == ${RECORD_CONTENT}:
        print(r['id'])
        break
" 2>/dev/null || echo "")

if [ -n "$RECORD_ID" ]; then
  pass "TXT record verified (ID ${RECORD_ID})"
else
  fail "TXT record not found after creation"
fi

# 5. Test idempotency (create same record again)
echo "--- Test idempotency ---"
IDEM_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"${RECORD_NAME}\",\"type\":\"TXT\",\"content\":${RECORD_CONTENT},\"ttl\":120,\"priority\":0,\"disabled\":0}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records")

if [ "$IDEM_CODE" = "201" ] || [ "$IDEM_CODE" = "200" ] || [ "$IDEM_CODE" = "409" ]; then
  pass "Idempotent create handled (HTTP ${IDEM_CODE})"
else
  fail "Unexpected response for duplicate create (HTTP ${IDEM_CODE})"
fi

# 6. Delete all matching TXT records (simulates CleanUp — deletes all matches like the solver does)
echo "--- Delete TXT records (CleanUp) ---"
ALL_RECORD_IDS=$(curl -s \
  -H "X-API-Key: ${API_KEY}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records?type=TXT" | \
  python3 -c "
import sys, json
resp = json.load(sys.stdin)
records = resp['data']
for r in records:
    if r['name'] == '${RECORD_NAME}' and r['content'] == ${RECORD_CONTENT}:
        print(r['id'])
" 2>/dev/null || echo "")

DELETE_OK=true
for rid in $ALL_RECORD_IDS; do
  DELETE_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X DELETE \
    -H "X-API-Key: ${API_KEY}" \
    "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records/${rid}")

  if [ "$DELETE_CODE" != "204" ] && [ "$DELETE_CODE" != "200" ]; then
    DELETE_OK=false
  fi
done

if [ -z "$ALL_RECORD_IDS" ]; then
  fail "No records found to delete"
elif [ "$DELETE_OK" = true ]; then
  pass "Deleted all matching TXT records"
else
  fail "Failed to delete some TXT records"
fi

# 7. Verify record is gone
echo "--- Verify TXT record deleted ---"
REMAINING=$(curl -s \
  -H "X-API-Key: ${API_KEY}" \
  "${POWERADMIN_URL}/api/${API_VERSION}/zones/${ZONE_ID}/records?type=TXT" | \
  python3 -c "
import sys, json
resp = json.load(sys.stdin)
records = resp['data']
matches = [r for r in records if r['name'] == '${RECORD_NAME}' and r['content'] == ${RECORD_CONTENT}]
print(len(matches))
" 2>/dev/null || echo "unknown")

if [ "$REMAINING" = "0" ]; then
  pass "TXT record confirmed deleted"
else
  fail "TXT record still exists after deletion (count: ${REMAINING})"
fi

# 8. Test with v1 API (if testing v2)
if [ "$API_VERSION" = "v2" ]; then
  echo "--- Test v1 API fallback ---"
  V1_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: ${API_KEY}" \
    "${POWERADMIN_URL}/api/v1/zones")

  if [ "$V1_CODE" = "200" ]; then
    pass "V1 API also reachable (HTTP ${V1_CODE})"
  else
    fail "V1 API returned HTTP ${V1_CODE}"
  fi
fi

# Summary
echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
