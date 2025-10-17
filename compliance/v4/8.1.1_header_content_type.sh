#!/bin/bash

# OData v4 Compliance Test: 8.1.1 Header Content-Type
# Tests that Content-Type header is properly set according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.1.1 Header Content-Type"
echo "======================================"
echo ""
echo "Description: Validates that the service returns proper Content-Type headers"
echo "             with the correct media type and optional parameters."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentType"
echo ""

PASSED=0
FAILED=0
TOTAL=0

test_result() {
    local test_name="$1"
    local result="$2"
    local details="$3"
    
    TOTAL=$((TOTAL + 1))
    if [ "$result" = "PASS" ]; then
        PASSED=$((PASSED + 1))
        echo "✓ PASS: $test_name"
    else
        FAILED=$((FAILED + 1))
        echo "✗ FAIL: $test_name"
        if [ -n "$details" ]; then
            echo "  Details: $details"
        fi
    fi
}

# Test 1: Service Document should return application/json with odata.metadata=minimal
echo "Test 1: Service Document Content-Type"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    if echo "$CONTENT_TYPE" | grep -q "odata.metadata=minimal"; then
        test_result "Service Document returns application/json with odata.metadata=minimal" "PASS"
    else
        test_result "Service Document returns application/json with odata.metadata=minimal" "FAIL" "Missing odata.metadata parameter. Got: $CONTENT_TYPE"
    fi
else
    test_result "Service Document returns application/json" "FAIL" "Expected application/json, got: $CONTENT_TYPE"
fi
echo ""

# Test 2: Metadata Document should return application/xml
echo "Test 2: Metadata Document Content-Type (XML)"
echo "  Request: GET $SERVER_URL/\$metadata"
RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
    test_result "Metadata Document returns application/xml" "PASS"
else
    test_result "Metadata Document returns application/xml" "FAIL" "Expected application/xml, got: $CONTENT_TYPE"
fi
echo ""

# Test 3: Metadata Document with $format=json should return application/json
echo "Test 3: Metadata Document Content-Type (JSON)"
echo "  Request: GET $SERVER_URL/\$metadata?\$format=json"
RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata?\$format=json" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    test_result "Metadata Document with \$format=json returns application/json" "PASS"
else
    test_result "Metadata Document with \$format=json returns application/json" "FAIL" "Expected application/json, got: $CONTENT_TYPE"
fi
echo ""

# Test 4: Entity Collection should return application/json with odata.metadata
echo "Test 4: Entity Collection Content-Type"
echo "  Request: GET $SERVER_URL/Products"
RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    if echo "$CONTENT_TYPE" | grep -q "odata.metadata"; then
        test_result "Entity Collection returns application/json with odata.metadata" "PASS"
    else
        test_result "Entity Collection returns application/json with odata.metadata" "FAIL" "Missing odata.metadata parameter. Got: $CONTENT_TYPE"
    fi
else
    test_result "Entity Collection returns application/json" "FAIL" "Expected application/json, got: $CONTENT_TYPE"
fi
echo ""

# Test 5: Single Entity should return application/json with odata.metadata
echo "Test 5: Single Entity Content-Type"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    if echo "$CONTENT_TYPE" | grep -q "odata.metadata"; then
        test_result "Single Entity returns application/json with odata.metadata" "PASS"
    else
        test_result "Single Entity returns application/json with odata.metadata" "FAIL" "Missing odata.metadata parameter. Got: $CONTENT_TYPE"
    fi
else
    test_result "Single Entity returns application/json" "FAIL" "Expected application/json, got: $CONTENT_TYPE"
fi
echo ""

# Summary
echo "======================================"
echo "Summary: $PASSED/$TOTAL tests passed"
if [ $FAILED -gt 0 ]; then
    echo "Status: FAILING"
    exit 1
else
    echo "Status: PASSING"
    exit 0
fi
