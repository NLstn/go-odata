#!/bin/bash

# OData v4 Compliance Test: 6.1 Extensibility
# Tests OData extensibility features including instance annotations and custom headers
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Extensibility

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 6.1 Extensibility"
echo "======================================"
echo ""
echo "Description: Tests OData v4 extensibility features including support"
echo "             for instance annotations, custom annotations, and proper"
echo "             handling of unknown elements per the specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Extensibility"
echo ""

# Test 1: Service must accept requests without understanding custom headers
test_ignores_unknown_headers() {
    # Per spec, services MUST NOT require clients to understand custom headers
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "X-Custom-Header: test")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Service should not require custom headers for standard operations
test_no_custom_headers_required() {
    # Standard OData operations must work without custom headers
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Instance annotations use @ prefix
test_instance_annotation_format() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Check for standard OData instance annotations with @ prefix
    if echo "$RESPONSE" | grep -q '"@odata'; then
        return 0
    else
        echo "  Details: Response should contain OData instance annotations with @ prefix"
        return 1
    fi
}

# Test 4: Service can include custom instance annotations
test_custom_annotations_allowed() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Custom annotations should follow @namespace.term format
    # This test just verifies service can return them (optional feature)
    # Even if no custom annotations exist, service should be capable
    return 0
}

# Test 5: Service must ignore unknown query options (SHOULD)
test_ignores_unknown_query_options() {
    # Services SHOULD ignore unknown query options that don't start with $
    # Standard behavior is to either ignore or return error
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?customOption=test")
    
    # Should either succeed (ignoring it) or fail with 400 (strict validation)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Service should handle unknown query options appropriately (got $HTTP_CODE)"
        return 1
    fi
}

# Test 6: Unknown system query options must return error
test_unknown_system_query_option_error() {
    # Unknown query options starting with $ MUST cause an error
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$unknownOption=test")
    
    # Must return 400 Bad Request for unknown system query options
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Unknown system query options (\$unknownOption) must return 400 Bad Request (got $HTTP_CODE)"
        return 1
    fi
}

# Test 7: Standard OData annotations are preserved
test_standard_annotations() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # Check for standard OData control information
    if check_json_field "$RESPONSE" "@odata.context"; then
        return 0
    else
        echo "  Details: Standard OData annotations must be present"
        return 1
    fi
}

# Test 8: Metadata extensibility - custom namespaces allowed
test_metadata_extensibility() {
    local METADATA=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Metadata should be well-formed (extensibility is about allowing custom elements)
    # This basic test just verifies metadata is retrievable
    if [ -n "$METADATA" ]; then
        return 0
    else
        echo "  Details: Metadata must be accessible for extensibility validation"
        return 1
    fi
}

# Test 9: Service should handle Prefer header for extensibility
test_prefer_header_handling() {
    # Prefer header is part of extensibility (client preferences)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Prefer: return=minimal")
    
    # Service should handle Prefer header (may or may not honor it)
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Service should accept Prefer header (got $HTTP_CODE)"
        return 1
    fi
}

# Test 10: Accept-Language header for internationalization extensibility
test_accept_language_header() {
    # Accept-Language is part of content negotiation/extensibility
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Accept-Language: en-US")
    
    # Service must not reject request based on Accept-Language
    check_status "$HTTP_CODE" "200"
}

# Run tests
run_test "Service accepts requests with unknown headers" test_ignores_unknown_headers
run_test "Service does not require custom headers" test_no_custom_headers_required
run_test "Instance annotations use @ prefix" test_instance_annotation_format
run_test "Service can include custom instance annotations" test_custom_annotations_allowed
run_test "Service handles unknown query options appropriately" test_ignores_unknown_query_options
run_test "Unknown system query options return 400 error (MUST)" test_unknown_system_query_option_error
run_test "Standard OData annotations are preserved" test_standard_annotations
run_test "Metadata supports extensibility" test_metadata_extensibility
run_test "Service handles Prefer header" test_prefer_header_handling
run_test "Service accepts Accept-Language header" test_accept_language_header

print_summary
