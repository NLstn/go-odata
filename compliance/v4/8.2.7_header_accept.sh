#!/bin/bash

# OData v4 Compliance Test: 8.2.7 Header Accept
# Tests Accept header content negotiation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderAccept

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.7 Header Accept"
echo "======================================"
echo ""
echo "Description: Validates Accept header content negotiation and handling"
echo "             of different media types according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderAccept"
echo ""



# Test 1: Accept: application/json should return JSON
test_accept_json() {
    local RESPONSE=$(curl -s -i -H "Accept: application/json" "$SERVER_URL/Products(1)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$CONTENT_TYPE" | grep -q "application/json"; then
            return 0
        else
            echo "  Details: Content-Type is $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Accept application/json" test_accept_json

# Test 2: Accept: */* should return JSON (default)
test_accept_wildcard() {
    local RESPONSE=$(curl -s -i -H "Accept: */*" "$SERVER_URL/Products(1)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$CONTENT_TYPE" | grep -q "application/json"; then
            return 0
        else
            echo "  Details: Content-Type is $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Accept */* returns JSON" test_accept_wildcard

# Test 3: Unsupported media type should return 406
test_unsupported_accept() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -H "Accept: application/xml" "$SERVER_URL/Products(1)" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)

    if [ "$HTTP_CODE" = "406" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        # Server might support XML, check Content-Type
        local FULL_RESPONSE=$(curl -s -i -H "Accept: application/xml" "$SERVER_URL/Products(1)" 2>&1)
        local CONTENT_TYPE=$(echo "$FULL_RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
        if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
            echo "  Details: Server supports XML (optional)"
            return 0
        else
            echo "  Details: Expected 406 or XML response, got JSON with HTTP 200"
            return 1
        fi
    else
        echo "  Details: Expected HTTP 406, got $HTTP_CODE"
        return 1
    fi
}

run_test "Unsupported Accept returns 406" test_unsupported_accept

# Test 4: Accept with parameters
test_accept_with_params() {
    local RESPONSE=$(curl -s -i -H "Accept: application/json;odata.metadata=minimal" "$SERVER_URL/Products(1)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$CONTENT_TYPE" | grep -q "application/json"; then
            return 0
        else
            echo "  Details: Content-Type is $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Accept with odata.metadata parameter" test_accept_with_params

# Test 5: Accept with quality values
test_accept_quality() {
    local RESPONSE=$(curl -s -i -H "Accept: text/plain;q=0.5, application/json;q=1.0" "$SERVER_URL/Products(1)" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$CONTENT_TYPE" | grep -q "application/json"; then
            return 0
        else
            echo "  Details: Expected JSON, got $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Accept quality values respected" test_accept_quality

# Test 6: Metadata document should support application/xml
test_metadata_accept_xml() {
    local RESPONSE=$(curl -s -i -H "Accept: application/xml" "$SERVER_URL/\$metadata" 2>&1)
    local CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
            return 0
        else
            echo "  Details: Content-Type is $CONTENT_TYPE"
            return 1
        fi
    else
        echo "  Details: HTTP $HTTP_CODE"
        return 1
    fi
}

run_test "Metadata accepts application/xml" test_metadata_accept_xml


print_summary
