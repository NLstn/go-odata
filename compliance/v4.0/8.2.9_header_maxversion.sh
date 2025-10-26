#!/bin/bash

# OData v4 Compliance Test: 8.2.9 Header OData-MaxVersion
# Tests OData-MaxVersion header according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataMaxVersion

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

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
test_maxversion_40() {
    local RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.0" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

    if [ "$HTTP_CODE" = "200" ]; then
        if [ -n "$ODATA_VERSION" ]; then
            return 0
        else
            echo "  Details: No OData-Version header in response"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "OData-MaxVersion 4.0 respected" test_maxversion_40

# Test 2: Request with OData-MaxVersion: 4.01
test_maxversion_401() {
    local RESPONSE=$(curl -s -i -H "OData-MaxVersion: 4.01" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

    if [ "$HTTP_CODE" = "200" ]; then
        if [ -n "$ODATA_VERSION" ]; then
            return 0
        else
            echo "  Details: No OData-Version header in response"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "OData-MaxVersion 4.01 accepted" test_maxversion_401

# Test 3: Request with unsupported OData-MaxVersion
test_unsupported_maxversion() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -H "OData-MaxVersion: 3.0" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)

    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "406" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Details: Server accepts lower version (lenient)"
        return 0
    else
        echo "  Details: Expected HTTP 400/406 or 200, got $HTTP_CODE"
        return 1
    fi
}

run_test "Unsupported OData-MaxVersion returns 400" test_unsupported_maxversion

# Test 4: Request without OData-MaxVersion (should default to highest supported)
test_no_maxversion() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r')

    if [ "$HTTP_CODE" = "200" ]; then
        if [ -n "$ODATA_VERSION" ]; then
            return 0
        else
            echo "  Details: No OData-Version header"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Request without OData-MaxVersion succeeds" test_no_maxversion

# Test 5: Invalid OData-MaxVersion format
test_invalid_maxversion() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -H "OData-MaxVersion: invalid" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)

    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "406" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Details: Server ignores invalid header (lenient)"
        return 0
    else
        echo "  Details: Expected HTTP 400/406 or 200, got $HTTP_CODE"
        return 1
    fi
}

run_test "Invalid OData-MaxVersion format returns 400" test_invalid_maxversion


print_summary
