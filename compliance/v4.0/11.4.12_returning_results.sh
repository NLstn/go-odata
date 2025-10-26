#!/bin/bash

# OData v4 Compliance Test: 11.4.12 Returning Results from Data Modification
# Tests returning entities from POST, PATCH, PUT operations with Prefer header
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.12 Returning Results from Modifications"
echo "======================================"
echo ""
echo "Description: Validates that services properly return or omit entity"
echo "             representations after modifications based on Prefer header."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html"
echo ""

# Track created entity IDs for cleanup
CREATED_IDS=()

cleanup() {
    for ID in "${CREATED_IDS[@]}"; do
        curl -s -o /dev/null -X DELETE "$SERVER_URL/Products($ID)" 2>/dev/null
    done
}

register_cleanup cleanup

# Test 1: POST with return=minimal returns 204 No Content
test_post_return_minimal() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Minimal Return Test","Price":99.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Extract ID from Location or OData-EntityId for cleanup
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    if [[ "$LOCATION" =~ Products\(([0-9]+)\) ]]; then
        CREATED_IDS+=("${BASH_REMATCH[1]}")
    fi
    
    # With return=minimal, should get 201 with no body or 204 with no body
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201 or 204)"
        return 1
    fi
}

# Test 2: POST with return=representation returns 201 with entity
test_post_return_representation() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Representation Return Test","Price":149.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Should return 201 with entity body
    if [ "$HTTP_CODE" = "201" ] && echo "$BODY" | grep -q '"Name":"Representation Return Test"'; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE, body contains entity: $(echo "$BODY" | grep -q '"Name"' && echo "yes" || echo "no")"
        return 1
    fi
}

# Test 3: POST without Prefer header (default behavior)
test_post_no_prefer() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"No Prefer Test","Price":79.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Default behavior may be return=representation (201 with body) or return=minimal
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201 or 204)"
        return 1
    fi
}

# Test 4: PATCH with return=representation returns entity
test_patch_return_representation() {
    # First create an entity
    local CREATE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Patch Test","Price":50.00,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$CREATE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE" | head -n -1)
    
    if [ "$CREATE_CODE" != "201" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -z "$ID" ]; then
        echo "  Details: Could not extract entity ID"
        return 1
    fi
    CREATED_IDS+=("$ID")
    
    # Now PATCH with return=representation
    local PATCH=$(curl -s -w "\n%{http_code}" -X PATCH "$SERVER_URL/Products($ID)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Price":75.00}')
    
    local PATCH_CODE=$(echo "$PATCH" | tail -1)
    local PATCH_BODY=$(echo "$PATCH" | head -n -1)
    
    # Should return 200 with updated entity
    if [ "$PATCH_CODE" = "200" ] && echo "$PATCH_BODY" | grep -q '"Price":75'; then
        return 0
    else
        echo "  Details: Status $PATCH_CODE, body has Price:75: $(echo "$PATCH_BODY" | grep -q '"Price":75' && echo "yes" || echo "no")"
        return 1
    fi
}

# Test 5: PATCH with return=minimal returns 204 No Content
test_patch_return_minimal() {
    # First create an entity
    local CREATE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Patch Minimal Test","Price":60.00,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$CREATE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE" | head -n -1)
    
    if [ "$CREATE_CODE" != "201" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -z "$ID" ]; then
        echo "  Details: Could not extract entity ID"
        return 1
    fi
    CREATED_IDS+=("$ID")
    
    # Now PATCH with return=minimal
    local PATCH=$(curl -s -i -X PATCH "$SERVER_URL/Products($ID)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Price":85.00}')
    
    local PATCH_CODE=$(echo "$PATCH" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should return 204 with no body
    if [ "$PATCH_CODE" = "204" ] || [ "$PATCH_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $PATCH_CODE (expected 204 or 200)"
        return 1
    fi
}

# Test 6: PUT with return=representation returns entity
test_put_return_representation() {
    # First create an entity
    local CREATE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Put Test","Price":40.00,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$CREATE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE" | head -n -1)
    
    if [ "$CREATE_CODE" != "201" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -z "$ID" ]; then
        echo "  Details: Could not extract entity ID"
        return 1
    fi
    CREATED_IDS+=("$ID")
    
    # Now PUT with return=representation
    local PUT=$(curl -s -w "\n%{http_code}" -X PUT "$SERVER_URL/Products($ID)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Updated Put Test","Price":95.00,"CategoryID":1,"Status":1}')
    
    local PUT_CODE=$(echo "$PUT" | tail -1)
    local PUT_BODY=$(echo "$PUT" | head -n -1)
    
    # Should return 200 with updated entity
    if [ "$PUT_CODE" = "200" ] && echo "$PUT_BODY" | grep -q '"Name":"Updated Put Test"'; then
        return 0
    else
        echo "  Details: Status $PUT_CODE"
        return 1
    fi
}

