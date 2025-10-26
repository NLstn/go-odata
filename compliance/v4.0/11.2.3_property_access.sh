#!/bin/bash

# OData v4 Compliance Test: 11.2.3 Addressing Individual Properties
# Tests addressing and accessing individual properties according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_AddressingIndividualProperties

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Access a primitive property
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products(1)/Name")
    check_json_field "$BODY" "value"
}

# Test 2: Access a primitive property with $value
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name/\$value")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products(1)/Name/\$value")
    # Should return plain text without JSON wrapping
    if ! echo "$BODY" | grep -q '"value"' && [ -n "$BODY" ]; then
        return 0
    fi
    return 1
}

# Test 3: Access non-existent property should return 404
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/NonExistentProperty")
    check_status "$HTTP_CODE" "404"
}

# Test 4: Access property of non-existent entity should return 404
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)/Name")
    check_status "$HTTP_CODE" "404"
}

# Test 5: Property access should have proper Content-Type
test_5() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)/Name" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    check_contains "$CONTENT_TYPE" "application/json"
}

# Test 6: $value should have text/plain Content-Type for strings
test_6() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)/Name/\$value" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    check_contains "$CONTENT_TYPE" "text/plain"
}

# Run all tests
run_test "Access a primitive property (Name)" test_1
run_test "Access primitive property raw value with \$value" test_2
run_test "Access non-existent property returns 404" test_3
run_test "Access property of non-existent entity returns 404" test_4
run_test "Property access returns proper Content-Type" test_5
run_test "\$value has text/plain Content-Type for string property" test_6

print_summary
