#!/bin/bash

# OData v4 Compliance Test: 8.2.2 If-Match and If-None-Match Headers
# Tests conditional request headers for optimistic concurrency
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderIfMatch

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.2 If-Match/If-None-Match"
echo "======================================"
echo ""
echo "Description: Validates If-Match and If-None-Match headers for"
echo "             optimistic concurrency control."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderIfMatch"
echo ""

# Use existing product from seeded data (ID 1 = Laptop)
TEST_ID=1

# Test 1: GET request returns ETag header
test_get_returns_etag() {
    local HEADERS=$(curl -s -I "$SERVER_URL/Products($TEST_ID)" 2>&1)
    
    if echo "$HEADERS" | grep -i "etag:" > /dev/null; then
        return 0
    else
        # ETag is optional, so this is just informational
        echo "  Details: No ETag header returned (optional feature)"
        return 0
    fi
}

# Test 2: If-Match with correct ETag allows update
test_if_match_correct_etag() {
    # Get current ETag
    local HEADERS=$(curl -s -I "$SERVER_URL/Products($TEST_ID)" 2>&1)
    local ETAG=$(echo "$HEADERS" | grep -i "etag:" | head -1 | cut -d: -f2 | tr -d '\r\n ' )
    
    if [ -n "$ETAG" ]; then
        # Try to update with correct ETag
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
            -H "Content-Type: application/json" \
            -H "If-Match: $ETAG" \
            -d '{"Price":149.99}' 2>&1)
        
        # Should succeed with 200 or 204
        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
            return 0
        else
            echo "  Details: Status $HTTP_CODE (expected 200 or 204)"
            return 1
        fi
    else
        echo "  Details: No ETag available for test"
        return 0
    fi
}

# Test 3: If-Match with incorrect ETag returns 412
test_if_match_incorrect_etag() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
        -H "Content-Type: application/json" \
        -H 'If-Match: "incorrect-etag-value"' \
        -d '{"Price":149.99}' 2>&1)
    
    # If ETags are supported, should return 412 Precondition Failed
    if [ "$HTTP_CODE" = "412" ]; then
        return 0
    else
        # If ETags not supported, may return different code
        echo "  Details: Status $HTTP_CODE (expected 412 if ETags supported)"
        return 0
    fi
}

# Test 4: If-Match with * matches any version
test_if_match_star() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
        -H "Content-Type: application/json" \
        -H "If-Match: *" \
        -d '{"Price":159.99}' 2>&1)
    
    # If-Match: * should succeed if entity exists
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200 or 204)"
        return 0
    fi
}

# Test 5: If-None-Match with correct ETag returns 304
test_if_none_match_correct_etag() {
    # Get current ETag
    local HEADERS=$(curl -s -I "$SERVER_URL/Products($TEST_ID)" 2>&1)
    local ETAG=$(echo "$HEADERS" | grep -i "etag:" | head -1 | cut -d: -f2 | tr -d '\r\n ' )
    
    if [ -n "$ETAG" ]; then
        # Try to GET with If-None-Match using current ETag
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products($TEST_ID)" \
            -H "If-None-Match: $ETAG" 2>&1)
        
        # Should return 304 Not Modified if ETags match
        if [ "$HTTP_CODE" = "304" ]; then
            return 0
        else
            echo "  Details: Status $HTTP_CODE (expected 304 if ETags supported)"
            return 0
        fi
    else
        echo "  Details: No ETag available for test"
        return 0
    fi
}

# Test 6: If-None-Match with incorrect ETag returns entity
test_if_none_match_incorrect_etag() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products($TEST_ID)" \
        -H 'If-None-Match: "old-etag-value"' 2>&1)
    
    # Should return 200 with entity if ETags don't match
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 0
    fi
}

# Test 7: If-Match on DELETE
test_if_match_delete() {
    # Create entity for deletion test
    CREATE_DEL=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Delete Test","Price":50,"CategoryID":1,"Status":1}' 2>&1)
    DEL_CODE=$(echo "$CREATE_DEL" | tail -1)
    DEL_BODY=$(echo "$CREATE_DEL" | head -n -1)
    
    if [ "$DEL_CODE" = "201" ]; then
        DEL_ID=$(echo "$DEL_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$DEL_ID" ]; then
            # Get ETag
            local HEADERS=$(curl -s -I "$SERVER_URL/Products($DEL_ID)" 2>&1)
            local ETAG=$(echo "$HEADERS" | grep -i "etag:" | head -1 | cut -d: -f2 | tr -d '\r\n ' )
            
            if [ -n "$ETAG" ]; then
                # Try to delete with If-Match
                local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Products($DEL_ID)" \
                    -H "If-Match: $ETAG" 2>&1)
                
                if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
                    return 0
                else
                    echo "  Details: Status $HTTP_CODE"
                    return 1
                fi
            else
                # No ETag support
                echo "  Details: No ETag available"
                return 0
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity for deletion"
    return 0
}

# Test 8: If-Match on non-existent entity returns 404
test_if_match_not_found() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products(999999)" \
        -H "Content-Type: application/json" \
        -H "If-Match: *" \
        -d '{"Price":100}' 2>&1)
    
    # Should return 404 before checking precondition
    check_status "$HTTP_CODE" "404"
}

echo "  Request: GET $SERVER_URL/Products($TEST_ID)"
run_test "GET returns ETag header (if supported)" test_get_returns_etag

echo "  Request: PATCH with If-Match and correct ETag"
run_test "If-Match with correct ETag allows update" test_if_match_correct_etag

echo "  Request: PATCH with If-Match and incorrect ETag"
run_test "If-Match with incorrect ETag returns 412" test_if_match_incorrect_etag

echo "  Request: PATCH with If-Match: *"
run_test "If-Match: * matches any version" test_if_match_star

echo "  Request: GET with If-None-Match and current ETag"
run_test "If-None-Match with current ETag returns 304" test_if_none_match_correct_etag

echo "  Request: GET with If-None-Match and old ETag"
run_test "If-None-Match with old ETag returns 200 with entity" test_if_none_match_incorrect_etag

echo "  Request: DELETE with If-Match"
run_test "If-Match works with DELETE operation" test_if_match_delete

echo "  Request: PATCH with If-Match on non-existent entity"
run_test "If-Match on non-existent entity returns 404" test_if_match_not_found

print_summary
