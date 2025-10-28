#!/bin/bash

# OData v4 Compliance Test: 11.4.2.1 @odata.bind Annotation
# Tests binding navigation properties using @odata.bind annotation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BindingaNavigationProperty

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.2.1 @odata.bind Annotation"
echo "======================================"
echo ""
echo "Description: Tests the @odata.bind annotation for binding navigation properties"
echo "             when creating and updating entities. This allows setting relationships"
echo "             by reference rather than embedding full entity details."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BindingaNavigationProperty"
echo ""

# Track created IDs for cleanup
CREATED_PRODUCT_IDS=()
CREATED_CATEGORY_IDS=()

# Cleanup function
cleanup() {
    for ID in "${CREATED_PRODUCT_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($ID)" > /dev/null 2>&1
    done
    for ID in "${CREATED_CATEGORY_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Categories($ID)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: POST with @odata.bind using relative URL (single-valued navigation property)
test_bind_post_relative_url() {
    local PAYLOAD='{"Name":"BindTestProduct1","Price":99.99,"Category@odata.bind":"Categories(1)"}'
    local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
    
    # Check if product was created
    if echo "$BODY" | grep -q '"ID"' && echo "$BODY" | grep -q '"Name"[[:space:]]*:[[:space:]]*"BindTestProduct1"'; then
        # Verify CategoryID was set to 1
        if echo "$BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*1'; then
            local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
            [ -n "$ID" ] && CREATED_PRODUCT_IDS+=("$ID")
            return 0
        fi
    fi
    return 1
}

# Test 2: POST with @odata.bind using absolute URL (single-valued navigation property)
test_bind_post_absolute_url() {
    local PAYLOAD='{"Name":"BindTestProduct2","Price":199.99,"Category@odata.bind":"'$SERVER_URL'/Categories(1)"}'
    local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
    
    # Check if product was created
    if echo "$BODY" | grep -q '"ID"' && echo "$BODY" | grep -q '"Name"[[:space:]]*:[[:space:]]*"BindTestProduct2"'; then
        # Verify CategoryID was set to 1
        if echo "$BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*1'; then
            local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
            [ -n "$ID" ] && CREATED_PRODUCT_IDS+=("$ID")
            return 0
        fi
    fi
    return 1
}

# Test 3: POST with @odata.bind referencing non-existent entity should fail
test_bind_post_nonexistent() {
    local PAYLOAD='{"Name":"BindTestProduct3","Price":299.99,"Category@odata.bind":"Categories(999999)"}'
    local HTTP_CODE=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json" -o /dev/null -w "%{http_code}")
    
    # Should return 400 Bad Request
    [ "$HTTP_CODE" = "400" ]
}

# Test 4: POST with @odata.bind referencing wrong entity set should fail
test_bind_post_wrong_entity_set() {
    local PAYLOAD='{"Name":"BindTestProduct4","Price":399.99,"Category@odata.bind":"Products(1)"}'
    local HTTP_CODE=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json" -o /dev/null -w "%{http_code}")
    
    # Should return 400 Bad Request
    [ "$HTTP_CODE" = "400" ]
}

# Test 5: POST with invalid @odata.bind format should fail
test_bind_post_invalid_format() {
    local PAYLOAD='{"Name":"BindTestProduct5","Price":499.99,"Category@odata.bind":"InvalidFormat"}'
    local HTTP_CODE=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json" -o /dev/null -w "%{http_code}")
    
    # Should return 400 Bad Request
    [ "$HTTP_CODE" = "400" ]
}

# Test 6: PATCH with @odata.bind to update navigation property
test_bind_patch_update() {
    # First create a product with Category 1
    local CREATE_PAYLOAD='{"Name":"BindTestProduct6","Price":599.99,"Category@odata.bind":"Categories(1)"}'
    local CREATE_BODY=$(http_post "$SERVER_URL/Products" "$CREATE_PAYLOAD" -H "Content-Type: application/json")
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
    
    if [ -z "$ID" ]; then
        return 1
    fi
    
    CREATED_PRODUCT_IDS+=("$ID")
    
    # Now update to Category 2 using @odata.bind
    local UPDATE_PAYLOAD='{"Category@odata.bind":"Categories(2)"}'
    http_patch "$SERVER_URL/Products($ID)" "$UPDATE_PAYLOAD" -H "Content-Type: application/json" > /dev/null
    
    # Fetch the product and verify CategoryID is 2
    local UPDATED_BODY=$(http_get_body "$SERVER_URL/Products($ID)")
    
    if echo "$UPDATED_BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*2'; then
        return 0
    fi
    return 1
}

# Test 7: PATCH with @odata.bind and other properties together
test_bind_patch_mixed() {
    # First create a product
    local CREATE_PAYLOAD='{"Name":"BindTestProduct7","Price":699.99,"Category@odata.bind":"Categories(1)"}'
    local CREATE_BODY=$(http_post "$SERVER_URL/Products" "$CREATE_PAYLOAD" -H "Content-Type: application/json")
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
    
    if [ -z "$ID" ]; then
        return 1
    fi
    
    CREATED_PRODUCT_IDS+=("$ID")
    
    # Update both price and category
    local UPDATE_PAYLOAD='{"Price":799.99,"Category@odata.bind":"Categories(2)"}'
    http_patch "$SERVER_URL/Products($ID)" "$UPDATE_PAYLOAD" -H "Content-Type: application/json" > /dev/null
    
    # Fetch the product and verify both updates
    local UPDATED_BODY=$(http_get_body "$SERVER_URL/Products($ID)")
    
    if echo "$UPDATED_BODY" | grep -q '"Price"[[:space:]]*:[[:space:]]*799.99' && \
       echo "$UPDATED_BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*2'; then
        return 0
    fi
    return 1
}

# Test 8: POST with @odata.bind to non-existent navigation property should fail
test_bind_post_invalid_nav_prop() {
    local PAYLOAD='{"Name":"BindTestProduct8","Price":899.99,"NonExistentNav@odata.bind":"Categories(1)"}'
    local HTTP_CODE=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json" -o /dev/null -w "%{http_code}")
    
    # Should return 400 Bad Request
    [ "$HTTP_CODE" = "400" ]
}

# Test 9: POST with @odata.bind and direct foreign key should use @odata.bind
test_bind_post_precedence() {
    # If both @odata.bind and direct foreign key are provided, @odata.bind should take precedence
    local PAYLOAD='{"Name":"BindTestProduct9","Price":999.99,"CategoryID":1,"Category@odata.bind":"Categories(2)"}'
    local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
    
    # Check if product was created
    if echo "$BODY" | grep -q '"ID"' && echo "$BODY" | grep -q '"Name"[[:space:]]*:[[:space:]]*"BindTestProduct9"'; then
        # Verify CategoryID was set to 2 (from @odata.bind, not from direct assignment)
        if echo "$BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*2'; then
            local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
            [ -n "$ID" ] && CREATED_PRODUCT_IDS+=("$ID")
            return 0
        fi
    fi
    return 1
}

# Test 10: Verify @odata.bind works with root-relative URLs
test_bind_post_root_relative() {
    local PAYLOAD='{"Name":"BindTestProduct10","Price":1099.99,"Category@odata.bind":"/Categories(1)"}'
    local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
    
    # Check if product was created
    if echo "$BODY" | grep -q '"ID"' && echo "$BODY" | grep -q '"Name"[[:space:]]*:[[:space:]]*"BindTestProduct10"'; then
        # Verify CategoryID was set to 1
        if echo "$BODY" | grep -q '"CategoryID"[[:space:]]*:[[:space:]]*1'; then
            local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
            [ -n "$ID" ] && CREATED_PRODUCT_IDS+=("$ID")
            return 0
        fi
    fi
    return 1
}

# Run all tests
run_test "POST with @odata.bind (relative URL)" test_bind_post_relative_url
run_test "POST with @odata.bind (absolute URL)" test_bind_post_absolute_url
run_test "POST with @odata.bind to non-existent entity returns 400" test_bind_post_nonexistent
run_test "POST with @odata.bind to wrong entity set returns 400" test_bind_post_wrong_entity_set
run_test "POST with invalid @odata.bind format returns 400" test_bind_post_invalid_format
run_test "PATCH with @odata.bind updates navigation property" test_bind_patch_update
run_test "PATCH with @odata.bind and other properties" test_bind_patch_mixed
run_test "POST with @odata.bind to invalid navigation property returns 400" test_bind_post_invalid_nav_prop
run_test "POST with @odata.bind takes precedence over direct foreign key" test_bind_post_precedence
run_test "POST with @odata.bind (root-relative URL)" test_bind_post_root_relative

print_summary
