#!/bin/bash

# OData v4 Compliance Test: 11.4.2 Create an Entity (POST)
# Tests entity creation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CreateanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.2 Create an Entity (POST)"
echo "======================================"
echo ""
echo "Description: Validates entity creation via POST requests"
echo "             with proper status codes and headers."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CreateanEntity"
echo ""

# Test 1: POST should return 201 Created
echo "Test 1: POST entity returns 201 Created"
PAYLOAD='{"Name":"ComplianceTestProduct1","Price":99.99,"Category":"Test"}'
echo "  Request: POST $SERVER_URL/Products"
echo "  Body: $PAYLOAD"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "201" ]; then
    test_result "POST returns 201 Created" "PASS"
    CREATED_ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
else
    test_result "POST returns 201 Created" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Location header should be present in 201 response
echo "Test 2: Location header present in 201 response"
PAYLOAD='{"Name":"ComplianceTestProduct2","Price":199.99,"Category":"Test"}'
echo "  Request: POST $SERVER_URL/Products"

RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" 2>&1)

LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')

if [ -n "$LOCATION" ] && [ "$HTTP_CODE" = "201" ]; then
    test_result "201 response includes Location header" "PASS"
    CREATED_ID2=$(echo "$LOCATION" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
else
    test_result "201 response includes Location header" "FAIL" "Location header not found or status not 201"
fi
echo ""

# Test 3: Created entity should be returned in response body
echo "Test 3: Created entity returned in response body"
PAYLOAD='{"Name":"ComplianceTestProduct3","Price":299.99,"Category":"Test"}'
echo "  Request: POST $SERVER_URL/Products"

RESPONSE=$(curl -s -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" 2>&1)

if echo "$RESPONSE" | grep -q '"ID"'; then
    if echo "$RESPONSE" | grep -q '"Name"[[:space:]]*:[[:space:]]*"ComplianceTestProduct3"'; then
        test_result "Response body includes created entity" "PASS"
        CREATED_ID3=$(echo "$RESPONSE" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    else
        test_result "Response body includes created entity" "FAIL" "Entity data incomplete"
    fi
else
    test_result "Response body includes created entity" "FAIL" "No entity in response"
fi
echo ""

# Test 4: POST with Prefer: return=minimal should return 204 No Content
echo "Test 4: POST with Prefer: return=minimal returns 204"
PAYLOAD='{"Name":"ComplianceTestProduct4","Price":399.99,"Category":"Test"}'
echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"

RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -H "Prefer: return=minimal" \
    -d "$PAYLOAD" 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')
LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)

if [ "$HTTP_CODE" = "204" ]; then
    if [ -n "$LOCATION" ]; then
        test_result "POST with Prefer: return=minimal returns 204 with Location header" "PASS"
        CREATED_ID4=$(echo "$LOCATION" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
    else
        test_result "POST with Prefer: return=minimal returns 204 with Location header" "FAIL" "Location header missing"
    fi
else
    test_result "POST with Prefer: return=minimal returns 204" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 5: OData-EntityId header should be present in 204 response
echo "Test 5: OData-EntityId header in 204 response"
PAYLOAD='{"Name":"ComplianceTestProduct5","Price":499.99,"Category":"Test"}'
echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"

RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -H "Prefer: return=minimal" \
    -d "$PAYLOAD" 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')
ODATA_ENTITYID=$(echo "$RESPONSE" | grep -i "^OData-EntityId:" | head -1 | sed 's/OData-EntityId: //i' | tr -d '\r' | xargs)

if [ "$HTTP_CODE" = "204" ] && [ -n "$ODATA_ENTITYID" ]; then
    test_result "204 response includes OData-EntityId header" "PASS"
    CREATED_ID5=$(echo "$ODATA_ENTITYID" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
else
    test_result "204 response includes OData-EntityId header" "FAIL" "Header not found or status not 204"
fi
echo ""

# Clean up any remaining test entities
if [ -n "$CREATED_ID" ]; then
fi


print_summary
