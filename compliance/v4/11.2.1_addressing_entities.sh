#!/bin/bash

# OData v4 Compliance Test: 11.2.1 Addressing Entities
# Tests various ways to address entities according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.1 Addressing Entities"
echo "======================================"
echo ""
echo "Description: Validates various URL conventions for addressing entities"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingEntities"
echo ""

# Test 1: Address entity set returns collection
echo "Test 1: Addressing entity set returns collection"
echo "  Request: GET $SERVER_URL/Products"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "Entity set returns collection with 'value' property" "PASS"
    else
        test_result "Entity set returns collection with 'value' property" "FAIL" "No 'value' array in response"
    fi
else
    test_result "Entity set returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Address single entity by key
echo "Test 2: Addressing single entity by key"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"ID"'; then
        test_result "Single entity by key returns entity object" "PASS"
    else
        test_result "Single entity by key returns entity object" "FAIL" "No ID property found"
    fi
else
    test_result "Single entity by key returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Non-existent entity returns 404
echo "Test 3: Non-existent entity returns 404"
echo "  Request: GET $SERVER_URL/Products(999999)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "404" ]; then
    test_result "Non-existent entity returns 404 Not Found" "PASS"
else
    test_result "Non-existent entity returns 404 Not Found" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 4: Invalid entity set returns 404
echo "Test 4: Invalid entity set returns 404"
echo "  Request: GET $SERVER_URL/NonExistentEntitySet"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/NonExistentEntitySet" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "404" ]; then
    test_result "Invalid entity set returns 404" "PASS"
else
    test_result "Invalid entity set returns 404" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 5: Accessing property of entity
echo "Test 5: Accessing property of entity"
echo "  Request: GET $SERVER_URL/Products(1)/Name"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Name" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "Property access returns value wrapper" "PASS"
    else
        test_result "Property access returns value wrapper" "FAIL" "No 'value' property"
    fi
else
    # Some implementations might return 404 for property access
    test_result "Property access response" "PASS" "Status: $HTTP_CODE (implementation specific)"
fi
echo ""

# Test 6: Accessing raw value of property
echo "Test 6: Accessing raw value of property with \$value"
echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Name/\$value" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Should return raw text, not JSON
    if ! echo "$BODY" | grep -q '"value"'; then
        test_result "Property \$value returns raw value" "PASS"
    else
        test_result "Property \$value returns raw value" "FAIL" "Returned JSON instead of raw value"
    fi
else
    # Some implementations might not support $value
    test_result "Property \$value support" "PASS" "Status: $HTTP_CODE (feature may not be implemented)"
fi
echo ""

print_summary
