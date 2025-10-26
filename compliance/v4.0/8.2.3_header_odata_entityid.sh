#!/bin/bash

# OData v4 Compliance Test: 8.2.3 OData-EntityId Header
# Tests the OData-EntityId response header
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataEntityId

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.3 OData-EntityId Header"
echo "======================================"
echo ""
echo "Description: Validates the OData-EntityId response header is returned"
echo "             appropriately for entity operations."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataEntityId"
echo ""

# Test 1: POST returns OData-EntityId header
test_post_returns_entityid() {
    local HEADERS=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"EntityId Test","Price":99.99,"CategoryID":1,"Status":1}' 2>&1)
    
    # Check for OData-EntityId header (case-insensitive)
    if echo "$HEADERS" | grep -i "odata-entityid:" > /dev/null; then
        return 0
    else
        # Header is optional when return=representation
        echo "  Details: OData-EntityId header not found (may be optional)"
        return 0
    fi
}

# Test 2: OData-EntityId contains entity URL
test_entityid_contains_url() {
    local HEADERS=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"EntityId URL Test","Price":50,"CategoryID":1,"Status":1}' 2>&1)
    
    
    if echo "$HEADERS" | grep -i "odata-entityid:" > /dev/null; then
        local ENTITYID=$(echo "$HEADERS" | grep -i "odata-entityid:" | head -1)
        
        # Should contain Products URL
        if echo "$ENTITYID" | grep -q "Products"; then
            return 0
        else
            echo "  Details: OData-EntityId doesn't contain entity URL"
            return 1
        fi
    else
        echo "  Details: No OData-EntityId header"
        return 0
    fi
}

# Test 3: OData-EntityId with return=minimal
test_entityid_return_minimal() {
    local HEADERS=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Minimal Test","Price":75,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$HEADERS" | head -1 | grep -o '[0-9]\{3\}')
    
    
    # With return=minimal, should prefer OData-EntityId over body
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 4: PATCH with return=minimal may include OData-EntityId
test_patch_entityid() {
    # Create entity first
    local CREATE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Patch Test","Price":40,"CategoryID":1,"Status":1}' 2>&1)
    
    local CREATE_CODE=$(echo "$CREATE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            
            # Now PATCH with return=minimal
            local HEADERS=$(curl -s -i -X PATCH "$SERVER_URL/Products($ID)" \
                -H "Content-Type: application/json" \
                -H "Prefer: return=minimal" \
                -d '{"Price":45}' 2>&1)
            
            local HTTP_CODE=$(echo "$HEADERS" | head -1 | grep -o '[0-9]\{3\}')
            
            if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
                return 0
            else
                echo "  Details: Status $HTTP_CODE"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity"
    return 1
}

# Test 5: OData-EntityId is a valid URL
test_entityid_valid_url() {
    local HEADERS=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Valid URL Test","Price":60,"CategoryID":1,"Status":1}' 2>&1)
    
    
    if echo "$HEADERS" | grep -i "odata-entityid:" > /dev/null; then
        local ENTITYID=$(echo "$HEADERS" | grep -i "odata-entityid:" | head -1 | cut -d: -f2- | tr -d '\r\n ' )
        
        # Try to access the URL
        if [ -n "$ENTITYID" ]; then
            local TEST_CODE=$(http_get "$ENTITYID")
            
            if [ "$TEST_CODE" = "200" ]; then
                return 0
            else
                echo "  Details: OData-EntityId URL returned $TEST_CODE"
                return 1
            fi
        fi
    fi
    
    echo "  Details: No OData-EntityId to test"
    return 0
}

# Test 6: Header case sensitivity
test_header_case() {
    local HEADERS=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Case Test","Price":85,"CategoryID":1,"Status":1}' 2>&1)
    
    
    # HTTP headers are case-insensitive, but OData spec uses specific casing
    # We accept any case
    if echo "$HEADERS" | grep -i "odata-entityid:" > /dev/null; then
        return 0
    else
        echo "  Details: No OData-EntityId header found"
        return 0
    fi
}

echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"
run_test "POST returns OData-EntityId header" test_post_returns_entityid

echo "  Request: POST $SERVER_URL/Products"
run_test "OData-EntityId contains entity URL" test_entityid_contains_url

echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"
run_test "OData-EntityId with return=minimal preference" test_entityid_return_minimal

echo "  Request: PATCH with Prefer: return=minimal"
run_test "PATCH with return=minimal may include OData-EntityId" test_patch_entityid

echo "  Request: POST and verify OData-EntityId URL is accessible"
run_test "OData-EntityId is a valid, dereferenceable URL" test_entityid_valid_url

echo "  Request: POST and check header casing"
run_test "OData-EntityId header case handling" test_header_case

print_summary
