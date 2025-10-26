#!/bin/bash

# OData v4 Compliance Test: 11.4.4 Delete an Entity
# Tests DELETE operations according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_DeleteanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.4 Delete an Entity"
echo "======================================"
echo ""
echo "Description: Validates DELETE operations for removing entities"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_DeleteanEntity"
echo ""



# Test 1: DELETE returns 204 No Content on success
test_delete_success() {
    # Create entity to delete
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Product To Delete","Price":10.00,"CategoryID":1,"Status":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)

    if [ "$CREATE_CODE" = "201" ]; then
        local DELETE_ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$DELETE_ID" ]; then
            local DELETE_CODE=$(curl -g -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Products($DELETE_ID)")
            check_status "$DELETE_CODE" "204"
        else
            echo "  Details: Could not create test entity"
            return 1
        fi
    else
        echo "  Details: Could not create test entity (status: $CREATE_CODE)"
        return 1
    fi
}

run_test "DELETE returns 204 No Content" test_delete_success

# Test 2: DELETE to non-existent entity returns 404
test_delete_nonexistent() {
    local HTTP_CODE=$(curl -g -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
}

run_test "DELETE to non-existent entity returns 404" test_delete_nonexistent

# Test 3: Verify entity is actually deleted
test_verify_deleted() {
    # Create entity to delete
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Product To Verify Delete","Price":20.00,"CategoryID":1,"Status":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)

    if [ "$CREATE_CODE" = "201" ]; then
        local VERIFY_ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$VERIFY_ID" ]; then
            # Delete the entity
            curl -g -s -o /dev/null -X DELETE "$SERVER_URL/Products($VERIFY_ID)"
            
            # Try to retrieve it
            local VERIFY_CODE=$(http_get "$SERVER_URL/Products($VERIFY_ID)")
            check_status "$VERIFY_CODE" "404"
        else
            echo "  Details: Could not create test entity"
            return 1
        fi
    else
        echo "  Details: Could not create test entity"
        return 1
    fi
}

run_test "Deleted entity returns 404 on GET" test_verify_deleted


print_summary
