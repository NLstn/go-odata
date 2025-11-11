#!/bin/bash

# OData v4 Compliance Test: 11.3.5 Logical Operators in $filter
# Tests logical operators (and, or, not) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LogicalOperators

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.5 Logical Operators"
echo "======================================"
echo ""
echo "Description: Validates logical operators (and, or, not) in \$filter"
echo "             expressions according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LogicalOperators"
echo ""

# Test 1: AND operator
test_and_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%20100")
    
    # Verify all returned entities have 10 < Price < 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 10 && $1 < 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected 10 < Price < 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 2: OR operator
test_or_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20lt%2010%20or%20Price%20gt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20lt%2010%20or%20Price%20gt%20100")
    
    # Verify all returned entities have Price < 10 OR Price > 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 < 10 || $1 > 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected Price < 10 or Price > 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 3: NOT operator
test_not_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050)")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050)")
    
    # Verify all returned entities have NOT (Price > 50), i.e., Price <= 50
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 <= 50) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected Price <= 50)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
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
    
    if ! check_json_field "$RESPONSE" "value"; then
        return 1
    fi
    
    # Verify all returned entities have 10 < Price < 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 10 && $1 < 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (not in range (10, 100))"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 11: OR result returns correct entities
test_or_result_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20lt%2010%20or%20Price%20gt%20100")
    
    if ! check_json_field "$RESPONSE" "value"; then
        return 1
    fi
    
    # Verify all returned entities have Price < 10 OR Price > 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 < 10 || $1 > 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected < 10 or > 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 12: NOT result returns correct entities
test_not_result_correctness() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=not%20(Price%20gt%2050)")
    
    if ! check_json_field "$RESPONSE" "value"; then
        return 1
    fi
    
    # Verify all returned entities have NOT (Price > 50), i.e., Price <= 50
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 <= 50) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected <= 50)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
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
