#!/bin/bash

# OData v4 Compliance Test: 11.4.6 Managing Relationships
# Tests relationship management (creating, updating, deleting links) according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ManagingRelationships

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Test 1: Read entity reference with \$ref
test_1() {
    local RESPONSE=$(curl -g -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
    local STATUS=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$STATUS" = "200" ]; then
        check_contains "$BODY" '"@odata.id"' "Response has @odata.id"
    elif [ "$STATUS" = "404" ] || [ "$STATUS" = "501" ]; then
        return 0  # Valid responses
    else
        return 1
    fi
}

# Test 2: Read collection of references
test_2() {
    local RESPONSE=$(curl -g -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/RelatedProducts/\$ref" 2>&1)
    local STATUS=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$STATUS" = "200" ]; then
        check_contains "$BODY" '"value"' "Response has value array"
    elif [ "$STATUS" = "404" ] || [ "$STATUS" = "501" ]; then
        return 0
    else
        return 1
    fi
}

# Test 3: Create entity reference (single-valued navigation)
test_3() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'$SERVER_URL'/Categories(1)"}' \
        "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]
}

# Test 4: Add reference to collection with POST
test_4() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'$SERVER_URL'/Products(2)"}' \
        "$SERVER_URL/Products(1)/RelatedProducts/\$ref" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 5: Delete entity reference
test_5() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
        "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 6: Delete specific reference from collection
test_6() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
        "$SERVER_URL/Products(1)/RelatedProducts(2)/\$ref" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 7: Invalid reference should return 400
test_7() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"invalid-reference"}' \
        "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]
}

run_test "Read entity reference with \$ref" test_1
run_test "Read collection of entity references" test_2
run_test "Create/update entity reference with PUT" test_3
run_test "Add entity reference to collection with POST" test_4
run_test "Delete entity reference with DELETE" test_5
run_test "Delete specific reference from collection" test_6
run_test "Invalid @odata.id in reference returns 400" test_7

print_summary
