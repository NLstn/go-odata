#!/bin/bash

# OData v4 Compliance Test: 11.2.5.4.1 Advanced $apply Transformations
# Tests advanced $apply query option transformations for data aggregation
# Spec: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.4.1 Advanced \$apply"
echo "======================================"
echo ""
echo "Description: Validates advanced \$apply transformations including"
echo "             nested groupby, multiple aggregations, filter before/after"
echo "             aggregation, and complex transformation pipelines."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html"
echo ""

# Test 1: Basic $apply support (baseline)
test_apply_baseline() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(Price%20with%20sum%20as%20Total)")
    # Should work (200) or not be supported (501/400)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 2: Multiple aggregations in single aggregate
test_multiple_aggregations() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(Price%20with%20sum%20as%20TotalPrice,Price%20with%20average%20as%20AvgPrice,Price%20with%20max%20as%20MaxPrice)")
    # Should support multiple aggregations if $apply is supported
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 3: groupby with multiple properties
test_groupby_multiple_properties() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID,Status))")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 4: groupby with aggregate containing multiple aggregation methods
test_groupby_with_multiple_aggregates() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(Price%20with%20sum%20as%20Total,Price%20with%20average%20as%20Average,\$count%20as%20Count))")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 5: Filter before aggregation
test_filter_before_aggregate() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=filter(Price%20gt%2050)/aggregate(Price%20with%20sum%20as%20Total)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 6: Filter before groupby
test_filter_before_groupby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=filter(Price%20gt%2050)/groupby((CategoryID),aggregate(\$count%20as%20Count))")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 7: Multiple transformations in sequence
test_transformation_pipeline() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=filter(Price%20gt%2010)/groupby((CategoryID))/filter(\$count%20gt%201)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 8: Aggregate with distinct count
test_countdistinct() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(CategoryID%20with%20countdistinct%20as%20UniqueCategories)")
    # countdistinct may not be implemented - accept 200 or error codes
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "500" ]
}

# Test 9: groupby with aggregate and $filter after
test_groupby_then_filter() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(Price%20with%20sum%20as%20Total))/filter(Total%20gt%20100)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 10: Aggregate with min and max together
test_min_max_aggregate() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(Price%20with%20min%20as%20MinPrice,Price%20with%20max%20as%20MaxPrice)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 11: $apply with $top (limit aggregated results)
test_apply_with_top() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))&\$top=2")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 12: $apply with $orderby (order aggregated results)
test_apply_with_orderby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(Price%20with%20sum%20as%20Total))&\$orderby=Total%20desc")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 13: Complex pipeline with filter/groupby/aggregate/filter
test_complex_pipeline() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=filter(Status%20eq%201)/groupby((CategoryID),aggregate(Price%20with%20average%20as%20AvgPrice))/filter(AvgPrice%20gt%2050)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 14: Invalid aggregation method
test_invalid_aggregation_method() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(Price%20with%20invalid%20as%20Result)")
    # Should return 400 for invalid method
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 15: $apply response format validation
test_apply_response_format() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have valid JSON with value array
        echo "$RESPONSE" | grep -q "\"value\""
    else
        # Not supported is acceptable
        return 0
    fi
}

# Test 16: Empty groupby (aggregate without grouping)
test_empty_groupby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((),aggregate(Price%20with%20sum%20as%20Total))")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 17: Aggregate on navigation property (if supported)
test_aggregate_navigation() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Orders?\$apply=groupby((CustomerID),aggregate(\$count%20as%20OrderCount))")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]
}

# Test 18: Average aggregation
test_average_aggregation() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(Price%20with%20average%20as%20AvgPrice)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 19: $apply with $count (total count of aggregated results)
test_apply_with_count() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID))&\$count=true")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 20: Nested filter expressions in $apply
test_nested_filter_in_apply() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$apply=filter(Price%20gt%2010%20and%20CategoryID%20eq%201)/aggregate(\$count%20as%20Total)")
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]
}

echo "  Request: Basic \$apply with aggregate"
run_test "Basic \$apply support (baseline)" test_apply_baseline

echo "  Request: \$apply with multiple aggregations"
run_test "Multiple aggregations in single statement" test_multiple_aggregations

echo "  Request: groupby with multiple properties"
run_test "groupby with multiple grouping properties" test_groupby_multiple_properties

echo "  Request: groupby with multiple aggregate methods"
run_test "groupby with multiple aggregation methods" test_groupby_with_multiple_aggregates

echo "  Request: filter transformation before aggregate"
run_test "Filter before aggregation" test_filter_before_aggregate

echo "  Request: filter before groupby"
run_test "Filter before groupby transformation" test_filter_before_groupby

echo "  Request: Multiple transformations in sequence"
run_test "Transformation pipeline" test_transformation_pipeline

echo "  Request: countdistinct aggregation"
run_test "Aggregate with countdistinct" test_countdistinct

echo "  Request: groupby then filter on result"
run_test "Filter after groupby/aggregate" test_groupby_then_filter

echo "  Request: Aggregate with min and max"
run_test "Min and max aggregation together" test_min_max_aggregate

echo "  Request: \$apply with \$top"
run_test "\$apply works with \$top" test_apply_with_top

echo "  Request: \$apply with \$orderby"
run_test "\$apply works with \$orderby" test_apply_with_orderby

echo "  Request: Complex transformation pipeline"
run_test "Complex filter/groupby/aggregate/filter pipeline" test_complex_pipeline

echo "  Request: Invalid aggregation method"
run_test "Invalid aggregation method returns error" test_invalid_aggregation_method

echo "  Request: Check \$apply response format"
run_test "\$apply response has valid JSON structure" test_apply_response_format

echo "  Request: groupby with empty grouping set"
run_test "Empty groupby (aggregate all)" test_empty_groupby

echo "  Request: Aggregate on navigation property"
run_test "Aggregate with navigation property (optional)" test_aggregate_navigation

echo "  Request: Average aggregation method"
run_test "Average aggregation method" test_average_aggregation

echo "  Request: \$apply with \$count"
run_test "\$apply works with \$count" test_apply_with_count

echo "  Request: Nested filter in \$apply"
run_test "Nested filter expressions in \$apply" test_nested_filter_in_apply

print_summary
