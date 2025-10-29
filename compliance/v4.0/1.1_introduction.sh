#!/bin/bash

# OData v4 Compliance Test: 1.1 Introduction
# Tests basic service requirements from the OData v4 introduction section
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Introduction

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 1.1 Introduction"
echo "======================================"
echo ""
echo "Description: Tests basic service requirements defined in the OData v4"
echo "             introduction section, including service availability,"
echo "             protocol version support, and basic resource accessibility."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Introduction"
echo ""

# Test 1: Service root must be accessible
test_service_root_accessible() {
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Service document must be valid JSON
test_service_document_json() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    # Check if response is valid JSON by trying to parse it
    if echo "$RESPONSE" | python3 -m json.tool >/dev/null 2>&1; then
        return 0
    else
        echo "  Details: Service document is not valid JSON"
        return 1
    fi
}

# Test 3: Metadata document must be accessible
test_metadata_accessible() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Service document must contain @odata.context
test_service_context() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    check_json_field "$RESPONSE" "@odata.context"
}

# Test 5: Service must support at least one entity set
test_has_entity_sets() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    if check_json_field "$RESPONSE" "value"; then
        # Check if value array is not empty
        if echo "$RESPONSE" | grep -q '"value"\s*:\s*\['; then
            # Check if array has at least one element
            if ! echo "$RESPONSE" | grep -q '"value"\s*:\s*\[\s*\]'; then
                return 0
            fi
        fi
    fi
    echo "  Details: Service document must contain at least one entity set in value array"
    return 1
}

# Test 6: Service must be accessible via HTTP
test_http_access() {
    # This test verifies that basic HTTP GET works
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Service must be accessible via HTTP GET (got $HTTP_CODE)"
        return 1
    fi
}

# Run tests
run_test "Service root is accessible" test_service_root_accessible
run_test "Service document returns valid JSON" test_service_document_json
run_test "Metadata document is accessible" test_metadata_accessible
run_test "Service document contains @odata.context" test_service_context
run_test "Service exposes at least one entity set" test_has_entity_sets
run_test "Service is accessible via HTTP" test_http_access

print_summary
