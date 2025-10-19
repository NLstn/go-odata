#!/bin/bash

# OData v4 Compliance Test: 5.1.1 Primitive Data Types
# Tests handling of various primitive data types (String, Int32, Decimal, Boolean, DateTime, etc.)
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_PrimitiveTypes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.1.1 Primitive Data Types"
echo "======================================"
echo ""
echo "Description: Validates handling of OData primitive data types in requests and responses"
echo "             (Edm.String, Edm.Int32, Edm.Decimal, Edm.Boolean, Edm.DateTimeOffset, etc.)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_PrimitiveTypes"
echo ""

# Test 1: String data type
test_string_type() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Name eq 'Laptop'")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Name eq 'Laptop'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$RESPONSE" | grep -q '"Name"'; then
            return 0
        else
            echo "  Details: Response missing Name field"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        echo "  Response: $RESPONSE"
        return 1
    fi
}

# Test 2: Int32 data type
test_int32_type() {
    local FILTER="ID eq 1"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Decimal data type
test_decimal_type() {
    # Use actual price from sample data (999.99 for Laptop)
    local FILTER="Price eq 999.99"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Boolean data type (using Status as int, but testing boolean filters)
test_boolean_type() {
    local FILTER="Status eq 1"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 5: DateTimeOffset data type
test_datetime_type() {
    local FILTER="CreatedAt lt '2025-12-31T23:59:59Z'"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Null value handling
test_null_value() {
    local FILTER="Category ne null"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Number precision - verify existing data maintains precision
test_number_precision() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)" 2>&1)
    # Check that price field exists and has decimal precision
    if echo "$RESPONSE" | grep -q '"Price":999.99'; then
        return 0
    else
        echo "  Details: Price precision not maintained or product not found"
        return 1
    fi
}

# Test 8: Special characters in strings
test_special_characters() {
    local FILTER="contains(Name,'&') or contains(Name,'/')"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Empty string handling
test_empty_string() {
    local FILTER="Name ne ''"
    local ENCODED_FILTER=$(printf %s "$FILTER" | jq -sRr @uri)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=$ENCODED_FILTER")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=Name eq 'Laptop'"
run_test "Edm.String type handles text values" test_string_type

echo "  Request: GET \$filter=ID eq 1"
run_test "Edm.Int32 type handles integer values" test_int32_type

echo "  Request: GET \$filter=Price eq 999.99"
run_test "Edm.Decimal type handles decimal values" test_decimal_type

echo "  Request: GET \$filter=Status eq 1"
run_test "Boolean-like integer filtering works" test_boolean_type

echo "  Request: GET \$filter=CreatedAt lt 2025-12-31T23:59:59Z"
run_test "Edm.DateTimeOffset type handles datetime values" test_datetime_type

echo "  Request: GET \$filter=Category ne null"
run_test "Null value handling in filters" test_null_value

echo "  Request: GET existing product with decimal price"
run_test "Decimal precision is maintained" test_number_precision

echo "  Request: GET \$filter with special characters"
run_test "Special characters in strings are handled" test_special_characters

echo "  Request: GET \$filter=Name ne ''"
run_test "Empty string handling in filters" test_empty_string

print_summary
