#!/bin/bash

# OData v4 Compliance Test: 11.5.1 Conditional Requests (ETag)
# Tests conditional request handling with ETags according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ConditionalRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.5.1 Conditional Requests (ETag)"
echo "======================================"
echo ""
echo "Description: Validates ETag support and conditional request handling"
echo "             including If-Match and If-None-Match headers."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ConditionalRequests"
echo ""



# Test 1: Entity with @odata.etag should include ETag header
echo "Test 1: Response includes ETag header for entity with @odata.etag"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
BODY=$(echo "$RESPONSE" | sed -n '/^$/,$p' | tail -n +2)
ETAG_HEADER=$(echo "$RESPONSE" | grep -i "^ETag:" | head -1 | sed 's/ETag: //i' | tr -d '\r')

if [ -n "$ETAG_HEADER" ]; then
    test_result "Entity response includes ETag header" "PASS"
elif echo "$BODY" | grep -q '"@odata.etag"'; then
    test_result "Entity response includes ETag header" "FAIL" "Entity has @odata.etag but no ETag header"
else
    # If entity doesn't support ETags, that's okay
    test_result "Entity response includes ETag header" "PASS" "Entity doesn't support ETags (optional)"
fi

# Test 2: If-None-Match with matching ETag should return 304
if [ -n "$ETAG_HEADER" ]; then
    echo ""
    echo "Test 2: If-None-Match with matching ETag returns 304 Not Modified"
    echo "  Request: GET $SERVER_URL/Products(1) with If-None-Match: $ETAG_HEADER"
    RESPONSE=$(curl -s -w "\n%{http_code}" -H "If-None-Match: $ETAG_HEADER" "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "304" ]; then
        test_result "If-None-Match returns 304" "PASS"
    else
        test_result "If-None-Match returns 304" "FAIL" "Expected HTTP 304, got $HTTP_CODE"
    fi
    
    # Test 3: If-None-Match with non-matching ETag should return 200
    echo ""
    echo "Test 3: If-None-Match with non-matching ETag returns 200"
    echo "  Request: GET $SERVER_URL/Products(1) with If-None-Match: \"different-etag\""
    RESPONSE=$(curl -s -w "\n%{http_code}" -H 'If-None-Match: "different-etag"' "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        test_result "If-None-Match with different ETag returns 200" "PASS"
    else
        test_result "If-None-Match with different ETag returns 200" "FAIL" "Expected HTTP 200, got $HTTP_CODE"
    fi
    
    # Test 4: If-Match with matching ETag should succeed for PATCH
    echo ""
    echo "Test 4: If-Match with matching ETag allows PATCH"
    echo "  Request: PATCH $SERVER_URL/Products(1) with If-Match: $ETAG_HEADER"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H "If-Match: $ETAG_HEADER" \
        -d '{"Description":"Test update"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        test_result "If-Match with matching ETag allows PATCH" "PASS"
    else
        test_result "If-Match with matching ETag allows PATCH" "FAIL" "Expected HTTP 200/204, got $HTTP_CODE"
    fi
    
    # Test 5: If-Match with non-matching ETag should return 412
    echo ""
    echo "Test 5: If-Match with non-matching ETag returns 412 Precondition Failed"
    echo "  Request: PATCH $SERVER_URL/Products(1) with If-Match: \"wrong-etag\""
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H 'If-Match: "wrong-etag"' \
        -d '{"Description":"Test update"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "412" ]; then
        test_result "If-Match with wrong ETag returns 412" "PASS"
    else
        test_result "If-Match with wrong ETag returns 412" "FAIL" "Expected HTTP 412, got $HTTP_CODE"
    fi
    
    # Test 6: If-Match: * should always succeed
    echo ""
    echo "Test 6: If-Match: * allows update regardless of ETag"
    echo "  Request: PATCH $SERVER_URL/Products(1) with If-Match: *"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H "If-Match: *" \
        -d '{"Description":"Test update with wildcard"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        test_result "If-Match: * allows PATCH" "PASS"
    else
        test_result "If-Match: * allows PATCH" "FAIL" "Expected HTTP 200/204, got $HTTP_CODE"
    fi
else
    echo ""
    echo "Skipping conditional request tests - entity does not support ETags"
fi


print_summary
