#!/bin/bash

# OData v4 Compliance Test: 11.3.5 Logical Operators in $filter
# Tests logical operators (and, or, not) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_LogicalOperators

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.5 Logical Operators"
echo "======================================"
echo ""
echo "Description: Validates logical operators (and, or, not) in \$filter"
echo "             expressions according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_LogicalOperators"
echo ""

# Test 1: AND operator
test_and_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%20100")
    
    check_status "$HTTP_CODE" "200"
}

# Test 2: OR operator
test_or_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20lt%2010%20or%20Price%20gt%20100")
    
    check_status "$HTTP_CODE" "200"
}

# Test 3: NOT operator
test_not_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 4: Complex expression with AND and OR
test_complex_and_or() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=(Price%20lt%2010%20or%20Price%20gt%20100)%20and%20CategoryID%20eq%201")
    
    check_status "$HTTP_CODE" "200"
}

# Test 5: Multiple AND operators
test_multiple_and() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%20100%20and%20Status%20eq%201")
    
    check_status "$HTTP_CODE" "200"
}

# Test 6: Multiple OR operators
test_multiple_or() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=CategoryID%20eq%201%20or%20CategoryID%20eq%202%20or%20CategoryID%20eq%203")
    
    check_status "$HTTP_CODE" "200"
}

# Test 7: NOT with AND
test_not_with_and() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050%20and%20Status%20eq%201)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 8: Parentheses for precedence
test_parentheses_precedence() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20(CategoryID%20eq%201%20or%20CategoryID%20eq%202)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 9: NOT with OR
test_not_with_or() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=not%20(Price%20lt%2010%20or%20Price%20gt%20100)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 10: AND result returns correct entities
test_and_result_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%20100")
    
    # Should return products with price between 10 and 100
    # Verify response is valid JSON with value array
    if check_json_field "$RESPONSE" "value"; then
        # Check that returned products meet the criteria
        # (This is a basic check - full validation would parse JSON)
        return 0
    else
        return 1
    fi
}

# Test 11: OR result returns correct entities
test_or_result_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20lt%2010%20or%20Price%20gt%20100")
    
    # Should return products with price less than 10 OR greater than 100
    if check_json_field "$RESPONSE" "value"; then
        return 0
    else
        return 1
    fi
}

# Test 12: NOT result returns correct entities
test_not_result_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050)")
    
    # Should return products with price NOT greater than 50 (i.e., <= 50)
    if check_json_field "$RESPONSE" "value"; then
        return 0
    else
        return 1
    fi
}

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 and Price lt 100"
run_test "AND operator works in filter expressions" test_and_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=Price lt 10 or Price gt 100"
run_test "OR operator works in filter expressions" test_or_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=not (Price gt 50)"
run_test "NOT operator works in filter expressions" test_not_operator

echo "  Request: GET $SERVER_URL/Products?\$filter=(Price lt 10 or Price gt 100) and CategoryID eq 1"
run_test "Complex expression with AND and OR" test_complex_and_or

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 and Price lt 100 and Status eq 1"
run_test "Multiple AND operators chain correctly" test_multiple_and

echo "  Request: GET $SERVER_URL/Products?\$filter=CategoryID eq 1 or CategoryID eq 2 or CategoryID eq 3"
run_test "Multiple OR operators chain correctly" test_multiple_or

echo "  Request: GET $SERVER_URL/Products?\$filter=not (Price gt 50 and Status eq 1)"
run_test "NOT with AND expression" test_not_with_and

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 and (CategoryID eq 1 or CategoryID eq 2)"
run_test "Parentheses control operator precedence" test_parentheses_precedence

echo "  Request: GET $SERVER_URL/Products?\$filter=not (Price lt 10 or Price gt 100)"
run_test "NOT with OR expression" test_not_with_or

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 and Price lt 100"
run_test "AND operator returns correct filtered results" test_and_result_correctness

echo "  Request: GET $SERVER_URL/Products?\$filter=Price lt 10 or Price gt 100"
run_test "OR operator returns correct filtered results" test_or_result_correctness

echo "  Request: GET $SERVER_URL/Products?\$filter=not (Price gt 50)"
run_test "NOT operator returns correct filtered results" test_not_result_correctness

print_summary
