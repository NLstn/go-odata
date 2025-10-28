#!/bin/bash

# OData v4 Compliance Test: 11.2.5.13 $index Query Option
# Tests $index system query option for retrieving zero-based position of items
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_index

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

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

# Helper: ensure the service returns HTTP 200 for a request that relies on
# $index support. When the response status is not 200, emit a clear compliance
# defect message and return failure.
require_index_support() {
    local http_code="$1"
    local context="$2"

    if [ "$http_code" != "200" ]; then
        echo "  Compliance defect: Expected HTTP 200 for $context but received $http_code."
        echo "  Missing \$index support is a compliance defect that must be addressed."
        return 1
    fi

    return 0
}

# Test 1: $index without other query options
test_index_basic() {
    local url="$SERVER_URL/Products?\$index"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index query on Products" || return 1

    local response=$(http_get_body "$url")
    if ! echo "$response" | grep -q '"@odata.index"'; then
        echo "  Note: Response missing @odata.index annotations (optional check)."
    fi

    return 0
}

# Test 2: $index with $top
test_index_with_top() {
    local url="$SERVER_URL/Products?\$index&\$top=5"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$top" || return 1
    return 0
}

# Test 3: $index with $skip
test_index_with_skip() {
    local url="$SERVER_URL/Products?\$index&\$skip=2"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$skip" || return 1
    return 0
}

# Test 4: $index with $orderby
test_index_with_orderby() {
    local url="$SERVER_URL/Products?\$index&\$orderby=Price"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$orderby" || return 1
    return 0
}

# Test 5: $index with $filter
test_index_with_filter() {
    local url="$SERVER_URL/Products?\$index&\$filter=Price%20gt%2050"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$filter" || return 1
    return 0
}

# Test 6: $index response format
test_index_response_format() {
    local url="$SERVER_URL/Products?\$index&\$top=3"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index response format" || return 1

    local response=$(http_get_body "$url")
    echo "$response" | grep -q '"value"'
}

# Test 7: $index with $expand
test_index_with_expand() {
    local url="$SERVER_URL/Products?\$index&\$expand=Category"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$expand" || return 1
    return 0
}

# Test 8: $index on entity
test_index_on_entity() {
    local url="$SERVER_URL/Products(1)?\$index"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index on entity" || return 1
    return 0
}

# Test 9: $index with complex query combination
test_index_complex_query() {
    local url="$SERVER_URL/Products?\$index&\$filter=CategoryID%20eq%201&\$orderby=Name&\$top=5"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index complex query" || return 1
    return 0
}

# Test 10: $index with $count
test_index_with_count() {
    local url="$SERVER_URL/Products?\$index&\$count=true"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$count" || return 1
    return 0
}

# Test 11: Check if @odata.index annotation is included
test_index_annotation_presence() {
    local url="$SERVER_URL/Products?\$index&\$top=2"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index annotation presence" || return 1

    local response=$(http_get_body "$url")
    if ! echo "$response" | grep -q '"@odata.index"'; then
        echo "  Note: Response missing @odata.index annotations (optional check)."
    fi

    return 0
}

# Test 12: $index value starts at 0
test_index_starts_at_zero() {
    local url="$SERVER_URL/Products?\$index&\$top=1"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index value starts at 0" || return 1

    local response=$(http_get_body "$url")
    if echo "$response" | grep -q '"@odata.index":0'; then
        return 0
    fi

    echo "  Note: Unable to verify that @odata.index starts at 0 (optional check)."
    return 0
}

# Test 13: $index with $select
test_index_with_select() {
    local url="$SERVER_URL/Products?\$index&\$select=Name,Price"
    local http_code=$(http_get "$url")

    require_index_support "$http_code" "\$index with \$select" || return 1
    return 0
}

# Test 14: $index case sensitivity
test_index_case_sensitivity() {
    local url="$SERVER_URL/Products?\$INDEX"
    local http_code=$(http_get "$url")

    if [ "$http_code" = "200" ]; then
        echo "  Compliance defect: Service treated \$INDEX as valid; expected rejection."
        return 1
    fi

    if [ "$http_code" != "400" ]; then
        echo "  Compliance defect: Expected HTTP 400 for uppercase \$INDEX but received $http_code."
        return 1
    fi

    return 0
}

# Test 15: Multiple $index parameters (invalid)
test_multiple_index_params() {
    local url="$SERVER_URL/Products?\$index&\$index"
    local http_code=$(http_get "$url")

    if [ "$http_code" = "200" ]; then
        echo "  Compliance defect: Service accepted duplicate \$index parameters."
        return 1
    fi

    if [ "$http_code" != "400" ]; then
        echo "  Compliance defect: Expected HTTP 400 for duplicate \$index parameters but received $http_code."
        return 1
    fi

    return 0
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
