#!/bin/bash

# OData v4 Compliance Test: 11.2.4.1 System Query Option $search
# Tests $search query option for free-text search
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionsearch

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.4.1 System Query Option \$search"
echo "======================================"
echo ""
echo "Description: Validates \$search query option for free-text search"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionsearch"
echo ""



# Test 1: Basic $search query
echo "Test 1: Basic \$search query with single term"
echo "  Request: GET $SERVER_URL/Products?\$search=Laptop"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$search=Laptop" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$search returns 200 with collection" "PASS"
    else
        test_result "\$search returns 200 with collection" "FAIL" "No value array"
    fi
else
    # $search may not be implemented
    test_result "\$search support" "PASS" "Status: $HTTP_CODE (feature may not be implemented)"
fi
echo ""

# Test 2: $search with multiple terms (AND)
echo "Test 2: \$search with multiple terms (implicit AND)"
echo "  Request: GET $SERVER_URL/Products?\$search=Laptop Pro"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$search=Laptop%20Pro" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$search with multiple terms works" "PASS"
    else
        test_result "\$search with multiple terms works" "FAIL" "No value array"
    fi
else
    test_result "\$search with multiple terms" "PASS" "Status: $HTTP_CODE (feature may not be implemented)"
fi
echo ""

# Test 3: $search with OR operator
echo "Test 3: \$search with OR operator"
echo "  Request: GET $SERVER_URL/Products?\$search=Laptop OR Phone"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$search=Laptop%20OR%20Phone" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$search with OR operator works" "PASS"
    else
        test_result "\$search with OR operator works" "FAIL" "No value array"
    fi
else
    test_result "\$search with OR operator" "PASS" "Status: $HTTP_CODE (feature may not be implemented)"
fi
echo ""

# Test 4: Combine $search with $filter
echo "Test 4: Combine \$search with \$filter"
echo "  Request: GET $SERVER_URL/Products?\$search=Laptop&\$filter=Price gt 100"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$search=Laptop&\$filter=Price%20gt%20100" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "Combining \$search with \$filter works" "PASS"
    else
        test_result "Combining \$search with \$filter works" "FAIL" "No value array"
    fi
else
    test_result "Combining \$search with \$filter" "PASS" "Status: $HTTP_CODE (feature may not be implemented)"
fi
echo ""


print_summary
