#!/bin/bash

# OData v4 Compliance Test: 8.3 Error Responses
# Tests error response format according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ErrorResponse

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.3 Error Responses"
echo "======================================"
echo ""
echo "Description: Validates error response format and structure"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ErrorResponse"
echo ""



# Test 1: 404 error contains error object
test_404_error_object() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" = "404" ]; then
        if echo "$BODY" | grep -q '"error"'; then
            return 0
        else
            echo "  Details: No 'error' object in response"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

run_test "404 error response contains 'error' object" test_404_error_object

# Test 2: Error object contains 'code' property
test_error_code_property() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(999999)" 2>&1)
    check_json_field "$RESPONSE" "code"
}

run_test "Error object contains 'code' property" test_error_code_property

# Test 3: Error object contains 'message' property
test_error_message_property() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(999999)" 2>&1)
    check_json_field "$RESPONSE" "message"
}

run_test "Error object contains 'message' property" test_error_message_property

# Test 4: Error response has correct Content-Type
test_error_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        return 0
    else
        echo "  Details: Content-Type: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Error response has application/json Content-Type" test_error_content_type

# Test 5: Invalid filter syntax returns 400 with error
test_invalid_query_error() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=invalid%20syntax" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" = "400" ]; then
        if echo "$BODY" | grep -q '"error"'; then
            return 0
        else
            echo "  Details: No error object"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (may accept invalid syntax)"
        return 1
    fi
}

run_test "Invalid query returns 400 with error object" test_invalid_query_error

# Test 6: Unsupported version returns 406 with error
test_unsupported_version() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products" \
        -H "OData-MaxVersion: 3.0" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" = "406" ]; then
        if echo "$BODY" | grep -q '"error"'; then
            return 0
        else
            echo "  Details: No error object"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

run_test "Unsupported version returns 406 with error" test_unsupported_version

# Test 7: Error response includes OData-Version header
test_error_odata_version_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)" 2>&1)
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

    if [ -n "$ODATA_VERSION" ]; then
        return 0
    else
        echo "  Details: No OData-Version header"
        return 1
    fi
}

run_test "Error response includes OData-Version header" test_error_odata_version_header


print_summary
