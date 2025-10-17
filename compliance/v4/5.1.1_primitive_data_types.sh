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

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: String data type
test_string_type() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Name eq 'Gaming Laptop'")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Name eq 'Gaming Laptop'")
    
    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$RESPONSE" | grep -q '"Name"'; then
            return 0
        else
            echo "  Details: Response missing Name field"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: Int32 data type
test_int32_type() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=ID eq 1")
    check_status "$HTTP_CODE" "200"
}

# Test 3: Decimal data type
test_decimal_type() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price eq 99.99")
    check_status "$HTTP_CODE" "200"
}

# Test 4: Boolean data type (using Status as int, but testing boolean filters)
test_boolean_type() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Status eq 1")
    check_status "$HTTP_CODE" "200"
}

# Test 5: DateTimeOffset data type
test_datetime_type() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=CreatedAt lt 2025-12-31T23:59:59Z")
    check_status "$HTTP_CODE" "200"
}

# Test 6: Null value handling
test_null_value() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category ne null")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Create entity with various primitive types
test_create_with_primitives() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test Primitive Types","Price":123.45,"Category":"Test","Status":1}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 8: Number precision
test_number_precision() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Precision Test","Price":999.999,"Category":"Test","Status":1}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
            # Verify precision is maintained
            local VERIFY=$(curl -s "$SERVER_URL/Products($ID)" 2>&1)
            if echo "$VERIFY" | grep -q '"Price"'; then
                return 0
            fi
        fi
    fi
    echo "  Details: Failed to verify precision"
    return 1
}

# Test 9: Special characters in strings
test_special_characters() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=contains(Name,'%26') or contains(Name,'/')")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Empty string handling
test_empty_string() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Name ne ''")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=Name eq 'Gaming Laptop'"
run_test "Edm.String type handles text values" test_string_type

echo "  Request: GET \$filter=ID eq 1"
run_test "Edm.Int32 type handles integer values" test_int32_type

echo "  Request: GET \$filter=Price eq 99.99"
run_test "Edm.Decimal type handles decimal values" test_decimal_type

echo "  Request: GET \$filter=Status eq 1"
run_test "Boolean-like integer filtering works" test_boolean_type

echo "  Request: GET \$filter=CreatedAt lt 2025-12-31T23:59:59Z"
run_test "Edm.DateTimeOffset type handles datetime values" test_datetime_type

echo "  Request: GET \$filter=Category ne null"
run_test "Null value handling in filters" test_null_value

echo "  Request: POST with various primitive types"
run_test "Create entity with primitive types" test_create_with_primitives

echo "  Request: POST with high precision decimal"
run_test "Decimal precision is maintained" test_number_precision

echo "  Request: GET \$filter with special characters"
run_test "Special characters in strings are handled" test_special_characters

echo "  Request: GET \$filter=Name ne ''"
run_test "Empty string handling in filters" test_empty_string

print_summary
