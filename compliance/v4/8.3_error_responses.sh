#!/bin/bash

# OData v4 Compliance Test: 8.3 Error Responses
# Tests error response format according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ErrorResponse

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
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
echo "Test 1: 404 error response contains error object"
echo "  Request: GET $SERVER_URL/Products(999999)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "404" ]; then
    if echo "$BODY" | grep -q '"error"'; then
        test_result "404 error response contains 'error' object" "PASS"
    else
        test_result "404 error response contains 'error' object" "FAIL" "No 'error' object in response"
    fi
else
    test_result "Non-existent entity returns 404" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Error object contains 'code' property
echo "Test 2: Error object contains 'code' property"
RESPONSE=$(curl -s "$SERVER_URL/Products(999999)" 2>&1)

if echo "$RESPONSE" | grep -q '"code"'; then
    test_result "Error object contains 'code' property" "PASS"
else
    test_result "Error object contains 'code' property" "FAIL" "No 'code' property in error"
fi
echo ""

# Test 3: Error object contains 'message' property
echo "Test 3: Error object contains 'message' property"
RESPONSE=$(curl -s "$SERVER_URL/Products(999999)" 2>&1)

if echo "$RESPONSE" | grep -q '"message"'; then
    test_result "Error object contains 'message' property" "PASS"
else
    test_result "Error object contains 'message' property" "FAIL" "No 'message' property in error"
fi
echo ""

# Test 4: Error response has correct Content-Type
echo "Test 4: Error response has application/json Content-Type"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    test_result "Error response has application/json Content-Type" "PASS"
else
    test_result "Error response has application/json Content-Type" "FAIL" "Content-Type: $CONTENT_TYPE"
fi
echo ""

# Test 5: Invalid filter syntax returns 400 with error
echo "Test 5: Invalid query syntax returns 400 Bad Request with error"
echo "  Request: GET $SERVER_URL/Products?\$filter=invalid syntax"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=invalid%20syntax" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "400" ]; then
    if echo "$BODY" | grep -q '"error"'; then
        test_result "Invalid query returns 400 with error object" "PASS"
    else
        test_result "Invalid query returns 400 with error object" "FAIL" "No error object"
    fi
else
    test_result "Invalid query returns 400" "FAIL" "Status code: $HTTP_CODE (may accept invalid syntax)"
fi
echo ""

# Test 6: Unsupported version returns 406 with error
echo "Test 6: Unsupported OData version returns 406 Not Acceptable"
echo "  Request: GET $SERVER_URL/Products with OData-MaxVersion: 3.0"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products" \
    -H "OData-MaxVersion: 3.0" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "406" ]; then
    if echo "$BODY" | grep -q '"error"'; then
        test_result "Unsupported version returns 406 with error" "PASS"
    else
        test_result "Unsupported version returns 406 with error" "FAIL" "No error object"
    fi
else
    test_result "Unsupported version returns 406" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 7: Error response includes OData-Version header
echo "Test 7: Error response includes OData-Version header"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(999999)" 2>&1)
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

if [ -n "$ODATA_VERSION" ]; then
    test_result "Error response includes OData-Version header" "PASS"
else
    test_result "Error response includes OData-Version header" "FAIL" "No OData-Version header"
fi
echo ""


print_summary
