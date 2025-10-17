#!/bin/bash

# OData v4 Compliance Test: 9.1 Service Document
# Tests that service document is properly formatted according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ServiceDocument

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 9.1 Service Document"
echo "======================================"
echo ""
echo "Description: Validates that the service document is properly formatted"
echo "             with correct context and entity set listings."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ServiceDocument"
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

# Test 1: Service document should be accessible at root
echo "Test 1: Service document accessible at /"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    test_result "Service document returns 200 OK" "PASS"
else
    test_result "Service document returns 200 OK" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Service document should have @odata.context
echo "Test 2: Service document contains @odata.context"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s "$SERVER_URL/" 2>&1)

if echo "$RESPONSE" | grep -q '"@odata.context"'; then
    CONTEXT=$(echo "$RESPONSE" | grep -o '"@odata.context"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
    if echo "$CONTEXT" | grep -q '/\$metadata'; then
        test_result "Service document contains @odata.context pointing to \$metadata" "PASS"
    else
        test_result "Service document contains @odata.context pointing to \$metadata" "FAIL" "Context: $CONTEXT"
    fi
else
    test_result "Service document contains @odata.context" "FAIL" "Property not found"
fi
echo ""

# Test 3: Service document should have value array
echo "Test 3: Service document contains value array"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s "$SERVER_URL/" 2>&1)

if echo "$RESPONSE" | grep -q '"value"'; then
    if echo "$RESPONSE" | grep -q '"value"[[:space:]]*:[[:space:]]*\['; then
        test_result "Service document contains value array" "PASS"
    else
        test_result "Service document contains value array" "FAIL" "value is not an array"
    fi
else
    test_result "Service document contains value array" "FAIL" "Property not found"
fi
echo ""

# Test 4: Service document entity sets should have required properties
echo "Test 4: Entity sets have required properties (name, kind, url)"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s "$SERVER_URL/" 2>&1)

# Check if entity sets have name property
if echo "$RESPONSE" | grep -q '"name"'; then
    NAME_CHECK="PASS"
else
    NAME_CHECK="FAIL"
fi

# Check if entity sets have url property
if echo "$RESPONSE" | grep -q '"url"'; then
    URL_CHECK="PASS"
else
    URL_CHECK="FAIL"
fi

# Check if entity sets have kind property
if echo "$RESPONSE" | grep -q '"kind"'; then
    KIND_CHECK="PASS"
else
    KIND_CHECK="FAIL"
fi

if [ "$NAME_CHECK" = "PASS" ] && [ "$URL_CHECK" = "PASS" ] && [ "$KIND_CHECK" = "PASS" ]; then
    test_result "Entity sets contain name, kind, and url properties" "PASS"
else
    test_result "Entity sets contain name, kind, and url properties" "FAIL" "Missing properties - name: $NAME_CHECK, url: $URL_CHECK, kind: $KIND_CHECK"
fi
echo ""

# Test 5: Entity set kind should be "EntitySet"
echo "Test 5: Entity sets have kind=\"EntitySet\""
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s "$SERVER_URL/" 2>&1)

if echo "$RESPONSE" | grep -q '"kind"[[:space:]]*:[[:space:]]*"EntitySet"'; then
    test_result "Entity sets have kind=\"EntitySet\"" "PASS"
else
    test_result "Entity sets have kind=\"EntitySet\"" "FAIL" "EntitySet kind not found"
fi
echo ""

# Test 6: Singleton should have kind="Singleton" (if any)
echo "Test 6: Singletons have kind=\"Singleton\" (if present)"
echo "  Request: GET $SERVER_URL/"
RESPONSE=$(curl -s "$SERVER_URL/" 2>&1)

# Check if there are any singletons
if echo "$RESPONSE" | grep -q '"kind"[[:space:]]*:[[:space:]]*"Singleton"'; then
    # Found singleton, check that it's properly formatted
    # Look for name property within the singleton object (before or after kind)
    SINGLETON_BLOCK=$(echo "$RESPONSE" | grep -A3 -B3 '"kind"[[:space:]]*:[[:space:]]*"Singleton"')
    if echo "$SINGLETON_BLOCK" | grep -q '"name"'; then
        test_result "Singletons have kind=\"Singleton\" with name property" "PASS"
    else
        test_result "Singletons have kind=\"Singleton\" with name property" "FAIL" "Singleton missing name"
    fi
elif echo "$RESPONSE" | grep -q '"Company"'; then
    # There should be a Company singleton based on the devserver
    test_result "Singletons have kind=\"Singleton\"" "FAIL" "Company singleton exists but kind is not set"
else
    # No singletons found, which is acceptable
    test_result "Singletons have kind=\"Singleton\" (if present)" "PASS" "No singletons to test"
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
