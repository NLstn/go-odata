#!/bin/bash

# OData v4 Compliance Test: 11.2.5.2 System Query Option $select and $orderby
# Tests $select and $orderby query options according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic $select with single property
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name")
    check_json_field "$BODY" "Name"
}

# Test 2: $select with multiple properties
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name,Price")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name,Price")
    if check_json_field "$BODY" "Name" && check_json_field "$BODY" "Price"; then
        return 0
    fi
    return 1
}

# Test 3: Basic $orderby ascending
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Price%20asc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=Price%20asc")
    check_json_field "$BODY" "value"
}

# Test 4: $orderby descending
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Price%20desc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=Price%20desc")
    check_json_field "$BODY" "value"
}

# Test 5: $orderby with multiple properties
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=CategoryID,Price%20desc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=CategoryID,Price%20desc")
    check_json_field "$BODY" "value"
}

# Test 6: Combining $select and $orderby
test_6() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name,Price&\$orderby=Price")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name,Price&\$orderby=Price")
    if check_json_field "$BODY" "Name" && check_json_field "$BODY" "Price"; then
        return 0
    fi
    return 1
}

# Run all tests
run_test "\$select with single property" test_1
run_test "\$select with multiple properties" test_2
run_test "\$orderby ascending" test_3
run_test "\$orderby descending" test_4
run_test "\$orderby with multiple properties" test_5
run_test "Combining \$select and \$orderby" test_6

print_summary
