#!/bin/bash

# OData v4 Compliance Test: 11.2.5.1 System Query Option $filter
# Tests $filter query option according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionfilter

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic eq (equals) operator
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ID%20eq%201")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=ID%20eq%201")
    check_json_field "$BODY" "value"
}

# Test 2: gt (greater than) operator
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%20100")
    check_json_field "$BODY" "value"
}

# Test 3: String contains function
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    check_json_field "$BODY" "value"
}

# Test 4: Boolean operators (and)
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%201000")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%201000")
    check_json_field "$BODY" "value"
}

# Test 5: Boolean operators (or)
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ID%20eq%201%20or%20ID%20eq%202")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=ID%20eq%201%20or%20ID%20eq%202")
    check_json_field "$BODY" "value"
}

# Test 6: Parentheses for grouping
test_6() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=(Price%20gt%20100)%20and%20(ID%20lt%2010)")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=(Price%20gt%20100)%20and%20(ID%20lt%2010)")
    check_json_field "$BODY" "value"
}

# Run all tests
run_test "\$filter with eq operator" test_1
run_test "\$filter with gt operator" test_2
run_test "\$filter with contains() function" test_3
run_test "\$filter with 'and' operator" test_4
run_test "\$filter with 'or' operator" test_5
run_test "\$filter with parentheses" test_6

print_summary
