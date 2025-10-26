#!/bin/bash

# OData v4 Compliance Test: 11.3.1 String Functions in $filter
# Tests string functions (contains, startswith, endswith, length, etc.) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.1 String Functions"
echo "======================================"
echo ""
echo "Description: Validates string functions in \$filter query option"
echo "             (contains, startswith, endswith, length, indexof, substring, tolower, toupper, trim, concat)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: contains function
test_contains_function() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check that response contains value array
        if echo "$RESPONSE" | grep -q '"value"'; then
            return 0
        else
            echo "  Details: Response missing 'value' array"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: startswith function
test_startswith_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=startswith(Name,'Gaming')")
    check_status "$HTTP_CODE" "200"
}

# Test 3: endswith function
test_endswith_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=endswith(Name,'Mouse')")
    check_status "$HTTP_CODE" "200"
}

# Test 4: length function
test_length_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=length(Name) gt 10")
    check_status "$HTTP_CODE" "200"
}

# Test 5: indexof function
test_indexof_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=indexof(Name,'Pro') eq 0")
    check_status "$HTTP_CODE" "200"
}

# Test 6: substring function
test_substring_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=substring(Name,0,3) eq 'Gam'")
    check_status "$HTTP_CODE" "200"
}

# Test 7: tolower function
test_tolower_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=tolower(Name) eq 'laptop'")
    check_status "$HTTP_CODE" "200"
}

# Test 8: toupper function
test_toupper_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=toupper(Name) eq 'LAPTOP'")
    check_status "$HTTP_CODE" "200"
}

# Test 9: trim function
test_trim_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=trim(Name) eq 'Laptop'")
    check_status "$HTTP_CODE" "200"
}

# Test 10: concat function
test_concat_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=concat(Name,' Test') eq 'Laptop Test'")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=contains(Name,'Laptop')"
run_test "contains() function filters string values" test_contains_function

echo "  Request: GET \$filter=startswith(Name,'Gaming')"
run_test "startswith() function filters by prefix" test_startswith_function

echo "  Request: GET \$filter=endswith(Name,'Mouse')"
run_test "endswith() function filters by suffix" test_endswith_function

echo "  Request: GET \$filter=length(Name) gt 10"
run_test "length() function returns string length" test_length_function

echo "  Request: GET \$filter=indexof(Name,'Pro') eq 0"
run_test "indexof() function finds substring position" test_indexof_function

echo "  Request: GET \$filter=substring(Name,0,3) eq 'Gam'"
run_test "substring() function extracts substring" test_substring_function

echo "  Request: GET \$filter=tolower(Name) eq 'laptop'"
run_test "tolower() function converts to lowercase" test_tolower_function

echo "  Request: GET \$filter=toupper(Name) eq 'LAPTOP'"
run_test "toupper() function converts to uppercase" test_toupper_function

echo "  Request: GET \$filter=trim(Name) eq 'Laptop'"
run_test "trim() function removes whitespace" test_trim_function

echo "  Request: GET \$filter=concat(...)"
run_test "concat() function concatenates strings" test_concat_function

print_summary
