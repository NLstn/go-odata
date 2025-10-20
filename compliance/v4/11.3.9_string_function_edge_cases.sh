#!/bin/bash

# OData v4 Compliance Test: 11.3.9 String Function Edge Cases
# Tests edge cases and boundary conditions for string functions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.9 String Function Edge Cases"
echo "======================================"
echo ""
echo "Description: Validates edge cases and boundary conditions for string"
echo "             functions in \$filter expressions."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: contains() with empty string
test_contains_empty_string() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'')")
    
    # Should return 200 - empty string is contained in all strings
    check_status "$HTTP_CODE" "200"
}

# Test 2: startswith() with empty string
test_startswith_empty_string() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=startswith(Name,'')")
    
    # Should return 200 - all strings start with empty string
    check_status "$HTTP_CODE" "200"
}

# Test 3: endswith() with empty string
test_endswith_empty_string() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=endswith(Name,'')")
    
    # Should return 200 - all strings end with empty string
    check_status "$HTTP_CODE" "200"
}

# Test 4: length() of empty string
test_length_empty_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=length(Category) eq 0")
    
    check_status "$HTTP_CODE" "200"
}

# Test 5: substring() with start beyond length
test_substring_beyond_length() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=substring(Name,100) eq ''")
    
    # Implementation dependent - may return 200 or 400
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 6: substring() with negative start (invalid)
test_substring_negative_start() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=substring(Name,-1) eq ''")
    
    # Should return 400 for invalid parameter
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400)"
        return 1
    fi
}

# Test 7: substring() with length of 0
test_substring_zero_length() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=substring(Name,0,0) eq ''")
    
    check_status "$HTTP_CODE" "200"
}

# Test 8: indexof() with substring not found returns -1
test_indexof_not_found() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=indexof(Name,'ZZZZZ') eq -1")
    
    check_status "$HTTP_CODE" "200"
}

# Test 9: indexof() with empty string returns 0
test_indexof_empty_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=indexof(Name,'') eq 0")
    
    check_status "$HTTP_CODE" "200"
}

# Test 10: tolower() on already lowercase string
test_tolower_lowercase() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=tolower(Name) eq tolower(Name)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 11: toupper() on already uppercase string
test_toupper_uppercase() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=toupper(Category) eq toupper(Category)")
    
    check_status "$HTTP_CODE" "200"
}

# Test 12: trim() on string without whitespace
test_trim_no_whitespace() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=trim(Name) eq Name")
    
    check_status "$HTTP_CODE" "200"
}

# Test 13: concat() with empty strings
test_concat_empty_strings() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=concat('','') eq ''")
    
    check_status "$HTTP_CODE" "200"
}

# Test 14: concat() with multiple arguments
test_concat_multiple() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=concat(concat(Name,' - '),Category) ne ''")
    
    check_status "$HTTP_CODE" "200"
}

# Test 15: Case sensitivity of contains()
test_contains_case_sensitive() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(tolower(Name),'laptop')")
    
    check_status "$HTTP_CODE" "200"
}

# Test 16: Nested string functions
test_nested_string_functions() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=length(trim(toupper(Name))) gt 0")
    
    check_status "$HTTP_CODE" "200"
}

# Test 17: String function on null property
test_string_function_on_null() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=length(Description) eq null")
    
    # Implementation dependent - null handling varies
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 18: Very long string in filter
test_very_long_string() {
    local LONG_STRING="$(printf 'A%.0s' {1..1000})"
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'$LONG_STRING')")
    
    check_status "$HTTP_CODE" "200"
}

# Test 19: Special regex characters in string functions
test_special_regex_chars() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'.*+?[]')")
    
    # Should treat as literal string, not regex
    check_status "$HTTP_CODE" "200"
}

# Test 20: Unicode in string functions
test_unicode_in_string_functions() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'café')")
    
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET with contains(Name,'')"
run_test "contains() with empty string" test_contains_empty_string

echo "  Request: GET with startswith(Name,'')"
run_test "startswith() with empty string" test_startswith_empty_string

echo "  Request: GET with endswith(Name,'')"
run_test "endswith() with empty string" test_endswith_empty_string

echo "  Request: GET with length(Category) eq 0"
run_test "length() of empty string" test_length_empty_string

echo "  Request: GET with substring(Name,100)"
run_test "substring() beyond string length" test_substring_beyond_length

echo "  Request: GET with substring(Name,-1)"
run_test "substring() with negative start returns 400" test_substring_negative_start

echo "  Request: GET with substring(Name,0,0)"
run_test "substring() with length of 0" test_substring_zero_length

echo "  Request: GET with indexof(Name,'ZZZZZ') eq -1"
run_test "indexof() not found returns -1" test_indexof_not_found

echo "  Request: GET with indexof(Name,'') eq 0"
run_test "indexof() with empty string" test_indexof_empty_string

echo "  Request: GET with tolower(Name) eq tolower(Name)"
run_test "tolower() on lowercase string" test_tolower_lowercase

echo "  Request: GET with toupper(Category) eq toupper(Category)"
run_test "toupper() on uppercase string" test_toupper_uppercase

echo "  Request: GET with trim(Name) eq Name"
run_test "trim() on string without whitespace" test_trim_no_whitespace

echo "  Request: GET with concat('','')"
run_test "concat() with empty strings" test_concat_empty_strings

echo "  Request: GET with nested concat()"
run_test "concat() with multiple arguments" test_concat_multiple

echo "  Request: GET with contains(tolower(Name),'laptop')"
run_test "Case-insensitive contains()" test_contains_case_sensitive

echo "  Request: GET with length(trim(toupper(Name)))"
run_test "Nested string functions" test_nested_string_functions

echo "  Request: GET with length(Description) eq null"
run_test "String function on null property" test_string_function_on_null

echo "  Request: GET with very long string"
run_test "Very long string in filter" test_very_long_string

echo "  Request: GET with special regex characters"
run_test "Special characters treated as literals" test_special_regex_chars

echo "  Request: GET with Unicode characters"
run_test "Unicode in string functions" test_unicode_in_string_functions

print_summary
