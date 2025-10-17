#!/bin/bash

# OData v4 Compliance Test: 11.2.5.6 System Query Option $expand
# Tests $expand query option for expanding related entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionexpand

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.6 System Query Option \$expand"
echo "======================================"
echo ""
echo "Description: Validates \$expand query option for expanding related"
echo "             entities according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionexpand"
echo ""



# Test 1: Basic $expand returns related entities inline
echo "Test 1: \$expand includes related entities inline"
echo "  Request: GET $SERVER_URL/Products?\$expand=Descriptions&\$top=1"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$expand=Descriptions&\$top=1" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Descriptions"'; then
        test_result "\$expand includes expanded navigation property" "PASS"
    else
        test_result "\$expand includes expanded navigation property" "FAIL" "No Descriptions property found"
    fi
else
    test_result "\$expand returns 200" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: $expand with $select on expanded entity
echo "Test 2: \$expand with nested \$select"
echo "  Request: GET $SERVER_URL/Products?\$expand=Descriptions(\$select=Description)&\$top=1"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$expand=Descriptions(\$select=Description)&\$top=1" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Descriptions"'; then
        test_result "\$expand with nested \$select works" "PASS"
    else
        test_result "\$expand with nested \$select works" "FAIL" "No Descriptions in response"
    fi
else
    test_result "\$expand with nested \$select" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Multiple $expand
echo "Test 3: Multiple navigation properties in \$expand"
echo "  Request: GET $SERVER_URL/Products(1)?\$expand=Descriptions"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)?\$expand=Descriptions" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"Descriptions"'; then
        test_result "Single entity \$expand works" "PASS"
    else
        test_result "Single entity \$expand works" "FAIL" "No Descriptions property"
    fi
else
    test_result "Single entity \$expand" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""


print_summary
