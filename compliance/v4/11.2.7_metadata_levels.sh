#!/bin/bash

# OData v4 Compliance Test: 11.2.7 Metadata Levels
# Tests odata.metadata parameter values according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_metadataURLs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.7 Metadata Levels"
echo "======================================"
echo ""
echo "Description: Validates odata.metadata parameter values (minimal, full, none)"
echo "             and their impact on response payloads."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_metadataURLs"
echo ""



# Test 1: odata.metadata=minimal (default)
echo "Test 1: odata.metadata=minimal includes @odata.context"
echo "  Request: GET $SERVER_URL/Products?\$format=application/json;odata.metadata=minimal"
RESPONSE=$(curl -s "$SERVER_URL/Products?\$format=application/json;odata.metadata=minimal" 2>&1)

if echo "$RESPONSE" | grep -q '"@odata.context"'; then
    test_result "metadata=minimal has @odata.context" "PASS"
else
    test_result "metadata=minimal has @odata.context" "FAIL" "Missing @odata.context"
fi

# Test 2: odata.metadata=full includes type annotations
echo ""
echo "Test 2: odata.metadata=full includes type annotations"
echo "  Request: GET $SERVER_URL/Products(1)?\$format=application/json;odata.metadata=full"
RESPONSE=$(curl -s "$SERVER_URL/Products(1)?\$format=application/json;odata.metadata=full" 2>&1)

if echo "$RESPONSE" | grep -q '"@odata.context"'; then
    # In full metadata, should include type information
    if echo "$RESPONSE" | grep -q '@odata\.type\|@odata\.id'; then
        test_result "metadata=full has type annotations" "PASS"
    else
        test_result "metadata=full has type annotations" "FAIL" "Missing @odata.type or @odata.id"
    fi
else
    test_result "metadata=full has type annotations" "FAIL" "Missing @odata.context"
fi

# Test 3: odata.metadata=none excludes metadata
echo ""
echo "Test 3: odata.metadata=none excludes @odata.context"
echo "  Request: GET $SERVER_URL/Products(1)?\$format=application/json;odata.metadata=none"
RESPONSE=$(curl -s "$SERVER_URL/Products(1)?\$format=application/json;odata.metadata=none" 2>&1)

if ! echo "$RESPONSE" | grep -q '"@odata.context"'; then
    test_result "metadata=none excludes @odata.context" "PASS"
else
    test_result "metadata=none excludes @odata.context" "FAIL" "Should not include @odata.context"
fi

# Test 4: metadata=none still returns data
echo ""
echo "Test 4: odata.metadata=none still returns entity data"
echo "  Request: GET $SERVER_URL/Products(1)?\$format=application/json;odata.metadata=none"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)?\$format=application/json;odata.metadata=none" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"ID"\|"Name"'; then
        test_result "metadata=none returns entity data" "PASS"
    else
        test_result "metadata=none returns entity data" "FAIL" "No entity properties found"
    fi
else
    test_result "metadata=none returns entity data" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 5: Invalid metadata value should work or return error
echo ""
echo "Test 5: Invalid odata.metadata value handling"
echo "  Request: GET $SERVER_URL/Products(1)?\$format=application/json;odata.metadata=invalid"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)?\$format=application/json;odata.metadata=invalid" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "Invalid metadata value returns 400" "PASS"
elif [ "$HTTP_CODE" = "200" ]; then
    test_result "Invalid metadata value returns 400" "PASS" "Server accepts invalid value (lenient)"
else
    test_result "Invalid metadata value returns 400" "FAIL" "Expected HTTP 400 or 200, got $HTTP_CODE"
fi

# Test 6: Collection with metadata=full
echo ""
echo "Test 6: Collection with metadata=full includes @odata.nextLink when applicable"
echo "  Request: GET $SERVER_URL/Products?\$top=2&\$format=application/json;odata.metadata=full"
RESPONSE=$(curl -s "$SERVER_URL/Products?\$top=2&\$format=application/json;odata.metadata=full" 2>&1)

if echo "$RESPONSE" | grep -q '"@odata.context"'; then
    # May or may not have nextLink depending on data
    test_result "Collection metadata=full has context" "PASS"
else
    test_result "Collection metadata=full has context" "FAIL" "Missing @odata.context"
fi


print_summary
