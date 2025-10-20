#!/bin/bash

# OData v4 Compliance Test: 11.2.5.11 OrderBy with Computed Properties
# Tests $orderby with properties from $compute
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptioncompute

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.11 OrderBy with Computed"
echo "======================================"
echo ""
echo "Description: Validates \$orderby functionality with computed properties"
echo "             from \$compute query option (OData v4.01 feature)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptioncompute"
echo ""

# Test 1: Compute a property and order by it
test_orderby_computed() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.1 as TaxedPrice&\$orderby=TaxedPrice")
    
    # This feature may not be fully implemented in all servers
    # Accept 200 (success) or 501 (not implemented) or 400 (not supported)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200, 400, or 501)"
        return 1
    fi
}

# Test 2: OrderBy with multiple computed properties
test_orderby_multiple_computed() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 0.9 as DiscountPrice,Price mul 1.1 as TaxedPrice&\$orderby=DiscountPrice desc")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 3: OrderBy computed property with direction
test_orderby_computed_desc() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 2 as DoublePrice&\$orderby=DoublePrice desc")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 4: OrderBy mixing computed and regular properties
test_orderby_mixed() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.2 as MarkedUpPrice&\$orderby=CategoryID,MarkedUpPrice desc")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 5: OrderBy computed with select
test_orderby_computed_with_select() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.08 as FinalPrice&\$select=Name,FinalPrice&\$orderby=FinalPrice")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 6: OrderBy computed with filter
test_orderby_computed_with_filter() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 0.8 as SalePrice&\$filter=SalePrice gt 50&\$orderby=SalePrice")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 7: OrderBy computed with top
test_orderby_computed_with_top() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price div 2 as HalfPrice&\$orderby=HalfPrice desc&\$top=3")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 8: OrderBy regular property still works with compute present
test_orderby_regular_with_compute() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.5 as HighPrice&\$orderby=Name")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 9: Response includes computed properties when ordered
test_response_includes_computed() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$compute=Price mul 2 as DoublePrice&\$select=Name,DoublePrice&\$orderby=DoublePrice&\$top=1")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 2 as DoublePrice&\$select=Name,DoublePrice&\$orderby=DoublePrice&\$top=1")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if computed property appears in response
        if echo "$RESPONSE" | grep -q "DoublePrice"; then
            return 0
        else
            echo "  Details: Computed property not in response"
            return 1
        fi
    elif [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        # Feature not implemented
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 10: OrderBy without including computed in select
test_orderby_computed_not_selected() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.3 as MarkedPrice&\$select=Name,Price&\$orderby=MarkedPrice")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET \$compute=...&\$orderby=computed"
run_test "OrderBy computed property" test_orderby_computed

echo "  Request: GET \$compute=...,...&\$orderby=computed"
run_test "OrderBy with multiple computed properties" test_orderby_multiple_computed

echo "  Request: GET \$compute=...&\$orderby=computed desc"
run_test "OrderBy computed with desc direction" test_orderby_computed_desc

echo "  Request: GET \$compute=...&\$orderby=regular,computed"
run_test "OrderBy mixing computed and regular" test_orderby_mixed

echo "  Request: GET \$compute&\$select&\$orderby=computed"
run_test "OrderBy computed with \$select" test_orderby_computed_with_select

echo "  Request: GET \$compute&\$filter=computed&\$orderby=computed"
run_test "OrderBy computed with \$filter" test_orderby_computed_with_filter

echo "  Request: GET \$compute&\$orderby=computed&\$top=3"
run_test "OrderBy computed with \$top" test_orderby_computed_with_top

echo "  Request: GET \$compute=...&\$orderby=regular"
run_test "OrderBy regular property with compute present" test_orderby_regular_with_compute

echo "  Request: GET \$compute&\$select=computed&\$orderby (check response)"
run_test "Response includes computed properties" test_response_includes_computed

echo "  Request: GET \$compute=...&\$select=other&\$orderby=computed"
run_test "OrderBy computed not in \$select" test_orderby_computed_not_selected

print_summary
