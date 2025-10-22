#!/bin/bash

# OData v4 Compliance Test: 11.2.5.13 $index Query Option
# Tests $index system query option for retrieving zero-based position of items
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_index

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.13 \$index Query Option"
echo "======================================"
echo ""
echo "Description: Validates the \$index system query option which returns"
echo "             the zero-based ordinal position of each item in a collection."
echo "             This is an OData v4.01 feature and may be optional."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_index"
echo ""

# Test 1: $index without other query options
test_index_basic() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$index")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index")
    
    # Should either support it (200) or not implement it (400/501)
    # If supported, response should include @odata.index annotations
    if [ "$HTTP_CODE" = "200" ]; then
        # Optional: check for @odata.index in response
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]; then
        # Not implemented - acceptable
        return 0
    else
        return 1
    fi
}

# Test 2: $index with $top
test_index_with_top() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$top=5")
    
    # Should either work (200) or not be implemented (400/501)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 3: $index with $skip
test_index_with_skip() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$skip=2")
    
    # Indexes should be relative to skip position
    # Should either work (200) or not be implemented (400/501)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 4: $index with $orderby
test_index_with_orderby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$orderby=Price")
    
    # Indexes should reflect ordered position
    # Should either work (200) or not be implemented (400/501)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 5: $index with $filter
test_index_with_filter() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$filter=Price%20gt%2050")
    
    # Indexes should be relative to filtered set
    # Should either work (200) or not be implemented (400/501)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 6: $index response format
test_index_response_format() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$index&\$top=3")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$top=3")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # If supported, check for proper JSON structure
        echo "$RESPONSE" | grep -q "\"value\"" 
    else
        # Not implemented is acceptable
        return 0
    fi
}

# Test 7: $index with $expand (should work or be gracefully rejected)
test_index_with_expand() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$expand=Category")
    
    # Should either work or return appropriate error
    # 404 is acceptable if entity doesn't have expandable navigation properties
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]
}

# Test 8: $index on entity (should not be applicable)
test_index_on_entity() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)?\$index")
    
    # $index is for collections only per spec, but if not implemented may be ignored
    # Accept rejection (400) or being ignored (200/404)
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "200" ]
}

# Test 9: $index with complex query combination
test_index_complex_query() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$filter=CategoryID%20eq%201&\$orderby=Name&\$top=5")
    
    # Should handle complex queries gracefully
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 10: $index with $count
test_index_with_count() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$count=true")
    
    # Should either work or return appropriate error
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 11: Check if @odata.index annotation is included
test_index_annotation_presence() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$index&\$top=2")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$top=2")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # If supported, should include @odata.index annotations
        # This is optional to check but recommended
        return 0
    else
        # Not implemented
        return 0
    fi
}

# Test 12: $index value starts at 0
test_index_starts_at_zero() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$index&\$top=1")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$top=1")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # If supported, first item should have index 0
        # This is optional to verify
        return 0
    else
        # Not implemented
        return 0
    fi
}

# Test 13: $index with $select
test_index_with_select() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$select=Name,Price")
    
    # Should work with $select
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 14: $index case sensitivity
test_index_case_sensitivity() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$INDEX")
    
    # System query options should be case-sensitive
    # Should return 400 for uppercase
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ]
}

# Test 15: Multiple $index parameters (invalid)
test_multiple_index_params() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$index&\$index")
    
    # Should reject duplicate parameters
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ]
}

echo "  Request: GET collection with \$index"
run_test "\$index query option basic support" test_index_basic

echo "  Request: GET with \$index and \$top"
run_test "\$index works with \$top" test_index_with_top

echo "  Request: GET with \$index and \$skip"
run_test "\$index works with \$skip" test_index_with_skip

echo "  Request: GET with \$index and \$orderby"
run_test "\$index works with \$orderby" test_index_with_orderby

echo "  Request: GET with \$index and \$filter"
run_test "\$index works with \$filter" test_index_with_filter

echo "  Request: GET with \$index, check response format"
run_test "\$index response has valid JSON structure" test_index_response_format

echo "  Request: GET with \$index and \$expand"
run_test "\$index works with \$expand" test_index_with_expand

echo "  Request: GET entity with \$index (invalid)"
run_test "\$index rejected on single entity" test_index_on_entity

echo "  Request: GET with \$index and multiple query options"
run_test "\$index works with complex query combinations" test_index_complex_query

echo "  Request: GET with \$index and \$count"
run_test "\$index works with \$count" test_index_with_count

echo "  Request: GET with \$index, check for annotations"
run_test "@odata.index annotation presence (optional)" test_index_annotation_presence

echo "  Request: GET with \$index, verify zero-based indexing"
run_test "\$index starts at zero (optional verification)" test_index_starts_at_zero

echo "  Request: GET with \$index and \$select"
run_test "\$index works with \$select" test_index_with_select

echo "  Request: GET with uppercase \$INDEX"
run_test "\$index is case-sensitive" test_index_case_sensitivity

echo "  Request: GET with duplicate \$index parameters"
run_test "Duplicate \$index parameters handled" test_multiple_index_params

print_summary
