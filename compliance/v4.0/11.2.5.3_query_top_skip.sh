#!/bin/bash

# OData v4 Compliance Test: 11.2.5.3 and 11.2.5.4 System Query Options $top and $skip
# Tests $top and $skip query options for paging according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionstopandskip

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: $top limits the number of items returned
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=2")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$top=2")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    
    # $top=2 should return EXACTLY 2 items (assuming at least 2 exist in the collection)
    # But we need to be lenient if there are fewer than 2 items total
    if [ "$COUNT" -gt 2 ]; then
        echo "  Details: Returned $COUNT items, expected max 2"
        return 1
    fi
    
    # Verify we got at least 1 item (assuming Products collection is not empty)
    if [ "$COUNT" -lt 1 ]; then
        echo "  Details: Returned $COUNT items, expected at least 1"
        return 1
    fi
    
    return 0
}

# Test 2: $skip skips the specified number of items
test_2() {
    # First get all products to know what to expect
    local ALL_BODY=$(http_get_body "$SERVER_URL/Products")
    local ALL_IDS=$(echo "$ALL_BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$' | head -3)
    
    # Get the second ID from the full list
    local SECOND_ID=$(echo "$ALL_IDS" | sed -n '2p')
    
    if [ -z "$SECOND_ID" ]; then
        echo "  Details: Not enough products to test $skip"
        return 0  # Pass if there aren't enough products
    fi
    
    # Now test $skip=1&$top=1 - should return the second item
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$skip=1&\$top=1")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local BODY=$(http_get_body "$SERVER_URL/Products?\$skip=1&\$top=1")
    
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify we got exactly 1 item
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -ne 1 ]; then
        echo "  Details: Expected exactly 1 item, got $COUNT"
        return 1
    fi
    
    # Verify it's the second item (ID should match)
    local RETURNED_ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$' | head -1)
    if [ "$RETURNED_ID" != "$SECOND_ID" ]; then
        echo "  Details: Expected ID=$SECOND_ID (2nd item), got ID=$RETURNED_ID"
        return 1
    fi
    
    return 0
}

# Test 3: $top=0 returns empty collection
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=0")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$top=0")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -eq 0 ]; then
        return 0
    fi
    echo "  Details: Returned $COUNT items"
    return 1
}

# Test 4: Combine $skip and $top for paging
test_4() {
    # First get all products to understand the collection
    local ALL_BODY=$(http_get_body "$SERVER_URL/Products")
    local TOTAL_COUNT=$(echo "$ALL_BODY" | grep -o '"ID"' | wc -l)
    
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$skip=2&\$top=3")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$skip=2&\$top=3")
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    
    # Should return at most 3 items
    if [ "$COUNT" -gt 3 ]; then
        echo "  Details: Returned $COUNT items, expected max 3"
        return 1
    fi
    
    # Calculate expected count: min(3, total - 2)
    local EXPECTED_MAX=$((TOTAL_COUNT - 2))
    if [ "$EXPECTED_MAX" -gt 3 ]; then
        EXPECTED_MAX=3
    fi
    if [ "$EXPECTED_MAX" -lt 0 ]; then
        EXPECTED_MAX=0
    fi
    
    # Verify count is reasonable
    if [ "$COUNT" -gt "$EXPECTED_MAX" ]; then
        echo "  Details: Returned $COUNT items, expected max $EXPECTED_MAX (total=$TOTAL_COUNT, skip=2, top=3)"
        return 1
    fi
    
    return 0
}

# Run all tests
run_test "\$top limits number of items" test_1
run_test "\$skip skips items" test_2
run_test "\$top=0 returns empty collection" test_3
run_test "Combine \$skip and \$top for paging" test_4

print_summary
