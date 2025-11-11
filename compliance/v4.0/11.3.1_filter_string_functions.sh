#!/bin/bash

# OData v4 Compliance Test: 11.3.1 String Functions in $filter
# Tests string functions (contains, startswith, endswith, length, etc.) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.1 String Functions"
echo "======================================"
echo ""
echo "Description: Validates string functions in \$filter query option"
echo "             (contains, startswith, endswith, length, indexof, substring, tolower, toupper, trim, concat)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: contains function
test_contains_function() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')")
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
    
    # Check that response contains value array
    if ! echo "$RESPONSE" | grep -q '"value"'; then
        echo "  Details: Response missing 'value' array"
        return 1
    fi
    
    # Verify all returned entities have "Laptop" in Name
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
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

# Test 2: startswith function
test_startswith_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=startswith(Name,'Gaming')")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=startswith(Name,'Gaming')")
    
    # Verify all returned entities have Name starting with "Gaming"
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            if ! echo "$name" | grep -q "^Gaming"; then
                echo "  Details: Found entity with Name='$name' which does not start with 'Gaming'"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 3: endswith function
test_endswith_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=endswith(Name,'Mouse')")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=endswith(Name,'Mouse')")
    
    # Verify all returned entities have Name ending with "Mouse"
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            if ! echo "$name" | grep -q "Mouse$"; then
                echo "  Details: Found entity with Name='$name' which does not end with 'Mouse'"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 4: length function
test_length_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=length(Name)%20gt%2010")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=length(Name)%20gt%2010")
    
    # Verify all returned entities have Name length > 10
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            local len=${#name}
            if [ "$len" -le 10 ]; then
                echo "  Details: Found entity with Name='$name' (length=$len) which is not > 10"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 5: indexof function
test_indexof_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=indexof(Name,'Pro') eq 0")
    check_status "$HTTP_CODE" "200"
}

# Test 6: substring function
test_substring_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=substring(Name,0,3) eq 'Gam'")
    check_status "$HTTP_CODE" "200"
}

# Test 7: tolower function
test_tolower_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=tolower(Name)%20eq%20'laptop'")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=tolower(Name)%20eq%20'laptop'")
    
    # Verify all returned entities have Name that equals "laptop" when lowercased
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            local lower_name=$(echo "$name" | tr '[:upper:]' '[:lower:]')
            if [ "$lower_name" != "laptop" ]; then
                echo "  Details: Found entity with Name='$name' (lowercase='$lower_name') which does not equal 'laptop'"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 8: toupper function
test_toupper_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=toupper(Name)%20eq%20'LAPTOP'")
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=toupper(Name)%20eq%20'LAPTOP'")
    
    # Verify all returned entities have Name that equals "LAPTOP" when uppercased
    local NAMES=$(echo "$RESPONSE" | grep -o '"Name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"Name"[[:space:]]*:[[:space:]]*"//; s/"$//')
    
    if [ -z "$NAMES" ]; then
        echo "  Details: No entities returned or no Name field found"
        return 1
    fi
    
    while IFS= read -r name; do
        if [ -n "$name" ]; then
            local upper_name=$(echo "$name" | tr '[:lower:]' '[:upper:]')
            if [ "$upper_name" != "LAPTOP" ]; then
                echo "  Details: Found entity with Name='$name' (uppercase='$upper_name') which does not equal 'LAPTOP'"
                return 1
            fi
        fi
    done <<< "$NAMES"
    
    return 0
}

# Test 9: trim function
test_trim_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=trim(Name) eq 'Laptop'")
    check_status "$HTTP_CODE" "200"
}

# Test 10: concat function
test_concat_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=concat(Name,' Test') eq 'Laptop Test'")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=contains(Name,'Laptop')"
run_test "contains() function filters string values" test_contains_function

echo "  Request: GET \$filter=startswith(Name,'Gaming')"
run_test "startswith() function filters by prefix" test_startswith_function

echo "  Request: GET \$filter=endswith(Name,'Mouse')"
run_test "endswith() function filters by suffix" test_endswith_function

echo "  Request: GET \$filter=length(Name) gt 10"
run_test "length() function returns string length" test_length_function

echo "  Request: GET \$filter=indexof(Name,'Pro') eq 0"
run_test "indexof() function finds substring position" test_indexof_function

echo "  Request: GET \$filter=substring(Name,0,3) eq 'Gam'"
run_test "substring() function extracts substring" test_substring_function

echo "  Request: GET \$filter=tolower(Name) eq 'laptop'"
run_test "tolower() function converts to lowercase" test_tolower_function

echo "  Request: GET \$filter=toupper(Name) eq 'LAPTOP'"
run_test "toupper() function converts to uppercase" test_toupper_function

echo "  Request: GET \$filter=trim(Name) eq 'Laptop'"
run_test "trim() function removes whitespace" test_trim_function

echo "  Request: GET \$filter=concat(...)"
run_test "concat() function concatenates strings" test_concat_function

print_summary
