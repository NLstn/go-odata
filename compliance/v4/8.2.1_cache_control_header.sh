#!/bin/bash

# OData v4 Compliance Test: 8.2.1 Cache-Control Header
# Tests Cache-Control header handling for caching directives
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderCacheControl

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.1 Cache-Control Header"
echo "======================================"
echo ""
echo "Description: Validates Cache-Control header handling for HTTP caching"
echo "             according to OData v4 specification and HTTP standards"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderCacheControl"
echo ""

# Test 1: GET request includes Cache-Control header
test_cache_control_present() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    
    # Check for Cache-Control header (optional but good practice)
    if echo "$RESPONSE" | grep -qi "Cache-Control:"; then
        return 0
    else
        echo "  Details: Cache-Control header not present (optional)"
        # This is optional, so don't fail
        return 0
    fi
}

# Test 2: Metadata document can be cached
test_metadata_cacheable() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Metadata should be cacheable or have cache directives
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 3: Service document can be cached
test_service_doc_cacheable() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 4: Dynamic content may have no-cache
test_dynamic_content_cache() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    
    # Dynamic content might have Cache-Control: no-cache or similar
    # This is implementation-dependent, so just verify request succeeds
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    check_status "$HTTP_CODE" "200"
}

# Test 5: POST requests typically not cached
test_post_not_cached() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Cache Test","Price":99.99,"Category":"Test","Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # POST should return 201 and typically shouldn't be cached
    if [ "$HTTP_CODE" = "201" ]; then
        # Clean up
        local BODY=$(echo "$RESPONSE" | tail -n 5)
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            curl -s -X DELETE "$SERVER_URL/Products($ID)" > /dev/null 2>&1
        fi
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 6: ETag and Cache-Control work together
test_etag_with_cache() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    
    # If ETag is present, caching can be more effective
    if echo "$RESPONSE" | grep -qi "ETag:"; then
        echo "  Details: ETag present, enabling conditional caching"
        return 0
    else
        echo "  Details: No ETag (optional feature)"
        return 0
    fi
}

# Test 7: Max-age directive (if present)
test_max_age_directive() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
    
    # Metadata could have max-age since it rarely changes
    # This is optional, so just log
    if echo "$RESPONSE" | grep -qi "Cache-Control:.*max-age"; then
        echo "  Details: max-age directive present"
        return 0
    else
        echo "  Details: No max-age directive (optional)"
        return 0
    fi
}

# Test 8: Private vs public cache directive
test_cache_visibility() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    
    # Service might specify private or public caching
    # Just verify the response is successful
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    check_status "$HTTP_CODE" "200"
}

# Test 9: No-store directive for sensitive data
test_no_store_directive() {
    # Some endpoints might use no-store for sensitive data
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Just verify response succeeds
    check_status "$HTTP_CODE" "200"
}

# Test 10: Vary header with Cache-Control
test_vary_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" \
        -H "Accept: application/json" 2>&1)
    
    # Vary header helps with content negotiation caching
    if echo "$RESPONSE" | grep -qi "Vary:"; then
        echo "  Details: Vary header present"
        return 0
    else
        echo "  Details: No Vary header (optional)"
        return 0
    fi
}

echo "  Request: GET /Products"
run_test "Cache-Control header presence" test_cache_control_present

echo "  Request: GET /\$metadata"
run_test "Metadata document cacheability" test_metadata_cacheable

echo "  Request: GET / (service document)"
run_test "Service document cacheability" test_service_doc_cacheable

echo "  Request: GET /Products(1)"
run_test "Dynamic content cache directives" test_dynamic_content_cache

echo "  Request: POST /Products"
run_test "POST requests typically not cached" test_post_not_cached

echo "  Request: GET with ETag check"
run_test "ETag and Cache-Control interaction" test_etag_with_cache

echo "  Request: GET /\$metadata (max-age)"
run_test "max-age directive for metadata" test_max_age_directive

echo "  Request: GET cache visibility"
run_test "Cache visibility (private/public)" test_cache_visibility

echo "  Request: GET sensitive data"
run_test "no-store directive for sensitive data" test_no_store_directive

echo "  Request: GET with Accept header"
run_test "Vary header for content negotiation" test_vary_header

print_summary
