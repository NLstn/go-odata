#!/bin/bash

# OData v4 Compliance Test: 5.1.2 Nullable Properties
# Tests handling of nullable properties
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_Nullable
#
# Key Requirements:
# - Properties marked as Nullable="true" in metadata MUST accept null values
# - Null values MUST be represented as JSON null in responses
# - Filters with "eq null" and "ne null" MUST work correctly
# - Attempting to set a Nullable="false" property to null MUST be rejected with 400
# - Accessing a null-valued property MUST return either:
#   - 204 No Content, or
#   - 200 OK with {"@odata.null":true} or {"value":null}

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


# Test 1: Create entity with null value in nullable property
test_create_with_null() {
    # LongText is a nullable property on ProductDescription
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/ProductDescriptions" \
        -H "Content-Type: application/json" \
        -d '{"ProductID":1,"LanguageKey":"IT","Description":"Italian description","LongText":null}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE, Body: $BODY"
        return 1
    fi
}

# Test 2: Filter for null values using eq null (using nullable property)
test_filter_eq_null() {
    # LongText is a nullable property
    local HTTP_CODE=$(http_get "$SERVER_URL/ProductDescriptions?\$filter=LongText%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 3: Filter for non-null values using ne null
test_filter_ne_null() {
    # LongText is a nullable property
    local HTTP_CODE=$(http_get "$SERVER_URL/ProductDescriptions?\$filter=LongText%20ne%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 4: Response includes null values as JSON null
test_response_null_representation() {
    # Create a ProductDescription with null LongText explicitly
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/ProductDescriptions" \
        -H "Content-Type: application/json" \
        -d '{"ProductID":2,"LanguageKey":"IT","Description":"Italian description","LongText":null}' 2>&1)
    
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        # ProductDescription has composite key: ProductID and LanguageKey
        local PRODUCT_ID=$(echo "$CREATE_BODY" | grep -o '"ProductID":[0-9]*' | head -1 | grep -o '[0-9]*')
        local LANG_KEY=$(echo "$CREATE_BODY" | grep -o '"LanguageKey":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        if [ -n "$PRODUCT_ID" ] && [ -n "$LANG_KEY" ]; then
            # Get the entity and verify LongText is represented as null
            local GET_RESPONSE=$(http_get_body "$SERVER_URL/ProductDescriptions(ProductID=$PRODUCT_ID,LanguageKey='$LANG_KEY')")
            
            # Should contain "LongText":null
            if echo "$GET_RESPONSE" | grep -q '"LongText":null'; then
                return 0
            else
                echo "  Details: LongText not represented as null in JSON response"
                echo "  Response: $GET_RESPONSE"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity with null LongText"
    return 1
}

# Test 5: Update property to null
test_update_to_null() {
    # Create a test entity with non-null LongText
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/ProductDescriptions" \
        -H "Content-Type: application/json" \
        -d '{"ProductID":3,"LanguageKey":"IT","Description":"Italian description","LongText":"Some long text"}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local PRODUCT_ID=$(echo "$BODY" | grep -o '"ProductID":[0-9]*' | head -1 | grep -o '[0-9]*')
        local LANG_KEY=$(echo "$BODY" | grep -o '"LanguageKey":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        if [ -n "$PRODUCT_ID" ] && [ -n "$LANG_KEY" ]; then
            # Now update LongText to null
            local UPDATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$SERVER_URL/ProductDescriptions(ProductID=$PRODUCT_ID,LanguageKey='$LANG_KEY')" \
                -H "Content-Type: application/json" \
                -d '{"LongText":null}' 2>&1)
            
            local UPDATE_CODE=$(echo "$UPDATE_RESPONSE" | tail -1)
            local UPDATE_BODY=$(echo "$UPDATE_RESPONSE" | head -n -1)
            
            if [ "$UPDATE_CODE" = "200" ] || [ "$UPDATE_CODE" = "204" ]; then
                return 0
            else
                echo "  Details: Update status $UPDATE_CODE, Body: $UPDATE_BODY"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity, Status: $HTTP_CODE"
    return 1
}

# Test 6: Null literal in URL (using nullable property)
test_null_literal_url() {
    # LongText is nullable
    local HTTP_CODE=$(http_get "$SERVER_URL/ProductDescriptions?\$filter=LongText%20eq%20null")
    
    check_status "$HTTP_CODE" "200"
}

# Test 7: Accessing null property returns null
test_access_null_property() {
    # Create an entity with null LongText
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/ProductDescriptions" \
        -H "Content-Type: application/json" \
        -d '{"ProductID":5,"LanguageKey":"IT","Description":"Italian description","LongText":null}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local PRODUCT_ID=$(echo "$BODY" | grep -o '"ProductID":[0-9]*' | head -1 | grep -o '[0-9]*')
        local LANG_KEY=$(echo "$BODY" | grep -o '"LanguageKey":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        if [ -n "$PRODUCT_ID" ] && [ -n "$LANG_KEY" ]; then
            # Access the null property
            local PROP_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/ProductDescriptions(ProductID=$PRODUCT_ID,LanguageKey='$LANG_KEY')/LongText" 2>&1)
            local PROP_CODE=$(echo "$PROP_RESPONSE" | tail -1)
            local PROP_BODY=$(echo "$PROP_RESPONSE" | head -n -1)
            
            # Per OData spec, accessing a null property should return 204 No Content
            # or 200 with {"@odata.null":true} or {"value":null}
            if [ "$PROP_CODE" = "204" ]; then
                return 0
            elif [ "$PROP_CODE" = "200" ]; then
                # Check if body represents null properly
                if echo "$PROP_BODY" | grep -q -E '("@odata\.null"\s*:\s*true|"value"\s*:\s*null)'; then
                    return 0
                else
                    echo "  Details: Status 200 but body doesn't properly represent null: $PROP_BODY"
                    return 1
                fi
            else
                echo "  Details: Status $PROP_CODE, Body: $PROP_BODY"
                return 1
            fi
        fi
    fi
    
    echo "  Details: Could not create test entity, Status: $HTTP_CODE"
    return 1
}

# Test 8: Cannot set non-nullable property to null
test_nonnullable_reject_null() {
    # Try to create entity with required non-nullable field as null
    # Description is non-nullable (Nullable="false" in metadata)
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/ProductDescriptions" \
        -H "Content-Type: application/json" \
        -d '{"ProductID":1,"LanguageKey":"ES","Description":null,"LongText":"Some text"}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Should return 400 Bad Request since Description is non-nullable
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Expected 400 for null in non-nullable property, got $HTTP_CODE"
        echo "  Body: $BODY"
        return 1
    fi
}

echo "  Request: POST $SERVER_URL/ProductDescriptions with null LongText (nullable property)"
run_test "Create entity with null value in nullable property" test_create_with_null

echo "  Request: GET $SERVER_URL/ProductDescriptions?\$filter=LongText eq null"
run_test "Filter for entities where property eq null" test_filter_eq_null

echo "  Request: GET $SERVER_URL/ProductDescriptions?\$filter=LongText ne null"
run_test "Filter for entities where property ne null" test_filter_ne_null

echo "  Request: GET $SERVER_URL/ProductDescriptions(ProductID,LanguageKey) and verify LongText:null in response"
run_test "Response represents null values as JSON null" test_response_null_representation

echo "  Request: PATCH $SERVER_URL/ProductDescriptions(ProductID,LanguageKey) to set LongText to null"
run_test "Update property to null value" test_update_to_null

echo "  Request: GET $SERVER_URL/ProductDescriptions?\$filter=LongText eq null"
run_test "Null literal in URL filter" test_null_literal_url

echo "  Request: GET $SERVER_URL/ProductDescriptions(ProductID,LanguageKey)/LongText where LongText is null"
run_test "Accessing null property returns appropriate response" test_access_null_property

echo "  Request: POST $SERVER_URL/ProductDescriptions with null in non-nullable property (Description)"
run_test "Setting non-nullable property to null handled correctly" test_nonnullable_reject_null

print_summary
