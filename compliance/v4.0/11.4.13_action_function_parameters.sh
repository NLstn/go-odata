#!/bin/bash

# OData v4 Compliance Test: 11.4.13 Action and Function Parameter Validation
# Tests parameter validation for actions and functions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Operations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.13 Action/Function Parameters"
echo "======================================"
echo ""
echo "Description: Validates parameter validation for actions and functions,"
echo "             including required parameters, type validation, and error handling"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Operations"
echo ""

# Test 1: Unbound function with valid parameters
test_function_valid_params() {
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts?count=3")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Unbound function without required parameter should fail
test_function_missing_required_param() {
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts")
    
    # Should return 400 Bad Request for missing required parameter
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 400 or 404)"
        return 1
    fi
}

# Test 3: Bound function with valid parameters
test_bound_function_valid_params() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/GetTotalPrice?taxRate=0.08")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Bound function on non-existent entity should fail
test_bound_function_invalid_entity() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)/GetTotalPrice?taxRate=0.08")
    
    # Should return 404 Not Found
    check_status "$HTTP_CODE" "404"
}

# Test 5: Bound action with valid parameters
test_action_valid_params() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/ApplyDiscount" \
        -H "Content-Type: application/json" \
        -d '{"percentage": 10}')
    
    # Should return 200 or 204
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 204)"
        return 1
    fi
}

# Test 6: Action with missing required parameter should fail
test_action_missing_param() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/ApplyDiscount" \
        -H "Content-Type: application/json" \
        -d '{}')
    
    # Should return 400 Bad Request
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "500" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 400 or 500)"
        return 1
    fi
}

# Test 7: Action with invalid parameter type should fail  
test_action_invalid_param_type() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/ApplyDiscount" \
        -H "Content-Type: application/json" \
        -d '{"percentage": "invalid"}')
    
    # Should return 400 Bad Request
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "500" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 400 or 500)"
        return 1
    fi
}

# Test 8: Unbound action executes successfully
test_unbound_action() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/ResetAllPrices" \
        -H "Content-Type: application/json")
    
    # Should return 200 or 204
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 204)"
        return 1
    fi
}

# Test 9: Function returns proper response structure
test_function_response_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/GetTopProducts?count=3")
    
    # Should have @odata.context
    if echo "$RESPONSE" | grep -q '@odata.context'; then
        return 0
    else
        echo "  Details: Response missing '@odata.context'"
        return 1
    fi
}

# Test 10: Function with numeric parameter validation
test_function_numeric_param() {
    # Test with valid numeric parameter
    local HTTP_CODE=$(http_get "$SERVER_URL/GetTopProducts?count=5")
    check_status "$HTTP_CODE" "200"
}

# Test 11: Action response includes proper OData headers
test_action_odata_headers() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products(1)/IncreasePrice" \
        -H "Content-Type: application/json" \
        -d '{"amount": 5.0}')
    
    # Check for OData-Version header
    if echo "$RESPONSE" | grep -qi "OData-Version"; then
        return 0
    else
        echo "  Details: Response missing 'OData-Version' header"
        return 1
    fi
}

# Test 12: Function parameter with decimal value
test_function_decimal_param() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/GetTotalPrice?taxRate=0.075")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET /GetTopProducts?count=3"
run_test "Unbound function with valid parameters" test_function_valid_params

echo "  Request: GET /GetTopProducts (no params)"
run_test "Function without required parameter fails" test_function_missing_required_param

echo "  Request: GET /Products(1)/GetTotalPrice?taxRate=0.08"
run_test "Bound function with valid parameters" test_bound_function_valid_params

echo "  Request: GET /Products(999999)/GetTotalPrice?taxRate=0.08"
run_test "Bound function on invalid entity fails" test_bound_function_invalid_entity

echo "  Request: POST /Products(1)/ApplyDiscount {percentage: 10}"
run_test "Bound action with valid parameters" test_action_valid_params

echo "  Request: POST /Products(1)/ApplyDiscount {}"
run_test "Action without required parameter fails" test_action_missing_param

echo "  Request: POST /Products(1)/ApplyDiscount {percentage: 'invalid'}"
run_test "Action with invalid parameter type fails" test_action_invalid_param_type

echo "  Request: POST /ResetAllPrices"
run_test "Unbound action executes successfully" test_unbound_action

echo "  Request: GET /GetTopProducts?count=3 (check structure)"
run_test "Function returns proper response structure" test_function_response_structure

echo "  Request: GET /GetTopProducts?count=5"
run_test "Function with numeric parameter validation" test_function_numeric_param

echo "  Request: POST /Products(1)/IncreasePrice (check headers)"
run_test "Action response includes OData headers" test_action_odata_headers

echo "  Request: GET /Products(1)/GetTotalPrice?taxRate=0.075"
run_test "Function parameter with decimal value" test_function_decimal_param

print_summary
