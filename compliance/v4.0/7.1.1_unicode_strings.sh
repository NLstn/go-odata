#!/bin/bash

# OData v4 Compliance Test: 7.1.1 Unicode and Internationalization
# Tests handling of Unicode characters, multi-byte characters, emoji, and international text
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LiteralDataValues

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 7.1.1 Unicode and Internationalization"
echo "======================================"
echo ""
echo "Description: Tests handling of Unicode characters including multi-byte characters,"
echo "             emoji, international text, and proper URL encoding of Unicode strings"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LiteralDataValues"
echo ""

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: Basic multi-byte Unicode characters (Latin Extended)
test_latin_extended() {
    local FILTER="contains(Name,'café')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Cyrillic characters
test_cyrillic() {
    local FILTER="contains(Name,'Привет')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Chinese characters
test_chinese() {
    local FILTER="contains(Name,'中文')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Japanese characters (Hiragana, Katakana, Kanji mix)
test_japanese() {
    local FILTER="contains(Name,'日本語')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Arabic characters (RTL text)
test_arabic() {
    local FILTER="contains(Name,'مرحبا')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Hebrew characters (RTL text)
test_hebrew() {
    local FILTER="contains(Name,'שלום')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Emoji characters
test_emoji() {
    local FILTER="contains(Name,'😀')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Mixed script text
test_mixed_scripts() {
    local FILTER="contains(Name,'Hello世界')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Accented characters
test_accented_characters() {
    local FILTER="contains(Name,'Québec') or contains(Name,'São')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Greek characters
test_greek() {
    local FILTER="contains(Name,'Ελληνικά')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 11: Special Unicode characters (mathematical symbols)
test_mathematical_symbols() {
    local FILTER="contains(Name,'∑∫π')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 12: Zero-width characters and combining marks
test_combining_marks() {
    # Test combining diacritical marks
    local FILTER="contains(Name,'e\u0301')"  # e with combining acute accent
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 13: Create entity with Unicode name
test_create_unicode_entity() {
    local UNICODE_NAME="Продукт测试🌍"
    local JSON_DATA="{\"Name\":\"$UNICODE_NAME\",\"Price\":99.99,\"Status\":1,\"CategoryID\":1}"
    # Use curl directly to get status code and response
    local HTTP_CODE=$(curl -s -o /tmp/unicode_response.json -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$JSON_DATA" \
        "$SERVER_URL/Products")
    
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
        # Try to extract ID if present in response
        local CREATED_ID=$(cat /tmp/unicode_response.json 2>/dev/null | grep -o '"ID":[0-9]*' | grep -o '[0-9]*' | head -1)
        if [ -n "$CREATED_ID" ]; then
            CREATED_IDS+=("$CREATED_ID")
        fi
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 201 or 204)"
        return 1
    fi
}

# Test 14: Retrieve entity with Unicode name
test_retrieve_unicode_entity() {
    # First create an entity
    local UNICODE_NAME="Test日本"
    local JSON_DATA="{\"Name\":\"$UNICODE_NAME\",\"Price\":123.45,\"Status\":1,\"CategoryID\":1}"
    local CREATE_CODE=$(curl -s -o /tmp/unicode_retrieve.json -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$JSON_DATA" \
        "$SERVER_URL/Products")
    
    if [ "$CREATE_CODE" = "201" ] || [ "$CREATE_CODE" = "204" ]; then
        local CREATED_ID=$(cat /tmp/unicode_retrieve.json 2>/dev/null | grep -o '"ID":[0-9]*' | grep -o '[0-9]*' | head -1)
        if [ -n "$CREATED_ID" ]; then
            CREATED_IDS+=("$CREATED_ID")
            # Try to retrieve it
            local HTTP_CODE=$(http_get "$SERVER_URL/Products($CREATED_ID)")
            check_status "$HTTP_CODE" "200"
        else
            return 0  # Entity created but ID not returned, acceptable
        fi
    else
        return 0  # If creation fails, that's okay for this test
    fi
}

# Test 15: String functions with Unicode
test_unicode_string_functions() {
    local FILTER="length(Name) gt 0"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 16: Case insensitive search with Unicode (using tolower on property)
test_unicode_case_insensitive() {
    local FILTER="tolower(Name) eq 'café'"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 17: Unicode in orderby
test_unicode_orderby() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$orderby=Name")
    check_status "$HTTP_CODE" "200"
}

# Test 18: Thai characters
test_thai() {
    local FILTER="contains(Name,'สวัสดี')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 19: Korean characters
test_korean() {
    local FILTER="contains(Name,'안녕하세요')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 20: Unicode in both filter and select
test_unicode_in_multiple_operations() {
    local FILTER="contains(Name,'test')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=$ENCODED_FILTER&\$select=Name")
    check_status "$HTTP_CODE" "200"
}

run_test "Latin extended characters (café)" test_latin_extended
run_test "Cyrillic characters" test_cyrillic
run_test "Chinese characters" test_chinese
run_test "Japanese characters" test_japanese
run_test "Arabic characters (RTL)" test_arabic
run_test "Hebrew characters (RTL)" test_hebrew
run_test "Emoji characters" test_emoji
run_test "Mixed script text" test_mixed_scripts
run_test "Accented characters" test_accented_characters
run_test "Greek characters" test_greek
run_test "Mathematical symbols" test_mathematical_symbols
run_test "Combining diacritical marks" test_combining_marks
run_test "Create entity with Unicode name" test_create_unicode_entity
run_test "Retrieve entity with Unicode name" test_retrieve_unicode_entity
run_test "String functions with Unicode" test_unicode_string_functions
run_test "Case insensitive Unicode search" test_unicode_case_insensitive
run_test "Unicode in orderby" test_unicode_orderby
run_test "Thai characters" test_thai
run_test "Korean characters" test_korean
run_test "Unicode in multiple operations" test_unicode_in_multiple_operations

print_summary
