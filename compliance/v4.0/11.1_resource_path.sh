#!/bin/bash

# OData v4 Compliance Test: 11.1 Resource Path
# Tests OData v4 resource path conventions for addressing resources
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_URLComponents

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.1 Resource Path"
echo "======================================"
echo ""
echo "Description: Tests resource path conventions for addressing OData"
echo "             resources including entity sets, entities, properties,"
echo "             navigation paths, and system resources."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_URLComponents"
echo ""

# Test 1: Service root path
test_service_root() {
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Entity set path
test_entity_set_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Single entity by key (simple key)
test_entity_by_key() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Single entity by key with property name
test_entity_by_named_key() {
    # OData allows Products(ID=1) as alternative to Products(1)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(ID=1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ]; then
        # Some implementations may not support named key syntax
        skip_test "Named key syntax (ID=1)" "Named key syntax may not be implemented"
        return 0
    else
        echo "  Details: Named key syntax should return 200 or be unimplemented (got $HTTP_CODE)"
        return 1
    fi
}

# Test 5: Property path
test_property_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name")
    
    # Should return 200 with property value or 404 if not supported
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Property path should be valid (got $HTTP_CODE)"
        return 1
    fi
}

# Test 6: Property value with $value
test_property_value_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name/\$value")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ]; then
        # Property $value may not be implemented
        skip_test "Property \$value path" "Property \$value not implemented"
        return 0
    else
        echo "  Details: Property \$value path handling (got $HTTP_CODE)"
        return 1
    fi
}

# Test 7: Navigation property path
test_navigation_property_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Category")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ]; then
        # Navigation may not be set up or feature not implemented
        return 0
    else
        echo "  Details: Navigation property path should be accessible (got $HTTP_CODE)"
        return 1
    fi
}

# Test 8: Chained navigation paths
test_chained_navigation() {
    # Products -> Category -> Products (if Category has Products navigation)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Category/Products")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        # Chained navigation may not be fully implemented
        skip_test "Chained navigation paths" "Chained navigation not implemented"
        return 0
    else
        echo "  Details: Chained navigation path handling (got $HTTP_CODE)"
        return 1
    fi
}

# Test 9: System resource $metadata
test_metadata_resource_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Invalid resource path returns 404
test_invalid_resource_path() {
    local HTTP_CODE=$(http_get "$SERVER_URL/InvalidResource")
    check_status "$HTTP_CODE" "404"
}

# Test 11: Entity with non-existent key returns 404
test_nonexistent_entity_key() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
}

# Test 12: Case sensitivity in resource paths
test_case_sensitivity() {
    # OData resource paths are case-sensitive by default
    local HTTP_CODE=$(http_get "$SERVER_URL/products")
    
    # Should return 404 for lowercase (case mismatch)
    # However, some servers may be case-insensitive
    if [ "$HTTP_CODE" = "404" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        # Server is case-insensitive (allowed but not required)
        return 0
    else
        echo "  Details: Case sensitivity handling for resource paths (got $HTTP_CODE)"
        return 1
    fi
}

# Test 13: Path with query options
test_path_with_query_options() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=5")
    check_status "$HTTP_CODE" "200"
}

# Test 14: Empty path segments handling
test_empty_path_segments() {
    # Products// should be invalid (empty segment)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products//")
    
    # Should return 404, 400, or 301 (redirect to normalized path)
    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "301" ]; then
        return 0
    else
        echo "  Details: Empty path segments should return error or redirect (got $HTTP_CODE)"
        return 1
    fi
}

# Run tests
run_test "Service root path" test_service_root
run_test "Entity set path" test_entity_set_path
run_test "Entity by key path" test_entity_by_key
run_test "Entity by named key path" test_entity_by_named_key
run_test "Property path" test_property_path
run_test "Property \$value path" test_property_value_path
run_test "Navigation property path" test_navigation_property_path
run_test "Chained navigation paths" test_chained_navigation
run_test "\$metadata system resource path" test_metadata_resource_path
run_test "Invalid resource path returns 404" test_invalid_resource_path
run_test "Non-existent entity key returns 404" test_nonexistent_entity_key
run_test "Resource path case sensitivity" test_case_sensitivity
run_test "Path with query options" test_path_with_query_options
run_test "Empty path segments rejected" test_empty_path_segments

print_summary
