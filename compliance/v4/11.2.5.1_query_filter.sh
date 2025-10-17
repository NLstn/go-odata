#!/bin/bash

# OData v4 Compliance Test: 11.2.5.1 System Query Option $filter
# Tests $filter query option according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionfilter

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.1 System Query Option \$filter"
echo "======================================"
echo ""
echo "Description: Validates \$filter query option with various operators"
echo "             and expressions according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionfilter"
echo ""

PASSED=0
FAILED=0
TOTAL=0

test_result() {
    local test_name="$1"
    local result="$2"
    local details="$3"
    
    TOTAL=$((TOTAL + 1))
    if [ "$result" = "PASS" ]; then
        PASSED=$((PASSED + 1))
        echo "✓ PASS: $test_name"
    else
        FAILED=$((FAILED + 1))
        echo "✗ FAIL: $test_name"
        if [ -n "$details" ]; then
            echo "  Details: $details"
        fi
    fi
}

# Test 1: Basic eq (equals) operator
echo "Test 1: \$filter with eq operator"
echo "  Request: GET $SERVER_URL/Products?\$filter=ID eq 1"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=ID%20eq%201" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with eq operator returns 200 with results" "PASS"
    else
        test_result "\$filter with eq operator returns 200 with results" "FAIL" "No value array in response"
    fi
else
    test_result "\$filter with eq operator returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: gt (greater than) operator
echo "Test 2: \$filter with gt operator"
echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 100"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%20100" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with gt operator returns valid response" "PASS"
    else
        test_result "\$filter with gt operator returns valid response" "FAIL" "No value array"
    fi
else
    test_result "\$filter with gt operator" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: String contains function
echo "Test 3: \$filter with contains() function"
echo "  Request: GET $SERVER_URL/Products?\$filter=contains(Name,'Laptop')"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=contains(Name,'Laptop')" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with contains() function works" "PASS"
    else
        test_result "\$filter with contains() function works" "FAIL" "No value array"
    fi
else
    test_result "\$filter with contains() function" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 4: Boolean operators (and)
echo "Test 4: \$filter with 'and' operator"
echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 and Price lt 1000"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=Price%20gt%2010%20and%20Price%20lt%201000" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with 'and' operator works" "PASS"
    else
        test_result "\$filter with 'and' operator works" "FAIL" "No value array"
    fi
else
    test_result "\$filter with 'and' operator" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 5: Boolean operators (or)
echo "Test 5: \$filter with 'or' operator"
echo "  Request: GET $SERVER_URL/Products?\$filter=ID eq 1 or ID eq 2"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=ID%20eq%201%20or%20ID%20eq%202" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with 'or' operator works" "PASS"
    else
        test_result "\$filter with 'or' operator works" "FAIL" "No value array"
    fi
else
    test_result "\$filter with 'or' operator" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 6: Parentheses for grouping
echo "Test 6: \$filter with parentheses"
echo "  Request: GET $SERVER_URL/Products?\$filter=(Price gt 100) and (ID lt 10)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$filter=(Price%20gt%20100)%20and%20(ID%20lt%2010)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$filter with parentheses works" "PASS"
    else
        test_result "\$filter with parentheses works" "FAIL" "No value array"
    fi
else
    test_result "\$filter with parentheses" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Summary
echo "======================================"
echo "Summary: $PASSED/$TOTAL tests passed"
if [ $FAILED -gt 0 ]; then
    echo "Status: FAILING"
    exit 1
else
    echo "Status: PASSING"
    exit 0
fi
