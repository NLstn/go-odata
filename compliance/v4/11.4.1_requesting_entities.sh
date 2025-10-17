#!/bin/bash

# OData v4 Compliance Test: 11.4.1 Requesting Individual Entities
# Tests requesting individual entities with various methods
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingIndividualEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.1 Requesting Individual Entities"
echo "======================================"
echo ""
echo "Description: Validates various methods to request individual entities"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingIndividualEntities"
echo ""



# Test 1: GET individual entity by key
echo "Test 1: GET individual entity by key"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"ID"'; then
        if ! echo "$BODY" | grep -q '"value"'; then
            test_result "GET individual entity returns entity object (not collection)" "PASS"
        else
            test_result "GET individual entity returns entity object" "FAIL" "Response contains 'value' array"
        fi
    else
        test_result "GET individual entity returns entity object" "FAIL" "No ID property"
    fi
else
    test_result "GET individual entity returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: HEAD request for individual entity
echo "Test 2: HEAD request for individual entity"
echo "  Request: HEAD $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -I "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    # HEAD should return headers but no body
    test_result "HEAD request returns 200 with headers only" "PASS"
else
    test_result "HEAD request returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Request with If-None-Match returns 304 for matching ETag
echo "Test 3: Conditional request with If-None-Match"
echo "  First, get entity to retrieve ETag..."
RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
ETAG=$(echo "$RESPONSE" | grep -i "^ETag:" | head -1 | sed 's/ETag: //i' | tr -d '\r')

if [ -n "$ETAG" ]; then
    echo "  Request: GET $SERVER_URL/Products(1) with If-None-Match: $ETAG"
    CONDITIONAL_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)" \
        -H "If-None-Match: $ETAG" 2>&1)
    CONDITIONAL_CODE=$(echo "$CONDITIONAL_RESPONSE" | tail -1)
    
    if [ "$CONDITIONAL_CODE" = "304" ]; then
        test_result "If-None-Match with matching ETag returns 304" "PASS"
    else
        test_result "If-None-Match with matching ETag returns 304" "FAIL" "Status: $CONDITIONAL_CODE (ETag support may not be implemented)"
    fi
else
    test_result "ETag support for conditional requests" "FAIL" "No ETag header in response (feature may not be implemented)"
fi
echo ""

# Test 4: Request non-existent entity returns 404
echo "Test 4: Request non-existent entity returns 404"
echo "  Request: GET $SERVER_URL/Products(999999)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(999999)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "404" ]; then
    # Should also return OData error format
    if echo "$BODY" | grep -q '"error"'; then
        test_result "Non-existent entity returns 404 with OData error" "PASS"
    else
        test_result "Non-existent entity returns 404" "PASS"
    fi
else
    test_result "Non-existent entity returns 404" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 5: Request entity with query options
echo "Test 5: Request individual entity with \$select"
echo "  Request: GET $SERVER_URL/Products(1)?\$select=Name,Price"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)?\$select=Name,Price" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Name"' && echo "$BODY" | grep -q '"Price"'; then
        test_result "Individual entity with \$select works" "PASS"
    else
        test_result "Individual entity with \$select works" "FAIL" "Selected properties not found"
    fi
else
    test_result "Individual entity with \$select" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""


print_summary
