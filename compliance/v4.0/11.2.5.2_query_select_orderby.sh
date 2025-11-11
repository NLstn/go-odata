#!/bin/bash

# OData v4 Compliance Test: 11.2.5.2 System Query Option $select and $orderby
# Tests $select and $orderby query options according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic $select with single property
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name")
    
    # Verify Name field is present
    if ! check_json_field "$BODY" "Name"; then
        return 1
    fi
    
    # Verify that ONLY Name field is present (not other fields like Price, Description, etc.)
    # $select should limit the returned properties
    # Check for presence of fields that should NOT be in the response
    if echo "$BODY" | grep -q '"Description"[[:space:]]*:'; then
        echo "  Details: Response contains 'Description' field which was not selected"
        return 1
    fi
    
    return 0
}

# Test 2: $select with multiple properties
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name,Price")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name,Price")
    
    # Verify both Name and Price are present
    if ! check_json_field "$BODY" "Name"; then
        return 1
    fi
    if ! check_json_field "$BODY" "Price"; then
        return 1
    fi
    
    # Verify that fields NOT in $select are not present
    if echo "$BODY" | grep -q '"Description"[[:space:]]*:'; then
        echo "  Details: Response contains 'Description' field which was not selected"
        return 1
    fi
    
    return 0
}

# Test 3: Basic $orderby ascending
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Price%20asc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=Price%20asc")
    
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify the results are actually ordered by Price ascending
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No Price values found in response"
        return 1
    fi
    
    # Check that each price is >= the previous price
    local prev_price=""
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            if [ -n "$prev_price" ]; then
                local IS_ORDERED=$(echo "$prev_price $price" | awk '{if ($1 <= $2) print "yes"; else print "no"}')
                if [ "$IS_ORDERED" != "yes" ]; then
                    echo "  Details: Results not ordered ascending: found $price after $prev_price"
                    return 1
                fi
            fi
            prev_price="$price"
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 4: $orderby descending
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Price%20desc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=Price%20desc")
    
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify the results are actually ordered by Price descending
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No Price values found in response"
        return 1
    fi
    
    # Check that each price is <= the previous price
    local prev_price=""
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            if [ -n "$prev_price" ]; then
                local IS_ORDERED=$(echo "$prev_price $price" | awk '{if ($1 >= $2) print "yes"; else print "no"}')
                if [ "$IS_ORDERED" != "yes" ]; then
                    echo "  Details: Results not ordered descending: found $price after $prev_price"
                    return 1
                fi
            fi
            prev_price="$price"
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 5: $orderby with multiple properties
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=CategoryID,Price%20desc")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$orderby=CategoryID,Price%20desc")
    check_json_field "$BODY" "value"
}

# Test 6: Combining $select and $orderby
test_6() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$select=Name,Price&\$orderby=Price")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$select=Name,Price&\$orderby=Price")
    
    # Verify selected fields are present
    if ! check_json_field "$BODY" "Name"; then
        return 1
    fi
    if ! check_json_field "$BODY" "Price"; then
        return 1
    fi
    
    # Verify ordering
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No Price values found in response"
        return 1
    fi
    
    # Check that prices are ordered ascending (default when direction not specified)
    local prev_price=""
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            if [ -n "$prev_price" ]; then
                local IS_ORDERED=$(echo "$prev_price $price" | awk '{if ($1 <= $2) print "yes"; else print "no"}')
                if [ "$IS_ORDERED" != "yes" ]; then
                    echo "  Details: Results not ordered: found $price after $prev_price"
                    return 1
                fi
            fi
            prev_price="$price"
        fi
    done <<< "$PRICES"
    
    return 0
}

# Run all tests
run_test "\$select with single property" test_1
run_test "\$select with multiple properties" test_2
run_test "\$orderby ascending" test_3
run_test "\$orderby descending" test_4
run_test "\$orderby with multiple properties" test_5
run_test "Combining \$select and \$orderby" test_6

print_summary
