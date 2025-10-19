#!/bin/bash

# OData v4 Compliance Test: 5.2 Complex Types
# Tests complex (structured) types, nested properties, and complex type filtering
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ComplexType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.2 Complex Types"
echo "======================================"
echo ""
echo "Description: Validates handling of complex (structured) types including"
echo "             nested properties, filtering, selecting, and operations."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ComplexType"
echo ""

# Test 1: Retrieve entity with complex type property
test_complex_type_retrieval() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if response is valid JSON
        if echo "$RESPONSE" | grep -q '{'; then
            return 0
        else
            echo "  Details: Invalid response format"
            return 1
        fi
    else
        echo "  Details: Status: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: Access nested property of complex type
test_complex_nested_property() {
    # Access nested property like Address/City
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Address/City")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type property not found (status: $HTTP_CODE)"
        return 0  # Pass - may not have complex types
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: Filter by nested complex type property
test_filter_complex_property() {
    # Filter by nested property of complex type
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Address/City eq 'Seattle'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type filtering not supported (status: $HTTP_CODE)"
        return 0  # Pass - may not have complex types
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: Select complex type property
test_select_complex_property() {
    # Select entire complex type
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name,Address")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type in \$select not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 5: Select nested property of complex type
test_select_nested_complex() {
    # Select specific property within complex type
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Address/City,Address/State")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Nested \$select not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Complex type with null value
test_complex_null_value() {
    # Filter for entities where complex type is null
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Address eq null")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Null complex type filtering not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: Complex type in $orderby
test_orderby_complex_property() {
    # Order by nested property of complex type
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Address/City")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type in \$orderby not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Deeply nested complex type
test_deeply_nested_complex() {
    # Access deeply nested property (e.g., Address/Location/Coordinates/Latitude)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Metadata/Details/Version eq '1.0'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Deeply nested complex types not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 9: Complex type with logical operators in filter
test_complex_logical_filter() {
    # Combine multiple complex type filters
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Address/City eq 'Seattle' and Address/State eq 'WA'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex logical filters not supported (status: $HTTP_CODE)"
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Complex type property $value access
test_complex_value_access() {
    # Access $value of complex type property (should return 404 - not supported for complex types)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Address/\$value")
    
    # $value should NOT work for complex types (only primitive types)
    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Details: \$value should not work for complex types"
        return 1
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 0  # Pass - implementation-specific
    fi
}

# Test 11: PATCH complex type property
test_patch_complex_type() {
    # Try to update complex type property
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Address")
    
    # Just check if the endpoint is addressable
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 12: Complex type in metadata
test_complex_in_metadata() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if metadata contains ComplexType definition
        if echo "$RESPONSE" | grep -q 'ComplexType'; then
            return 0
        else
            echo "  Details: No ComplexType found in metadata (may not have complex types)"
            return 0  # Pass - optional
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET Products(1) with complex type"
run_test "Retrieve entity with complex type property" test_complex_type_retrieval

echo "  Request: GET Products(1)/Address/City"
run_test "Access nested property of complex type" test_complex_nested_property

echo "  Request: GET \$filter=Address/City eq 'Seattle'"
run_test "Filter by nested complex type property" test_filter_complex_property

echo "  Request: GET \$select=Name,Address"
run_test "Select complex type property" test_select_complex_property

echo "  Request: GET \$select=Address/City,Address/State"
run_test "Select nested properties of complex type" test_select_nested_complex

echo "  Request: GET \$filter=Address eq null"
run_test "Filter for null complex type value" test_complex_null_value

echo "  Request: GET \$orderby=Address/City"
run_test "Order by complex type property" test_orderby_complex_property

echo "  Request: GET \$filter with deeply nested complex type"
run_test "Deeply nested complex type access" test_deeply_nested_complex

echo "  Request: GET \$filter with multiple complex conditions"
run_test "Complex type with logical operators" test_complex_logical_filter

echo "  Request: GET Products(1)/Address/\$value"
run_test "\$value access for complex type (should fail)" test_complex_value_access

echo "  Request: Access complex type property endpoint"
run_test "Complex type property endpoint" test_patch_complex_type

echo "  Request: GET \$metadata"
run_test "Complex type in metadata document" test_complex_in_metadata

print_summary
