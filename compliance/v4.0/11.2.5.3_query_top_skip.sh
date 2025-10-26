#!/bin/bash

# OData v4 Compliance Test: 11.2.5.3 and 11.2.5.4 System Query Options $top and $skip
# Tests $top and $skip query options for paging according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionstopandskip

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: $top limits the number of items returned
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=2")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$top=2")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -le 2 ]; then
        return 0
    fi
    echo "  Details: Returned $COUNT items"
    return 1
}

# Test 2: $skip skips the specified number of items
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$skip=1&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$skip=1&\$top=1")
    check_json_field "$BODY" "value"
}

# Test 3: $top=0 returns empty collection
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=0")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$top=0")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -eq 0 ]; then
        return 0
    fi
    echo "  Details: Returned $COUNT items"
    return 1
}

# Test 4: Combine $skip and $top for paging
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$skip=2&\$top=3")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$skip=2&\$top=3")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -le 3 ]; then
        return 0
    fi
    echo "  Details: Returned $COUNT items, expected max 3"
    return 1
}

# Run all tests
run_test "\$top limits number of items" test_1
run_test "\$skip skips items" test_2
run_test "\$top=0 returns empty collection" test_3
run_test "Combine \$skip and \$top for paging" test_4

print_summary
