#!/bin/bash

# OData v4 Compliance Test: 8.2.6 Header OData-Version
# Tests that OData-Version header is properly set and version negotiation works
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataVersion

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.6 Header OData-Version"
echo "======================================"
echo ""
echo "Description: Validates that the service returns proper OData-Version headers"
echo "             and correctly handles OData-MaxVersion header negotiation."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataVersion"
echo ""



# Test 1: Service should return OData-Version: 4.0 header
test_odata_version_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

    if [ -n "$ODATA_VERSION" ]; then
        if [ "$ODATA_VERSION" = "4.0" ] || [ "$ODATA_VERSION" = "4.01" ]; then
            return 0
        else
            echo "  Details: Got version: $ODATA_VERSION"
            return 1
        fi
    else
        echo "  Details: Header not found"
        return 1
    fi
}

run_test "Service returns OData-Version header with value 4.0 or 4.01" test_odata_version_header

# Test 2: Service should accept request with OData-MaxVersion: 4.0
test_maxversion_40() {
    local STATUS_CODE=$(http_get "$SERVER_URL/" -H "OData-MaxVersion: 4.0")
    check_status "$STATUS_CODE" "200"
}

run_test "Service accepts OData-MaxVersion: 4.0" test_maxversion_40

# Test 3: Service should accept request with OData-MaxVersion: 4.01
test_maxversion_401() {
    local STATUS_CODE=$(http_get "$SERVER_URL/" -H "OData-MaxVersion: 4.01")
    check_status "$STATUS_CODE" "200"
}

run_test "Service accepts OData-MaxVersion: 4.01" test_maxversion_401

# Test 4: Service should reject request with OData-MaxVersion: 3.0
test_maxversion_30() {
    local STATUS_CODE=$(http_get "$SERVER_URL/" -H "OData-MaxVersion: 3.0")
    check_status "$STATUS_CODE" "406"
}

run_test "Service rejects OData-MaxVersion: 3.0 with 406 Not Acceptable" test_maxversion_30

# Test 5: OData-Version header should be present in all responses
test_entity_collection_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

    if [ -n "$ODATA_VERSION" ]; then
        return 0
    else
        echo "  Details: Header not found"
        return 1
    fi
}

run_test "Entity collection response includes OData-Version header" test_entity_collection_header

# Test 6: OData-Version header should be present in error responses
test_error_response_header() {
    local RESPONSE=$(curl -s -i -H "OData-MaxVersion: 3.0" "$SERVER_URL/" 2>&1)
    local ODATA_VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r' | xargs)

    if [ -n "$ODATA_VERSION" ]; then
        return 0
    else
        echo "  Details: No OData-Version header"
        return 1
    fi
}

run_test "Error response includes OData-Version header" test_error_response_header


print_summary
