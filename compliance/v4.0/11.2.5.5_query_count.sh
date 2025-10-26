#!/bin/bash

# OData v4 Compliance Test: 11.2.5.5 System Query Option $count
# Tests $count query option according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptioncount

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: $count=true includes @odata.count in response
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$count=true")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$count=true")
    check_json_field "$BODY" "@odata.count"
}

# Test 2: $count=false does not include @odata.count
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$count=false")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$count=false")
    if ! echo "$BODY" | grep -q '"@odata.count"'; then
        return 0
    fi
    echo "  Details: @odata.count should not be present"
    return 1
}

# Test 3: $count with $filter returns filtered count
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$count=true&\$filter=Price%20gt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$count=true&\$filter=Price%20gt%20100")
    if check_json_field "$BODY" "@odata.count"; then
        local COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
        if [ -n "$COUNT" ]; then
            return 0
        fi
        echo "  Details: Count not numeric"
    fi
    return 1
}

# Test 4: $count with $top still returns total count
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$count=true&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$count=true&\$top=1")
    if check_json_field "$BODY" "@odata.count"; then
        local COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
        local ITEMS=$(echo "$BODY" | grep -o '"ID"' | wc -l)
        if [ -n "$COUNT" ] && [ "$COUNT" -ge "$ITEMS" ]; then
            return 0
        fi
        echo "  Details: Count=$COUNT, Items=$ITEMS"
    fi
    return 1
}

# Run all tests
run_test "\$count=true includes @odata.count in response" test_1
run_test "\$count=false excludes @odata.count" test_2
run_test "\$count with \$filter returns filtered count" test_3
run_test "\$count with \$top returns total count, not page count" test_4

print_summary
