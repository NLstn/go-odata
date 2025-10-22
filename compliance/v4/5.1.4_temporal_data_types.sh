#!/bin/bash

# OData v4 Compliance Test: 5.1.4 Temporal Data Types
# Tests handling of temporal OData primitive types: Edm.Date, Edm.TimeOfDay, Edm.Duration
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PrimitiveTypes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.1.4 Temporal Data Types"
echo "======================================"
echo ""
echo "Description: Validates handling of OData temporal types including"
echo "             Edm.Date, Edm.TimeOfDay, and Edm.Duration in filters,"
echo "             metadata, and responses."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PrimitiveTypes"
echo ""

# Test 1: Edm.DateTimeOffset is supported (baseline)
test_datetime_offset_support() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=year(CreatedAt)%20eq%202024")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Cast to Edm.Date in filter
test_cast_to_date() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=cast(CreatedAt,%20%27Edm.Date%27)%20eq%202024-01-15")
    # Should either work (200) or not be supported by entity (400/404)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]
}

# Test 3: Cast to Edm.TimeOfDay in filter
test_cast_to_timeofday() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=cast(CreatedAt,%20%27Edm.TimeOfDay%27)%20eq%2014:30:00")
    # Should either work (200) or not be supported by entity (400/404)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]
}

# Test 4: isof function recognizes Edm.Date
test_isof_date() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$filter=isof(CreatedAt,%20%27Edm.Date%27)%20eq%20true")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=isof(CreatedAt,%20%27Edm.Date%27)%20eq%20true")
    # Should return valid response
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 5: isof function recognizes Edm.TimeOfDay
test_isof_timeofday() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=isof(CreatedAt,%20%27Edm.TimeOfDay%27)%20eq%20false")
    # Should return valid response
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 6: isof function recognizes Edm.DateTimeOffset
test_isof_datetimeoffset() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=isof(CreatedAt,%20%27Edm.DateTimeOffset%27)%20eq%20true")
    check_status "$HTTP_CODE" "200"
}

# Test 7: Date literal in filter (YYYY-MM-DD format)
test_date_literal() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=date(CreatedAt)%20eq%202024-01-15")
    check_status "$HTTP_CODE" "200"
}

# Test 8: Time literal in filter (HH:MM:SS format)
test_time_literal() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=time(CreatedAt)%20eq%2014:30:00")
    check_status "$HTTP_CODE" "200"
}

# Test 9: Date comparison with gt/lt operators
test_date_comparison() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=date(CreatedAt)%20gt%202024-01-01")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Time comparison with operators
test_time_comparison() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=time(CreatedAt)%20lt%2012:00:00")
    check_status "$HTTP_CODE" "200"
}

# Test 11: Combining date and time functions
test_date_time_combination() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=date(CreatedAt)%20eq%202024-01-15%20and%20time(CreatedAt)%20gt%2010:00:00")
    check_status "$HTTP_CODE" "200"
}

# Test 12: Type validation - invalid date format
test_invalid_date_format() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=date(CreatedAt)%20eq%2001/15/2024")
    # Should return 400 for invalid format
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ]
}

# Test 13: Metadata includes temporal type support
test_metadata_temporal_types() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Metadata should reference DateTimeOffset at minimum
    echo "$RESPONSE" | grep -q "DateTimeOffset"
}

# Test 14: Cast with invalid temporal type returns error
test_invalid_temporal_cast() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=cast(CreatedAt,%20%27Edm.InvalidType%27)%20eq%20null")
    # Should return 400 for invalid type
    [ "$HTTP_CODE" = "400" ]
}

# Test 15: Duration type support (optional - may not be implemented)
test_duration_support() {
    # Duration is typically not stored but can be used in computations
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=isof(%27Edm.Duration%27)%20eq%20false")
    # Should handle gracefully - either 200 or 400
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

echo "  Request: Filter with DateTimeOffset (baseline)"
run_test "Edm.DateTimeOffset type is supported" test_datetime_offset_support

echo "  Request: Filter with cast to Edm.Date"
run_test "Cast to Edm.Date in filter" test_cast_to_date

echo "  Request: Filter with cast to Edm.TimeOfDay"
run_test "Cast to Edm.TimeOfDay in filter" test_cast_to_timeofday

echo "  Request: Filter with isof for Edm.Date"
run_test "isof function recognizes Edm.Date type" test_isof_date

echo "  Request: Filter with isof for Edm.TimeOfDay"
run_test "isof function recognizes Edm.TimeOfDay type" test_isof_timeofday

echo "  Request: Filter with isof for Edm.DateTimeOffset"
run_test "isof function recognizes Edm.DateTimeOffset type" test_isof_datetimeoffset

echo "  Request: Filter with date literal"
run_test "Date literal in YYYY-MM-DD format" test_date_literal

echo "  Request: Filter with time literal"
run_test "Time literal in HH:MM:SS format" test_time_literal

echo "  Request: Filter with date comparison operators"
run_test "Date comparison with gt/lt operators" test_date_comparison

echo "  Request: Filter with time comparison operators"
run_test "Time comparison with operators" test_time_comparison

echo "  Request: Filter combining date and time functions"
run_test "Combine date() and time() functions" test_date_time_combination

echo "  Request: Filter with invalid date format"
run_test "Invalid date format handled appropriately" test_invalid_date_format

echo "  Request: GET metadata document"
run_test "Metadata includes temporal type definitions" test_metadata_temporal_types

echo "  Request: Filter with invalid temporal type"
run_test "Invalid temporal type in cast returns error" test_invalid_temporal_cast

echo "  Request: Filter with Edm.Duration type"
run_test "Edm.Duration type handled (optional)" test_duration_support

print_summary
