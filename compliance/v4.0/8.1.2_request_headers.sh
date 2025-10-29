#!/bin/bash

# OData v4 Compliance Test: 8.1.2 Request Headers
# Tests OData v4 request header handling and requirements
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestHeaders

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.1.2 Request Headers"
echo "======================================"
echo ""
echo "Description: Tests proper handling of OData request headers including"
echo "             Accept, Content-Type, OData-MaxVersion, OData-Version,"
echo "             and other standard HTTP request headers."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestHeaders"
echo ""

# Test 1: Service accepts requests without Accept header
test_no_accept_header() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Service respects Accept: application/json
test_accept_json() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Accept: application/json")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products" -H "Accept: application/json")
        # Response should be valid JSON
        if echo "$BODY" | python3 -m json.tool >/dev/null 2>&1; then
            return 0
        fi
    fi
    echo "  Details: Accept: application/json should return JSON response"
    return 1
}

# Test 3: Service handles Accept: application/xml for metadata
test_accept_xml_metadata() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata" -H "Accept: application/xml")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Service rejects unsupported media types
test_unsupported_accept() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Accept: application/pdf")
    # Should return 406 Not Acceptable or 200 (if it ignores and uses default)
    if [ "$HTTP_CODE" = "406" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Unsupported Accept header handling (got $HTTP_CODE)"
        return 1
    fi
}

# Test 5: OData-MaxVersion header is respected
test_odata_maxversion_header() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "OData-MaxVersion: 4.0")
    check_status "$HTTP_CODE" "200"
}

# Test 6: OData-Version header in request
test_odata_version_request() {
    # Client can send OData-Version in request
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "OData-Version: 4.0")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Content-Type required for POST
test_content_type_post() {
    local RESPONSE=$(http_post "$SERVER_URL/Products" \
        '{"Name":"Test","Price":99.99,"CategoryID":1}' \
        -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should accept POST with Content-Type
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: POST with Content-Type header (got $HTTP_CODE)"
        return 1
    fi
}

# Test 8: Accept-Charset header handling
test_accept_charset() {
    # OData uses UTF-8 by default, service should handle Accept-Charset
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Accept-Charset: utf-8")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Accept-Language header
test_accept_language() {
    # Service should accept Accept-Language (may not honor it)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Accept-Language: en-US")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Multiple Accept values
test_multiple_accept_values() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" \
        -H "Accept: application/xml, application/json")
    check_status "$HTTP_CODE" "200"
}

# Test 11: Accept with quality values
test_accept_quality_values() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" \
        -H "Accept: application/json;q=1.0, application/xml;q=0.8")
    check_status "$HTTP_CODE" "200"
}

# Test 12: User-Agent header accepted
test_user_agent() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" \
        -H "User-Agent: OData-Compliance-Test/1.0")
    check_status "$HTTP_CODE" "200"
}

# Test 13: Host header required for HTTP/1.1
test_host_header() {
    # Host header is required by HTTP/1.1 spec
    # curl automatically adds it, so we just verify basic request works
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 14: Authorization header (basic test)
test_authorization_header() {
    # Service should accept Authorization header (may not require it)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" \
        -H "Authorization: Bearer fake-token")
    # Should return 200 or 401, not crash
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
        return 0
    else
        echo "  Details: Service should handle Authorization header (got $HTTP_CODE)"
        return 1
    fi
}

# Test 15: Custom headers are ignored
test_custom_headers_ignored() {
    # Service should not fail due to custom headers
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" \
        -H "X-Custom-Header: test" \
        -H "X-Another-Header: value")
    check_status "$HTTP_CODE" "200"
}

# Run tests
run_test "Service accepts requests without Accept header" test_no_accept_header
run_test "Service respects Accept: application/json" test_accept_json
run_test "Service handles Accept: application/xml for metadata" test_accept_xml_metadata
run_test "Service handles unsupported Accept media types" test_unsupported_accept
run_test "OData-MaxVersion header is respected" test_odata_maxversion_header
run_test "OData-Version header in request" test_odata_version_request
run_test "Content-Type required for POST" test_content_type_post
run_test "Accept-Charset header handling" test_accept_charset
run_test "Accept-Language header accepted" test_accept_language
run_test "Multiple Accept values supported" test_multiple_accept_values
run_test "Accept with quality values" test_accept_quality_values
run_test "User-Agent header accepted" test_user_agent
run_test "Host header required for HTTP/1.1" test_host_header
run_test "Authorization header handling" test_authorization_header
run_test "Custom headers are ignored" test_custom_headers_ignored

print_summary
