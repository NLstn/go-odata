#!/bin/bash

# OData v4 Compliance Test: 11.3.2 Date and Time Functions in $filter
# Tests date/time functions (year, month, day, hour, minute, second, date, time, etc.) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.2 Date/Time Functions"
echo "======================================"
echo ""
echo "Description: Validates date and time functions in \$filter query option"
echo "             (year, month, day, hour, minute, second, date, time, now, etc.)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuiltinFilterOperations"
echo ""

# Test 1: year function
test_year_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=year(CreatedAt) eq 2024")
    check_status "$HTTP_CODE" "200"
}

# Test 2: month function
test_month_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=month(CreatedAt) eq 1")
    check_status "$HTTP_CODE" "200"
}

# Test 3: day function
test_day_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=day(CreatedAt) eq 15")
    check_status "$HTTP_CODE" "200"
}

# Test 4: hour function
test_hour_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=hour(CreatedAt) lt 12")
    check_status "$HTTP_CODE" "200"
}

# Test 5: minute function
test_minute_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=minute(CreatedAt) eq 30")
    check_status "$HTTP_CODE" "200"
}

# Test 6: second function
test_second_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=second(CreatedAt) lt 60")
    check_status "$HTTP_CODE" "200"
}

# Test 7: date function
test_date_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=date(CreatedAt) eq 2024-01-15")
    check_status "$HTTP_CODE" "200"
}

# Test 8: time function
test_time_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=time(CreatedAt) lt 12:00:00")
    check_status "$HTTP_CODE" "200"
}

# Test 9: now function
test_now_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=CreatedAt lt now()")
    check_status "$HTTP_CODE" "200"
}

# Test 10: Combined date functions
test_combined_date_functions() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=year(CreatedAt) eq 2024 and month(CreatedAt) ge 6")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET \$filter=year(CreatedAt) eq 2024"
run_test "year() function extracts year from date" test_year_function

echo "  Request: GET \$filter=month(CreatedAt) eq 1"
run_test "month() function extracts month from date" test_month_function

echo "  Request: GET \$filter=day(CreatedAt) eq 15"
run_test "day() function extracts day from date" test_day_function

echo "  Request: GET \$filter=hour(CreatedAt) lt 12"
run_test "hour() function extracts hour from datetime" test_hour_function

echo "  Request: GET \$filter=minute(CreatedAt) eq 30"
run_test "minute() function extracts minute from datetime" test_minute_function

echo "  Request: GET \$filter=second(CreatedAt) lt 60"
run_test "second() function extracts second from datetime" test_second_function

echo "  Request: GET \$filter=date(CreatedAt) eq 2024-01-15"
run_test "date() function extracts date portion" test_date_function

echo "  Request: GET \$filter=time(CreatedAt) lt 12:00:00"
run_test "time() function extracts time portion" test_time_function

echo "  Request: GET \$filter=CreatedAt lt now()"
run_test "now() function returns current datetime" test_now_function

echo "  Request: GET \$filter=year(CreatedAt) eq 2024 and month(CreatedAt) ge 6"
run_test "Combined date functions work together" test_combined_date_functions

print_summary
