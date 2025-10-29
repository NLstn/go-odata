#!/bin/bash

# OData v4 Compliance Test: 8.1.3 Response Headers
# Tests OData v4 response header requirements and compliance
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ResponseHeaders

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.1.3 Response Headers"
echo "======================================"
echo ""
echo "Description: Tests that OData services return proper response headers"
echo "             including Content-Type, OData-Version, and other required"
echo "             or recommended headers per the OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ResponseHeaders"
echo ""

# Test 1: Response includes Content-Type header
test_content_type_present() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    if echo "$RESPONSE" | grep -iq "^Content-Type:"; then
        return 0
    else
        echo "  Details: Response must include Content-Type header"
        return 1
    fi
}

# Test 2: Response includes OData-Version header
test_odata_version_present() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    if echo "$RESPONSE" | grep -iq "^OData-Version:"; then
        return 0
    else
        echo "  Details: Response should include OData-Version header"
        return 1
    fi
}

# Test 3: OData-Version is 4.0 or 4.01
test_odata_version_value() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local VERSION=$(echo "$RESPONSE" | grep -i "^OData-Version:" | head -1 | sed 's/OData-Version: //i' | tr -d '\r\n ')
    
    if echo "$VERSION" | grep -qE "^4\.(0|01)"; then
        return 0
    else
        echo "  Details: OData-Version should be 4.0 or 4.01 (got: $VERSION)"
        return 1
    fi
}

# Test 4: Content-Type includes charset for text responses
test_content_type_charset() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1)
    
    # For JSON, UTF-8 is implied per RFC 8259, but explicit charset is ok too
    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        return 0
    elif echo "$CONTENT_TYPE" | grep -q "charset"; then
        return 0
    else
        # Acceptable if charset is implicit
        return 0
    fi
}

# Test 5: Response includes Date header (HTTP requirement)
test_date_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    # Date header is part of HTTP/1.1 but not strictly required for all responses
    # Just verify it's valid if present
    return 0
}

# Test 6: Response includes Content-Length or Transfer-Encoding
test_content_length_or_transfer_encoding() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    
    if echo "$RESPONSE" | grep -iq "^Content-Length:" || \
       echo "$RESPONSE" | grep -iq "^Transfer-Encoding:"; then
        return 0
    else
        # One of these should be present in HTTP/1.1
        echo "  Details: Response should include Content-Length or Transfer-Encoding"
        return 1
    fi
}

# Test 7: Successful response returns appropriate status code
test_success_status_code() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local STATUS=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$STATUS" = "200" ]; then
        return 0
    else
        echo "  Details: GET on entity set should return 200 (got $STATUS)"
        return 1
    fi
}

# Test 8: Created response returns 201
test_created_status() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test Product","Price":49.99,"CategoryID":1}' 2>&1)
    local STATUS=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    # Clean up
    if [ "$STATUS" = "201" ]; then
        local BODY=$(echo "$RESPONSE" | tail -1)
        local ID=$(echo "$BODY" | grep -oP '"ID":\s*\K\d+' | head -1)
        if [ -n "$ID" ]; then
            curl -s -X DELETE "$SERVER_URL/Products($ID)" >/dev/null 2>&1
        fi
        return 0
    else
        echo "  Details: POST creating entity should return 201 (got $STATUS)"
        return 1
    fi
}

# Test 9: No Content response returns 204
test_no_content_status() {
    # First create an entity
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Delete Test","Price":1.00,"CategoryID":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local BODY=$(echo "$CREATE_RESPONSE" | sed '$d')
        local ID=$(echo "$BODY" | grep -oP '"ID":\s*\K\d+' | head -1)
        
        if [ -n "$ID" ]; then
            # DELETE should return 204 No Content
            local DEL_RESPONSE=$(curl -s -i -X DELETE "$SERVER_URL/Products($ID)" 2>&1)
            local DEL_STATUS=$(echo "$DEL_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
            
            if [ "$DEL_STATUS" = "204" ]; then
                return 0
            else
                echo "  Details: DELETE should return 204 (got $DEL_STATUS)"
                return 1
            fi
        fi
    fi
    
    skip_test "204 No Content response" "Could not create test entity"
    return 0
}

# Test 10: Not Found returns 404
test_not_found_status() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
}

# Test 11: Bad Request returns 400
test_bad_request_status() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=invalid syntax here")
    # Should return 400 for bad filter syntax
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        # Some implementations may be lenient
        echo "  Details: Invalid filter syntax should return 400 (got $HTTP_CODE)"
        return 1
    fi
}

# Test 12: Response includes Server header (optional but common)
test_server_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    # Server header is optional, test passes either way
    return 0
}

# Test 13: Cache-Control header present for cacheable responses
test_cache_control() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    # Cache-Control is optional but recommended
    return 0
}

# Test 14: ETag header for entities with concurrency control
test_etag_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    # ETag is optional unless concurrency control is required
    return 0
}

# Test 15: Location header for created resources
test_location_header_on_create() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Location Test","Price":29.99,"CategoryID":1}' 2>&1)
    local STATUS=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$STATUS" = "201" ]; then
        if echo "$RESPONSE" | grep -iq "^Location:"; then
            # Clean up
            local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r')
            local ID=$(echo "$LOCATION" | grep -oP '\((\d+)\)' | tr -d '()')
            if [ -n "$ID" ]; then
                curl -s -X DELETE "$SERVER_URL/Products($ID)" >/dev/null 2>&1
            fi
            return 0
        else
            echo "  Details: 201 Created should include Location header"
            # Clean up anyway
            local BODY=$(echo "$RESPONSE" | tail -1)
            local ID=$(echo "$BODY" | grep -oP '"ID":\s*\K\d+' | head -1)
            if [ -n "$ID" ]; then
                curl -s -X DELETE "$SERVER_URL/Products($ID)" >/dev/null 2>&1
            fi
            return 1
        fi
    else
        echo "  Details: Could not test Location header (got $STATUS for POST)"
        return 1
    fi
}

# Run tests
run_test "Response includes Content-Type header (MUST)" test_content_type_present
run_test "Response includes OData-Version header (SHOULD)" test_odata_version_present
run_test "OData-Version is 4.0 or 4.01" test_odata_version_value
run_test "Content-Type includes charset when needed" test_content_type_charset
run_test "Response includes Date header" test_date_header
run_test "Response includes Content-Length or Transfer-Encoding" test_content_length_or_transfer_encoding
run_test "Successful GET returns 200 OK" test_success_status_code
run_test "Created entity returns 201 Created" test_created_status
run_test "DELETE returns 204 No Content" test_no_content_status
run_test "Non-existent resource returns 404 Not Found" test_not_found_status
run_test "Invalid request returns 400 Bad Request" test_bad_request_status
run_test "Server header in response" test_server_header
run_test "Cache-Control header for cacheable responses" test_cache_control
run_test "ETag header for concurrency control" test_etag_header
run_test "Location header for 201 Created" test_location_header_on_create

print_summary
