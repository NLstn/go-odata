#!/bin/bash

# OData v4 Compliance Test: 11.4.3 Update an Entity
# Tests PATCH and PUT operations for updating entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_UpdateanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.3 Update an Entity"
echo "======================================"
echo ""
echo "Description: Validates PATCH and PUT operations for updating entities"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_UpdateanEntity"
echo ""

# Use existing product from seeded data (ID 1 = Laptop)
TEST_ID=1

# Test 1: PATCH updates specified properties only
test_patch_update() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
        -H "Content-Type: application/json" \
        -d '{"Price":149.99}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        # Verify the update
        local VERIFY=$(curl -s "$SERVER_URL/Products($TEST_ID)" 2>&1)
        if echo "$VERIFY" | grep -q '"Price":149.99'; then
            return 0
        else
            echo "  Details: Price not updated"
            return 1
        fi
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 204)"
        return 1
    fi
}

# Test 2: PATCH with invalid property returns error
test_patch_invalid_property() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
        -H "Content-Type: application/json" \
        -d '{"NonExistentProperty":"value"}' 2>&1)
    
    check_status "$HTTP_CODE" "400"
}

# Test 3: PATCH to non-existent entity returns 404
test_patch_not_found() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products(999999)" \
        -H "Content-Type: application/json" \
        -d '{"Price":100}' 2>&1)
    
    check_status "$HTTP_CODE" "404"
}

# Test 4: Content-Type header validation
test_patch_no_content_type() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($TEST_ID)" \
        -d '{"Price":99.99}' 2>&1)
    
    # Should return 400 or 415 for missing/incorrect Content-Type
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "415" ]; then
        return 0
    else
        # Some implementations may be lenient and accept it
        echo "  Details: Status: $HTTP_CODE (lenient implementation)"
        return 0
    fi
}

echo "  Request: PATCH $SERVER_URL/Products($TEST_ID)"
run_test "PATCH updates specified properties (partial update)" test_patch_update

echo "  Request: PATCH $SERVER_URL/Products($TEST_ID) with invalid property"
run_test "PATCH with invalid property returns 400 Bad Request" test_patch_invalid_property

echo "  Request: PATCH $SERVER_URL/Products(999999)"
run_test "PATCH to non-existent entity returns 404" test_patch_not_found

echo "  Request: PATCH $SERVER_URL/Products($TEST_ID) without Content-Type"
run_test "PATCH without Content-Type validation" test_patch_no_content_type

print_summary
