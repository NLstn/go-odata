#!/bin/bash

# OData v4 Compliance Test: 11.3.4 Type Functions in $filter
# Tests type checking and casting functions (isof, cast) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.4 Type Functions"
echo "======================================"
echo ""
echo "Description: Validates type checking and casting functions in \$filter query option"
echo "             (isof, cast)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: isof function with property
test_isof_function_property() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof(Price,Edm.Decimal)")
    check_status "$HTTP_CODE" "200"
}

# Test 2: isof function with entity type
test_isof_function_entity() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof('Product')")
    check_status "$HTTP_CODE" "200"
}

# Test 3: cast function
test_cast_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=cast(Status,Edm.String) eq '1'")
    check_status "$HTTP_CODE" "200"
}

# Test 4: isof with null check
test_isof_null_check() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof(Name,Edm.String) eq true")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Negative isof test
test_isof_negative() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof(Price,Edm.String) eq false")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=isof(Price,Edm.Decimal)"
run_test "isof() function checks property type" test_isof_function_property

echo "  Request: GET \$filter=isof('Product')"
run_test "isof() function checks entity type" test_isof_function_entity

echo "  Request: GET \$filter=cast(Status,Edm.String) eq '1'"
run_test "cast() function casts to specified type" test_cast_function

echo "  Request: GET \$filter=isof(Name,Edm.String) eq true"
run_test "isof() with null check returns true" test_isof_null_check

echo "  Request: GET \$filter=isof(Price,Edm.String) eq false"
run_test "isof() returns false for wrong type" test_isof_negative

print_summary
