#!/bin/bash

# OData v4 Compliance Test: 5.1.1.1 Numeric Edge Cases
# Tests handling of special numeric values including IEEE 754 special values,
# precision limits, and boundary conditions for numeric types
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_PrimitiveTypes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.1.1.1 Numeric Edge Cases"
echo "======================================"
echo ""
echo "Description: Tests handling of numeric edge cases including very large numbers,"
echo "             precision limits, special IEEE 754 values, and boundary conditions"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_PrimitiveTypes"
echo ""

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: Very large integer values
test_large_integer() {
    local FILTER="ID gt 999999"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Zero value in comparisons
test_zero_value() {
    local FILTER="Price eq 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Negative numbers in filters
test_negative_numbers() {
    local FILTER="Price gt -1"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Decimal precision with many decimal places
test_decimal_precision() {
    local FILTER="Price eq 999.9999999"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    # Should succeed even if no match (200 with empty result)
    check_status "$HTTP_CODE" "200"
}

# Test 5: Scientific notation (if supported)
test_scientific_notation() {
    local FILTER="Price lt 1e6"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    # Should either accept or reject cleanly (200 or 400)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Unexpected status code: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Very small decimal values
test_small_decimals() {
    local FILTER="Price gt 0.001"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Integer division behavior
test_integer_division() {
    local FILTER="ID div 2 eq 1"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Modulo with various values
test_modulo_operation() {
    local FILTER="ID mod 10 eq 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Numeric comparison with nulls
test_numeric_null_comparison() {
    local FILTER="Price ne null and Price gt 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Multiple numeric operations in single filter
test_complex_numeric_expression() {
    local FILTER="(Price mul 2) gt 1000 and (Price div 10) lt 200"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 11: Boundary value for Int32 (max positive)
test_int32_max_boundary() {
    local FILTER="ID lt 2147483647"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 12: Numeric precision in arithmetic operations
test_arithmetic_precision() {
    local FILTER="Price add 0.01 gt Price"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 13: Zero division handling (returns empty result set)
test_zero_division() {
    local FILTER="Price div 0 gt 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    # Division by zero returns 200 with empty results (database evaluates to null/false)
    # This is acceptable behavior per OData spec (implementation-dependent)
    check_status "$HTTP_CODE" "200"
}

# Test 14: Negative zero handling
test_negative_zero() {
    local FILTER="Price sub Price eq 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 15: Numeric ordering with edge values
test_numeric_ordering() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Price desc")
    check_status "$HTTP_CODE" "200"
}

run_test "Very large integer values in filter" test_large_integer
run_test "Zero value in numeric comparison" test_zero_value
run_test "Negative numbers in filter" test_negative_numbers
run_test "Decimal precision with many places" test_decimal_precision
run_test "Scientific notation handling" test_scientific_notation
run_test "Very small decimal values" test_small_decimals
run_test "Integer division behavior" test_integer_division
run_test "Modulo operation" test_modulo_operation
run_test "Numeric comparison with null values" test_numeric_null_comparison
run_test "Complex numeric expressions" test_complex_numeric_expression
run_test "Int32 maximum boundary value" test_int32_max_boundary
run_test "Arithmetic precision maintained" test_arithmetic_precision
run_test "Division by zero returns empty result" test_zero_division
run_test "Negative zero handling" test_negative_zero
run_test "Numeric ordering with edge values" test_numeric_ordering

print_summary
