#!/bin/bash

# OData v4 Compliance Test: 9.2 Metadata Document
# Tests metadata document structure and format
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_MetadataDocumentRequest

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Metadata document is accessible at $metadata
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Metadata Content-Type is application/xml
test_2() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    check_contains "$CONTENT_TYPE" "application/xml"
}

# Test 3: Metadata contains Edmx element
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "<edmx:Edmx"; then
        return 0
    elif echo "$RESPONSE" | grep -q "<Edmx"; then
        return 0
    fi
    return 1
}

# Test 4: Metadata contains DataServices element
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "<edmx:DataServices"; then
        return 0
    elif echo "$RESPONSE" | grep -q "<DataServices"; then
        return 0
    fi
    return 1
}

# Test 5: Metadata contains Schema element
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "<Schema"
}

# Test 6: Metadata contains EntityType definitions
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "<EntityType"
}

# Test 7: Metadata contains EntityContainer
test_7() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "<EntityContainer"
}

# Test 8: Metadata is valid XML
test_8() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    # Try to validate with xmllint if available
    if command -v xmllint &> /dev/null; then
        echo "$RESPONSE" | xmllint --noout - 2>&1
        return $?
    else
        # Basic check: look for matching tags
        if echo "$RESPONSE" | grep -q "</edmx:Edmx>"; then
            return 0
        elif echo "$RESPONSE" | grep -q "</Edmx>"; then
            return 0
        fi
        return 1
    fi
}

# Run all tests
run_test "Metadata document accessible at \$metadata" test_1
run_test "Metadata Content-Type is application/xml" test_2
run_test "Metadata contains Edmx root element" test_3
run_test "Metadata contains DataServices element" test_4
run_test "Metadata contains Schema element" test_5
run_test "Metadata contains EntityType definitions" test_6
run_test "Metadata contains EntityContainer" test_7
run_test "Metadata document is valid XML" test_8

print_summary
