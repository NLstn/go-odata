#!/bin/bash

# OData v4 Compliance Test: 11.4.2 Create an Entity (POST)
# Tests entity creation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CreateanEntity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

# Track created IDs for cleanup
CREATED_IDS=()

# Cleanup function
cleanup() {
    for ID in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($ID)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: POST should return 201 Created
test_1() {
    local PAYLOAD='{"Name":"ComplianceTestProduct1","Price":99.99,"Category":"Test"}'
    local HTTP_CODE=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json" -o /dev/null -w "%{http_code}")
    
    if [ "$HTTP_CODE" = "201" ]; then
        local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
        local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
        [ -n "$ID" ] && CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

# Test 2: Location header should be present in 201 response
test_2() {
    local PAYLOAD='{"Name":"ComplianceTestProduct2","Price":199.99,"Category":"Test"}'
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>&1)
    
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')
    
    if [ -n "$LOCATION" ] && [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$LOCATION" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
        [ -n "$ID" ] && CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

# Test 3: Created entity should be returned in response body
test_3() {
    local PAYLOAD='{"Name":"ComplianceTestProduct3","Price":299.99,"Category":"Test"}'
    local BODY=$(http_post "$SERVER_URL/Products" "$PAYLOAD" -H "Content-Type: application/json")
    
    if echo "$BODY" | grep -q '"ID"' && echo "$BODY" | grep -q '"Name"[[:space:]]*:[[:space:]]*"ComplianceTestProduct3"'; then
        local ID=$(echo "$BODY" | grep -o '"ID"[[:space:]]*:[[:space:]]*[0-9]*' | grep -o '[0-9]*' | head -1)
        [ -n "$ID" ] && CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

# Test 4: POST with Prefer: return=minimal should return 204 No Content
test_4() {
    local PAYLOAD='{"Name":"ComplianceTestProduct4","Price":399.99,"Category":"Test"}'
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d "$PAYLOAD" 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')
    local LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | head -1 | sed 's/Location: //i' | tr -d '\r' | xargs)
    
    if [ "$HTTP_CODE" = "204" ] && [ -n "$LOCATION" ]; then
        local ID=$(echo "$LOCATION" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
        [ -n "$ID" ] && CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

# Test 5: OData-EntityId header should be present in 204 response
test_5() {
    local PAYLOAD='{"Name":"ComplianceTestProduct5","Price":499.99,"Category":"Test"}'
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d "$PAYLOAD" 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP" | head -1 | awk '{print $2}')
    local ODATA_ENTITYID=$(echo "$RESPONSE" | grep -i "^OData-EntityId:" | head -1 | sed 's/OData-EntityId: //i' | tr -d '\r' | xargs)
    
    if [ "$HTTP_CODE" = "204" ] && [ -n "$ODATA_ENTITYID" ]; then
        local ID=$(echo "$ODATA_ENTITYID" | grep -o 'Products([0-9]*)' | grep -o '[0-9]*')
        [ -n "$ID" ] && CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

# Run all tests
run_test "POST entity returns 201 Created" test_1
run_test "Location header present in 201 response" test_2
run_test "Created entity returned in response body" test_3
run_test "POST with Prefer: return=minimal returns 204" test_4
run_test "OData-EntityId header in 204 response" test_5

print_summary
