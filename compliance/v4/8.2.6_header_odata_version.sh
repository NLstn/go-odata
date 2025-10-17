#!/bin/bash

# OData v4 Compliance Test: 8.2.6 Header OData-Version
# Tests that OData-Version header is properly set and version negotiation works
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataVersion

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.6 Header OData-Version"
echo "======================================"
echo ""
echo "Description: Validates that the service returns proper OData-Version headers"
echo "             and correctly handles OData-MaxVersion header negotiation."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataVersion"
echo ""



# Test 1: Service should return OData-Version: 4.0 header
echo "Test 1: OData-Version header present in response"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

if [ -n "$ODATA_VERSION" ]; then
    if [ "$ODATA_VERSION" = "4.0" ] || [ "$ODATA_VERSION" = "4.01" ]; then
        test_result "Service returns OData-Version header with value 4.0 or 4.01" "PASS"
    else
        test_result "Service returns OData-Version header with value 4.0 or 4.01" "FAIL" "Got version: $ODATA_VERSION"
    fi
else
    test_result "Service returns OData-Version header" "FAIL" "Header not found"
fi
echo ""

# Test 2: Service should accept request with OData-MaxVersion: 4.0
echo "Test 2: Accept OData-MaxVersion: 4.0"
echo "  Request: GET $SERVER_URL/ with OData-MaxVersion: 4.0"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.0" "$SERVER_URL/" 2>&1)
STATUS_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')

if [ "$STATUS_CODE" = "200" ]; then
    test_result "Service accepts OData-MaxVersion: 4.0" "PASS"
else
    test_result "Service accepts OData-MaxVersion: 4.0" "FAIL" "Status code: $STATUS_CODE"
fi
echo ""

# Test 3: Service should accept request with OData-MaxVersion: 4.01
echo "Test 3: Accept OData-MaxVersion: 4.01"
echo "  Request: GET $SERVER_URL/ with OData-MaxVersion: 4.01"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.01" "$SERVER_URL/" 2>&1)
STATUS_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')

if [ "$STATUS_CODE" = "200" ]; then
    test_result "Service accepts OData-MaxVersion: 4.01" "PASS"
else
    test_result "Service accepts OData-MaxVersion: 4.01" "FAIL" "Status code: $STATUS_CODE"
fi
echo ""

# Test 4: Service should reject request with OData-MaxVersion: 3.0
echo "Test 4: Reject OData-MaxVersion: 3.0"
echo "  Request: GET $SERVER_URL/ with OData-MaxVersion: 3.0"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 3.0" "$SERVER_URL/" 2>&1)
STATUS_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')

if [ "$STATUS_CODE" = "406" ]; then
    test_result "Service rejects OData-MaxVersion: 3.0 with 406 Not Acceptable" "PASS"
else
    test_result "Service rejects OData-MaxVersion: 3.0 with 406 Not Acceptable" "FAIL" "Status code: $STATUS_CODE"
fi
echo ""

# Test 5: OData-Version header should be present in all responses
echo "Test 5: OData-Version header present in entity collection response"
echo "  Request: GET $SERVER_URL/Products"
RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

if [ -n "$ODATA_VERSION" ]; then
    test_result "Entity collection response includes OData-Version header" "PASS"
else
    test_result "Entity collection response includes OData-Version header" "FAIL" "Header not found"
fi
echo ""

# Test 6: OData-Version header should be present in error responses
echo "Test 6: OData-Version header present in error response"
echo "  Request: GET $SERVER_URL/NonExistentEntity with OData-MaxVersion: 3.0"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 3.0" "$SERVER_URL/" 2>&1)
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

if [ -n "$ODATA_VERSION" ]; then
    test_result "Error response includes OData-Version header" "PASS"
else
    test_result "Error response includes OData-Version header" "FAIL" "Header not found"
fi
echo ""


print_summary
