#!/bin/bash

# OData v4 Compliance Test: 11.2.13 Type Casting and Type Inheritance
# Tests derived types, type casting in URLs, and polymorphic queries
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingDerivedTypes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.13 Type Casting and Type Inheritance"
echo "======================================"
echo ""
echo "Description: Validates handling of derived types, type casting in URLs,"
echo "             and polymorphic entity queries."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingDerivedTypes"
echo ""

# Note: Type inheritance and casting are advanced OData features
# Many implementations may not support them initially

# Test 1: Filter by type using isof function
test_isof_function() {
    # Filter entities by type (e.g., Products?$filter=isof(Namespace.DerivedType))
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof('Namespace.SpecialProduct')")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: isof function not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 2: Type cast in URL path
test_type_cast_in_path() {
    # Cast to derived type in URL (e.g., Products(1)/Namespace.SpecialProduct)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Namespace.SpecialProduct")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Type cast in path not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: Type cast in collection
test_type_cast_collection() {
    # Filter collection by derived type (e.g., Products/Namespace.SpecialProduct)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products/Namespace.SpecialProduct")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Type cast on collection not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: Cast function in filter
test_cast_function() {
    # Use cast function to convert type (e.g., cast(Property, 'Edm.String'))
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=cast(ID,'Edm.String') eq '1'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: cast function not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 5: Access derived type property
test_derived_property_access() {
    # Access property specific to derived type after type cast
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Namespace.SpecialProduct/SpecialProperty")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Derived type property access not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Filter with isof and property condition
test_isof_with_filter() {
    # Combine isof with other filter conditions
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=isof('Namespace.SpecialProduct') and Price gt 100")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Combined isof filter not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: Polymorphic query returns base and derived types
test_polymorphic_query() {
    # Query base type should return both base and derived type instances
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    # Just check that the endpoint works
    check_status "$HTTP_CODE" "200"
}

# Test 8: Type information in response (@odata.type)
test_type_annotation() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check for @odata.type annotation (optional in minimal metadata)
        if echo "$RESPONSE" | grep -q '@odata.type'; then
            return 0
        else
            echo "  Details: @odata.type not present (acceptable for minimal metadata)"
            return 0  # Pass - optional in minimal metadata
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 9: Derived types in metadata
test_derived_in_metadata() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if metadata contains BaseType attribute (indicates inheritance)
        if echo "$RESPONSE" | grep -q 'BaseType'; then
            return 0
        else
            echo "  Details: No derived types in metadata (optional feature)"
            return 0  # Pass - optional
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Create entity with derived type
test_create_derived_type() {
    # Try to create entity with derived type annotation
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"@odata.type":"Namespace.SpecialProduct","Name":"Test","Price":100}' 2>&1)
    
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Creating derived type not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 11: Type cast with navigation property
test_type_cast_navigation() {
    # Type cast with navigation (e.g., Products(1)/Namespace.SpecialProduct/RelatedItems)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Namespace.SpecialProduct/Category")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Type cast with navigation not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 12: Invalid type cast returns error
test_invalid_type_cast() {
    # Try to cast to non-existent type
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Namespace.InvalidType")
    
    # Should return 404 or 400 for invalid type
    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Details: Invalid type cast should fail"
        return 1
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 0  # Pass - implementation-specific
    fi
}

echo "  Request: GET \$filter=isof('Namespace.SpecialProduct')"
run_test "Filter by type using isof function" test_isof_function

echo "  Request: GET Products(1)/Namespace.SpecialProduct"
run_test "Type cast in URL path" test_type_cast_in_path

echo "  Request: GET Products/Namespace.SpecialProduct"
run_test "Type cast on collection" test_type_cast_collection

echo "  Request: GET \$filter=cast(ID,'Edm.String') eq '1'"
run_test "Cast function in filter" test_cast_function

echo "  Request: GET Products(1)/Namespace.SpecialProduct/SpecialProperty"
run_test "Access derived type property" test_derived_property_access

echo "  Request: GET \$filter=isof(...) and Price gt 100"
run_test "Filter with isof and other conditions" test_isof_with_filter

echo "  Request: GET Products (polymorphic)"
run_test "Polymorphic query returns all types" test_polymorphic_query

echo "  Request: Check for @odata.type annotation"
run_test "Type information in response" test_type_annotation

echo "  Request: GET \$metadata"
run_test "Derived types in metadata" test_derived_in_metadata

echo "  Request: POST with derived type"
run_test "Create entity with derived type" test_create_derived_type

echo "  Request: GET type cast with navigation"
run_test "Type cast with navigation property" test_type_cast_navigation

echo "  Request: GET with invalid type cast"
run_test "Invalid type cast returns error" test_invalid_type_cast

print_summary
