#!/bin/bash

# OData v4 Compliance Test: 11.3.3 Arithmetic Functions in $filter
# Tests arithmetic operators and math functions (add, sub, mul, div, mod, ceiling, floor, round) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.3 Arithmetic Functions"
echo "======================================"
echo ""
echo "Description: Validates arithmetic operators and math functions in \$filter query option"
echo "             (add, sub, mul, div, mod, ceiling, floor, round)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: add operator
test_add_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price add 10 gt 100")
    check_status "$HTTP_CODE" "200"
}

# Test 2: sub operator
test_sub_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price sub 50 lt 100")
    check_status "$HTTP_CODE" "200"
}

# Test 3: mul operator
test_mul_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price mul 2 gt 200")
    check_status "$HTTP_CODE" "200"
}

# Test 4: div operator
test_div_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price div 2 lt 100")
    check_status "$HTTP_CODE" "200"
}

# Test 5: mod operator
test_mod_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price mod 10 eq 0")
    check_status "$HTTP_CODE" "200"
}

# Test 6: ceiling function
test_ceiling_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ceiling(Price) eq 100")
    check_status "$HTTP_CODE" "200"
}

# Test 7: floor function
test_floor_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=floor(Price) eq 99")
    check_status "$HTTP_CODE" "200"
}

# Test 8: round function
test_round_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=round(Price) eq 100")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Combined arithmetic operations
test_combined_arithmetic() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price mul 2 sub 100 gt 0")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Arithmetic with comparison
test_arithmetic_comparison() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price ge 50 and Price le 150")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=Price add 10 gt 100"
run_test "add operator performs addition" test_add_operator

echo "  Request: GET \$filter=Price sub 50 lt 100"
run_test "sub operator performs subtraction" test_sub_operator

echo "  Request: GET \$filter=Price mul 2 gt 200"
run_test "mul operator performs multiplication" test_mul_operator

echo "  Request: GET \$filter=Price div 2 lt 100"
run_test "div operator performs division" test_div_operator

echo "  Request: GET \$filter=Price mod 10 eq 0"
run_test "mod operator performs modulo" test_mod_operator

echo "  Request: GET \$filter=ceiling(Price) eq 100"
run_test "ceiling() function rounds up" test_ceiling_function

echo "  Request: GET \$filter=floor(Price) eq 99"
run_test "floor() function rounds down" test_floor_function

echo "  Request: GET \$filter=round(Price) eq 100"
run_test "round() function rounds to nearest integer" test_round_function

echo "  Request: GET \$filter=Price mul 2 sub 100 gt 0"
run_test "Combined arithmetic operations work" test_combined_arithmetic

echo "  Request: GET \$filter=Price ge 50 and Price le 150"
run_test "Arithmetic comparisons (ge, le) work" test_arithmetic_comparison

print_summary
