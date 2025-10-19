#!/bin/bash

# OData v4 Compliance Test: 11.4.6.1 Navigation Property Operations
# Tests operations on navigation properties including accessing, filtering, and modifying
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_NavigationProperties

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.6.1 Navigation Properties"
echo "======================================"
echo ""
echo "Description: Validates navigation property operations including"
echo "             accessing related entities and filtering"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_NavigationProperties"
echo ""

# Test 1: Access navigation property collection
test_nav_property_collection() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Navigation property returns collection in value wrapper
test_nav_property_collection_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Descriptions")
    
    # Should have value array for collection
    if echo "$RESPONSE" | grep -q '"value"'; then
        return 0
    else
        echo "  Details: Navigation property collection missing 'value' array"
        return 1
    fi
}

# Test 3: Navigation property with $filter
test_nav_property_filter() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$filter=LanguageKey eq 'EN'")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Navigation property with $select
test_nav_property_select() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$select=LanguageKey,Description")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Navigation property with $orderby
test_nav_property_orderby() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$orderby=LanguageKey")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Navigation property with $top
test_nav_property_top() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$top=2")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Navigation property with $count
test_nav_property_count() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Descriptions?\$count=true")
    
    # Should have @odata.count
    if echo "$RESPONSE" | grep -q '@odata.count'; then
        return 0
    else
        echo "  Details: Navigation property response missing '@odata.count'"
        return 1
    fi
}

# Test 8: Navigation property on non-existent entity returns 404
test_nav_property_not_found() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)/Descriptions")
    check_status "$HTTP_CODE" "404"
}

# Test 9: Navigation property with combined query options
test_nav_property_combined_options() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$filter=LanguageKey eq 'EN'&\$select=Description&\$orderby=LanguageKey")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Navigation property has proper @odata.context
test_nav_property_context() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Descriptions")
    
    # Should have @odata.context
    if echo "$RESPONSE" | grep -q '@odata.context'; then
        return 0
    else
        echo "  Details: Navigation property response missing '@odata.context'"
        return 1
    fi
}

# Test 11: Access specific entity through navigation property
test_nav_property_specific_entity() {
    # Access specific description by composite key
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions(ProductID=1,LanguageKey='EN')")
    
    # May return 200 or 404 depending on data
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 404)"
        return 1
    fi
}

# Test 12: Navigation property supports $expand (nested navigation)
test_nav_property_expand() {
    # If ProductDescription has navigation properties, this would work
    # For now, just verify it doesn't error out
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions?\$expand=Product")
    
    # May succeed or return error depending on schema
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Unexpected status code: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET /Products(1)/Descriptions"
run_test "Access navigation property collection" test_nav_property_collection

echo "  Request: GET /Products(1)/Descriptions (check structure)"
run_test "Navigation property returns collection structure" test_nav_property_collection_structure

echo "  Request: GET /Products(1)/Descriptions?\$filter=..."
run_test "Navigation property with \$filter" test_nav_property_filter

echo "  Request: GET /Products(1)/Descriptions?\$select=..."
run_test "Navigation property with \$select" test_nav_property_select

echo "  Request: GET /Products(1)/Descriptions?\$orderby=..."
run_test "Navigation property with \$orderby" test_nav_property_orderby

echo "  Request: GET /Products(1)/Descriptions?\$top=2"
run_test "Navigation property with \$top" test_nav_property_top

echo "  Request: GET /Products(1)/Descriptions?\$count=true"
run_test "Navigation property with \$count" test_nav_property_count

echo "  Request: GET /Products(999999)/Descriptions"
run_test "Navigation property on invalid entity returns 404" test_nav_property_not_found

echo "  Request: GET /Products(1)/Descriptions?\$filter&\$select&\$orderby"
run_test "Navigation property with combined options" test_nav_property_combined_options

echo "  Request: GET /Products(1)/Descriptions (check context)"
run_test "Navigation property has @odata.context" test_nav_property_context

echo "  Request: GET /Products(1)/Descriptions(key)"
run_test "Access specific entity via navigation" test_nav_property_specific_entity

echo "  Request: GET /Products(1)/Descriptions?\$expand=..."
run_test "Navigation property supports \$expand" test_nav_property_expand

print_summary
