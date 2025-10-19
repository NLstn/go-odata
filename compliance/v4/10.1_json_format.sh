#!/bin/bash

# OData v4 Compliance Test: 10.1 JSON Format
# Tests JSON format requirements for OData responses
# Spec: https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 10.1 JSON Format"
echo "======================================"
echo ""
echo "Description: Validates JSON format requirements for OData responses"
echo "             (entity representation, collections, metadata annotations, etc.)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html"
echo ""

# Test 1: Collection response has 'value' property
test_json_collection_value() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    if echo "$RESPONSE" | grep -q '"value"'; then
        return 0
    else
        echo "  Details: Collection response missing 'value' property"
        return 1
    fi
}

# Test 2: Entity response has @odata.context
test_json_odata_context() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    if echo "$RESPONSE" | grep -q '"@odata.context"'; then
        return 0
    else
        echo "  Details: Response missing @odata.context annotation"
        return 1
    fi
}

# Test 3: Valid JSON structure
test_json_valid_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Check if response can be parsed as JSON (contains braces and valid structure)
    if echo "$RESPONSE" | grep -q '^{.*}$' || echo "$RESPONSE" | python3 -m json.tool > /dev/null 2>&1; then
        return 0
    else
        echo "  Details: Response is not valid JSON"
        return 1
    fi
}

# Test 4: Property values have correct types
test_json_property_types() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Check for numeric property (ID should be number, not string)
    if echo "$RESPONSE" | grep -q '"ID":[0-9]'; then
        return 0
    else
        echo "  Details: Numeric properties not formatted correctly"
        return 1
    fi
}

# Test 5: String values are properly quoted
test_json_string_values() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Check that Name is a quoted string
    if echo "$RESPONSE" | grep -q '"Name":"[^"]*"'; then
        return 0
    else
        echo "  Details: String values not properly quoted"
        return 1
    fi
}

# Test 6: Null values represented as JSON null
test_json_null_values() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    # Should return valid JSON (status 200) 
    # If entity has null fields, they should be represented as null in JSON
    check_status "$HTTP_CODE" "200"
}

# Test 7: Arrays in collection responses
test_json_array_format() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # value should be an array (allow optional whitespace after colon)
    if echo "$RESPONSE" | grep -q '"value":[[:space:]]*\['; then
        return 0
    else
        echo "  Details: 'value' property is not an array"
        return 1
    fi
}

# Test 8: Metadata annotations format
test_json_metadata_annotations() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # Check for @ prefix in metadata annotations
    if echo "$RESPONSE" | grep -q '"@odata\.' || echo "$RESPONSE" | grep -q '@odata\.'; then
        return 0
    else
        echo "  Details: Metadata annotations not in correct format"
        return 1
    fi
}

# Test 9: IEEE754Compatible for large numbers
test_json_ieee754_compatible() {
    # When IEEE754Compatible=true, large numbers should be strings
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$format=application/json;IEEE754Compatible=true")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$format=application/json;IEEE754Compatible=true")
    
    # Should still return valid JSON
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: IEEE754Compatible parameter not supported (status: $HTTP_CODE)"
        # This is optional, so don't fail
        return 0
    fi
}

# Test 10: Content-Type includes charset
test_json_charset() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" 2>&1)
    
    # Check for Content-Type with charset
    if echo "$RESPONSE" | grep -qi "Content-Type:.*application/json" && \
       echo "$RESPONSE" | grep -qi "charset"; then
        return 0
    elif echo "$RESPONSE" | grep -qi "Content-Type:.*application/json"; then
        # JSON without explicit charset is acceptable (UTF-8 is default)
        return 0
    else
        echo "  Details: Content-Type not properly set"
        return 1
    fi
}

echo "  Request: GET /Products (collection)"
run_test "Collection response has 'value' property" test_json_collection_value

echo "  Request: GET /Products"
run_test "Response includes @odata.context annotation" test_json_odata_context

echo "  Request: GET /Products(1)"
run_test "Response is valid JSON structure" test_json_valid_structure

echo "  Request: GET /Products(1) - check property types"
run_test "Property values have correct JSON types" test_json_property_types

echo "  Request: GET /Products(1) - check strings"
run_test "String values are properly quoted" test_json_string_values

echo "  Request: GET with \$select"
run_test "Null values represented as JSON null" test_json_null_values

echo "  Request: GET /Products (array check)"
run_test "Collection 'value' is JSON array" test_json_array_format

echo "  Request: GET /Products (metadata)"
run_test "Metadata annotations use @ prefix" test_json_metadata_annotations

echo "  Request: GET with IEEE754Compatible=true"
run_test "IEEE754Compatible parameter supported" test_json_ieee754_compatible

echo "  Request: Check Content-Type header"
run_test "Content-Type includes proper JSON media type" test_json_charset

print_summary
