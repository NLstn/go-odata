#!/bin/bash

# OData v4 Compliance Test: 11.2.5.2 System Query Option $select and $orderby
# Tests $select and $orderby query options according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.2/11.2.5.3 Query Options \$select and \$orderby"
echo "======================================"
echo ""
echo "Description: Validates \$select and \$orderby query options"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html"
echo ""



# Test 1: Basic $select with single property
echo "Test 1: \$select with single property"
echo "  Request: GET $SERVER_URL/Products?\$select=Name"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$select=Name" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Name"'; then
        test_result "\$select returns selected property" "PASS"
    else
        test_result "\$select returns selected property" "FAIL" "Name property not in response"
    fi
else
    test_result "\$select with single property" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: $select with multiple properties
echo "Test 2: \$select with multiple properties"
echo "  Request: GET $SERVER_URL/Products?\$select=Name,Price"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$select=Name,Price" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    NAME_FOUND=$(echo "$BODY" | grep -c '"Name"')
    PRICE_FOUND=$(echo "$BODY" | grep -c '"Price"')
    if [ "$NAME_FOUND" -gt 0 ] && [ "$PRICE_FOUND" -gt 0 ]; then
        test_result "\$select returns multiple selected properties" "PASS"
    else
        test_result "\$select returns multiple selected properties" "FAIL" "Missing properties"
    fi
else
    test_result "\$select with multiple properties" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Basic $orderby ascending
echo "Test 3: \$orderby ascending"
echo "  Request: GET $SERVER_URL/Products?\$orderby=Price asc"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$orderby=Price%20asc" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$orderby asc returns results" "PASS"
    else
        test_result "\$orderby asc returns results" "FAIL" "No value array"
    fi
else
    test_result "\$orderby ascending" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 4: $orderby descending
echo "Test 4: \$orderby descending"
echo "  Request: GET $SERVER_URL/Products?\$orderby=Price desc"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$orderby=Price%20desc" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$orderby desc returns results" "PASS"
    else
        test_result "\$orderby desc returns results" "FAIL" "No value array"
    fi
else
    test_result "\$orderby descending" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 5: $orderby with multiple properties
echo "Test 5: \$orderby with multiple properties"
echo "  Request: GET $SERVER_URL/Products?\$orderby=Category,Price desc"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$orderby=Category,Price%20desc" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$orderby with multiple properties works" "PASS"
    else
        test_result "\$orderby with multiple properties works" "FAIL" "No value array"
    fi
else
    test_result "\$orderby with multiple properties" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 6: Combining $select and $orderby
echo "Test 6: Combining \$select and \$orderby"
echo "  Request: GET $SERVER_URL/Products?\$select=Name,Price&\$orderby=Price"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$select=Name,Price&\$orderby=Price" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Name"' && echo "$BODY" | grep -q '"Price"'; then
        test_result "Combining \$select and \$orderby works" "PASS"
    else
        test_result "Combining \$select and \$orderby works" "FAIL" "Missing properties"
    fi
else
    test_result "Combining \$select and \$orderby" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""


print_summary
