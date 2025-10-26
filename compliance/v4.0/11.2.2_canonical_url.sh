#!/bin/bash

# OData v4 Compliance Test: 11.2.2 Canonical URL
# Tests canonical URL representation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CanonicalURL

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Store ODATA_ID for later tests
ODATA_ID=""

# Test 1: Entity should have @odata.id with canonical URL
test_1() {
    local BODY=$(http_get_body "$SERVER_URL/Products(1)")
    if check_json_field "$BODY" "@odata.id"; then
        ODATA_ID=$(echo "$BODY" | grep -o '"@odata.id":"[^"]*"' | head -1 | cut -d'"' -f4)
        [ -n "$ODATA_ID" ]
        return $?
    fi
    return 1
}

# Test 2: Canonical URL should be dereferenceable
test_2() {
    if [ -z "$ODATA_ID" ]; then
        local BODY=$(http_get_body "$SERVER_URL/Products(1)")
        ODATA_ID=$(echo "$BODY" | grep -o '"@odata.id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    if [ -n "$ODATA_ID" ]; then
        local HTTP_CODE
        if echo "$ODATA_ID" | grep -q "^http"; then
            HTTP_CODE=$(curl -g -s -o /dev/null -w "%{http_code}" "$ODATA_ID")
        else
            HTTP_CODE=$(http_get "$SERVER_URL/$ODATA_ID")
        fi
        check_status "$HTTP_CODE" "200"
    else
        return 1
    fi
}

# Test 3: Collection should have @odata.id for each entity
test_3() {
    local BODY=$(http_get_body "$SERVER_URL/Products?\$top=3")
    local ENTITY_COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    local ODATA_ID_COUNT=$(echo "$BODY" | grep -o '"@odata.id"' | wc -l)
    
    if [ "$ENTITY_COUNT" -gt 0 ] && [ "$ODATA_ID_COUNT" -eq "$ENTITY_COUNT" ]; then
        return 0
    fi
    echo "  Details: Found $ENTITY_COUNT entities but $ODATA_ID_COUNT @odata.id fields"
    return 1
}

# Test 4: Canonical URL format should match entity set and key
test_4() {
    local BODY=$(http_get_body "$SERVER_URL/Products(1)")
    if check_json_field "$BODY" "@odata.id"; then
        local ID=$(echo "$BODY" | grep -o '"@odata.id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if echo "$ID" | grep -qE 'Products\([0-9]+\)'; then
            return 0
        fi
        echo "  Details: URL format does not match expected pattern: $ID"
    fi
    return 1
}

# Run all tests
run_test "Entity has @odata.id with canonical URL" test_1
run_test "@odata.id URL is dereferenceable" test_2
run_test "Each entity in collection has @odata.id" test_3
run_test "Canonical URL format matches entity set and key pattern" test_4

print_summary
