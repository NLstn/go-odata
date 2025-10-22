#!/bin/bash

# OData v4 Compliance Test: 8.2.5 Location Header
# Tests that Location header is properly set for resource creation
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderLocation

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.5 Location Header"
echo "======================================"
echo ""
echo "Description: Validates that the Location header is correctly returned"
echo "             when creating new resources (201 Created responses)."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderLocation"
echo ""

# Track created entity IDs for cleanup
CREATED_IDS=()

cleanup() {
    for ID in "${CREATED_IDS[@]}"; do
        curl -s -o /dev/null -X DELETE "$SERVER_URL/Products($ID)" 2>/dev/null
    done
}

register_cleanup cleanup

# Test 1: POST entity returns Location header
test_location_on_create() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Location Test Product","Price":99.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID from Location header for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # With return=minimal, should get 201 or 204, and Location header should be present
    if ([ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]) && [ -n "$LOCATION" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE, Location='$LOCATION'"
        return 1
    fi
}

# Test 2: Location header points to correct resource
test_location_dereferenceable() {
    local CREATE_RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Dereference Test","Price":79.99,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$CREATE_RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    if [ -z "$LOCATION" ]; then
        echo "  Details: No Location header found"
        return 1
    fi
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # Try to GET the Location URL
    local GET_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$LOCATION")
    
    if [ "$GET_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Location '$LOCATION' returned status $GET_CODE"
        return 1
    fi
}

# Test 3: Location header format with canonical URL
test_location_format() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Format Test","Price":59.99,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # Location should contain the entity set name and key
    if echo "$LOCATION" | grep -q "Products([0-9]*)" && echo "$LOCATION" | grep -q "^http"; then
        return 0
    else
        echo "  Details: Location format invalid: '$LOCATION'"
        return 1
    fi
}

# Test 4: Location header with return=representation
test_location_with_representation() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Representation Test","Price":149.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # Location should be present even with return=representation
    if [ "$HTTP_CODE" = "201" ] && [ -n "$LOCATION" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE, Location='$LOCATION'"
        return 1
    fi
}

# Test 5: Location header with composite keys (if applicable)
test_location_composite_key() {
    # This test checks if composite keys are properly formatted in Location
    # Skip if no composite key entities exist
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Composite Key Test","Price":39.99,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # For single key, should be in format EntitySet(key)
    if echo "$LOCATION" | grep -q "Products([0-9]*)" ; then
        return 0
    else
        echo "  Details: Unexpected Location format: '$LOCATION'"
        return 1
    fi
}

# Test 6: No Location header on PATCH (update)
test_no_location_on_patch() {
    # First create an entity
    local CREATE_RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Patch Test","Price":88.88,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$CREATE_RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    if [ -z "$LOCATION" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        local ID="${BASH_REMATCH[1]}"
        CREATED_IDS+=("$ID")
        
        # Try PATCH - should not return Location
        local PATCH_RESPONSE=$(curl -s -i -X PATCH "$SERVER_URL/Products($ID)" \
            -H "Content-Type: application/json" \
            -d '{"Price":77.77}')
        
        local PATCH_LOCATION=$(echo "$PATCH_RESPONSE" | grep -i "^Location:" | head -1)
        
        # Location should not be present for PATCH
        if [ -z "$PATCH_LOCATION" ]; then
            return 0
        else
            echo "  Details: Unexpected Location header on PATCH"
            return 1
        fi
    else
        echo "  Details: Could not extract ID from Location"
        return 1
    fi
}

# Test 7: No Location header on PUT (replace)
test_no_location_on_put() {
    # First create an entity
    local CREATE_RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Put Test","Price":66.66,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$CREATE_RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    if [ -z "$LOCATION" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        local ID="${BASH_REMATCH[1]}"
        CREATED_IDS+=("$ID")
        
        # Try PUT - should not return Location
        local PUT_RESPONSE=$(curl -s -i -X PUT "$SERVER_URL/Products($ID)" \
            -H "Content-Type: application/json" \
            -d '{"Name":"Updated","Price":55.55,"CategoryID":1,"Status":1}')
        
        local PUT_LOCATION=$(echo "$PUT_RESPONSE" | grep -i "^Location:" | head -1)
        
        # Location should not be present for PUT (update, not upsert creating new entity)
        if [ -z "$PUT_LOCATION" ]; then
            return 0
        else
            echo "  Details: Unexpected Location header on PUT (update)"
            return 1
        fi
    else
        echo "  Details: Could not extract ID from Location"
        return 1
    fi
}

# Test 8: Location header absolute URL format
test_location_absolute_url() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Absolute URL Test","Price":123.45,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # Location should be an absolute URL (starts with http:// or https://)
    if echo "$LOCATION" | grep -qE "^https?://"; then
        return 0
    else
        echo "  Details: Location is not an absolute URL: '$LOCATION'"
        return 1
    fi
}

# Test 9: Location consistency with OData-EntityId
test_location_entityid_consistency() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Consistency Test","Price":111.11,"CategoryID":1,"Status":1}')
    
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    local ENTITY_ID=$(echo "$RESPONSE" | grep -i "^OData-EntityId:" | head -1 | sed 's/OData-EntityId: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # If both are present, they should point to the same resource
    if [ -n "$LOCATION" ]; then
        if [ -z "$ENTITY_ID" ] || [ "$LOCATION" = "$ENTITY_ID" ]; then
            return 0
        else
            echo "  Details: Location '$LOCATION' != OData-EntityId '$ENTITY_ID'"
            return 1
        fi
    else
        echo "  Details: No Location header"
        return 1
    fi
}

# Test 10: Deep insert with Location
test_location_deep_insert() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Categories" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Deep Insert Cat","Products":[{"Name":"Related Product","Price":22.22,"Status":1}]}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    # Extract ID for cleanup (Category)
    if [[ "$LOCATION" =~ Categories\(([0-9]+)\) ]]; then
        # Note: Deep insert cleanup is complex; assuming cascade delete or accepting orphaned records
        # In production, proper cleanup would delete both category and related products
        :
    fi
    
    # Location should point to the main created entity (Category)
    # Accept both 201 and 204 status codes
    if ([ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]) && echo "$LOCATION" | grep -q "Categories([0-9]*)"; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE, Location='$LOCATION'"
        return 1
    fi
}

echo "  Request: POST /Products with Prefer: return=minimal"
run_test "POST returns Location header with 201 Created" test_location_on_create

echo "  Request: GET Location URL from previous POST"
run_test "Location header URL is dereferenceable" test_location_dereferenceable

echo "  Request: POST /Products and verify Location format"
run_test "Location header format includes entity set and key" test_location_format

echo "  Request: POST /Products with Prefer: return=representation"
run_test "Location header present with return=representation" test_location_with_representation

echo "  Request: POST /Products and check key format"
run_test "Location header with proper key format" test_location_composite_key

echo "  Request: PATCH /Products(ID)"
run_test "PATCH does not return Location header" test_no_location_on_patch

echo "  Request: PUT /Products(ID) (update)"
run_test "PUT (update) does not return Location header" test_no_location_on_put

echo "  Request: POST /Products and check URL format"
run_test "Location header is absolute URL" test_location_absolute_url

echo "  Request: POST /Products and compare Location with OData-EntityId"
run_test "Location consistent with OData-EntityId header" test_location_entityid_consistency

echo "  Request: POST /Categories with nested Products"
run_test "Deep insert returns Location for main entity" test_location_deep_insert

print_summary
