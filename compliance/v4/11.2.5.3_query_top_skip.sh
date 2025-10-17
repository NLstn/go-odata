#!/bin/bash

# OData v4 Compliance Test: 11.2.5.3 and 11.2.5.4 System Query Options $top and $skip
# Tests $top and $skip query options for paging according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionstopandskip

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.3-4 System Query Options \$top and \$skip"
echo "======================================"
echo ""
echo "Description: Validates \$top and \$skip query options for paging"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionstopandskip"
echo ""



# Test 1: $top limits the number of items returned
echo "Test 1: \$top limits number of items"
echo "  Request: GET $SERVER_URL/Products?\$top=2"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$top=2" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -le 2 ]; then
        test_result "\$top=2 returns at most 2 items" "PASS"
    else
        test_result "\$top=2 returns at most 2 items" "FAIL" "Returned $COUNT items"
    fi
else
    test_result "\$top returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: $skip skips the specified number of items
echo "Test 2: \$skip skips items"
echo "  Request: GET $SERVER_URL/Products?\$skip=1&\$top=1"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$skip=1&\$top=1" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$skip with \$top returns valid response" "PASS"
    else
        test_result "\$skip with \$top returns valid response" "FAIL" "No value array"
    fi
else
    test_result "\$skip returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: $top=0 returns empty collection
echo "Test 3: \$top=0 returns empty collection"
echo "  Request: GET $SERVER_URL/Products?\$top=0"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$top=0" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -eq 0 ]; then
        test_result "\$top=0 returns empty collection" "PASS"
    else
        test_result "\$top=0 returns empty collection" "FAIL" "Returned $COUNT items"
    fi
else
    test_result "\$top=0 returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 4: Combine $skip and $top for paging
echo "Test 4: Combine \$skip and \$top for paging"
echo "  Request: GET $SERVER_URL/Products?\$skip=2&\$top=3"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$skip=2&\$top=3" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    if [ "$COUNT" -le 3 ]; then
        test_result "Combined \$skip and \$top works" "PASS"
    else
        test_result "Combined \$skip and \$top works" "FAIL" "Returned $COUNT items, expected max 3"
    fi
else
    test_result "Combined \$skip and \$top" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""


print_summary
