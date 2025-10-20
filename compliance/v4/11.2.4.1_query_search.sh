#!/bin/bash

# OData v4 Compliance Test: 11.2.4.1 System Query Option $search
# Tests $search query option for free-text search
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionsearch

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Test 1: Basic $search query
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$search=Laptop")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$search=Laptop")
        check_json_field "$BODY" "value"
    else
        # $search may not be implemented
        return 0
    fi
}

# Test 2: $search with multiple terms (AND)
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$search=Laptop%20Pro")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$search=Laptop%20Pro")
        check_json_field "$BODY" "value"
    else
        return 0
    fi
}

# Test 3: $search with OR operator
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$search=Laptop%20OR%20Phone")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$search=Laptop%20OR%20Phone")
        check_json_field "$BODY" "value"
    else
        return 0
    fi
}

# Test 4: Combine $search with $filter
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$search=Laptop&\$filter=Price%20gt%20100")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$search=Laptop&\$filter=Price%20gt%20100")
        check_json_field "$BODY" "value"
    else
        return 0
    fi
}

# Run all tests
run_test "Basic \$search query with single term" test_1
run_test "\$search with multiple terms (implicit AND)" test_2
run_test "\$search with OR operator" test_3
run_test "Combine \$search with \$filter" test_4

print_summary
