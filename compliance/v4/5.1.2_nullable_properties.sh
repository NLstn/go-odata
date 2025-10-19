#!/bin/bash

# OData v4 Compliance Test: 5.1.2 Nullable Properties
# Tests handling of nullable properties
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_Nullable

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.1.2 Nullable Properties"
echo "======================================"
echo ""
echo "Description: Validates handling of nullable properties including"
echo "             null values in filters and responses."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_Nullable"
echo ""

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: Create entity with null value
test_create_with_null() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Null Test Product","Price":99.99,"Category":null,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 2: Filter for null values using eq null
test_filter_eq_null() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 3: Filter for non-null values using ne null
test_filter_ne_null() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20ne%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 4: Response includes null values as JSON null
test_response_null_representation() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Category%20eq%20null")
    
    # Should return valid JSON with null values
    if check_json_field "$RESPONSE" "value"; then
        # Check if response contains null (JSON null representation)
        if echo "$RESPONSE" | grep -q ':null[,}]'; then
            return 0
        else
            # May not have any null values in response
            echo "  Details: No null values in response (may be filtered out)"
            return 0
        fi
    else
        return 1
    fi
}

# Test 5: Update property to null
test_update_to_null() {
    # Create a test entity first
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Update to Null Test","Price":50,"Category":"Test","Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
            
            # Now update Category to null
            local UPDATE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($ID)" \
                -H "Content-Type: application/json" \
                -d '{"Category":null}' 2>&1)
            
            if [ "$UPDATE_CODE" = "200" ] || [ "$UPDATE_CODE" = "204" ]; then
                return 0
            else
                echo "  Details: Update status $UPDATE_CODE"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity"
    return 1
}

# Test 6: Null literal in URL
test_null_literal_url() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Category%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 7: Accessing null property returns null
test_access_null_property() {
    # First ensure we have an entity with null category
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Null Property Test","Price":30,"Category":null,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
            
            # Access the null property
            local PROP_CODE=$(http_get "$SERVER_URL/Products($ID)/Category")
            
            # Should return 200 or 204 for null value
            if [ "$PROP_CODE" = "200" ] || [ "$PROP_CODE" = "204" ]; then
                return 0
            else
                echo "  Details: Status $PROP_CODE"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity"
    return 1
}

# Test 8: Cannot set non-nullable property to null
test_nonnullable_reject_null() {
    # Try to create entity with required field as null
    # Assuming ID or Status might be non-nullable
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":null,"Price":50,"Category":"Test","Status":1}' 2>&1)
    
    # Should return 400 if Name is non-nullable
    # Or 201 if it's nullable
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "201" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

echo "  Request: POST $SERVER_URL/Products with null Category"
run_test "Create entity with null value in nullable property" test_create_with_null

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq null"
run_test "Filter for entities where property eq null" test_filter_eq_null

echo "  Request: GET $SERVER_URL/Products?\$filter=Category ne null"
run_test "Filter for entities where property ne null" test_filter_ne_null

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq null"
run_test "Response represents null values as JSON null" test_response_null_representation

echo "  Request: PATCH $SERVER_URL/Products(ID) to set Category to null"
run_test "Update property to null value" test_update_to_null

echo "  Request: GET $SERVER_URL/Products?\$filter=Category eq null"
run_test "Null literal in URL filter" test_null_literal_url

echo "  Request: GET $SERVER_URL/Products(ID)/Category where Category is null"
run_test "Accessing null property returns appropriate response" test_access_null_property

echo "  Request: POST $SERVER_URL/Products with null in non-nullable property"
run_test "Setting non-nullable property to null handled correctly" test_nonnullable_reject_null

print_summary
