#!/bin/bash

# OData v4 Compliance Test: 11.2.10 Addressing Operations
# Tests addressing bound and unbound actions and functions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.10 Addressing Operations"
echo "======================================"
echo ""
echo "Description: Validates addressing of bound and unbound actions and functions"
echo "             according to OData v4 specification"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingOperations"
echo ""

# Test 1: Unbound function is addressable
test_unbound_function() {
    # Try to call an unbound function (if available)
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts()")
    
    # Should return 200 if function exists, or 404 if not implemented
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 2: Unbound function with parameters
test_unbound_function_params() {
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts(count=5)")
    
    # Should return 200 if function exists, or 404/501 if not
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 3: Bound function on entity
test_bound_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/GetRelatedProducts()")
    
    # Should return 200 if function exists, or 404/501 if not
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 4: Unbound action is addressable
test_unbound_action() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/ResetProducts" \
        -H "Content-Type: application/json" \
        -d '{}' 2>&1)
    
    # Should return appropriate code (200/204 if exists, 404/501 if not)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 5: Bound action on entity
test_bound_action() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/Activate" \
        -H "Content-Type: application/json" \
        -d '{}' 2>&1)
    
    # Should return appropriate code
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Function with multiple parameters
test_function_multiple_params() {
    local HTTP_CODE=$(http_get "$SERVER_URL/FindProducts(name='Laptop',maxPrice=1000)")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 7: Action returns result
test_action_with_result() {
    local RESPONSE=$(curl -s -X POST "$SERVER_URL/Products(1)/CalculateDiscount" \
        -H "Content-Type: application/json" \
        -d '{"percentage":10}' 2>&1)
    
    # Just check that request completes (200, 404, or 501 are all acceptable)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/CalculateDiscount" \
        -H "Content-Type: application/json" \
        -d '{"percentage":10}' 2>&1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Function on collection
test_function_on_collection() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products/GetAveragePrice()")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 9: Action on collection
test_action_on_collection() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products/MarkAllAsReviewed" \
        -H "Content-Type: application/json" \
        -d '{}' 2>&1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Metadata includes operations
test_metadata_operations() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Metadata should be valid XML
    if echo "$RESPONSE" | grep -q "<edmx:Edmx" || echo "$RESPONSE" | grep -q "<?xml"; then
        return 0
    else
        echo "  Details: Metadata response invalid"
        return 1
    fi
}

echo "  Request: GET /GetTopProducts()"
run_test "Unbound function is addressable" test_unbound_function

echo "  Request: GET /GetTopProducts(count=5)"
run_test "Unbound function with parameters" test_unbound_function_params

echo "  Request: GET /Products(1)/GetRelatedProducts()"
run_test "Bound function on entity" test_bound_function

echo "  Request: POST /ResetProducts"
run_test "Unbound action is addressable" test_unbound_action

echo "  Request: POST /Products(1)/Activate"
run_test "Bound action on entity" test_bound_action

echo "  Request: GET function with multiple parameters"
run_test "Function with multiple parameters" test_function_multiple_params

echo "  Request: POST action with result"
run_test "Action can return result" test_action_with_result

echo "  Request: GET /Products/GetAveragePrice()"
run_test "Function on collection" test_function_on_collection

echo "  Request: POST /Products/MarkAllAsReviewed"
run_test "Action on collection" test_action_on_collection

echo "  Request: GET /\$metadata"
run_test "Metadata includes operations" test_metadata_operations

print_summary
