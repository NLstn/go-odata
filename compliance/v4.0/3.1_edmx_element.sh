#!/bin/bash

# OData v4.0 Compliance Test: 3.1 Element edmx:Edmx
# Tests the root edmx:Edmx element of the CSDL XML document
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752501

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4.0 Compliance Test"
echo "Section: 3.1 Element edmx:Edmx"
echo "======================================"
echo ""
echo "Description: Validates the edmx:Edmx root element structure and attributes"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752501"
echo ""

# Test: Validate complete edmx:Edmx element structure
test_edmx_element_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check 1: edmx:Edmx root element is present
    if ! echo "$RESPONSE" | grep -q '<edmx:Edmx'; then
        echo "  Details: Missing edmx:Edmx root element"
        return 1
    fi
    
    # Check 2: Proper EDMX namespace declaration
    if ! echo "$RESPONSE" | grep -q 'xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"'; then
        echo "  Details: Missing or invalid edmx namespace declaration"
        echo "  Expected: xmlns:edmx=\"http://docs.oasis-open.org/odata/ns/edmx\""
        return 1
    fi
    
    # Check 3: Version attribute is present
    if ! echo "$RESPONSE" | grep -q '<edmx:Edmx.*Version='; then
        echo "  Details: edmx:Edmx element must have Version attribute"
        return 1
    fi
    
    # Check 4: Element is properly closed
    if ! echo "$RESPONSE" | grep -q '</edmx:Edmx>'; then
        echo "  Details: edmx:Edmx element must be properly closed with </edmx:Edmx>"
        return 1
    fi
    
    # Check 5: Contains exactly one edmx:DataServices element
    local COUNT=$(echo "$RESPONSE" | grep -o '<edmx:DataServices' | wc -l)
    if [ "$COUNT" -ne 1 ]; then
        echo "  Details: edmx:Edmx must contain exactly one edmx:DataServices element (found: $COUNT)"
        return 1
    fi
    
    return 0
}

echo "  Request: GET \$metadata"
run_test "edmx:Edmx element has correct structure and attributes" test_edmx_element_structure

print_summary
