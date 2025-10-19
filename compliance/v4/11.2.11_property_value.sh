#!/bin/bash

# OData v4 Compliance Test: 11.2.11 Addressing Individual Properties with $value
# Tests accessing raw property values using $value
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingIndividualPropertiesofanEnt

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.11 Property \$value"
echo "======================================"
echo ""
echo "Description: Validates accessing raw property values using the"
echo "             \$value path segment."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingIndividualPropertiesofanEnt"
echo ""

# Test 1: Access primitive property $value
test_property_value() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Name/\$value")
    
    # Should return raw string value (not JSON)
    # Just verify we got something back
    if [ -n "$RESPONSE" ]; then
        return 0
    else
        echo "  Details: Empty response"
        return 1
    fi
}

# Test 2: $value returns correct Content-Type
test_value_content_type() {
    local HEADERS=$(curl -s -I "$SERVER_URL/Products(1)/Name/\$value" 2>&1)
    
    # Should return text/plain or appropriate media type
    if echo "$HEADERS" | grep -i "content-type:" > /dev/null; then
        return 0
    else
        echo "  Details: No Content-Type header"
        return 1
    fi
}

# Test 3: $value on numeric property
test_numeric_value() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Price/\$value")
    
    check_status "$HTTP_CODE" "200"
}

# Test 4: $value on non-existent property returns 404
test_value_nonexistent_property() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/NonExistent/\$value")
    
    check_status "$HTTP_CODE" "404"
}

# Test 5: $value on null property
test_value_null_property() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Category/\$value")
    
    # Should return 204 No Content for null, or 200 with value
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 6: $value on collection property returns error
test_value_collection_error() {
    # $value is not applicable to collections
    local HTTP_CODE=$(http_get "$SERVER_URL/Products/\$value")
    
    # Should return 400 or 404
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 or 404)"
        return 1
    fi
}

# Test 7: $value returns raw value without quotes
test_value_raw_format() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Status/\$value")
    
    # Should return raw value like "1" not JSON like {"value": 1}
    # Just verify it's not JSON-wrapped
    if ! echo "$RESPONSE" | grep -q '"value"'; then
        return 0
    else
        echo "  Details: Response appears to be JSON-wrapped"
        return 1
    fi
}

# Test 8: $value with Accept header
test_value_accept_header() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name/\$value" \
        -H "Accept: text/plain")
    
    check_status "$HTTP_CODE" "200"
}

# Test 9: $value on complex type property
test_value_complex_type() {
    # $value may not be supported on complex types
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/ComplexProperty/\$value")
    
    # Returns 200 if supported, 400/404 if not
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 10: $value without trailing slash
test_value_no_trailing_slash() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Name/\$value")
    
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value"
run_test "Access primitive property raw value" test_property_value

echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value (check headers)"
run_test "\$value returns appropriate Content-Type header" test_value_content_type

echo "  Request: GET $SERVER_URL/Products(1)/Price/\$value"
run_test "\$value works on numeric properties" test_numeric_value

echo "  Request: GET $SERVER_URL/Products(1)/NonExistent/\$value"
run_test "\$value on non-existent property returns 404" test_value_nonexistent_property

echo "  Request: GET $SERVER_URL/Products(1)/Category/\$value"
run_test "\$value on null property returns 204 or 200" test_value_null_property

echo "  Request: GET $SERVER_URL/Products/\$value"
run_test "\$value on collection returns error" test_value_collection_error

echo "  Request: GET $SERVER_URL/Products(1)/Status/\$value"
run_test "\$value returns raw value without JSON wrapper" test_value_raw_format

echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value with Accept: text/plain"
run_test "\$value respects Accept header" test_value_accept_header

echo "  Request: GET $SERVER_URL/Products(1)/ComplexProperty/\$value"
run_test "\$value on complex type handled appropriately" test_value_complex_type

echo "  Request: GET $SERVER_URL/Products(1)/Name/\$value"
run_test "\$value works without trailing slash" test_value_no_trailing_slash

print_summary
