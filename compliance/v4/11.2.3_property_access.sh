#!/bin/bash

# OData v4 Compliance Test: 11.2.3 Addressing Individual Properties
# Tests addressing and accessing individual properties according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_AddressingIndividualProperties

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.3 Addressing Individual Properties"
echo "======================================"
echo ""
echo "Description: Validates addressing and accessing individual properties"
echo "             of entities according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_AddressingIndividualProperties"
echo ""



# Test 1: Access a primitive property
echo "Test 1: Access a primitive property (Name)"
echo "  Request: GET $SERVER_URL/Products(1)/Name"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Name" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Should return a JSON object with "value" property
    if echo "$BODY" | grep -q '"value"'; then
        test_result "Access primitive property" "PASS"
    else
        test_result "Access primitive property" "FAIL" "Response does not contain 'value' property"
    fi
else
    test_result "Access primitive property" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Access a primitive property with \$value
echo ""
echo "Test 2: Access primitive property raw value with \$value"
echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Name/\$value" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Should return plain text without JSON wrapping
    if ! echo "$BODY" | grep -q '"value"' && [ -n "$BODY" ]; then
        test_result "Access property \$value" "PASS"
    else
        test_result "Access property \$value" "FAIL" "Expected plain value, got JSON"
    fi
else
    test_result "Access property \$value" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 3: Access non-existent property should return 404
echo ""
echo "Test 3: Access non-existent property returns 404"
echo "  Request: GET $SERVER_URL/Products(1)/NonExistentProperty"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/NonExistentProperty" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "404" ]; then
    test_result "Non-existent property returns 404" "PASS"
else
    test_result "Non-existent property returns 404" "FAIL" "Expected HTTP 404, got $HTTP_CODE"
fi

# Test 4: Access property of non-existent entity should return 404
echo ""
echo "Test 4: Access property of non-existent entity returns 404"
echo "  Request: GET $SERVER_URL/Products(999999)/Name"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)/Name" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "404" ]; then
    test_result "Property of non-existent entity returns 404" "PASS"
else
    test_result "Property of non-existent entity returns 404" "FAIL" "Expected HTTP 404, got $HTTP_CODE"
fi

# Test 5: Property access should have proper Content-Type
echo ""
echo "Test 5: Property access returns proper Content-Type"
echo "  Request: GET $SERVER_URL/Products(1)/Name"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)/Name" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    test_result "Property has application/json Content-Type" "PASS"
else
    test_result "Property has application/json Content-Type" "FAIL" "Got Content-Type: $CONTENT_TYPE"
fi

# Test 6: \$value should have text/plain Content-Type for strings
echo ""
echo "Test 6: \$value has text/plain Content-Type for string property"
echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)/Name/\$value" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "text/plain"; then
    test_result "\$value has text/plain Content-Type" "PASS"
else
    test_result "\$value has text/plain Content-Type" "FAIL" "Got Content-Type: $CONTENT_TYPE"
fi


print_summary
