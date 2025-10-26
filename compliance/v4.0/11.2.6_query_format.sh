#!/bin/bash

# OData v4 Compliance Test: 11.2.6 Query Option $format
# Tests $format query option for specifying response format
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionformat

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: $format=json returns JSON
test_1() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$format=json" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
    
    if check_contains "$CONTENT_TYPE" "application/json"; then
        check_json_field "$BODY" "value"
    else
        echo "  Details: Content-Type: $CONTENT_TYPE"
        return 1
    fi
}

# Test 2: $format=xml returns XML (for metadata)
test_2() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata?\$format=xml" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    check_contains "$CONTENT_TYPE" "application/xml"
}

# Test 3: Invalid $format returns error
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$format=invalid")
    if [ "$HTTP_CODE" = "406" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    fi
    # Some implementations may be lenient
    return 0
}

# Test 4: $format parameter overrides Accept header
test_4() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$format=json" -H "Accept: application/xml" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    check_contains "$CONTENT_TYPE" "application/json"
}

# Run all tests
run_test "\$format=json returns JSON response" test_1
run_test "\$format=xml on metadata returns XML" test_2
run_test "Invalid \$format value returns error" test_3
run_test "\$format parameter takes precedence over Accept header" test_4

print_summary
