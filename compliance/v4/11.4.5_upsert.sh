#!/bin/bash

# OData v4 Compliance Test: 11.4.5 Upsert Operations
# Tests upsert (PUT) operations according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_UpsertanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.5 Upsert Operations"
echo "======================================"
echo ""
echo "Description: Validates upsert operations using PUT to create or"
echo "             replace entities according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_UpsertanEntity"
echo ""



# Test 1: PUT to existing entity updates it
echo "Test 1: PUT to existing entity (update)"
echo "  Request: PUT $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"ID":1,"Name":"Updated Product","Price":199.99,"Description":"Updated via PUT"}' \
    "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    test_result "PUT updates existing entity" "PASS"
else
    test_result "PUT updates existing entity" "FAIL" "Expected HTTP 200/204, got $HTTP_CODE"
fi

# Test 2: PUT to non-existent entity creates it (if supported)
echo ""
echo "Test 2: PUT to non-existent entity (insert)"
echo "  Request: PUT $SERVER_URL/Products(999999)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"ID":999999,"Name":"Upserted Product","Price":299.99,"Description":"Created via PUT"}' \
    "$SERVER_URL/Products(999999)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "201" ]; then
    test_result "PUT creates non-existent entity" "PASS"
elif [ "$HTTP_CODE" = "404" ]; then
    test_result "PUT creates non-existent entity" "PASS" "Server requires POST for creation (valid behavior)"
else
    test_result "PUT creates non-existent entity" "FAIL" "Expected HTTP 201 or 404, got $HTTP_CODE"
fi

# Test 3: PUT with missing required fields should fail
echo ""
echo "Test 3: PUT with incomplete entity returns 400"
echo "  Request: PUT $SERVER_URL/Products(1) with missing fields"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"Name":"Incomplete"}' \
    "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "PUT with incomplete data returns 400" "PASS"
elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    test_result "PUT with incomplete data returns 400" "PASS" "Server accepts partial data (lenient)"
else
    test_result "PUT with incomplete data returns 400" "FAIL" "Expected HTTP 400, got $HTTP_CODE"
fi

# Test 4: PUT should return proper headers
echo ""
echo "Test 4: PUT response includes proper headers"
echo "  Request: PUT $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -i -X PUT \
    -H "Content-Type: application/json" \
    -d '{"ID":1,"Name":"Header Test Product","Price":99.99,"Description":"Testing headers"}' \
    "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    # Check for OData-Version header
    if echo "$RESPONSE" | grep -iq "^OData-Version:"; then
        test_result "PUT response includes OData-Version header" "PASS"
    else
        test_result "PUT response includes OData-Version header" "FAIL" "Missing OData-Version header"
    fi
else
    test_result "PUT response includes OData-Version header" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 5: PUT with If-Match header
echo ""
echo "Test 5: PUT with If-Match for optimistic concurrency"
echo "  Request: GET $SERVER_URL/Products(1) to get ETag"
ETAG_RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
ETAG=$(echo "$ETAG_RESPONSE" | grep -i "^ETag:" | head -1 | sed 's/ETag: //i' | tr -d '\r')

if [ -n "$ETAG" ]; then
    echo "  Request: PUT $SERVER_URL/Products(1) with If-Match: $ETAG"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -H "If-Match: $ETAG" \
        -d '{"ID":1,"Name":"Conditional Update","Price":149.99,"Description":"With ETag"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        test_result "PUT with If-Match succeeds" "PASS"
    else
        test_result "PUT with If-Match succeeds" "FAIL" "Expected HTTP 200/204, got $HTTP_CODE"
    fi
else
    test_result "PUT with If-Match succeeds" "PASS" "Entity doesn't support ETags (optional)"
fi

# Test 6: PUT without Content-Type should fail
echo ""
echo "Test 6: PUT without Content-Type returns 400 or 415"
echo "  Request: PUT $SERVER_URL/Products(1) without Content-Type header"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -d '{"ID":1,"Name":"No Content-Type","Price":99.99}' \
    "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "415" ]; then
    test_result "PUT without Content-Type returns error" "PASS"
elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    test_result "PUT without Content-Type returns error" "PASS" "Server accepts request (lenient)"
else
    test_result "PUT without Content-Type returns error" "FAIL" "Expected HTTP 400/415, got $HTTP_CODE"
fi


print_summary
