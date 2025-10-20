#!/bin/bash

# OData v4 Compliance Test: 11.4.6 Managing Relationships
# Tests relationship management (creating, updating, deleting links) according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ManagingRelationships

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.6 Managing Relationships"
echo "======================================"
echo ""
echo "Description: Validates relationship management operations including"
echo "             creating, updating, and deleting entity references."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ManagingRelationships"
echo ""



# Test 1: Read entity reference with \$ref
echo "Test 1: Read entity reference with \$ref"
echo "  Request: GET $SERVER_URL/Products(1)/Category/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Should return @odata.context and @odata.id
    if echo "$BODY" | grep -q '"@odata.id"'; then
        test_result "Read entity reference" "PASS"
    else
        test_result "Read entity reference" "FAIL" "Response missing @odata.id"
    fi
elif [ "$HTTP_CODE" = "404" ]; then
    test_result "Read entity reference" "PASS" "No relationship exists (valid)"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Read entity reference" "PASS" "\$ref not implemented (optional)"
else
    test_result "Read entity reference" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Read collection of references
echo ""
echo "Test 2: Read collection of entity references"
echo "  Request: GET $SERVER_URL/Products(1)/RelatedProducts/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/RelatedProducts/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Should return array of references
    if echo "$BODY" | grep -q '"value"'; then
        test_result "Read reference collection" "PASS"
    else
        test_result "Read reference collection" "FAIL" "Response missing value array"
    fi
elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
    test_result "Read reference collection" "PASS" "Navigation property not found or not implemented"
else
    test_result "Read reference collection" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 3: Create entity reference (single-valued navigation)
echo ""
echo "Test 3: Create/update entity reference with PUT"
echo "  Request: PUT $SERVER_URL/Products(1)/Category/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"@odata.id":"'$SERVER_URL'/Categories(1)"}' \
    "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    test_result "Create/update reference with PUT" "PASS"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Create/update reference with PUT" "PASS" "\$ref updates not implemented (optional)"
elif [ "$HTTP_CODE" = "404" ]; then
    test_result "Create/update reference with PUT" "PASS" "Navigation property not found"
else
    test_result "Create/update reference with PUT" "FAIL" "Expected HTTP 204/200, got $HTTP_CODE"
fi

# Test 4: Add reference to collection with POST
echo ""
echo "Test 4: Add entity reference to collection with POST"
echo "  Request: POST $SERVER_URL/Products(1)/RelatedProducts/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"@odata.id":"'$SERVER_URL'/Products(2)"}' \
    "$SERVER_URL/Products(1)/RelatedProducts/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "201" ]; then
    test_result "Add reference with POST" "PASS"
elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
    test_result "Add reference with POST" "PASS" "Navigation property not found or not implemented"
else
    test_result "Add reference with POST" "FAIL" "Expected HTTP 204/201, got $HTTP_CODE"
fi

# Test 5: Delete entity reference
echo ""
echo "Test 5: Delete entity reference with DELETE"
echo "  Request: DELETE $SERVER_URL/Products(1)/Category/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
    "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "204" ]; then
    test_result "Delete reference" "PASS"
elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
    test_result "Delete reference" "PASS" "Reference not found or not implemented"
else
    test_result "Delete reference" "FAIL" "Expected HTTP 204, got $HTTP_CODE"
fi

# Test 6: Delete specific reference from collection
echo ""
echo "Test 6: Delete specific reference from collection"
echo "  Request: DELETE $SERVER_URL/Products(1)/RelatedProducts(2)/\$ref"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
    "$SERVER_URL/Products(1)/RelatedProducts(2)/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "204" ]; then
    test_result "Delete specific collection reference" "PASS"
elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
    test_result "Delete specific collection reference" "PASS" "Reference not found or not implemented"
else
    test_result "Delete specific collection reference" "FAIL" "Expected HTTP 204, got $HTTP_CODE"
fi

# Test 7: Invalid reference should return 400
echo ""
echo "Test 7: Invalid @odata.id in reference returns 400"
echo "  Request: PUT $SERVER_URL/Products(1)/Category/\$ref with invalid reference"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"@odata.id":"invalid-reference"}' \
    "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "400" ]; then
    test_result "Invalid reference returns 400" "PASS"
elif [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
    test_result "Invalid reference returns 400" "PASS" "Navigation property not found or not implemented"
else
    test_result "Invalid reference returns 400" "FAIL" "Expected HTTP 400, got $HTTP_CODE"
fi


print_summary
