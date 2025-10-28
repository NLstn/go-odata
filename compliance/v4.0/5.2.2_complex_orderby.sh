#!/bin/bash

# OData v4 Compliance Test: 5.2.2 Complex Type Ordering
# Ensures nested complex properties can be used in $orderby expressions
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

ORDER_ENCODED=$(printf %s "Dimensions/Length desc" | jq -sRr @uri)
REQUEST_URL="${SERVER_URL}/Products?%24orderby=${ORDER_ENCODED}"

log_test_start "5.2.2" "Complex type ordering by nested property"

HTTP_CODE=$(http_get "$REQUEST_URL")
RESPONSE=$(http_get_body "$REQUEST_URL")

if [ "$HTTP_CODE" != "200" ]; then
    log_test_failure "Expected HTTP 200, got $HTTP_CODE" "$RESPONSE"
    exit 1
fi

FIRST_ID=""
if command -v jq >/dev/null 2>&1; then
    FIRST_ID=$(echo "$RESPONSE" | jq -r '.value[0].ID // empty')
else
    FIRST_ID=$(python - <<'PY'
import json, sys
try:
    data = json.load(sys.stdin)
    value = data.get('value', [])
    if value:
        first = value[0]
        print(first.get('ID', ''))
    else:
        print('')
except Exception:
    print('')
PY
)
fi

if [ -z "$FIRST_ID" ]; then
    log_test_failure "Unable to determine first entity ID from response" "$RESPONSE"
    exit 1
fi

if [ "$FIRST_ID" != "1" ]; then
    log_test_failure "Expected first entity ID to be 1 when ordering by Dimensions/Length desc" "$RESPONSE"
    exit 1
fi

log_test_success
