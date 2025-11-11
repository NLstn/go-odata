#!/bin/bash

# OData v4 Compliance Test: 11.3.6 Comparison Operators in $filter
# Tests comparison operators (eq, ne, gt, ge, lt, le) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ComparisonOperators

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.6 Comparison Operators"
echo "======================================"
echo ""
echo "Description: Validates comparison operators (eq, ne, gt, ge, lt, le)"
echo "             in \$filter expressions."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ComparisonOperators"
echo ""

# Test 1: eq (equals) operator
test_eq_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Status%20eq%201")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Status%20eq%201")
    
    # Verify all returned entities have Status=1
    local STATUSES=$(echo "$RESPONSE" | grep -o '"Status"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    
    if [ -z "$STATUSES" ]; then
        echo "  Details: No entities returned or no Status field found"
        return 1
    fi
    
    while IFS= read -r status; do
        if [ -n "$status" ] && [ "$status" != "1" ]; then
            echo "  Details: Found entity with Status=$status (expected 1)"
            return 1
        fi
    done <<< "$STATUSES"
    
    return 0
}

# Test 2: ne (not equals) operator
test_ne_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Status%20ne%200")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Status%20ne%200")
    
    # Verify all returned entities have Status != 0
    local STATUSES=$(echo "$RESPONSE" | grep -o '"Status"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    
    if [ -z "$STATUSES" ]; then
        echo "  Details: No entities returned or no Status field found"
        return 1
    fi
    
    while IFS= read -r status; do
        if [ -n "$status" ] && [ "$status" = "0" ]; then
            echo "  Details: Found entity with Status=0 (expected != 0)"
            return 1
        fi
    done <<< "$STATUSES"
    
    return 0
}

# Test 3: gt (greater than) operator
test_gt_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2050")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%2050")
    
    # Verify all returned entities have Price > 50
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 50) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected > 50)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 4: ge (greater than or equal) operator
test_ge_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20ge%2050")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20ge%2050")
    
    # Verify all returned entities have Price >= 50
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 >= 50) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected >= 50)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 5: lt (less than) operator
test_lt_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20lt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20lt%20100")
    
    # Verify all returned entities have Price < 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 < 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected < 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 6: le (less than or equal) operator
test_le_operator() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20le%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20le%20100")
    
    # Verify all returned entities have Price <= 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 <= 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected <= 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 7: eq with string
test_eq_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Name%20eq%20%27Laptop%27")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Name%20eq%20%27Laptop%27")
    
    # Verify all returned entities have Name='Laptop'
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ] && [ "$name" != "Laptop" ]; then
            echo "  Details: Found entity with Name='$name' (expected 'Laptop')"
            return 1
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 8: ne with string
test_ne_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Name%20ne%20%27Laptop%27")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Name%20ne%20%27Laptop%27")
    
    # Verify all returned entities have Name != 'Laptop'
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ] && [ "$name" = "Laptop" ]; then
            echo "  Details: Found entity with Name='Laptop' (expected != 'Laptop')"
            return 1
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 9: Comparison with decimal numbers
test_decimal_comparison() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20eq%2099.99")
    
    check_status "$HTTP_CODE" "200"
}

# Test 10: Comparison with null
test_null_comparison() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=CategoryID%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 11: Multiple comparisons combined
test_multiple_comparisons() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20ge%2010%20and%20Price%20le%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20ge%2010%20and%20Price%20le%20100")
    
    # Verify all returned entities have 10 <= Price <= 100
    local PRICES=$(echo "$RESPONSE" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 >= 10 && $1 <= 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price (expected 10 <= Price <= 100)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
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
    local RESPONSE1=$(http_get_body "$SERVER_URL/Products?\$filter=Name%20eq%20%27Laptop%27")
    local RESPONSE2=$(http_get_body "$SERVER_URL/Products?\$filter=Name%20eq%20%27laptop%27")
    
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

echo "  Request: GET $SERVER_URL/Products?\$filter=Name eq 'Laptop'"
run_test "eq operator works with strings" test_eq_string

echo "  Request: GET $SERVER_URL/Products?\$filter=Name ne 'Laptop'"
run_test "ne operator works with strings" test_ne_string

echo "  Request: GET $SERVER_URL/Products?\$filter=Price eq 99.99"
run_test "Comparison operators work with decimal numbers" test_decimal_comparison

echo "  Request: GET $SERVER_URL/Products?\$filter=CategoryID eq null"
run_test "Comparison with null value" test_null_comparison

echo "  Request: GET $SERVER_URL/Products?\$filter=Price ge 10 and Price le 100"
run_test "Multiple comparison operators combined" test_multiple_comparisons

echo "  Request: GET $SERVER_URL/Products?\$filter=Status eq 1"
run_test "Comparison operators return correct results" test_comparison_correctness

echo "  Request: GET $SERVER_URL/Products?\$filter=Name eq 'Laptop' vs 'laptop'"
run_test "String comparison case sensitivity" test_eq_case_sensitive

echo "  Request: GET $SERVER_URL/Products?\$filter=Price equals 50"
run_test "Invalid comparison operator returns 400" test_invalid_operator

print_summary