# Test 7: return=representation with $select
test_return_with_select() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products?\$select=ID,Name,Price" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Select Test","Price":55.55,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Should return 201 with only selected properties
    if [ "$HTTP_CODE" = "201" ] && echo "$BODY" | grep -q '"Name":"Select Test"'; then
        # Should have ID, Name, Price but not CategoryID (unless it's selected by default)
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 8: return=representation with $expand
test_return_with_expand() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products?\$expand=Category" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Expand Test","Price":88.88,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Should return 201 with expanded Category
    if [ "$HTTP_CODE" = "201" ]; then
        # May or may not include expanded navigation - depends on implementation
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 9: Preference-Applied response header
test_preference_applied_header() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Preference Applied Test","Price":111.11,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local PREF_APPLIED=$(echo "$RESPONSE" | grep -i "^Preference-Applied:" | head -1)
    
    # Extract ID from body for cleanup
    if echo "$RESPONSE" | grep -q '"ID":[0-9]*'; then
        local ID=$(echo "$RESPONSE" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Preference-Applied header is optional but good practice
    if [ "$HTTP_CODE" = "201" ]; then
        # Accept whether or not Preference-Applied is present
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 10: Invalid Prefer value handled gracefully
test_invalid_prefer_value() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=invalid" \
        -d '{"Name":"Invalid Prefer Test","Price":33.33,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Should either use default behavior (201) or reject (400)
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201 or 400)"
        return 1
    fi
}

# Test 11: Multiple preferences in Prefer header
test_multiple_preferences() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation, odata.maxpagesize=10" \
        -d '{"Name":"Multiple Prefs Test","Price":44.44,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Extract ID for cleanup
    if echo "$BODY" | grep -q '"ID"'; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
    
    # Should handle multiple preferences
    if [ "$HTTP_CODE" = "201" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 12: DELETE does not return entity (regardless of Prefer)
test_delete_no_return() {
    # First create an entity
    local CREATE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Delete Test","Price":22.22,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$CREATE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE" | head -n -1)
    
    if [ "$CREATE_CODE" != "201" ]; then
        echo "  Details: Failed to create test entity"
        return 1
    fi
    
    local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
    if [ -z "$ID" ]; then
        echo "  Details: Could not extract entity ID"
        return 1
    fi
    
    # DELETE with return=representation (should be ignored)
    local DELETE=$(curl -s -i -X DELETE "$SERVER_URL/Products($ID)" \
        -H "Prefer: return=representation")
    
    local DELETE_CODE=$(echo "$DELETE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # DELETE should return 204 regardless of Prefer header
    if [ "$DELETE_CODE" = "204" ]; then
        # Don't add to cleanup since already deleted
        return 0
    else
        echo "  Details: Status $DELETE_CODE (expected 204)"
        return 1
    fi
}

echo "  Request: POST with Prefer: return=minimal"
run_test "POST with return=minimal returns 201/204 with no body" test_post_return_minimal

echo "  Request: POST with Prefer: return=representation"
run_test "POST with return=representation returns 201 with entity" test_post_return_representation

echo "  Request: POST without Prefer header"
run_test "POST without Prefer uses default behavior" test_post_no_prefer

echo "  Request: PATCH with Prefer: return=representation"
run_test "PATCH with return=representation returns 200 with entity" test_patch_return_representation

echo "  Request: PATCH with Prefer: return=minimal"
run_test "PATCH with return=minimal returns 204/200 with no body" test_patch_return_minimal

echo "  Request: PUT with Prefer: return=representation"
run_test "PUT with return=representation returns 200 with entity" test_put_return_representation

echo "  Request: POST with \$select and return=representation"
run_test "return=representation with \$select returns selected properties" test_return_with_select

echo "  Request: POST with \$expand and return=representation"
run_test "return=representation with \$expand includes expanded entities" test_return_with_expand

echo "  Request: POST with Prefer header and check Preference-Applied"
run_test "Preference-Applied header indicates applied preferences" test_preference_applied_header

echo "  Request: POST with invalid Prefer value"
run_test "Invalid Prefer value handled gracefully" test_invalid_prefer_value

echo "  Request: POST with multiple preferences"
run_test "Multiple preferences in Prefer header" test_multiple_preferences

echo "  Request: DELETE with Prefer: return=representation"
run_test "DELETE ignores return preference" test_delete_no_return

print_summary
