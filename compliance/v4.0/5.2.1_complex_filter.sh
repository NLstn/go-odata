#!/bin/bash

# OData v4 Compliance Test: 5.2.1 Complex Type Filtering
# Ensures nested complex properties can be used in $filter expressions
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.2.1 Complex Type Filtering"
echo "======================================"
echo ""
echo "Description: Validates that nested complex properties can participate in $filter expressions"
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

test_filter_nested_complex_property() {
    local filter_expr="ShippingAddress/City eq 'Seattle'"
    local encoded_filter=$(encode_query_option "$filter_expr")
    local request_url="${SERVER_URL}/Products?%24filter=${encoded_filter}"

    local http_code=$(http_get "$request_url")
    local response=$(http_get_body "$request_url")

    if ! check_status "$http_code" "200"; then
        echo "  Response: $response"
        return 1
    fi

    local result_count=""
    if command -v jq >/dev/null 2>&1; then
        result_count=$(echo "$response" | jq '.value | length')
    else
        result_count=$(printf '%s' "$response" | python - <<'PY'
import json
import sys
try:
    data = json.load(sys.stdin)
    print(len(data.get("value", [])))
except Exception:
    print("")
PY
)
    fi

    if [ -z "$result_count" ] || [ "$result_count" = "0" ]; then
        echo "  Details: Expected at least one entity to match the filter"
        echo "  Response: $response"
        return 1
    fi

    if ! echo "$response" | grep -q '"City":"Seattle"'; then
        echo "  Details: Filtered response missing expected City value"
        echo "  Response: $response"
        return 1
    fi

    return 0
}

run_test "Filter by nested complex property" test_filter_nested_complex_property

print_summary
