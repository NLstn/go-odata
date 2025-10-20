#!/bin/bash

# OData v4 Compliance Test: 11.5.1 Conditional Requests (ETag)
# Tests conditional request handling with ETags according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ConditionalRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Global variable to store ETag
ETAG_HEADER=""

# Test 1: Entity with @odata.etag should include ETag header
test_1() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    local BODY=$(echo "$RESPONSE" | sed -n '/^$/,$p' | tail -n +2)
    ETAG_HEADER=$(echo "$RESPONSE" | grep -i "^ETag:" | head -1 | sed 's/ETag: //i' | tr -d '\r')
    
    if [ -n "$ETAG_HEADER" ]; then
        return 0
    elif echo "$BODY" | grep -q '"@odata.etag"'; then
        return 1  # Has @odata.etag but no ETag header
    else
        return 0  # ETags optional
    fi
}

# Test 2: If-None-Match with matching ETag should return 304
test_2() {
    if [ -z "$ETAG_HEADER" ]; then
        return 0  # Skip if no ETag support
    fi
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -H "If-None-Match: $ETAG_HEADER" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "304" ]
}

# Test 3: If-None-Match with non-matching ETag should return 200
test_3() {
    if [ -z "$ETAG_HEADER" ]; then
        return 0
    fi
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -H 'If-None-Match: "different-etag"' "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "200" ]
}

# Test 4: If-Match with matching ETag should succeed for PATCH
test_4() {
    if [ -z "$ETAG_HEADER" ]; then
        return 0
    fi
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H "If-Match: $ETAG_HEADER" \
        -d '{"Name":"Test update"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]
}

# Test 5: If-Match with non-matching ETag should return 412
test_5() {
    if [ -z "$ETAG_HEADER" ]; then
        return 0
    fi
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H 'If-Match: "wrong-etag"' \
        -d '{"Name":"Test update"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "412" ]
}

# Test 6: If-Match: * should always succeed
test_6() {
    if [ -z "$ETAG_HEADER" ]; then
        return 0
    fi
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH \
        -H "Content-Type: application/json" \
        -H "If-Match: *" \
        -d '{"Name":"Test update with wildcard"}' \
        "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]
}

run_test "Response includes ETag header for entity with @odata.etag" test_1
run_test "If-None-Match with matching ETag returns 304 Not Modified" test_2
run_test "If-None-Match with non-matching ETag returns 200" test_3
run_test "If-Match with matching ETag allows PATCH" test_4
run_test "If-Match with non-matching ETag returns 412 Precondition Failed" test_5
run_test "If-Match: * allows update regardless of ETag" test_6

print_summary
