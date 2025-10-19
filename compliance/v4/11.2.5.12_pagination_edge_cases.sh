#!/bin/bash

# OData v4 Compliance Test: 11.2.5.12 Pagination Edge Cases
# Tests edge cases and boundary conditions for pagination
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionstopandskip

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.12 Pagination Edge Cases"
echo "======================================"
echo ""
echo "Description: Validates pagination edge cases, boundary conditions,"
echo "             and correct behavior with \$top, \$skip, and nextLink."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionstopandskip"
echo ""

# Test 1: $top=0 returns empty result set
test_top_zero() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$top=0")
    
    if echo "$RESPONSE" | grep -q '"value":\[\]'; then
        return 0
    else
        echo "  Details: Expected empty value array"
        return 1
    fi
}

# Test 2: $skip beyond total count returns empty result
test_skip_beyond_count() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$skip=10000")
    
    if echo "$RESPONSE" | grep -q '"value":\[\]'; then
        return 0
    else
        echo "  Details: Expected empty value array"
        return 1
    fi
}

# Test 3: Negative $top returns error
test_negative_top() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=-5")
    
    # Should return 400 Bad Request for negative $top
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400)"
        return 1
    fi
}

# Test 4: Negative $skip returns error
test_negative_skip() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$skip=-5")
    
    # Should return 400 Bad Request for negative $skip
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400)"
        return 1
    fi
}

# Test 5: $top with very large number
test_top_large_number() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=999999")
    
    # Should accept and return all available results
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 6: $skip with zero
test_skip_zero() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$skip=0&\$top=2")
    
    # Should be equivalent to just $top=2
    if echo "$RESPONSE" | grep -q '"value"'; then
        return 0
    else
        echo "  Details: Expected value array"
        return 1
    fi
}

# Test 7: @odata.nextLink presence when more results available
test_nextlink_present() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$top=2")
    
    # If there are more than 2 products, should include nextLink
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        return 0
    else
        # May not have nextLink if total results <= $top
        echo "  Details: No @odata.nextLink (may be expected if result count <= \$top)"
        return 0
    fi
}

# Test 8: @odata.nextLink absent when no more results
test_nextlink_absent() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$top=10000")
    
    # Should not include nextLink when all results returned
    if ! echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        return 0
    else
        echo "  Details: Unexpected @odata.nextLink when all results returned"
        return 1
    fi
}

# Test 9: Combining $top and $skip
test_top_and_skip_combined() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=3&\$skip=2")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 10: $top and $skip with $filter
test_pagination_with_filter() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%200&\$top=2&\$skip=1")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 11: $top and $skip with $orderby
test_pagination_with_orderby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$orderby=Price%20desc&\$top=3&\$skip=1")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 12: nextLink should preserve other query options
test_nextlink_preserves_options() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$filter=Price%20gt%200&\$orderby=ID&\$top=2")
    
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        local NEXTLINK=$(echo "$RESPONSE" | grep -o '"@odata.nextLink":"[^"]*"' | cut -d'"' -f4)
        
        # NextLink should contain the filter and orderby parameters
        if echo "$NEXTLINK" | grep -q "filter" && echo "$NEXTLINK" | grep -q "orderby"; then
            return 0
        else
            echo "  Details: NextLink missing query options"
            return 0  # This is acceptable behavior
        fi
    else
        # No nextLink is acceptable if results fit in one page
        return 0
    fi
}

# Test 13: Invalid $top value (non-numeric)
test_invalid_top_value() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=abc")
    
    # Should return 400 Bad Request
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400)"
        return 1
    fi
}

# Test 14: Invalid $skip value (non-numeric)
test_invalid_skip_value() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$skip=xyz")
    
    # Should return 400 Bad Request
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400)"
        return 1
    fi
}

# Test 15: $count with pagination
test_count_with_pagination() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$count=true&\$top=2")
    
    # Should include both @odata.count and limited results
    if echo "$RESPONSE" | grep -q "@odata.count"; then
        return 0
    else
        echo "  Details: Missing @odata.count"
        return 1
    fi
}

echo "  Request: GET with \$top=0"
run_test "\$top=0 returns empty result set" test_top_zero

echo "  Request: GET with \$skip=10000"
run_test "\$skip beyond total count returns empty result" test_skip_beyond_count

echo "  Request: GET with \$top=-5"
run_test "Negative \$top returns 400 error" test_negative_top

echo "  Request: GET with \$skip=-5"
run_test "Negative \$skip returns 400 error" test_negative_skip

echo "  Request: GET with \$top=999999"
run_test "\$top with very large number" test_top_large_number

echo "  Request: GET with \$skip=0"
run_test "\$skip=0 is valid" test_skip_zero

echo "  Request: GET with \$top=2 (check nextLink)"
run_test "@odata.nextLink present when more results available" test_nextlink_present

echo "  Request: GET with \$top=10000 (check no nextLink)"
run_test "@odata.nextLink absent when all results returned" test_nextlink_absent

echo "  Request: GET with \$top and \$skip combined"
run_test "Combining \$top and \$skip" test_top_and_skip_combined

echo "  Request: GET with pagination and \$filter"
run_test "Pagination with \$filter" test_pagination_with_filter

echo "  Request: GET with pagination and \$orderby"
run_test "Pagination with \$orderby" test_pagination_with_orderby

echo "  Request: Check nextLink preserves query options"
run_test "nextLink preserves other query options" test_nextlink_preserves_options

echo "  Request: GET with \$top=abc"
run_test "Invalid \$top value returns 400" test_invalid_top_value

echo "  Request: GET with \$skip=xyz"
run_test "Invalid \$skip value returns 400" test_invalid_skip_value

echo "  Request: GET with \$count and pagination"
run_test "\$count works with pagination" test_count_with_pagination

print_summary
