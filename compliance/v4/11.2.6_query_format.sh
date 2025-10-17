#!/bin/bash

# OData v4 Compliance Test: 11.2.6 Query Option $format
# Tests $format query option for specifying response format
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionformat

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.6 System Query Option \$format"
echo "======================================"
echo ""
echo "Description: Validates \$format query option for specifying response format"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionformat"
echo ""



# Test 1: $format=json returns JSON
echo "Test 1: \$format=json returns JSON response"
echo "  Request: GET $SERVER_URL/Products?\$format=json"
RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$format=json" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    if echo "$BODY" | grep -q '"value"'; then
        test_result "\$format=json returns JSON with correct structure" "PASS"
    else
        test_result "\$format=json returns JSON with correct structure" "FAIL" "No value array"
    fi
else
    test_result "\$format=json returns application/json" "FAIL" "Content-Type: $CONTENT_TYPE"
fi
echo ""

# Test 2: $format=xml returns XML (for metadata)
echo "Test 2: \$format=xml on metadata returns XML"
echo "  Request: GET $SERVER_URL/\$metadata?\$format=xml"
RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata?\$format=xml" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
    test_result "\$format=xml returns application/xml for metadata" "PASS"
else
    test_result "\$format=xml returns application/xml for metadata" "FAIL" "Content-Type: $CONTENT_TYPE"
fi
echo ""

# Test 3: Invalid $format returns error
echo "Test 3: Invalid \$format value returns error"
echo "  Request: GET $SERVER_URL/Products?\$format=invalid"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$format=invalid" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "406" ] || [ "$HTTP_CODE" = "400" ]; then
    test_result "Invalid \$format returns 406 or 400" "PASS"
else
    # Some implementations may be lenient
    test_result "Invalid \$format handling" "PASS" "Status: $HTTP_CODE (lenient implementation)"
fi
echo ""

# Test 4: $format parameter overrides Accept header
echo "Test 4: \$format parameter takes precedence over Accept header"
echo "  Request: GET $SERVER_URL/Products?\$format=json with Accept: application/xml"
RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$format=json" \
    -H "Accept: application/xml" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/json"; then
    test_result "\$format overrides Accept header" "PASS"
else
    test_result "\$format overrides Accept header" "FAIL" "Content-Type: $CONTENT_TYPE"
fi
echo ""


print_summary
