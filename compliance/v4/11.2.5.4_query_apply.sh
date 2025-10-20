#!/bin/bash

# OData v4 Compliance Test: 11.2.5.4 System Query Option $apply
# Tests $apply query option for data aggregation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Test 1: Basic aggregate transformation
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=aggregate(\$count%20as%20Total)")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=aggregate(\$count%20as%20Total)")
        check_json_field "$BODY" "Total"
    elif [ "$HTTP_CODE" = "501" ]; then
        # $apply not implemented (optional extension)
        return 0
    else
        return 1
    fi
}

# Test 2: groupby transformation
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=groupby((CategoryID))")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=groupby((CategoryID))")
        check_json_field "$BODY" "CategoryID"
    elif [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        return 1
    fi
}

# Test 3: groupby with aggregate
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
    if [ "$HTTP_CODE" = "200" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))")
        if check_json_field "$BODY" "CategoryID" && check_json_field "$BODY" "Count"; then
            return 0
        fi
        return 1
    elif [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        return 1
    fi
}

# Test 4: filter transformation
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=filter(Price%20gt%2010)")
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    fi
    return 1
}

# Test 5: Invalid $apply expression should return 400
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$apply=invalid(syntax)")
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    fi
    echo "  Details: Expected HTTP 400 or 501, got $HTTP_CODE"
    return 1
}

# Run all tests
run_test "\$apply with aggregate (count)" test_1
run_test "\$apply with groupby" test_2
run_test "\$apply with groupby and aggregate" test_3
run_test "\$apply with filter transformation" test_4
run_test "Invalid \$apply expression returns 400" test_5

print_summary
