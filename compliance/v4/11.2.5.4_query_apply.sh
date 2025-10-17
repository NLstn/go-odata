#!/bin/bash

# OData v4 Compliance Test: 11.2.5.4 System Query Option $apply
# Tests $apply query option for data aggregation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.4 System Query Option \$apply"
echo "======================================"
echo ""
echo "Description: Validates \$apply query option for data aggregation"
echo "             including groupby, aggregate, and filter transformations."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html"
echo ""



# Test 1: Basic aggregate transformation
echo "Test 1: \$apply with aggregate (count)"
echo "  Request: GET $SERVER_URL/Products?\$apply=aggregate(\$count as Total)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$apply=aggregate(\$count%20as%20Total)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Total"'; then
        test_result "Aggregate \$count" "PASS"
    else
        test_result "Aggregate \$count" "FAIL" "Response missing 'Total' property"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Aggregate \$count" "PASS" "\$apply not implemented (optional extension)"
else
    test_result "Aggregate \$count" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: groupby transformation
echo ""
echo "Test 2: \$apply with groupby"
echo "  Request: GET $SERVER_URL/Products?\$apply=groupby((CategoryID))"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID))" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"CategoryID"'; then
        test_result "groupby transformation" "PASS"
    else
        test_result "groupby transformation" "FAIL" "Response missing 'CategoryID' property"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "groupby transformation" "PASS" "\$apply not implemented (optional extension)"
else
    test_result "groupby transformation" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 3: groupby with aggregate
echo ""
echo "Test 3: \$apply with groupby and aggregate"
echo "  Request: GET $SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count as Count))"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$apply=groupby((CategoryID),aggregate(\$count%20as%20Count))" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"CategoryID"' && echo "$BODY" | grep -q '"Count"'; then
        test_result "groupby with aggregate" "PASS"
    else
        test_result "groupby with aggregate" "FAIL" "Response missing expected properties"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "groupby with aggregate" "PASS" "\$apply not implemented (optional extension)"
else
    test_result "groupby with aggregate" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 4: filter transformation
echo ""
echo "Test 4: \$apply with filter transformation"
echo "  Request: GET $SERVER_URL/Products?\$apply=filter(Price gt 10)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$apply=filter(Price%20gt%2010)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    test_result "filter transformation" "PASS"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "filter transformation" "PASS" "\$apply not implemented (optional extension)"
else
    test_result "filter transformation" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 5: Invalid $apply expression should return 400
echo ""
echo "Test 5: Invalid \$apply expression returns 400"
echo "  Request: GET $SERVER_URL/Products?\$apply=invalid(syntax)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$apply=invalid(syntax)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "Invalid \$apply returns 400" "PASS"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Invalid \$apply returns 400" "PASS" "\$apply not implemented (optional extension)"
else
    test_result "Invalid \$apply returns 400" "FAIL" "Expected HTTP 400 or 501, got $HTTP_CODE"
fi


print_summary
