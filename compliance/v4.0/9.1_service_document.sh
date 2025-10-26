#!/bin/bash

# OData v4 Compliance Test: 9.1 Service Document
# Tests that service document is properly formatted according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ServiceDocument

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Service document should be accessible at root
test_1() {
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    check_status "$HTTP_CODE" "200"
}

# Test 2: Service document should have @odata.context
test_2() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    if check_contains "$RESPONSE" '"@odata.context"'; then
        check_contains "$RESPONSE" '\$metadata'
    else
        return 1
    fi
}

# Test 3: Service document should have value array
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    check_json_field "$RESPONSE" "value"
}

# Test 4: Service document entity sets should have required properties
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    if check_json_field "$RESPONSE" "name" && \
       check_json_field "$RESPONSE" "url" && \
       check_json_field "$RESPONSE" "kind"; then
        return 0
    fi
    return 1
}

# Test 5: Entity set kind should be "EntitySet"
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    check_contains "$RESPONSE" '"kind"[[:space:]]*:[[:space:]]*"EntitySet"'
}

# Test 6: Singleton should have kind="Singleton" (if any)
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/")
    # If there are singletons, verify they have the correct kind
    # If no singletons, test passes
    if echo "$RESPONSE" | grep -q '"kind"[[:space:]]*:[[:space:]]*"Singleton"'; then
        local SINGLETON_BLOCK=$(echo "$RESPONSE" | grep -A3 -B3 '"kind"[[:space:]]*:[[:space:]]*"Singleton"')
        check_contains "$SINGLETON_BLOCK" '"name"'
    else
        # No singletons, test passes
        return 0
    fi
}

# Run all tests
run_test "Service document accessible at /" test_1
run_test "Service document contains @odata.context" test_2
run_test "Service document contains value array" test_3
run_test "Entity sets have required properties (name, kind, url)" test_4
run_test "Entity sets have kind=\"EntitySet\"" test_5
run_test "Singletons have kind=\"Singleton\" (if present)" test_6

print_summary
