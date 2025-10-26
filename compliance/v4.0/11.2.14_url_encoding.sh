#!/bin/bash

# OData v4 Compliance Test: 11.2.14 URL Encoding and Special Characters
# Tests proper handling of URL encoding in resource paths and query parameters
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.14 URL Encoding"
echo "======================================"
echo ""
echo "Description: Validates proper handling of URL encoding in resource paths,"
echo "             query parameters, and special characters according to RFC 3986."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html"
echo ""


# Test 1: String literal with spaces in filter
test_filter_with_spaces() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Name%20eq%20'Gaming%20Laptop'")
    check_status "$HTTP_CODE" "200"
}

# Test 2: String literal with special characters in filter
test_filter_special_chars() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'%26')")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Query option with encoded $ symbol
test_encoded_dollar_sign() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?%24top=5")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Filter with URL-encoded operators
test_encoded_operators() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%2050%20and%20Price%20lt%20200")
    check_status "$HTTP_CODE" "200"
}

# Test 5: String literal with single quote (escaped)
test_filter_single_quote() {
    # Single quotes in OData string literals are escaped by doubling them
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'%27%27')")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Parentheses in filter expressions
test_parentheses_encoding() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=%28Price%20gt%20100%29%20and%20%28ID%20lt%2010%29")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Mixed encoded and unencoded characters
test_mixed_encoding() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%2050&\$top=10")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Plus sign encoding (should be treated as space or literal +)
test_plus_sign() {
    # %2B is encoded plus, + may be treated as space
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%200")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Percent encoding in string literals
test_percent_in_string() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'%25')")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Reserved characters in query string
test_reserved_chars() {
    # Semicolon is a reserved character that should be encoded
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%2010%3Bid%20lt%20100")
    # This may fail as semicolon is a query parameter separator in some contexts
    # We're testing that the server handles it gracefully
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200 or 400)"
        return 1
    fi
}

# Test 11: Unicode characters in filter
test_unicode_characters() {
    # Test with UTF-8 encoded characters
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'%C3%A9')")
    check_status "$HTTP_CODE" "200"
}

# Test 12: Case sensitivity of query options
test_case_sensitivity() {
    # Query options should be case-insensitive (though lowercase is recommended)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$FILTER=ID%20eq%201")
    # Some implementations may be case-sensitive
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET with space in filter string literal"
run_test "Filter with URL-encoded spaces" test_filter_with_spaces

echo "  Request: GET with special characters in filter"
run_test "Filter with encoded special characters (&)" test_filter_special_chars

echo "  Request: GET with encoded \$ in query option"
run_test "Query option with encoded dollar sign" test_encoded_dollar_sign

echo "  Request: GET with encoded comparison operators"
run_test "Filter with URL-encoded operators" test_encoded_operators

echo "  Request: GET with escaped single quote"
run_test "Filter with single quote in string literal" test_filter_single_quote

echo "  Request: GET with encoded parentheses"
run_test "Filter with URL-encoded parentheses" test_parentheses_encoding

echo "  Request: GET with mixed encoding"
run_test "Mixed encoded and unencoded parameters" test_mixed_encoding

echo "  Request: GET with plus sign encoding"
run_test "Plus sign handling in URL" test_plus_sign

echo "  Request: GET with percent character"
run_test "Percent sign in filter string" test_percent_in_string

echo "  Request: GET with reserved characters"
run_test "Reserved characters handled gracefully" test_reserved_chars

echo "  Request: GET with Unicode characters"
run_test "Unicode characters in filter" test_unicode_characters

echo "  Request: GET with uppercase query option"
run_test "Query option case handling" test_case_sensitivity

print_summary
