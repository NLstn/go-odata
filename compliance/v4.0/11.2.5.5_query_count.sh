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
    
    if ! check_json_field "$BODY" "@odata.count"; then
        return 1
    fi
    
    # Extract the count and verify it matches the number of items in the response
    local COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    local ITEMS=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    
    if [ -z "$COUNT" ]; then
        echo "  Details: Count not numeric"
        return 1
    fi
    
    # The count should equal the number of items returned (when no $top is used)
    if [ "$COUNT" -ne "$ITEMS" ]; then
        echo "  Details: Count=$COUNT but response contains $ITEMS items"
        return 1
    fi
    
    # Verify all items actually match the filter (Price > 100)
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found item with Price=$price which is not > 100"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 4: $count with $top still returns total count
test_4() {
    # First get total count without $top
    local TOTAL_BODY=$(http_get_body "$SERVER_URL/Products?\$count=true")
    local TOTAL_COUNT=$(echo "$TOTAL_BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    
    # Now get with $top=1
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$count=true&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$count=true&\$top=1")
    
    if ! check_json_field "$BODY" "@odata.count"; then
        return 1
    fi
    
    local COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    local ITEMS=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    
    # The @odata.count should be the TOTAL count, not the page count
    if [ "$COUNT" != "$TOTAL_COUNT" ]; then
        echo "  Details: Count=$COUNT but total count is $TOTAL_COUNT (should match total, not page size)"
        return 1
    fi
    
    # The number of items in the response should be 1 (due to $top=1)
    if [ "$ITEMS" -ne 1 ]; then
        echo "  Details: Expected 1 item in response (due to \$top=1), got $ITEMS"
        return 1
    fi
    
    # The count should be greater than the items (assuming there are multiple products)
    if [ "$COUNT" -lt "$ITEMS" ]; then
        echo "  Details: Count=$COUNT should be >= Items=$ITEMS"
        return 1
    fi
    
    return 0
}

# Run all tests
run_test "\$count=true includes @odata.count in response" test_1
run_test "\$count=false excludes @odata.count" test_2
run_test "\$count with \$filter returns filtered count" test_3
run_test "\$count with \$top returns total count, not page count" test_4

print_summary
