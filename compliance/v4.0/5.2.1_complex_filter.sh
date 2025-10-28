#!/bin/bash

# OData v4 Compliance Test: 5.2.1 Complex Type Filtering
# Ensures nested complex properties can be used in $filter expressions
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

FILTER_ENCODED=$(printf %s "ShippingAddress/City eq 'Seattle'" | jq -sRr @uri)
REQUEST_URL="${SERVER_URL}/Products?%24filter=${FILTER_ENCODED}"

log_test_start "5.2.1" "Complex type filtering by nested property"

HTTP_CODE=$(http_get "$REQUEST_URL")
RESPONSE=$(http_get_body "$REQUEST_URL")

if [ "$HTTP_CODE" != "200" ]; then
    log_test_failure "Expected HTTP 200, got $HTTP_CODE" "$RESPONSE"
    exit 1
fi

RESULT_COUNT=""
if command -v jq >/dev/null 2>&1; then
    RESULT_COUNT=$(echo "$RESPONSE" | jq '.value | length')
else
    RESULT_COUNT=$(python - <<'PY'
import json, sys
try:
    data = json.load(sys.stdin)
    print(len(data.get('value', [])))
except Exception:
    print('')
PY
)
fi

if [ -z "$RESULT_COUNT" ] || [ "$RESULT_COUNT" = "0" ]; then
    log_test_failure "Expected at least one product in filtered results" "$RESPONSE"
    exit 1
fi

if ! echo "$RESPONSE" | grep -q '"City":"Seattle"'; then
    log_test_failure "Filtered results do not include expected city" "$RESPONSE"
    exit 1
fi

log_test_success
