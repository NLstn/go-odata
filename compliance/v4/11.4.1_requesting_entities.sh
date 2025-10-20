#!/bin/bash

# OData v4 Compliance Test: 11.4.1 Requesting Individual Entities
# Tests requesting individual entities with various methods
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingIndividualEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Test 1: GET individual entity by key
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    check_status "$HTTP_CODE" "200"
    local BODY=$(http_get_body "$SERVER_URL/Products(1)")
    check_contains "$BODY" '"ID"' "Entity has ID property"
    if echo "$BODY" | grep -q '"value"'; then
        return 1
    fi
}

# Test 2: HEAD request for individual entity
test_2() {
    local RESPONSE=$(curl -s -I "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    [ "$HTTP_CODE" = "200" ]
}

# Test 3: Request with If-None-Match returns 304 for matching ETag
test_3() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    local ETAG=$(echo "$RESPONSE" | grep -i "^ETag:" | head -1 | sed 's/ETag: //i' | tr -d '\r')
    
    if [ -n "$ETAG" ]; then
        local CONDITIONAL_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)" \
            -H "If-None-Match: $ETAG" 2>&1)
        local CONDITIONAL_CODE=$(echo "$CONDITIONAL_RESPONSE" | tail -1)
        [ "$CONDITIONAL_CODE" = "304" ]
    else
        return 0  # ETag support optional
    fi
}

# Test 4: Request non-existent entity returns 404
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
    local BODY=$(http_get_body "$SERVER_URL/Products(999999)")
    # OData error format optional but recommended
    return 0
}

# Test 5: Request entity with query options
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)?\$select=Name,Price")
    check_status "$HTTP_CODE" "200"
    local BODY=$(http_get_body "$SERVER_URL/Products(1)?\$select=Name,Price")
    check_contains "$BODY" '"Name"' "Response has Name"
    check_contains "$BODY" '"Price"' "Response has Price"
}

run_test "GET individual entity by key" test_1
run_test "HEAD request for individual entity" test_2
run_test "Conditional request with If-None-Match" test_3
run_test "Request non-existent entity returns 404" test_4
run_test "Request individual entity with \$select" test_5

print_summary
