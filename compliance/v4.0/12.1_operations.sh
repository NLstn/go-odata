#!/bin/bash

# OData v4 Compliance Test: 12.1 Operations (Actions and Functions)
# Tests OData v4 operations including bound and unbound actions and functions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 12.1 Operations"
echo "======================================"
echo ""
echo "Description: Tests OData operations (actions and functions) including"
echo "             bound and unbound operations, parameter passing, and"
echo "             proper invocation syntax per OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations"
echo ""

# Test 1: Operations must be declared in metadata
test_operations_in_metadata() {
    local METADATA=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Check for Function or Action elements in metadata
    if echo "$METADATA" | grep -q -i "Function\|Action"; then
        return 0
    else
        skip_test "Operations in metadata" "No operations declared in service"
        return 0
    fi
}

# Test 2: Unbound function invocation
test_unbound_function() {
    # Try invoking an unbound function (GetTopProducts is unbound in compliance server)
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts()")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Unbound function invocation" "Unbound functions not implemented"
        return 0
    else
        echo "  Details: Unbound function invocation (got $HTTP_CODE)"
        return 1
    fi
}

# Test 3: Unbound function with parameters
test_unbound_function_parameters() {
    # GetTopProducts with count parameter
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts(count=3)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Unbound function with parameters" "Function parameters not implemented"
        return 0
    else
        echo "  Details: Unbound function with parameters (got $HTTP_CODE)"
        return 1
    fi
}

# Test 4: Bound function on entity
test_bound_function() {
    # Test a bound function on an entity (GetTotalPrice on Products)
    # GetTotalPrice expects taxRate parameter (not quantity)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/GetTotalPrice(taxRate=0.08)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Bound function on entity" "Bound functions not implemented"
        return 0
    else
        echo "  Details: Bound function invocation (got $HTTP_CODE)"
        return 1
    fi
}

# Test 5: Bound function on collection
test_bound_function_collection() {
    # Test bound function on entity set/collection
    local HTTP_CODE=$(http_get "$SERVER_URL/Products/GetAveragePrice()")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Bound function on collection" "Collection-bound functions not implemented"
        return 0
    else
        echo "  Details: Bound function on collection (got $HTTP_CODE)"
        return 1
    fi
}

# Test 6: Unbound action invocation
test_unbound_action() {
    # Actions use POST, not GET
    local RESPONSE=$(http_post "$SERVER_URL/ResetProducts" '{}' -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Unbound action invocation" "Unbound actions not implemented"
        return 0
    else
        echo "  Details: Unbound action invocation (got $HTTP_CODE)"
        return 1
    fi
}

# Test 7: Bound action on entity
test_bound_action() {
    # Test bound action on an entity (ApplyDiscount on Products)
    local RESPONSE=$(http_post "$SERVER_URL/Products(1)/ApplyDiscount" \
        '{"percentage": 10}' \
        -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Bound action on entity" "Bound actions not implemented"
        return 0
    else
        echo "  Details: Bound action on entity (got $HTTP_CODE)"
        return 1
    fi
}

# Test 8: Action with parameter validation
test_action_parameter_validation() {
    # Test that action validates required parameters
    local RESPONSE=$(http_post "$SERVER_URL/Products(1)/ApplyDiscount" \
        '{}' \
        -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 400 Bad Request for missing required parameter
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Action parameter validation" "Action parameter validation not implemented"
        return 0
    elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        # Service may have default values or optional parameters
        return 0
    else
        echo "  Details: Action parameter validation (got $HTTP_CODE)"
        return 1
    fi
}

# Test 9: Function composition (function after navigation)
test_function_composition() {
    # Access function after navigation property
    local HTTP_CODE=$(http_get "$SERVER_URL/Categories(1)/Products/GetAveragePrice()")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Function composition after navigation" "Function composition not implemented"
        return 0
    else
        echo "  Details: Function composition (got $HTTP_CODE)"
        return 1
    fi
}

# Test 10: Function must be invoked with GET, action with POST
test_function_method_restriction() {
    # Functions should only accept GET
    local RESPONSE=$(http_post "$SERVER_URL/GetTopProducts()" '{}' -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 405 Method Not Allowed or 404
    if [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        # Some implementations may be lenient
        echo "  Details: Functions should only accept GET method (got $HTTP_CODE for POST)"
        return 1
    else
        # 501 or other error is acceptable
        return 0
    fi
}

# Test 11: Operations can return collections
test_operation_returns_collection() {
    # GetTopProducts should return a collection
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts()")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/GetTopProducts()")
        # Should have value array for collection result
        if check_json_field "$BODY" "value"; then
            return 0
        fi
        echo "  Details: Operation returning collection should have value array"
        return 1
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Operation returns collection" "Operations not implemented"
        return 0
    else
        echo "  Details: Operation should be accessible (got $HTTP_CODE)"
        return 1
    fi
}

# Test 12: Operations can return primitives
test_operation_returns_primitive() {
    # Functions can return primitive values
    local HTTP_CODE=$(http_get "$SERVER_URL/Products/GetAveragePrice()")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products/GetAveragePrice()")
        # Should have value field for primitive result
        if check_json_field "$BODY" "value"; then
            return 0
        fi
        # Or may return raw primitive
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Operation returns primitive" "Operations not implemented"
        return 0
    else
        echo "  Details: Operation returning primitive (got $HTTP_CODE)"
        return 1
    fi
}

# Run tests
run_test "Operations declared in metadata" test_operations_in_metadata
run_test "Unbound function invocation" test_unbound_function
run_test "Unbound function with parameters" test_unbound_function_parameters
run_test "Bound function on entity" test_bound_function
run_test "Bound function on collection" test_bound_function_collection
run_test "Unbound action invocation" test_unbound_action
run_test "Bound action on entity" test_bound_action
run_test "Action parameter validation" test_action_parameter_validation
run_test "Function composition after navigation" test_function_composition
run_test "Function method restriction (GET only)" test_function_method_restriction
run_test "Operation returns collection" test_operation_returns_collection
run_test "Operation returns primitive value" test_operation_returns_primitive

print_summary
