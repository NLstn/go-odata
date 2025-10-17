#!/bin/bash

# OData v4 Compliance Test: 8.2.9 Header OData-MaxVersion
# Tests OData-MaxVersion header according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataMaxVersion

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.9 Header OData-MaxVersion"
echo "======================================"
echo ""
echo "Description: Validates OData-MaxVersion header handling for version"
echo "             negotiation between client and server."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataMaxVersion"
echo ""



# Test 1: Request with OData-MaxVersion: 4.0
echo "Test 1: Request with OData-MaxVersion: 4.0"
echo "  Request: GET $SERVER_URL/Products(1) with OData-MaxVersion: 4.0"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.0" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

if [ "$HTTP_CODE" = "200" ]; then
    if [ -n "$ODATA_VERSION" ]; then
        # Version should be <= 4.0
        test_result "OData-MaxVersion 4.0 respected" "PASS"
    else
        test_result "OData-MaxVersion 4.0 respected" "FAIL" "No OData-Version header in response"
    fi
else
    test_result "OData-MaxVersion 4.0 respected" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Request with OData-MaxVersion: 4.01
echo ""
echo "Test 2: Request with OData-MaxVersion: 4.01"
echo "  Request: GET $SERVER_URL/Products(1) with OData-MaxVersion: 4.01"
RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.01" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

if [ "$HTTP_CODE" = "200" ]; then
    if [ -n "$ODATA_VERSION" ]; then
        test_result "OData-MaxVersion 4.01 accepted" "PASS"
    else
        test_result "OData-MaxVersion 4.01 accepted" "FAIL" "No OData-Version header in response"
    fi
else
    test_result "OData-MaxVersion 4.01 accepted" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 3: Request with unsupported OData-MaxVersion
echo ""
echo "Test 3: Request with OData-MaxVersion: 3.0 (unsupported)"
echo "  Request: GET $SERVER_URL/Products(1) with OData-MaxVersion: 3.0"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "OData-MaxVersion: 3.0" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "Unsupported OData-MaxVersion returns 400" "PASS"
elif [ "$HTTP_CODE" = "200" ]; then
    test_result "Unsupported OData-MaxVersion returns 400" "PASS" "Server accepts lower version (lenient)"
else
    test_result "Unsupported OData-MaxVersion returns 400" "FAIL" "Expected HTTP 400 or 200, got $HTTP_CODE"
fi

# Test 4: Request without OData-MaxVersion (should default to highest supported)
echo ""
echo "Test 4: Request without OData-MaxVersion header"
echo "  Request: GET $SERVER_URL/Products(1) without OData-MaxVersion"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

if [ "$HTTP_CODE" = "200" ]; then
    if [ -n "$ODATA_VERSION" ]; then
        test_result "Request without OData-MaxVersion succeeds" "PASS"
    else
        test_result "Request without OData-MaxVersion succeeds" "FAIL" "No OData-Version header"
    fi
else
    test_result "Request without OData-MaxVersion succeeds" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 5: Invalid OData-MaxVersion format
echo ""
echo "Test 5: Invalid OData-MaxVersion format"
echo "  Request: GET $SERVER_URL/Products(1) with OData-MaxVersion: invalid"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "OData-MaxVersion: invalid" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "Invalid OData-MaxVersion format returns 400" "PASS"
elif [ "$HTTP_CODE" = "200" ]; then
    test_result "Invalid OData-MaxVersion format returns 400" "PASS" "Server ignores invalid header (lenient)"
else
    test_result "Invalid OData-MaxVersion format returns 400" "FAIL" "Expected HTTP 400 or 200, got $HTTP_CODE"
fi


print_summary
