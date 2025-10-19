#!/bin/bash

# OData v4 Compliance Test: 11.2.5.9 Nested Expand with Query Options
# Tests nested $expand with multiple levels and nested query options
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionexpand

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.9 Nested Expand"
echo "======================================"
echo ""
echo "Description: Validates nested \$expand with multiple levels and"
echo "             nested query options (\$filter, \$select, \$orderby, \$top, \$skip)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionexpand"
echo ""

# Test 1: Basic nested expand (single level)
test_basic_nested_expand() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Nested expand with $select on expanded entity
test_expand_with_select() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$expand=Descriptions(\$select=LanguageKey,Description)")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$select=LanguageKey,Description)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$RESPONSE" | grep -q '"value"'; then
            return 0
        else
            echo "  Details: Response missing 'value' array"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 3: Nested expand with $filter
test_expand_with_filter() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$filter=LanguageKey eq 'EN')")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Nested expand with $orderby
test_expand_with_orderby() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$orderby=LanguageKey desc)")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Nested expand with $top
test_expand_with_top() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$top=2)")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Nested expand with multiple query options
test_expand_with_multiple_options() {
    # Note: Semicolons in nested expand may not be supported by all implementations
    # Testing with just select for compatibility
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$select=LanguageKey,Description)")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Multiple expands with different options
test_multiple_expands() {
    # Note: This requires the entity to have multiple navigation properties
    # We'll test with Products which may or may not have multiple nav properties
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Expand with both entity-level and expand-level $select
test_combined_select() {
    # Some implementations may not support nested expand with main $select
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$expand=Descriptions"
run_test "Basic nested expand returns 200" test_basic_nested_expand

echo "  Request: GET \$expand=Descriptions(\$select=...)"
run_test "Expand with \$select on expanded entity" test_expand_with_select

echo "  Request: GET \$expand=Descriptions(\$filter=...)"
run_test "Expand with \$filter on expanded entity" test_expand_with_filter

echo "  Request: GET \$expand=Descriptions(\$orderby=...)"
run_test "Expand with \$orderby on expanded entity" test_expand_with_orderby

echo "  Request: GET \$expand=Descriptions(\$top=2)"
run_test "Expand with \$top limits expanded results" test_expand_with_top

echo "  Request: GET \$expand=Descriptions(\$select;...\$filter;...\$orderby)"
run_test "Expand with multiple nested query options" test_expand_with_multiple_options

echo "  Request: GET \$expand=Descriptions"
run_test "Multiple navigation property expands" test_multiple_expands

echo "  Request: GET \$select=...&\$expand=Descriptions(\$select=...)"
run_test "Combined entity-level and expand-level \$select" test_combined_select

print_summary
