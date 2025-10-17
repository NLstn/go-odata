#!/bin/bash

# OData v4 Compliance Test: 11.2.5.5 System Query Option $count
# Tests $count query option according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptioncount

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.5 System Query Option \$count"
echo "======================================"
echo ""
echo "Description: Validates \$count query option to include count of matching"
echo "             entities in the response according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptioncount"
echo ""



# Test 1: $count=true includes @odata.count in response
echo "Test 1: \$count=true includes @odata.count in response"
echo "  Request: GET $SERVER_URL/Products?\$count=true"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$count=true" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"@odata.count"'; then
        test_result "\$count=true includes @odata.count property" "PASS"
    else
        test_result "\$count=true includes @odata.count property" "FAIL" "No @odata.count in response"
    fi
else
    test_result "\$count=true returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: $count=false does not include @odata.count
echo "Test 2: \$count=false excludes @odata.count"
echo "  Request: GET $SERVER_URL/Products?\$count=false"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$count=false" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if ! echo "$BODY" | grep -q '"@odata.count"'; then
        test_result "\$count=false excludes @odata.count" "PASS"
    else
        test_result "\$count=false excludes @odata.count" "FAIL" "@odata.count should not be present"
    fi
else
    test_result "\$count=false returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: $count with $filter returns filtered count
echo "Test 3: \$count with \$filter returns filtered count"
echo "  Request: GET $SERVER_URL/Products?\$count=true&\$filter=Price gt 100"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$count=true&\$filter=Price%20gt%20100" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"@odata.count"'; then
        COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
        if [ -n "$COUNT" ]; then
            test_result "\$count with \$filter returns count of filtered items" "PASS"
        else
            test_result "\$count with \$filter returns count of filtered items" "FAIL" "Count not numeric"
        fi
    else
        test_result "\$count with \$filter includes @odata.count" "FAIL" "No @odata.count"
    fi
else
    test_result "\$count with \$filter" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 4: $count with $top still returns total count
echo "Test 4: \$count with \$top returns total count, not page count"
echo "  Request: GET $SERVER_URL/Products?\$count=true&\$top=1"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$count=true&\$top=1" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"@odata.count"'; then
        COUNT=$(echo "$BODY" | grep -o '"@odata.count"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*$')
        ITEMS=$(echo "$BODY" | grep -o '"ID"' | wc -l)
        if [ -n "$COUNT" ] && [ "$COUNT" -ge "$ITEMS" ]; then
            test_result "\$count with \$top returns total count" "PASS"
        else
            test_result "\$count with \$top returns total count" "FAIL" "Count=$COUNT, Items=$ITEMS"
        fi
    else
        test_result "\$count with \$top includes @odata.count" "FAIL" "No @odata.count"
    fi
else
    test_result "\$count with \$top" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""


print_summary
