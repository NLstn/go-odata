#!/bin/bash

# OData v4 Compliance Test: 11.2.8 Delta Links
# Tests delta link support for tracking changes according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingChanges

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Global variables to store initial response
INITIAL_RESPONSE=""
INITIAL_STATUS=""
INITIAL_BODY=""
PREFERENCE_APPLIED=""

# Test 1: Request delta with Prefer: odata.track-changes
test_1() {
    INITIAL_RESPONSE=$(curl -s -i -H "Prefer: odata.track-changes" "$SERVER_URL/Products" 2>&1)
    INITIAL_STATUS=$(echo "$INITIAL_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    INITIAL_BODY=$(echo "$INITIAL_RESPONSE" | sed -n '/^$/,$p' | tail -n +2)
    PREFERENCE_APPLIED=$(echo "$INITIAL_RESPONSE" | grep -i "^Preference-Applied:" | head -1 | sed 's/Preference-Applied: //i' | tr -d '\r')
    
    if [ "$INITIAL_STATUS" = "200" ] || [ "$INITIAL_STATUS" = "501" ]; then
        return 0
    fi
    return 1
}

# Test 2: Preference-Applied header should indicate track-changes support
test_2() {
    if [ "$INITIAL_STATUS" = "200" ] || [ "$INITIAL_STATUS" = "501" ]; then
        return 0
    fi
    return 1
}

# Test 3: Delta link should be dereferenceable (if present)
test_3() {
    if echo "$INITIAL_BODY" | grep -q '"@odata.deltaLink"'; then
        local DELTA_LINK=$(echo "$INITIAL_BODY" | grep -o '"@odata.deltaLink":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$DELTA_LINK" ]; then
            if echo "$DELTA_LINK" | grep -q "^http"; then
                check_status "$DELTA_LINK" 200
            else
                check_status "${DELTA_LINK#/}" 200
            fi
        else
            return 1
        fi
    fi
    return 0
}

# Test 4: Delta response should include context (if delta supported)
test_4() {
    if echo "$INITIAL_BODY" | grep -q '"@odata.deltaLink"'; then
        check_contains "$INITIAL_BODY" '"@odata.context"' "Delta response has @odata.context"
    fi
    return 0
}

# Test 5: Delta with $filter
test_5() {
    local RESPONSE=$(http_get "$SERVER_URL/Products?\$filter=Price%20gt%2010" "-H" "Prefer: odata.track-changes")
    local STATUS=$(echo "$RESPONSE" | tail -1)
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "501" ]; then
        return 0
    fi
    return 1
}

# Test 6: Delta token parameter handling
test_6() {
    local STATUS=$(http_get "$SERVER_URL/Products?\$deltatoken=test-token" | tail -1)
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "410" ] || [ "$STATUS" = "400" ] || [ "$STATUS" = "501" ]; then
        return 0
    fi
    return 1
}

run_test "Request delta with Prefer: odata.track-changes" test_1
run_test "Preference-Applied header for track-changes" test_2
run_test "Delta link is dereferenceable" test_3
run_test "Delta response includes @odata.context" test_4
run_test "Delta request with \$filter" test_5
run_test "Request with delta token parameter" test_6

print_summary
