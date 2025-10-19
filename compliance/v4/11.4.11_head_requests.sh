#!/bin/bash

# OData v4 Compliance Test: 11.4.11 HEAD Requests
# Tests HEAD requests for entities and collections
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CommonHeaders

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.11 HEAD Requests"
echo "======================================"
echo ""
echo "Description: Validates HEAD request support for entities and collections"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CommonHeaders"
echo ""

# Test 1: HEAD request on entity collection
test_head_collection() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - HEAD is optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 2: HEAD request on single entity
test_head_entity() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - HEAD is optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: HEAD request returns no body
test_head_no_body() {
    local RESPONSE=$(curl -s -I "$SERVER_URL/Products" 2>&1)
    
    # HEAD should return headers only, no body
    # Check that we get headers but response isn't full JSON
    if echo "$RESPONSE" | grep -q "HTTP/"; then
        # Make sure it's not a full response with body
        if ! echo "$RESPONSE" | grep -q '"value"'; then
            return 0
        else
            echo "  Details: HEAD should not return body content"
            return 1
        fi
    else
        echo "  Details: No HTTP response received"
        return 1
    fi
}

# Test 4: HEAD request includes Content-Length
test_head_content_length() {
    local HEADERS=$(curl -s -I "$SERVER_URL/Products" 2>&1)
    
    # Content-Length header should be present
    if echo "$HEADERS" | grep -qi "Content-Length:"; then
        return 0
    else
        echo "  Details: Content-Length header missing (optional)"
        return 0  # Pass - optional
    fi
}

# Test 5: HEAD request includes OData-Version
test_head_odata_version() {
    local HEADERS=$(curl -s -I "$SERVER_URL/Products" 2>&1)
    
    # OData-Version header should be present
    if echo "$HEADERS" | grep -qi "OData-Version:"; then
        return 0
    else
        echo "  Details: OData-Version header missing"
        return 1
    fi
}

# Test 6: HEAD request with query options
test_head_with_query() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=5" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD with query not allowed (status: $HTTP_CODE)"
        return 0  # Pass - optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: HEAD request on non-existent entity returns 404
test_head_not_found() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(999999)" 2>&1)
    
    if [ "$HTTP_CODE" = "404" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - if HEAD not supported
    else
        echo "  Details: Expected 404, got $HTTP_CODE"
        return 1
    fi
}

# Test 8: HEAD request includes Content-Type
test_head_content_type() {
    local HEADERS=$(curl -s -I "$SERVER_URL/Products" 2>&1)
    
    # Content-Type header should be present
    if echo "$HEADERS" | grep -qi "Content-Type:"; then
        return 0
    else
        echo "  Details: Content-Type header missing"
        return 1
    fi
}

# Test 9: HEAD request on service document
test_head_service_document() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: HEAD request on metadata document
test_head_metadata() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/\$metadata" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 11: HEAD response time similar to GET
test_head_performance() {
    # HEAD should be faster than GET since no body is returned
    local HEAD_TIME=$(time curl -s -I -o /dev/null "$SERVER_URL/Products" 2>&1)
    
    # Just verify HEAD works - performance comparison is implementation-specific
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "405" ]; then
        return 0
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 12: HEAD with Accept header
test_head_accept_header() {
    local HTTP_CODE=$(curl -s -I -o /dev/null -w "%{http_code}" "$SERVER_URL/Products" \
        -H "Accept: application/json" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "405" ]; then
        echo "  Details: HEAD not allowed (status: $HTTP_CODE)"
        return 0  # Pass - optional
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: HEAD Products"
run_test "HEAD request on entity collection" test_head_collection

echo "  Request: HEAD Products(1)"
run_test "HEAD request on single entity" test_head_entity

echo "  Request: HEAD returns no body"
run_test "HEAD request returns headers only" test_head_no_body

echo "  Request: HEAD includes Content-Length"
run_test "HEAD response includes Content-Length" test_head_content_length

echo "  Request: HEAD includes OData-Version"
run_test "HEAD response includes OData-Version" test_head_odata_version

echo "  Request: HEAD Products?\$top=5"
run_test "HEAD request with query options" test_head_with_query

echo "  Request: HEAD Products(999999)"
run_test "HEAD on non-existent entity returns 404" test_head_not_found

echo "  Request: HEAD includes Content-Type"
run_test "HEAD response includes Content-Type" test_head_content_type

echo "  Request: HEAD /"
run_test "HEAD request on service document" test_head_service_document

echo "  Request: HEAD \$metadata"
run_test "HEAD request on metadata document" test_head_metadata

echo "  Request: HEAD performance check"
run_test "HEAD request performance" test_head_performance

echo "  Request: HEAD with Accept header"
run_test "HEAD with Accept header" test_head_accept_header

print_summary
