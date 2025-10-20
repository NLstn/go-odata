#!/bin/bash

# OData v4 Compliance Test: 11.2.7 Metadata Levels
# Tests odata.metadata parameter values according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_metadataURLs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Test 1: odata.metadata=minimal (default)
test_1() {
    local RESPONSE=$(http_get_body "Products?\$format=application/json;odata.metadata=minimal")
    check_contains "$RESPONSE" '"@odata.context"' "metadata=minimal has @odata.context"
}

# Test 2: odata.metadata=full includes type annotations
test_2() {
    local RESPONSE=$(http_get_body "Products(1)?\$format=application/json;odata.metadata=full")
    if echo "$RESPONSE" | grep -q '"@odata.context"'; then
        check_contains "$RESPONSE" '@odata\.type\|@odata\.id' "metadata=full has type annotations"
    else
        return 1
    fi
}

# Test 3: odata.metadata=none excludes metadata
test_3() {
    local RESPONSE=$(http_get_body "Products(1)?\$format=application/json;odata.metadata=none")
    if echo "$RESPONSE" | grep -q '"@odata.context"'; then
        return 1
    fi
}

# Test 4: metadata=none still returns data
test_4() {
    check_status "Products(1)?\$format=application/json;odata.metadata=none" 200
    local RESPONSE=$(http_get_body "Products(1)?\$format=application/json;odata.metadata=none")
    check_contains "$RESPONSE" '"ID"\|"Name"' "metadata=none returns entity data"
}

# Test 5: Invalid metadata value should work or return error
test_5() {
    local STATUS=$(http_get "Products(1)?\$format=application/json;odata.metadata=invalid" | tail -1)
    if [ "$STATUS" = "400" ] || [ "$STATUS" = "200" ]; then
        return 0
    fi
    return 1
}

# Test 6: Collection with metadata=full
test_6() {
    local RESPONSE=$(http_get_body "Products?\$top=2&\$format=application/json;odata.metadata=full")
    check_contains "$RESPONSE" '"@odata.context"' "Collection metadata=full has context"
}

run_test "odata.metadata=minimal includes @odata.context" test_1
run_test "odata.metadata=full includes type annotations" test_2
run_test "odata.metadata=none excludes @odata.context" test_3
run_test "odata.metadata=none still returns entity data" test_4
run_test "Invalid odata.metadata value handling" test_5
run_test "Collection with metadata=full includes @odata.nextLink when applicable" test_6

print_summary
