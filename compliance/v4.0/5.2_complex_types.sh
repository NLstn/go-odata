#!/bin/bash

# OData v4 Compliance Test: 5.2 Complex Types
# Tests complex (structured) types, nested properties, and complex type filtering
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType
#
# NOTE: This test validates complex type support in OData v4. According to the specification:
# 1. Complex types MUST be included in entity responses when they have values
# 2. Complex types MAY be null if marked as nullable
# 3. Nested property access (e.g., /Address/City) SHOULD be supported
# 4. Filtering by complex type properties SHOULD be supported
# 5. Selecting complex type properties SHOULD be supported
#
# The test validates:
# - Basic retrieval of entities with complex types (REQUIRED)
# - Proper error handling for unsupported features (400/404, not 500)
# - Null handling for nullable complex types
# - That the server doesn't crash on complex type operations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.2 Complex Types"
echo "======================================"
echo ""
echo "Description: Validates handling of complex (structured) types including"
echo "             nested properties, filtering, selecting, and operations."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType"
echo ""

# Test 1: Retrieve entity with complex type property
test_complex_type_retrieval() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if response includes complex type properties
        if echo "$RESPONSE" | grep -q 'ShippingAddress'; then
            # Verify it's a JSON object, not a string
            if echo "$RESPONSE" | grep -q '"ShippingAddress".*{'; then
                return 0
            else
                echo "  Details: ShippingAddress is not a complex object"
                return 1
            fi
        else
            echo "  Details: Response missing ShippingAddress complex type"
            return 1
        fi
    else
        echo "  Details: Status: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: Access nested property of complex type
test_complex_nested_property() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/ShippingAddress/City")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/ShippingAddress/City")

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 for nested property, got $HTTP_CODE"
        return 1
    fi

    if ! echo "$RESPONSE" | grep -q '"value":"Seattle"'; then
        echo "  Details: Nested property response missing expected value"
        return 1
    fi

    return 0
}

