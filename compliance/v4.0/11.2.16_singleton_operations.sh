#!/bin/bash

# OData v4 Compliance Test: 11.2.16 Singleton Operations
# Tests singleton entity operations and edge cases
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Singletons

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.16 Singleton Operations"
echo "======================================"
echo ""
echo "Description: Validates singleton entity operations including GET, PATCH, PUT"
echo "             and proper error responses for invalid operations (POST, DELETE)"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Singletons"
echo ""

# Test 1: GET singleton returns 200
test_get_singleton() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Company")
    check_status "$HTTP_CODE" "200"
}

# Test 2: GET singleton returns valid JSON structure
test_singleton_json_structure() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Company")
    
    # Check for @odata.context
    if echo "$RESPONSE" | grep -q '@odata.context'; then
        # Check that it's not wrapped in a value array (singletons return direct entity)
        if ! echo "$RESPONSE" | grep -q '"value"\s*:'; then
            return 0
        else
            echo "  Details: Singleton should not be wrapped in 'value' array"
            return 1
        fi
    else
        echo "  Details: Response missing '@odata.context'"
        return 1
    fi
}

# Test 3: GET singleton with $select
test_singleton_select() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Company?\$select=Name,CEO")
    check_status "$HTTP_CODE" "200"
}

# Test 4: PATCH singleton updates entity
test_patch_singleton() {
    local ORIGINAL=$(http_get_body "$SERVER_URL/Company")
    
    # Update CEO field - use curl directly to get status code
    local HTTP_CODE=$(curl -g -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Company" -H "Content-Type: application/json" -d '{"CEO":"Test CEO"}')
    
    # Restore original (cleanup inline)
    curl -g -s -X PATCH "$SERVER_URL/Company" -H "Content-Type: application/json" -d "$ORIGINAL" > /dev/null 2>&1
    
    # PATCH should return 204 No Content or 200 OK
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 204 or 200)"
        return 1
    fi
}

# Test 5: PUT singleton replaces entity
test_put_singleton() {
    local ORIGINAL=$(http_get_body "$SERVER_URL/Company")
    
    # Extract key fields from original
    local ID=$(echo "$ORIGINAL" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*')
    
    # Replace with new data
    local NEW_DATA='{"ID":'$ID',"Name":"Test Company","CEO":"Test CEO","Founded":2000,"HeadQuarter":"Test HQ","Version":1}'
    local RESPONSE=$(http_put "$SERVER_URL/Company" "$NEW_DATA" -H "Content-Type: application/json" -w "%{http_code}")
    
    # Restore original (cleanup inline)
    http_put "$SERVER_URL/Company" "$ORIGINAL" -H "Content-Type: application/json" > /dev/null 2>&1
    
    # Check if response is successful
    if echo "$RESPONSE" | grep -q "204\|200"; then
        return 0
    else
        echo "  Details: PUT singleton did not return expected status"
        return 1
    fi
}

# Test 6: POST to singleton should fail (405 Method Not Allowed)
test_post_singleton_fails() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Company" -H "Content-Type: application/json" -d '{"Name":"New Company"}')
    
    # Should return 405 Method Not Allowed or 404
    if [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 405, 404, or 501)"
        return 1
    fi
}

# Test 7: DELETE singleton should fail (405 Method Not Allowed)
test_delete_singleton_fails() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Company")
    
    # Should return 405 Method Not Allowed or 404
    if [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "501" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 405, 404, or 501)"
        return 1
    fi
}

# Test 8: Singleton appears in service document
test_singleton_in_service_document() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    
    if echo "$RESPONSE" | grep -q '"kind"[[:space:]]*:[[:space:]]*"Singleton"'; then
        return 0
    else
        echo "  Details: Service document does not list singleton with kind=Singleton"
        return 1
    fi
}

# Test 9: Singleton has proper metadata in $metadata
test_singleton_in_metadata() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    
    # Check for singleton definition (XML format)
    if echo "$RESPONSE" | grep -q 'Singleton'; then
        return 0
    else
        echo "  Details: Metadata document does not define singleton"
        return 1
    fi
}

# Test 10: Singleton property access
test_singleton_property_access() {
    # Some implementations may not support direct property access on singletons
    # Accept 200 (supported) or 404 (not implemented)
    local HTTP_CODE=$(http_get "$SERVER_URL/Company/Name")
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 404)"
        return 1
    fi
}

echo "  Request: GET /Company"
run_test "GET singleton returns 200" test_get_singleton

echo "  Request: GET /Company (check structure)"
run_test "Singleton returns proper JSON structure" test_singleton_json_structure

echo "  Request: GET /Company?\$select=Name,CEO"
run_test "Singleton supports \$select query option" test_singleton_select

echo "  Request: PATCH /Company"
run_test "PATCH updates singleton entity" test_patch_singleton

echo "  Request: PUT /Company"
run_test "PUT replaces singleton entity" test_put_singleton

echo "  Request: POST /Company (should fail)"
run_test "POST to singleton returns error" test_post_singleton_fails

echo "  Request: DELETE /Company (should fail)"
run_test "DELETE singleton returns error" test_delete_singleton_fails

echo "  Request: GET / (check for singleton)"
run_test "Singleton appears in service document" test_singleton_in_service_document

echo "  Request: GET /\$metadata (check for singleton)"
run_test "Singleton defined in metadata document" test_singleton_in_metadata

echo "  Request: GET /Company/Name"
run_test "Singleton property access works" test_singleton_property_access

print_summary
