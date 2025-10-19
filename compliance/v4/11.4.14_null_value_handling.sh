#!/bin/bash

# OData v4 Compliance Test: 11.4.14 Null Value Handling
# Tests null value handling in requests and responses
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.14 Null Value Handling"
echo "======================================"
echo ""
echo "Description: Validates that the service properly handles null values"
echo "             in entity creation, updates, and filtering."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html"
echo ""

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: Create entity with null property
test_create_with_null() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Null Test Product","Price":99.99,"Category":"Test","Description":null,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 2: Retrieve entity with null property
test_retrieve_null_property() {
    if [ ${#CREATED_IDS[@]} -gt 0 ]; then
        local ID=${CREATED_IDS[0]}
        local RESPONSE=$(curl -s "$SERVER_URL/Products($ID)")
        
        # Check that Description is either absent or null
        if echo "$RESPONSE" | grep -q '"Description":null' || ! echo "$RESPONSE" | grep -q '"Description"'; then
            return 0
        else
            echo "  Details: Expected null or absent Description property"
            return 1
        fi
    else
        echo "  Details: No test entity available"
        return 1
    fi
}

# Test 3: Update property to null using PATCH
test_patch_to_null() {
    if [ ${#CREATED_IDS[@]} -gt 0 ]; then
        local ID=${CREATED_IDS[0]}
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($ID)" \
            -H "Content-Type: application/json" \
            -d '{"Description":null}')
        
        if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
            return 0
        else
            echo "  Details: Status $HTTP_CODE (expected 204 or 200)"
            return 1
        fi
    else
        echo "  Details: No test entity available"
        return 1
    fi
}

# Test 4: Filter for null values using eq null
test_filter_eq_null() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Description%20eq%20null")
    check_status "$HTTP_CODE" "200"
}

# Test 5: Filter for non-null values using ne null
test_filter_ne_null() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Description%20ne%20null")
    check_status "$HTTP_CODE" "200"
}

# Test 6: PUT with null property
test_put_with_null() {
    if [ ${#CREATED_IDS[@]} -gt 0 ]; then
        local ID=${CREATED_IDS[0]}
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$SERVER_URL/Products($ID)" \
            -H "Content-Type: application/json" \
            -d '{"Name":"Updated Product","Price":149.99,"Category":"Test","Description":null,"Status":1}')
        
        if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
            return 0
        else
            echo "  Details: Status $HTTP_CODE (expected 204 or 200)"
            return 1
        fi
    else
        echo "  Details: No test entity available"
        return 1
    fi
}

# Test 7: JSON null vs string "null"
test_null_vs_string_null() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"String Null Test","Price":50,"Category":"Test","Description":"null","Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
            # Verify that Description is the string "null" not null
            if echo "$BODY" | grep -q '"Description":"null"'; then
                return 0
            else
                echo "  Details: Expected Description to be string 'null'"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Failed to create entity"
    return 1
}

# Test 8: Select null properties
test_select_null_property() {
    if [ ${#CREATED_IDS[@]} -gt 0 ]; then
        local ID=${CREATED_IDS[0]}
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products($ID)?\$select=Description")
        check_status "$HTTP_CODE" "200"
    else
        echo "  Details: No test entity available"
        return 1
    fi
}

echo "  Request: POST with null property"
run_test "Create entity with explicit null property" test_create_with_null

echo "  Request: GET entity with null property"
run_test "Retrieve entity returns null property correctly" test_retrieve_null_property

echo "  Request: PATCH to set property to null"
run_test "Update property to null using PATCH" test_patch_to_null

echo "  Request: GET with \$filter=Description eq null"
run_test "Filter for null values using eq null" test_filter_eq_null

echo "  Request: GET with \$filter=Description ne null"
run_test "Filter for non-null values using ne null" test_filter_ne_null

echo "  Request: PUT with null property"
run_test "Replace entity with null property using PUT" test_put_with_null

echo "  Request: POST with string 'null' vs JSON null"
run_test "Distinguish between JSON null and string 'null'" test_null_vs_string_null

echo "  Request: GET with \$select on null property"
run_test "Select null property explicitly" test_select_null_property

print_summary
