#!/bin/bash

# OData v4 Compliance Test: 11.4.4 Delete an Entity
# Tests DELETE operations according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_DeleteanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.4 Delete an Entity"
echo "======================================"
echo ""
echo "Description: Validates DELETE operations for removing entities"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_DeleteanEntity"
echo ""



# Test 1: DELETE returns 204 No Content on success
echo "Test 1: DELETE entity returns 204 No Content"
echo "  Setting up: Creating entity to delete..."
CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -d '{"Name":"Product To Delete","Price":10.00,"Category":"Test","Status":1}' 2>&1)
CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)

if [ "$CREATE_CODE" = "201" ]; then
    DELETE_ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -n "$DELETE_ID" ]; then
        echo "  Request: DELETE $SERVER_URL/Products($DELETE_ID)"
        DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$SERVER_URL/Products($DELETE_ID)" 2>&1)
        DELETE_CODE=$(echo "$DELETE_RESPONSE" | tail -1)
        
        if [ "$DELETE_CODE" = "204" ]; then
            test_result "DELETE returns 204 No Content" "PASS"
        else
            test_result "DELETE returns 204 No Content" "FAIL" "Status code: $DELETE_CODE"
        fi
    else
        test_result "DELETE returns 204 No Content" "FAIL" "Could not create test entity"
    fi
else
    test_result "DELETE returns 204 No Content" "FAIL" "Could not create test entity (status: $CREATE_CODE)"
fi
echo ""

# Test 2: DELETE to non-existent entity returns 404
echo "Test 2: DELETE to non-existent entity returns 404"
echo "  Request: DELETE $SERVER_URL/Products(999999)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$SERVER_URL/Products(999999)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "404" ]; then
    test_result "DELETE to non-existent entity returns 404" "PASS"
else
    test_result "DELETE to non-existent entity returns 404" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Verify entity is actually deleted
echo "Test 3: Verify deleted entity cannot be retrieved"
echo "  Setting up: Creating entity to delete..."
CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -d '{"Name":"Product To Verify Delete","Price":20.00,"Category":"Test","Status":1}' 2>&1)
CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)

if [ "$CREATE_CODE" = "201" ]; then
    VERIFY_ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -n "$VERIFY_ID" ]; then
        # Delete the entity
        curl -s -X DELETE "$SERVER_URL/Products($VERIFY_ID)" > /dev/null 2>&1
        
        # Try to retrieve it
        echo "  Request: GET $SERVER_URL/Products($VERIFY_ID) (should return 404)"
        VERIFY_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products($VERIFY_ID)" 2>&1)
        VERIFY_CODE=$(echo "$VERIFY_RESPONSE" | tail -1)
        
        if [ "$VERIFY_CODE" = "404" ]; then
            test_result "Deleted entity returns 404 on GET" "PASS"
        else
            test_result "Deleted entity returns 404 on GET" "FAIL" "Status code: $VERIFY_CODE (expected 404)"
        fi
    else
        test_result "Deleted entity returns 404 on GET" "FAIL" "Could not create test entity"
    fi
else
    test_result "Deleted entity returns 404 on GET" "FAIL" "Could not create test entity"
fi
echo ""


print_summary
