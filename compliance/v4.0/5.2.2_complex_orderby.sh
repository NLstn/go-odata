#!/bin/bash

# OData v4 Compliance Test: 5.2.2 Complex Type Ordering
# Ensures nested complex properties can be used in $orderby expressions
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.2.2 Complex Type Ordering"
echo "======================================"
echo ""
echo "Description: Validates that nested complex properties can be used in $orderby expressions"
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType"
echo ""

encode_query_option() {
    local raw_value="$1"
    if command -v jq >/dev/null 2>&1; then
        printf %s "$raw_value" | jq -sRr @uri
    else
        RAW_VALUE="$raw_value" python - <<'PY'
import os
import urllib.parse
print(urllib.parse.quote(os.environ["RAW_VALUE"], safe=""))
PY
    fi
}

test_orderby_nested_complex_property() {
    local order_expr="Dimensions/Length desc"
    local encoded_order=$(encode_query_option "$order_expr")
    local request_url="${SERVER_URL}/Products?%24orderby=${encoded_order}"

    local http_code=$(http_get "$request_url")
    local response=$(http_get_body "$request_url")

    if ! check_status "$http_code" "200"; then
        echo "  Response: $response"
        return 1
    fi

    local first_id=""
    if command -v jq >/dev/null 2>&1; then
        first_id=$(echo "$response" | jq -r '.value[0].ID // empty')
    else
        first_id=$(printf '%s' "$response" | python - <<'PY'
import json
import sys
try:
    data = json.load(sys.stdin)
    value = data.get("value", [])
    if value:
        first = value[0]
        print(first.get("ID", ""))
    else:
        print("")
except Exception:
    print("")
PY
)
    fi

    if [ -z "$first_id" ]; then
        echo "  Details: Unable to determine first entity ID from response"
        echo "  Response: $response"
        return 1
    fi

    if [ "$first_id" != "10" ]; then
        echo "  Details: Expected first entity ID to be 10 when ordering by descending Length"
        echo "  Response: $response"
        return 1
    fi

    return 0
}

run_test "Order by nested complex property" test_orderby_nested_complex_property

print_summary
