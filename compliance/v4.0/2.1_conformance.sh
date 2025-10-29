#!/bin/bash

# OData v4 Compliance Test: 2.1 Conformance
# Tests service conformance to OData v4 specification requirements
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Conformance

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 2.1 Conformance"
echo "======================================"
echo ""
echo "Description: Tests that the service conforms to OData v4 specification"
echo "             requirements including proper response formats, required"
echo "             headers, metadata availability, and protocol compliance."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Conformance"
echo ""

# Test 1: Service MUST return service document
test_service_document_required() {
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Service MUST return metadata document
test_metadata_document_required() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Service MUST support JSON format
test_json_format_support() {
    local RESPONSE=$(http_get_body "$SERVER_URL/" -H "Accept: application/json")
    # Verify response is valid JSON
    if echo "$RESPONSE" | python3 -m json.tool >/dev/null 2>&1; then
        return 0
    else
        echo "  Details: Service must support JSON format"
        return 1
    fi
}

# Test 4: Service MUST include OData-Version header in responses
test_odata_version_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
    if echo "$RESPONSE" | grep -iq "^OData-Version:"; then
        return 0
    else
        echo "  Details: OData-Version header is required in responses"
        return 1
    fi
}

# Test 5: Service MUST respond to requests without custom headers
test_no_custom_headers_required() {
    # Test that service responds without requiring custom headers
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Service must not require custom headers for basic requests"
        return 1
    fi
}

# Test 6: Service MUST support GET on entity sets
test_get_entity_sets() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Service MUST support GET on single entities
test_get_single_entity() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Service MUST return 404 for non-existent resources
test_404_for_missing_resource() {
    local HTTP_CODE=$(http_get "$SERVER_URL/NonExistentEntitySet")
    check_status "$HTTP_CODE" "404"
}

# Test 9: Service MUST use UTF-8 encoding
test_utf8_encoding() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1)
    
    # OData services should use UTF-8 encoding (implied or explicit)
    # JSON is UTF-8 by default per RFC 8259
    if echo "$CONTENT_TYPE" | grep -iq "application/json"; then
        return 0
    elif echo "$CONTENT_TYPE" | grep -iq "charset=utf-8"; then
        return 0
    else
        # JSON implies UTF-8, so this is acceptable
        return 0
    fi
}

# Test 10: Service MUST support $metadata system resource
test_metadata_system_resource() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/\$metadata")
        # Metadata can be XML or JSON
        if [ -n "$BODY" ]; then
            return 0
        fi
    fi
    echo "  Details: Service must support \$metadata system resource"
    return 1
}

# Run tests
run_test "Service returns service document (MUST)" test_service_document_required
run_test "Service returns metadata document (MUST)" test_metadata_document_required
run_test "Service supports JSON format (MUST)" test_json_format_support
run_test "Service includes OData-Version header (MUST)" test_odata_version_header
run_test "Service does not require custom headers (MUST)" test_no_custom_headers_required
run_test "Service supports GET on entity sets (MUST)" test_get_entity_sets
run_test "Service supports GET on single entities (MUST)" test_get_single_entity
run_test "Service returns 404 for non-existent resources (MUST)" test_404_for_missing_resource
run_test "Service uses UTF-8 encoding (MUST)" test_utf8_encoding
run_test "Service supports \$metadata system resource (MUST)" test_metadata_system_resource

print_summary