# Test 3: Filter by nested complex type property
test_filter_complex_property() {
    # Filter by nested property of complex type
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24filter=ShippingAddress/City%20eq%20'Seattle'")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 for complex type filter, got $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 4: Select complex type property
test_select_complex_property() {
    # Select entire complex type
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24select=Name,ShippingAddress")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type in \$select not supported (status: $HTTP_CODE)"
        return 0  # Pass - this is optional per spec
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for unsupported features"
        return 1  # Fail - should not crash
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 5: Select nested property of complex type
test_select_nested_complex() {
    # Select specific property within complex type
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24select=ShippingAddress/City,ShippingAddress/State")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Nested \$select not supported (status: $HTTP_CODE)"
        return 0  # Pass - this is optional per spec
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for unsupported features"
        return 1  # Fail - should not crash
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Complex type with null value
test_complex_null_value() {
    # Retrieve entity with null complex type (Product 4 has null ShippingAddress)
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(4)")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(4)")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check that ShippingAddress is either absent or explicitly null
        if echo "$RESPONSE" | grep -q '"ShippingAddress":null' || ! echo "$RESPONSE" | grep -q 'ShippingAddress'; then
            return 0
        else
            echo "  Details: Null complex type not properly represented"
            return 1
        fi
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: Filter for null complex type
test_filter_null_complex() {
    # Filter for entities where complex type is null
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24filter=ShippingAddress%20eq%20null")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Null complex type filtering not supported (status: $HTTP_CODE)"
        return 0  # Pass - this is optional per spec
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for unsupported features"
        return 1  # Fail - should not crash
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Complex type in $orderby
test_orderby_complex_property() {
    # Order by nested property of complex type
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24orderby=ShippingAddress/City")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex type in \$orderby not supported (status: $HTTP_CODE)"
        return 0  # Pass - this is optional per spec
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for unsupported features"
        return 1  # Fail - should not crash
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 9: Complex type with logical operators in filter
test_complex_logical_filter() {
    # Combine multiple complex type filters
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products?%24filter=ShippingAddress/City%20eq%20'Seattle'%20and%20ShippingAddress/State%20eq%20'WA'")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Complex logical filters not supported (status: $HTTP_CODE)"
        return 0  # Pass - this is optional per spec
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for unsupported features"
        return 1  # Fail - should not crash
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Complex type property $value access
test_complex_value_access() {
    # Access $value of complex type property (should return 404 - not supported for complex types)
    # URL encode the $ as %24 to avoid shell interpretation issues
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/Products(1)/ShippingAddress/%24value")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
    # $value should NOT work for complex types (only primitive types)
    # Per OData spec, accessing $value on a complex type should return 400 or 404
    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Details: \$value should not work for complex types (got 200)"
        return 1
    elif [ "$HTTP_CODE" = "500" ]; then
        echo "  Details: Server error (500) - should return 400/404 for invalid operations"
        return 1
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 0  # Pass - implementation-specific
    fi
}

# Test 11: Access complex type property directly
test_access_complex_type() {
    # Access the complex type property itself
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/ShippingAddress")
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/ShippingAddress")

    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 when accessing complex property, got $HTTP_CODE"
        return 1
    fi

    if ! echo "$RESPONSE" | grep -q '"City":"Seattle"'; then
        echo "  Details: Complex property response missing expected City value"
        return 1
    fi

    return 0
}

# Test 12: Complex type in metadata
test_complex_in_metadata() {
    # URL encode the $ as %24 to avoid shell interpretation issues
    local RESPONSE=$(curl -s "${SERVER_URL}/%24metadata")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVER_URL}/%24metadata")
    
    # 000 means curl failed - this is a test failure
    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi
    
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

# Test 13: HEAD on complex type property
test_complex_property_head() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "$SERVER_URL/Products(1)/ShippingAddress")

    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi

    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    echo "  Details: Expected 200 for HEAD on complex property, got $HTTP_CODE"
    return 1
}

# Test 14: OPTIONS on complex type property
test_complex_property_options() {
    local RESPONSE=$(curl -s -D - -o /dev/null -w "%{http_code}" -X OPTIONS "$SERVER_URL/Products(1)/ShippingAddress")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    local HEADERS=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Curl failed (connection error or invalid URL)"
        return 1
    fi

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 for OPTIONS on complex property, got $HTTP_CODE"
        return 1
    fi

    if ! echo "$HEADERS" | grep -qi 'Allow: GET, HEAD, OPTIONS'; then
        echo "  Details: OPTIONS response missing Allow: GET, HEAD, OPTIONS"
        return 1
    fi

    return 0
}

echo "  Request: GET Products(1) with complex type"
run_test "Retrieve entity with complex type property" test_complex_type_retrieval

echo "  Request: GET Products(1)/ShippingAddress/City"
run_test "Access nested property of complex type" test_complex_nested_property

echo "  Request: GET \$filter=ShippingAddress/City eq 'Seattle'"
run_test "Filter by nested complex type property" test_filter_complex_property

echo "  Request: GET \$select=Name,ShippingAddress"
run_test "Select complex type property" test_select_complex_property

echo "  Request: GET \$select=ShippingAddress/City,ShippingAddress/State"
run_test "Select nested properties of complex type" test_select_nested_complex

echo "  Request: GET Products(4) with null complex type"
run_test "Retrieve entity with null complex type" test_complex_null_value

echo "  Request: GET \$filter=ShippingAddress eq null"
run_test "Filter for null complex type value" test_filter_null_complex

echo "  Request: GET \$orderby=ShippingAddress/City"
run_test "Order by complex type property" test_orderby_complex_property

echo "  Request: GET \$filter with multiple complex conditions"
run_test "Complex type with logical operators" test_complex_logical_filter

echo "  Request: GET Products(1)/ShippingAddress/\$value"
run_test "\$value access for complex type (should fail)" test_complex_value_access

echo "  Request: Access complex type property endpoint"
run_test "Access complex type property directly" test_access_complex_type

echo "  Request: HEAD Products(1)/ShippingAddress"
run_test "HEAD on complex type property" test_complex_property_head

echo "  Request: OPTIONS Products(1)/ShippingAddress"
run_test "OPTIONS on complex type property" test_complex_property_options

echo "  Request: GET \$metadata"
run_test "Complex type in metadata document" test_complex_in_metadata

print_summary
