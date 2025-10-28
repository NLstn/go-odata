#!/bin/bash

# OData v4 Compliance Test: 11.2.5.8 System Query Option $compute
# Tests $compute system query option for computed properties
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptioncompute

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.8 System Query Option \$compute"
echo "======================================"
echo ""
echo "Description: Validates \$compute query option for adding computed properties"
echo "             to query results according to OData v4.01 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptioncompute"
echo ""

# Note: $compute is a relatively new addition in OData v4.01
# Many implementations may not support it initially

# Test 1: Simple $compute with arithmetic
test_compute_arithmetic() {
    # Compute a new property based on arithmetic expression
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.1 as PriceWithTax")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute arithmetic request (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 2: $compute with string function
test_compute_string_function() {
    # Compute using string functions
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=toupper(Name) as UpperName")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute string function request (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: $compute with $select
test_compute_with_select() {
    # Combine $compute with $select
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 2 as DoublePrice&\$select=Name,DoublePrice")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute with \$select (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: $compute with $filter
test_compute_with_filter() {
    # Use computed property in $filter
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.1 as PriceWithTax&\$filter=PriceWithTax gt 100")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute with \$filter (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 5: $compute with $orderby
test_compute_with_orderby() {
    # Order by computed property
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price div 2 as HalfPrice&\$orderby=HalfPrice")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute with \$orderby (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Multiple computed properties
test_multiple_computed() {
    # Define multiple computed properties
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Price mul 1.1 as WithTax,Price mul 0.9 as Discounted")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected multiple \$compute properties (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: $compute with date functions
test_compute_date_functions() {
    # Compute using date functions
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=year(CreatedAt) as CreatedYear")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute with date functions (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Invalid $compute syntax
test_invalid_compute_syntax() {
    # Invalid syntax should return 400
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=InvalidSyntax")
    
    # Should return 400 when syntax is invalid
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        echo "  Failure: Invalid syntax accepted"
        return 1
    elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service did not recognize \$compute invalid syntax request (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 9: $compute with nested properties
test_compute_nested_properties() {
    # Compute based on nested property
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$compute=Address/City as Location")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute with nested properties (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: $compute in $expand
test_compute_in_expand() {
    # Use $compute within $expand (advanced)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Category(\$compute=ID mul 2 as DoubleID)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Failure: Service rejected \$compute in \$expand (status: $HTTP_CODE)"
        return 1
    else
        echo "  Failure: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET \$compute=Price mul 1.1 as PriceWithTax"
run_test "Simple \$compute with arithmetic (OData v4.01)" test_compute_arithmetic

echo "  Request: GET \$compute=toupper(Name) as UpperName"
run_test "\$compute with string function" test_compute_string_function

echo "  Request: GET \$compute with \$select"
run_test "\$compute combined with \$select" test_compute_with_select

echo "  Request: GET \$compute with \$filter"
run_test "\$compute combined with \$filter" test_compute_with_filter

echo "  Request: GET \$compute with \$orderby"
run_test "\$compute combined with \$orderby" test_compute_with_orderby

echo "  Request: GET \$compute with multiple properties"
run_test "Multiple computed properties" test_multiple_computed

echo "  Request: GET \$compute with date functions"
run_test "\$compute with date functions" test_compute_date_functions

echo "  Request: GET \$compute with invalid syntax"
run_test "Invalid \$compute syntax returns error" test_invalid_compute_syntax

echo "  Request: GET \$compute with nested properties"
run_test "\$compute with nested properties" test_compute_nested_properties

echo "  Request: GET \$compute in \$expand"
run_test "\$compute within \$expand (advanced)" test_compute_in_expand

print_summary
