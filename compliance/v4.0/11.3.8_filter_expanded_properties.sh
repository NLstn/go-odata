#!/bin/bash

# OData v4 Compliance Test: 11.3.8 Filter on Expanded Properties
# Tests filtering based on properties of expanded navigation entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ExpandSystemQueryOption

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.8 Filter on Expanded Properties"
echo "======================================"
echo ""
echo "Description: Validates filtering entities based on properties"
echo "             of their expanded navigation properties"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ExpandSystemQueryOption"
echo ""

# Test 1: Filter on collection navigation property using any()
test_filter_any_on_navigation() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN')")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Filter on collection navigation property using all()
test_filter_all_on_navigation() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/all(d: d/LanguageKey ne 'XX')")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Filter with any() and complex condition
test_filter_any_complex() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN' and contains(d/Description,'Laptop'))")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Expand with filter applied to expanded entities
test_expand_with_nested_filter() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$filter=LanguageKey eq 'EN')")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Filter main entities AND filter expanded entities
test_filter_both_levels() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price gt 100&\$expand=Descriptions(\$filter=LanguageKey eq 'EN')")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Any with string function on navigation property
test_any_with_string_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: contains(d/Description,'Gaming'))")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Multiple navigation property filters with any()
test_multiple_any_filters() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')")
    
    # This should work - Product has both EN and DE descriptions
    check_status "$HTTP_CODE" "200"
}

# Test 8: Filter using navigation property with or condition
test_navigation_filter_or() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN' or d/LanguageKey eq 'DE')")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Nested any() - checking if any description meets criteria
test_nested_any_condition() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: length(d/Description) gt 10)")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Expand and filter on same navigation property
test_expand_and_filter_same_nav() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN')&\$expand=Descriptions(\$filter=LanguageKey eq 'EN')")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Descriptions/any(d: d/LanguageKey eq 'EN')&\$expand=Descriptions(\$filter=LanguageKey eq 'EN')")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have value array
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

# Test 11: Filter with not and any on navigation
test_not_any_on_navigation() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=not Descriptions/any(d: d/LanguageKey eq 'XX')")
    check_status "$HTTP_CODE" "200"
}

# Test 12: Complex filter combining entity and navigation properties
test_complex_combined_filter() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price gt 50 and Descriptions/any(d: contains(d/Description,'Pro'))")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=Descriptions/any(d: d/LanguageKey eq 'EN')"
run_test "Filter using any() on navigation property" test_filter_any_on_navigation

echo "  Request: GET \$filter=Descriptions/all(d: ...)"
run_test "Filter using all() on navigation property" test_filter_all_on_navigation

echo "  Request: GET \$filter=Descriptions/any(d: ... and ...)"
run_test "Filter with any() and complex condition" test_filter_any_complex

echo "  Request: GET \$expand=Descriptions(\$filter=...)"
run_test "Expand with filter on expanded entities" test_expand_with_nested_filter

echo "  Request: GET \$filter=...&\$expand=Descriptions(\$filter=...)"
run_test "Filter both main and expanded entities" test_filter_both_levels

echo "  Request: GET \$filter=Descriptions/any(d: contains(...))"
run_test "any() with string function on navigation" test_any_with_string_function

echo "  Request: GET \$filter=...any(...) and ...any(...)"
run_test "Multiple any() filters on same navigation" test_multiple_any_filters

echo "  Request: GET \$filter=Descriptions/any(d: ... or ...)"
run_test "Navigation filter with or condition" test_navigation_filter_or

echo "  Request: GET \$filter=Descriptions/any(d: length(...) gt 10)"
run_test "Nested condition in any() with function" test_nested_any_condition

echo "  Request: GET \$filter=...any(...)&\$expand=...(\$filter=...)"
run_test "Expand and filter same navigation property" test_expand_and_filter_same_nav

echo "  Request: GET \$filter=not Descriptions/any(...)"
run_test "Filter with not and any on navigation" test_not_any_on_navigation

echo "  Request: GET \$filter=Price gt 50 and Descriptions/any(...)"
run_test "Complex filter combining entity and nav properties" test_complex_combined_filter

print_summary
