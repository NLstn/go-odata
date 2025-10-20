#!/bin/bash

# OData v4 Compliance Test: 8.1.1 Header Content-Type
# Tests that Content-Type header is properly set according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.1.1 Header Content-Type"
echo "======================================"
echo ""
echo "Description: Validates that the service returns proper Content-Type headers"
echo "             with the correct media type and optional parameters."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentType"
echo ""



# Test 1: Service Document should return application/json with odata.metadata=minimal
test_service_doc_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        if echo "$CONTENT_TYPE" | grep -q "odata.metadata=minimal"; then
            return 0
        else
            echo "  Details: Missing odata.metadata parameter. Got: $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: Expected application/json, got: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Service Document returns application/json with odata.metadata=minimal" test_service_doc_content_type

# Test 2: Metadata Document should return application/xml
test_metadata_xml_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
        return 0
    else
        echo "  Details: Expected application/xml, got: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Metadata Document returns application/xml" test_metadata_xml_content_type

# Test 3: Metadata Document with $format=json should return application/json
test_metadata_json_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata?\$format=json" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        return 0
    else
        echo "  Details: Expected application/json, got: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Metadata Document with \$format=json returns application/json" test_metadata_json_content_type

# Test 4: Entity Collection should return application/json with odata.metadata
test_entity_collection_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        if echo "$CONTENT_TYPE" | grep -q "odata.metadata"; then
            return 0
        else
            echo "  Details: Missing odata.metadata parameter. Got: $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: Expected application/json, got: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Entity Collection returns application/json with odata.metadata" test_entity_collection_content_type

# Test 5: Single Entity should return application/json with odata.metadata
test_single_entity_content_type() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        if echo "$CONTENT_TYPE" | grep -q "odata.metadata"; then
            return 0
        else
            echo "  Details: Missing odata.metadata parameter. Got: $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: Expected application/json, got: $CONTENT_TYPE"
        return 1
    fi
}

run_test "Single Entity returns application/json with odata.metadata" test_single_entity_content_type


print_summary
