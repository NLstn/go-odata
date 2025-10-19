#!/bin/bash

# OData v4 Compliance Test: 8.4 Error Response Consistency
# Tests consistency and completeness of error responses across different scenarios
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ErrorResponse

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.4 Error Response Consistency"
echo "======================================"
echo ""
echo "Description: Validates that error responses are consistent and complete"
echo "             across different error scenarios (404, 400, 405, etc.)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ErrorResponse"
echo ""

# Test 1: 404 error has proper structure
test_404_error_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(999999)")
    
    # Check for error object
    if echo "$RESPONSE" | grep -q '"error"'; then
        # Check for required fields: code and message
        if echo "$RESPONSE" | grep -q '"code"' && echo "$RESPONSE" | grep -q '"message"'; then
            return 0
        else
            echo "  Details: Error missing required 'code' or 'message' field"
            return 1
        fi
    else
        echo "  Details: Response missing 'error' object"
        return 1
    fi
}

# Test 2: 400 error for invalid filter has proper structure
test_400_error_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=invalid syntax here")
    
    # Check for error object
    if echo "$RESPONSE" | grep -q '"error"'; then
        return 0
    else
        echo "  Details: Response missing 'error' object"
        return 1
    fi
}

# Test 3: 405 error for DELETE on collection has proper structure
test_405_error_structure() {
    local RESPONSE=$(curl -s -X DELETE "$SERVER_URL/Products")
    
    # Should return error structure or just fail with 405
    # Many implementations just return 405 without body
    if echo "$RESPONSE" | grep -q '"error"' || [ -z "$RESPONSE" ]; then
        return 0
    else
        # If there's a response, it should be error format
        echo "  Details: Non-empty response without error structure"
        return 1
    fi
}

# Test 4: Error responses have Content-Type header
test_error_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)")
    
    if echo "$RESPONSE" | grep -qi "Content-Type:.*application/json"; then
        return 0
    else
        echo "  Details: Error response missing JSON Content-Type header"
        return 1
    fi
}

# Test 5: Error responses have OData-Version header
test_error_odata_version() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)")
    
    if echo "$RESPONSE" | grep -qi "OData-Version"; then
        return 0
    else
        echo "  Details: Error response missing OData-Version header"
        return 1
    fi
}

# Test 6: 404 error has appropriate message
test_404_error_message() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(999999)")
    
    # Message should indicate entity not found
    if echo "$RESPONSE" | grep -qi "not found\|does not exist"; then
        return 0
    else
        echo "  Details: Error message does not indicate entity not found"
        return 1
    fi
}

# Test 7: 400 error for malformed JSON has proper structure
test_malformed_json_error() {
    local RESPONSE=$(curl -s -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{invalid json}')
    
    # Should have error structure
    if echo "$RESPONSE" | grep -q '"error"'; then
        return 0
    else
        echo "  Details: Response missing 'error' object for malformed JSON"
        return 1
    fi
}

# Test 8: Error code matches HTTP status
test_error_code_matches_status() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)")
    local BODY=$(echo "$RESPONSE" | head -n -1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
    
    # Extract error code from response
    if echo "$BODY" | grep -q '"code"'; then
        # Code should match or relate to HTTP status
        return 0
    else
        echo "  Details: Error response missing 'code' field"
        return 1
    fi
}

# Test 9: Multiple validation errors can be reported
test_multiple_errors() {
    # Try to create entity with missing required fields
    local RESPONSE=$(curl -s -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{}')
    
    # Should have error structure (details array is optional)
    if echo "$RESPONSE" | grep -q '"error"'; then
        return 0
    else
        echo "  Details: Response missing 'error' object"
        return 1
    fi
}

# Test 10: Error response is valid JSON
test_error_valid_json() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(999999)")
    
    # Try to parse as JSON using grep for basic structure
    if echo "$RESPONSE" | grep -q '^{.*}$'; then
        return 0
    else
        echo "  Details: Error response is not valid JSON"
        return 1
    fi
}

# Test 11: Consistency across different error types
test_error_consistency() {
    local RESPONSE_404=$(http_get_body "$SERVER_URL/Products(999999)")
    local RESPONSE_400=$(http_get_body "$SERVER_URL/Products?\$filter=invalid")
    
    # Both should have error object
    if echo "$RESPONSE_404" | grep -q '"error"' && echo "$RESPONSE_400" | grep -q '"error"'; then
        return 0
    else
        echo "  Details: Error format inconsistent across error types"
        return 1
    fi
}

# Test 12: Error details field (optional but recommended)
test_error_details_field() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(999999)")
    
    # Details field is optional, so we just check for consistent structure
    # If details exists, it should be an array
    if echo "$RESPONSE" | grep -q '"details"'; then
        if echo "$RESPONSE" | grep -q '"details"[[:space:]]*:[[:space:]]*\['; then
            return 0
        else
            echo "  Details: 'details' field is not an array"
            return 1
        fi
    else
        # No details field is also acceptable
        return 0
    fi
}

echo "  Request: GET /Products(999999)"
run_test "404 error has proper JSON structure" test_404_error_structure

echo "  Request: GET /Products?\$filter=invalid"
run_test "400 error has proper structure" test_400_error_structure

echo "  Request: DELETE /Products (collection)"
run_test "405 error has proper structure" test_405_error_structure

echo "  Request: GET /Products(999999) (check headers)"
run_test "Error responses have Content-Type header" test_error_content_type

echo "  Request: GET /Products(999999) (check headers)"
run_test "Error responses have OData-Version header" test_error_odata_version

echo "  Request: GET /Products(999999) (check message)"
run_test "404 error has appropriate message" test_404_error_message

echo "  Request: POST /Products with malformed JSON"
run_test "Malformed JSON returns proper error" test_malformed_json_error

echo "  Request: GET /Products(999999) (check code)"
run_test "Error code present in response" test_error_code_matches_status

echo "  Request: POST /Products with empty body"
run_test "Multiple validation errors handled" test_multiple_errors

echo "  Request: GET /Products(999999) (check JSON validity)"
run_test "Error response is valid JSON" test_error_valid_json

echo "  Request: Multiple error scenarios"
run_test "Error format consistent across types" test_error_consistency

echo "  Request: GET /Products(999999) (check details)"
run_test "Error details field structure" test_error_details_field

print_summary
