#!/bin/bash

# OData v4.0 Compliance Test: 3.2 Element edmx:DataServices
# Tests the edmx:DataServices element of the CSDL XML document
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752502

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4.0 Compliance Test"
echo "Section: 3.2 Element edmx:DataServices"
echo "======================================"
echo ""
echo "Description: Validates the edmx:DataServices element contains one or more edm:Schema elements"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752502"
echo ""

# Test: Validate complete edmx:DataServices element structure
test_dataservices_element_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check 1: edmx:DataServices element is present
    if ! echo "$RESPONSE" | grep -q '<edmx:DataServices'; then
        echo "  Details: Missing edmx:DataServices element"
        return 1
    fi
    
    # Check 2: edmx:DataServices is properly closed
    if ! echo "$RESPONSE" | grep -q '</edmx:DataServices>'; then
        echo "  Details: edmx:DataServices must be properly closed"
        return 1
    fi
    
    # Check 3: MUST contain at least one Schema element
    if ! echo "$RESPONSE" | grep -q '<Schema'; then
        echo "  Details: edmx:DataServices must contain at least one Schema element"
        return 1
    fi
    
    # Check 4: Schema elements have proper EDM namespace
    if ! echo "$RESPONSE" | grep -q 'xmlns="http://docs.oasis-open.org/odata/ns/edm"'; then
        echo "  Details: Schema elements must use EDM namespace"
        echo "  Expected: xmlns=\"http://docs.oasis-open.org/odata/ns/edm\""
        return 1
    fi
    
    # Check 5: Schema elements have Namespace attribute
    if ! echo "$RESPONSE" | grep -q '<Schema.*Namespace='; then
        echo "  Details: Schema elements must have Namespace attribute"
        return 1
    fi
    
    # Check 6: Schemas contain entity model elements
    if ! echo "$RESPONSE" | grep -qE '<(EntityType|ComplexType|EntityContainer|EnumType|Action|Function)'; then
        echo "  Details: Schema should contain entity model elements"
        return 1
    fi
    
    return 0
}

echo "  Request: GET \$metadata"
run_test "edmx:DataServices has correct structure with Schema elements" test_dataservices_element_structure

print_summary
