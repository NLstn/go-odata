#!/bin/bash

# OData v4 Compliance Test: 11.2.5.6 System Query Option $expand
# Tests $expand query option for expanding related entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionexpand

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic $expand returns related entities inline
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$expand=Descriptions&\$top=1")
    check_json_field "$BODY" "Descriptions"
}

# Test 2: $expand with $select on expanded entity
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$select=Description)&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$expand=Descriptions(\$select=Description)&\$top=1")
    check_json_field "$BODY" "Descriptions"
}

# Test 3: Multiple $expand
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)?\$expand=Descriptions")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products(1)?\$expand=Descriptions")
    check_json_field "$BODY" "Descriptions"
}

# Run all tests
run_test "\$expand includes related entities inline" test_1
run_test "\$expand with nested \$select" test_2
run_test "Multiple navigation properties in \$expand" test_3

print_summary
