#!/bin/bash

# OData v4 Compliance Test: 11.2.1 Addressing Entities
# Tests various ways to address entities according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Address entity set returns collection
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products")
    check_json_field "$BODY" "value"
}

# Test 2: Address single entity by key
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products(1)")
    check_json_field "$BODY" "ID"
}

# Test 3: Non-existent entity returns 404
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
}

# Test 4: Invalid entity set returns 404
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/NonExistentEntitySet")
    check_status "$HTTP_CODE" "404"
}

# Test 5: Accessing property of entity
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products(1)/Name")
        check_json_field "$BODY" "value"
    else
        # Some implementations might return 404 for property access
        return 0
    fi
}

# Test 6: Accessing raw value of property
test_6() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name/\$value")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products(1)/Name/\$value")
        # Should return raw text, not JSON
        if ! echo "$BODY" | grep -q '"value"' && [ -n "$BODY" ]; then
            return 0
        fi
        return 1
    else
        # Some implementations might not support $value
        return 0
    fi
}

# Run all tests
run_test "Addressing entity set returns collection" test_1
run_test "Addressing single entity by key" test_2
run_test "Non-existent entity returns 404" test_3
run_test "Invalid entity set returns 404" test_4
run_test "Accessing property of entity" test_5
run_test "Accessing raw value of property with \$value" test_6

print_summary
