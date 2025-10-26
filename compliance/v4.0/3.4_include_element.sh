#!/bin/bash

# OData v4.0 Compliance Test: 3.4 Element edmx:Include
# Tests the edmx:Include element for including schemas from referenced documents
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752504

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4.0 Compliance Test"
echo "Section: 3.4 Element edmx:Include"
echo "======================================"
echo ""
echo "Description: Validates edmx:Include elements that include schemas from referenced documents"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752504"
echo ""

# Test: Validate edmx:Include element structure if present
test_include_element_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check 1: edmx:Include is optional - test passes if not present
    if ! echo "$RESPONSE" | grep -q '<edmx:Include'; then
        return 0
    fi
    
    # Includes exist - validate their structure
    
    # Check 2: Includes MUST have Namespace attribute
    local INCLUDES=$(echo "$RESPONSE" | grep '<edmx:Include')
    if ! echo "$INCLUDES" | grep -q 'Namespace='; then
        echo "  Details: edmx:Include elements must have Namespace attribute"
        return 1
    fi
    
    # Check 3: Namespace attributes must not be empty
    if echo "$RESPONSE" | grep -q 'Namespace=""'; then
        echo "  Details: Namespace attribute must not be empty"
        return 1
    fi
    
    # Check 4: If Alias is present, it must not be empty
    if echo "$RESPONSE" | grep -q '<edmx:Include.*Alias=""'; then
        echo "  Details: Alias attribute must not be empty if specified"
        return 1
    fi
    
    return 0
}

echo "  Request: GET \$metadata"
run_test "edmx:Include elements have correct structure if present" test_include_element_structure

print_summary
