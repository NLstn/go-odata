#!/bin/bash

# OData v4 Compliance Test: 11.2.5.4 System Query Option $apply
# Tests $apply query option for data aggregation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic aggregate transformation
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=aggregate(\$count%20as%20Total)")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=aggregate(\$count%20as%20Total)")
        check_json_field "$BODY" "Total"
        return $?
    fi

    echo "  Details: Expected HTTP 200 for compliant $apply aggregate, got $HTTP_CODE"
    echo "           This indicates a compliance failure for required $apply support."
    return 1
}

# Test 2: groupby transformation
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=groupby((CategoryID))")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=groupby((CategoryID))")
        check_json_field "$BODY" "CategoryID"
        return $?
    fi

    echo "  Details: Expected HTTP 200 for compliant groupby transformation, got $HTTP_CODE"
    echo "           This indicates a compliance failure for required $apply groupby support."
    return 1
}

# Test 3: groupby with aggregate
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
        if check_json_field "$BODY" "CategoryID" && check_json_field "$BODY" "Count"; then
            return 0
        fi
        echo "  Details: Response body missing expected fields for compliant groupby/aggregate."
        return 1
    fi

    echo "  Details: Expected HTTP 200 for compliant groupby/aggregate, got $HTTP_CODE"
    echo "           This indicates a compliance failure for required $apply support."
    return 1
}

# Test 4: filter transformation
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=filter(Price%20gt%2010)")
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    echo "  Details: Expected HTTP 200 for compliant filter transformation, got $HTTP_CODE"
    echo "           This indicates a compliance failure for required $apply support."
    return 1
}

# Test 5: Invalid $apply expression should return 400
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=invalid(syntax)")
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    fi

    echo "  Details: Expected HTTP 400 for invalid $apply expression, got $HTTP_CODE"
    echo "           This indicates a compliance failure for required error handling."
    return 1
}

# Run all tests
run_test "\$apply with aggregate (count)" test_1
run_test "\$apply with groupby" test_2
run_test "\$apply with groupby and aggregate" test_3
run_test "\$apply with filter transformation" test_4
run_test "Invalid \$apply expression returns 400" test_5

print_summary
