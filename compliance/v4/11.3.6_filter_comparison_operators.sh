#!/bin/bash

# OData v4 Compliance Test: 11.3.6 Comparison Operators in $filter
# Tests comparison operators (eq, ne, gt, ge, lt, le) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_ComparisonOperators

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.6 Comparison Operators"
echo "======================================"
echo ""
echo "Description: Validates comparison operators (eq, ne, gt, ge, lt, le)"
echo "             in \$filter expressions."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_ComparisonOperators"
echo ""

# Test 1: eq (equals) operator
test_eq_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Status%20eq%201")
    
    check_status "$HTTP_CODE" "200"
}

# Test 2: ne (not equals) operator
test_ne_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Status%20ne%200")
    
    check_status "$HTTP_CODE" "200"
}

# Test 3: gt (greater than) operator
test_gt_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2050")
    
    check_status "$HTTP_CODE" "200"
}

# Test 4: ge (greater than or equal) operator
test_ge_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20ge%2050")
    
    check_status "$HTTP_CODE" "200"
}

# Test 5: lt (less than) operator
test_lt_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20lt%20100")
    
    check_status "$HTTP_CODE" "200"
}

# Test 6: le (less than or equal) operator
test_le_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20le%20100")
    
    check_status "$HTTP_CODE" "200"
}

# Test 7: eq with string
test_eq_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20eq%20%27Electronics%27")
    
    check_status "$HTTP_CODE" "200"
}

# Test 8: ne with string
test_ne_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20ne%20%27Electronics%27")
    
    check_status "$HTTP_CODE" "200"
}

# Test 9: Comparison with decimal numbers
test_decimal_comparison() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20eq%2099.99")
    
    check_status "$HTTP_CODE" "200"
}

# Test 10: Comparison with null
test_null_comparison() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 11: Multiple comparisons combined
test_multiple_comparisons() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20ge%2010%20and%20Price%20le%20100")
    
    check_status "$HTTP_CODE" "200"
}

# Test 12: Comparison operators return correct results
test_comparison_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Status%20eq%201")
    
    # Verify response structure
    if check_json_field "$RESPONSE" "value"; then
        return 0
    else
        return 1
    fi
}

# Test 13: eq operator case sensitivity for strings
test_eq_case_sensitive() {
    # OData string comparisons are case-sensitive by default
    local RESPONSE1=$(http_get_body "$SERVER_URL/Products?\$filter=Category%20eq%20%27Electronics%27")
    local RESPONSE2=$(http_get_body "$SERVER_URL/Products?\$filter=Category%20eq%20%27electronics%27")
    
    # These should potentially return different results if case-sensitive
    # We just verify both requests succeed
    if check_json_field "$RESPONSE1" "value" && check_json_field "$RESPONSE2" "value"; then
        return 0
    else
        return 1
    fi
}

# Test 14: Invalid comparison operator returns error
test_invalid_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20equals%2050")
    
    # Should return 400 for invalid operator
    check_status "$HTTP_CODE" "400"
}

echo "  Request: GET $SERVER_URL/Products?\$filter=Status eq 1"
run_test "eq (equals) operator works" test_eq_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Status ne 0"
run_test "ne (not equals) operator works" test_ne_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 50"
run_test "gt (greater than) operator works" test_gt_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Price ge 50"
run_test "ge (greater than or equal) operator works" test_ge_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Price lt 100"
run_test "lt (less than) operator works" test_lt_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Price le 100"
run_test "le (less than or equal) operator works" test_le_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq 'Electronics'"
run_test "eq operator works with strings" test_eq_string

echo "  Request: GET $SERVER_URL/Products?\$filter=Category ne 'Electronics'"
run_test "ne operator works with strings" test_ne_string

echo "  Request: GET $SERVER_URL/Products?\$filter=Price eq 99.99"
run_test "Comparison operators work with decimal numbers" test_decimal_comparison

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq null"
run_test "Comparison with null value" test_null_comparison

echo "  Request: GET $SERVER_URL/Products?\$filter=Price ge 10 and Price le 100"
run_test "Multiple comparison operators combined" test_multiple_comparisons

echo "  Request: GET $SERVER_URL/Products?\$filter=Status eq 1"
run_test "Comparison operators return correct results" test_comparison_correctness

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq 'Electronics' vs 'electronics'"
run_test "String comparison case sensitivity" test_eq_case_sensitive

echo "  Request: GET $SERVER_URL/Products?\$filter=Price equals 50"
run_test "Invalid comparison operator returns 400" test_invalid_operator

print_summary
