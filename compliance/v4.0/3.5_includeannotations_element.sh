#!/bin/bash

# OData v4.0 Compliance Test: 3.5 Element edmx:IncludeAnnotations
# Tests the edmx:IncludeAnnotations element for including annotations from referenced documents
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752505

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4.0 Compliance Test"
echo "Section: 3.5 Element edmx:IncludeAnnotations"
echo "======================================"
echo ""
echo "Description: Validates edmx:IncludeAnnotations elements for including annotations from references"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752505"
echo ""

# Test: Validate edmx:IncludeAnnotations element structure if present
test_includeannotations_element_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check 1: edmx:IncludeAnnotations is optional - test passes if not present
    if ! echo "$RESPONSE" | grep -q '<edmx:IncludeAnnotations'; then
        return 0
    fi
    
    # IncludeAnnotations exist - validate their structure
    
    # Check 2: MUST have TermNamespace attribute
    local INCLUDES=$(echo "$RESPONSE" | grep '<edmx:IncludeAnnotations')
    if ! echo "$INCLUDES" | grep -q 'TermNamespace='; then
        echo "  Details: edmx:IncludeAnnotations elements must have TermNamespace attribute"
        return 1
    fi
    
    # Check 3: TermNamespace attribute must not be empty
    if echo "$RESPONSE" | grep -q 'TermNamespace=""'; then
        echo "  Details: TermNamespace attribute must not be empty"
        return 1
    fi
    
    # Check 4: If Qualifier is present, it must not be empty
    if echo "$RESPONSE" | grep -q '<edmx:IncludeAnnotations.*Qualifier=""'; then
        echo "  Details: Qualifier attribute must not be empty if specified"
        return 1
    fi
    
    # Check 5: If TargetNamespace is present, it must not be empty
    if echo "$RESPONSE" | grep -q '<edmx:IncludeAnnotations.*TargetNamespace=""'; then
        echo "  Details: TargetNamespace attribute must not be empty if specified"
        return 1
    fi
    
    return 0
}

echo "  Request: GET \$metadata"
run_test "edmx:IncludeAnnotations elements have correct structure if present" test_includeannotations_element_structure

print_summary
