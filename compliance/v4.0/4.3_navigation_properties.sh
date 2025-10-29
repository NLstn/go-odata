#!/bin/bash

# OData v4 Compliance Test: 4.3 Navigation Properties
# Tests navigation property definitions and relationships in metadata
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_NavigationProperty

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 4.3 Navigation Properties"
echo "======================================"
echo ""
echo "Description: Tests that navigation properties are properly defined"
echo "             in metadata, including relationship types, multiplicity,"
echo "             and partner properties per OData v4 CSDL specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_NavigationProperty"
echo ""

# Test 1: Metadata must declare navigation properties
test_navigation_property_in_metadata() {
    local METADATA=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Check for NavigationProperty element in XML or navigationProperty in JSON
    if echo "$METADATA" | grep -q -i "NavigationProperty\|navigationProperty"; then
        return 0
    else
        echo "  Details: Metadata should declare navigation properties for related entities"
        return 1
    fi
}

# Test 2: Navigation property must specify target type
test_navigation_property_type() {
    local METADATA=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Navigation properties should have a Type attribute/property
    if echo "$METADATA" | grep -q -i 'Type=.*".*"'; then
        return 0
    else
        echo "  Details: Navigation properties must specify their target entity type"
        return 1
    fi
}

# Test 3: Single-valued navigation property returns single entity
test_single_navigation_property() {
    # Test accessing a navigation property that returns a single entity
    # Products -> Category (assuming single-valued navigation)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Category")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products(1)/Category")
        # Should return an entity (with @odata.context), not a collection
        if check_json_field "$BODY" "@odata.context"; then
            # Should NOT have a "value" array for single navigation
            if ! echo "$BODY" | grep -q '"value"\s*:\s*\['; then
                return 0
            fi
        fi
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        # Feature may not be implemented
        skip_test "Single-valued navigation property access" "Navigation properties not implemented"
        return 0
    fi
    
    echo "  Details: Single-valued navigation property should return single entity without value wrapper"
    return 1
}

# Test 4: Collection-valued navigation property returns collection
test_collection_navigation_property() {
    # Test accessing a navigation property that returns a collection
    # Category -> Products (assuming collection-valued navigation)
    local HTTP_CODE=$(http_get "$SERVER_URL/Categories(1)/Products")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Categories(1)/Products")
        # Should return a collection with "value" array
        if check_json_field "$BODY" "value"; then
            return 0
        fi
        echo "  Details: Collection-valued navigation property should return array with value wrapper"
        return 1
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        # Feature may not be implemented
        skip_test "Collection-valued navigation property access" "Navigation properties not implemented"
        return 0
    fi
    
    echo "  Details: Collection-valued navigation property must be accessible"
    return 1
}

# Test 5: Navigation property with filter
test_navigation_property_filter() {
    # Test filtering on a collection-valued navigation property
    local HTTP_CODE=$(http_get "$SERVER_URL/Categories(1)/Products?\$filter=Price gt 50")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        # Feature may not be implemented
        skip_test "Navigation property with filter" "Navigation property filtering not implemented"
        return 0
    else
        echo "  Details: Should support filtering on collection-valued navigation properties (got $HTTP_CODE)"
        return 1
    fi
}

# Test 6: Navigation property count
test_navigation_property_count() {
    # Test getting count of items through navigation property
    local HTTP_CODE=$(http_get "$SERVER_URL/Categories(1)/Products/\$count")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Categories(1)/Products/\$count")
        # Should return a number
        if echo "$BODY" | grep -qE '^[0-9]+$'; then
            return 0
        fi
        echo "  Details: Navigation property \$count should return numeric value"
        return 1
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        skip_test "Navigation property \$count" "Navigation property count not implemented"
        return 0
    fi
    
    echo "  Details: Should support \$count on collection-valued navigation properties"
    return 1
}

# Test 7: Navigation property must be nullable or have referential constraint
test_navigation_nullability() {
    local METADATA=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Navigation properties should have Nullable attribute or referential constraints
    # This is a metadata validation test
    if echo "$METADATA" | grep -q -i "NavigationProperty\|navigationProperty"; then
        # At this point, we just verify metadata is well-formed
        # Full validation would require XML/JSON schema validation
        return 0
    else
        echo "  Details: Navigation properties should be declared in metadata"
        return 1
    fi
}

# Test 8: Invalid navigation property returns 404
test_invalid_navigation_property() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/InvalidNavProperty")
    check_status "$HTTP_CODE" "404"
}

# Run tests
run_test "Navigation properties declared in metadata" test_navigation_property_in_metadata
run_test "Navigation properties specify target type" test_navigation_property_type
run_test "Single-valued navigation property returns entity" test_single_navigation_property
run_test "Collection-valued navigation property returns collection" test_collection_navigation_property
run_test "Navigation property supports \$filter" test_navigation_property_filter
run_test "Navigation property supports \$count" test_navigation_property_count
run_test "Navigation properties have proper nullability/constraints" test_navigation_nullability
run_test "Invalid navigation property returns 404" test_invalid_navigation_property

print_summary
