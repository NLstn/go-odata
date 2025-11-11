#!/bin/bash

# OData v4 Compliance Test: 11.2.5.1 System Query Option $filter
# Tests $filter query option according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionfilter

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Basic eq (equals) operator
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ID%20eq%201")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=ID%20eq%201")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify the filter actually worked - should return exactly 1 entity with ID=1
    local COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -ne 1 ]; then
        echo "  Details: Expected 1 entity, got $COUNT entities"
        return 1
    fi
    
    # Verify the returned entity has ID=1
    if ! echo "$BODY" | grep -q '"ID"[[:space:]]*:[[:space:]]*1[^0-9]'; then
        echo "  Details: Returned entity does not have ID=1"
        return 1
    fi
    
    return 0
}

# Test 2: gt (greater than) operator
test_2() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%20100")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%20100")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify all returned entities have Price > 100
    # Extract all Price values and check they are > 100
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    # Check each price is greater than 100
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            # Use awk for floating point comparison
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price which is not > 100"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 3: String contains function
test_3() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify all returned entities have "Laptop" in their Name
    # Extract the value array and check each Name field
    local NAMES=$(echo "$BODY" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    # Check each name contains "Laptop"
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            if ! echo "$name" | grep -q "Laptop"; then
                echo "  Details: Found entity with Name='$name' which does not contain 'Laptop'"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 4: Boolean operators (and)
test_4() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%201000")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%201000")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify all returned entities have 10 < Price < 1000
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    if [ -z "$PRICES" ]; then
        echo "  Details: No entities returned or no Price field found"
        return 1
    fi
    
    # Check each price is between 10 and 1000
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 10 && $1 < 1000) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price which is not in range (10, 1000)"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Test 5: Boolean operators (or)
test_5() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ID%20eq%201%20or%20ID%20eq%202")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=ID%20eq%201%20or%20ID%20eq%202")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify all returned entities have ID=1 or ID=2
    local IDS=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    
    if [ -z "$IDS" ]; then
        echo "  Details: No entities returned or no ID field found"
        return 1
    fi
    
    # Check each ID is either 1 or 2
    while IFS= read -r id; do
        if [ -n "$id" ]; then
            if [ "$id" != "1" ] && [ "$id" != "2" ]; then
                echo "  Details: Found entity with ID=$id which is not 1 or 2"
                return 1
            fi
        fi
    done <<< "$IDS"
    
    # Also verify we got at least 1 entity (could be 1 or 2 entities total)
    local COUNT=$(echo "$IDS" | grep -c .)
    if [ "$COUNT" -lt 1 ]; then
        echo "  Details: Expected at least 1 entity, got $COUNT"
        return 1
    fi
    
    return 0
}

# Test 6: Parentheses for grouping
test_6() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=(Price%20gt%20100)%20and%20(ID%20lt%2010)")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    local BODY=$(http_get_body "$SERVER_URL/Products?\$filter=(Price%20gt%20100)%20and%20(ID%20lt%2010)")
    
    # Verify response structure
    if ! check_json_field "$BODY" "value"; then
        return 1
    fi
    
    # Verify all returned entities have Price > 100 AND ID < 10
    # First check IDs
    local IDS=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
    local PRICES=$(echo "$BODY" | grep -o '"Price"[[:space:]]*:[[:space:]]*[0-9.]*' | grep -o '[0-9.]*$')
    
    # If no results, that's valid (no products match the criteria)
    if [ -z "$IDS" ]; then
        return 0
    fi
    
    # Check each ID < 10
    while IFS= read -r id; do
        if [ -n "$id" ]; then
            if [ "$id" -ge 10 ]; then
                echo "  Details: Found entity with ID=$id which is not < 10"
                return 1
            fi
        fi
    done <<< "$IDS"
    
    # Check each Price > 100
    while IFS= read -r price; do
        if [ -n "$price" ]; then
            local IS_VALID=$(echo "$price" | awk '{if ($1 > 100) print "yes"; else print "no"}')
            if [ "$IS_VALID" != "yes" ]; then
                echo "  Details: Found entity with Price=$price which is not > 100"
                return 1
            fi
        fi
    done <<< "$PRICES"
    
    return 0
}

# Run all tests
run_test "\$filter with eq operator" test_1
run_test "\$filter with gt operator" test_2
run_test "\$filter with contains() function" test_3
run_test "\$filter with 'and' operator" test_4
run_test "\$filter with 'or' operator" test_5
run_test "\$filter with parentheses" test_6

print_summary
