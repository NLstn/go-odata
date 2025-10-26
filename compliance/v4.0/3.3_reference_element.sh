#!/bin/bash

# OData v4.0 Compliance Test: 3.3 Element edmx:Reference
# Tests the edmx:Reference element for referencing external CSDL documents
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752503

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4.0 Compliance Test"
echo "Section: 3.3 Element edmx:Reference"
echo "======================================"
echo ""
echo "Description: Validates edmx:Reference elements that reference external CSDL documents"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752503"
echo ""

# Test: Validate edmx:Reference element structure if present
test_reference_element_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check 1: edmx:Reference is optional - test passes if not present
    if ! echo "$RESPONSE" | grep -q '<edmx:Reference'; then
        return 0
    fi
    
    # References exist - validate their structure
    
    # Check 2: References MUST have Uri attribute
    local REFERENCES=$(echo "$RESPONSE" | grep '<edmx:Reference')
    if ! echo "$REFERENCES" | grep -q 'Uri='; then
        echo "  Details: edmx:Reference elements must have Uri attribute"
        return 1
    fi
    
    # Check 3: Uri attributes must not be empty
    if echo "$RESPONSE" | grep -q 'Uri=""'; then
        echo "  Details: Uri attributes must not be empty"
        return 1
    fi
    
    # Check 4: References should contain Include or IncludeAnnotations (SHOULD requirement)
    # This is informational only, so we just log it
    if ! echo "$RESPONSE" | grep -qE '<edmx:(Include|IncludeAnnotations)'; then
        echo "  Info: edmx:Reference should contain edmx:Include or edmx:IncludeAnnotations"
    fi
    
    return 0
}

echo "  Request: GET \$metadata"
run_test "edmx:Reference elements have correct structure if present" test_reference_element_structure

print_summary
