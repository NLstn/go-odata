#!/bin/bash

# OData v4 Compliance Test: 11.2.8 Delta Links
# Tests delta link support for tracking changes according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestingChanges

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

INITIAL_TOKEN=""
CURRENT_TOKEN=""
CREATED_PRODUCT_ID=""

extract_body() {
    local response="$1"
    printf '%s' "${response#*$'\r\n\r\n'}"
}

request_initial_delta() {
    local RESPONSE=$(curl -s -i -H "Prefer: odata.track-changes" "$SERVER_URL/Products" 2>&1)
    local STATUS=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    if [ "$STATUS" != "200" ]; then
        echo "  Details: Expected 200, got $STATUS"
        return 1
    fi

    local PREF_APPLIED=$(echo "$RESPONSE" | grep -i "^Preference-Applied:" | head -1 | sed 's/Preference-Applied: //i' | tr -d '\r')
    echo "$PREF_APPLIED" | grep -qi "odata.track-changes" || {
        echo "  Details: Preference-Applied header missing odata.track-changes"
        return 1
    }

    local BODY=$(extract_body "$RESPONSE")
    local DELTA_LINK=$(echo "$BODY" | grep -o '"@odata.deltaLink":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$DELTA_LINK" ]; then
        echo "  Details: Delta link not found in initial response"
        return 1
    fi

    local NORMALIZED_LINK=$(echo "$DELTA_LINK" | sed 's/%24/$/g')
    INITIAL_TOKEN=$(echo "$NORMALIZED_LINK" | sed -n 's/.*$deltatoken=\([^&]*\).*/\1/p')
    if [ -z "$INITIAL_TOKEN" ]; then
        echo "  Details: Failed to extract delta token"
        return 1
    fi

    CURRENT_TOKEN="$INITIAL_TOKEN"
    return 0
}

verify_delta_includes_creation() {
    local PAYLOAD='{ "Name": "Track Changes Widget", "Price": 42.42, "CategoryID": 1, "Status": 1 }'
    local CREATE_RESPONSE=$(curl -s -i -H "Content-Type: application/json" -H "X-User-Role: admin" -X POST -d "$PAYLOAD" "$SERVER_URL/Products" 2>&1)
    local CREATE_STATUS=$(echo "$CREATE_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    if [ "$CREATE_STATUS" != "201" ]; then
        echo "  Details: Expected 201 when creating product, got $CREATE_STATUS"
        return 1
    fi

    local CREATE_BODY=$(extract_body "$CREATE_RESPONSE")
    CREATED_PRODUCT_ID=$(echo "$CREATE_BODY" | grep -o '"ID"[: ]*[0-9]*' | head -1 | sed 's/[^0-9]//g')
    if [ -z "$CREATED_PRODUCT_ID" ]; then
        echo "  Details: Failed to parse created product ID"
        return 1
    fi

    local DELTA_RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$deltatoken=$CURRENT_TOKEN" 2>&1)
    local DELTA_STATUS=$(echo "$DELTA_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    if [ "$DELTA_STATUS" != "200" ]; then
        echo "  Details: Expected 200 for delta request, got $DELTA_STATUS"
        return 1
    fi

    local DELTA_BODY=$(extract_body "$DELTA_RESPONSE")
    echo "$DELTA_BODY" | grep -q '"Name":"Track Changes Widget"' || {
        echo "  Details: Delta response missing created entity"
        return 1
    }

    local NEXT_LINK=$(echo "$DELTA_BODY" | grep -o '"@odata.deltaLink":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$NEXT_LINK" ]; then
        echo "  Details: Delta response missing delta link"
        return 1
    fi

    local NORMALIZED_NEXT=$(echo "$NEXT_LINK" | sed 's/%24/$/g')
    CURRENT_TOKEN=$(echo "$NORMALIZED_NEXT" | sed -n 's/.*$deltatoken=\([^&]*\).*/\1/p')
    if [ -z "$CURRENT_TOKEN" ]; then
        echo "  Details: Failed to extract next delta token"
        return 1
    fi

    return 0
}

verify_delta_includes_deletion() {
    if [ -z "$CREATED_PRODUCT_ID" ]; then
        echo "  Details: Created product ID not set"
        return 1
    fi

    local DELETE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "X-User-Role: admin" -X DELETE "$SERVER_URL/Products($CREATED_PRODUCT_ID)")
    if [ "$DELETE_STATUS" != "204" ]; then
        echo "  Details: Expected 204 when deleting product, got $DELETE_STATUS"
        return 1
    fi

    local DELTA_RESPONSE=$(curl -s -i "$SERVER_URL/Products?\$deltatoken=$CURRENT_TOKEN" 2>&1)
    local DELTA_STATUS=$(echo "$DELTA_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    if [ "$DELTA_STATUS" != "200" ]; then
        echo "  Details: Expected 200 for delta deletion request, got $DELTA_STATUS"
        return 1
    fi

    local DELTA_BODY=$(extract_body "$DELTA_RESPONSE")
    echo "$DELTA_BODY" | grep -q '"@odata.removed"' || {
        echo "  Details: Delta response missing @odata.removed entry"
        return 1
    }
    echo "$DELTA_BODY" | grep -q "\"ID\":$CREATED_PRODUCT_ID" || {
        echo "  Details: Removal entry missing correct ID"
        return 1
    }

    return 0
}

run_test "Initial delta request applies track-changes preference" request_initial_delta
run_test "Delta feed includes newly created entity" verify_delta_includes_creation
run_test "Delta feed reports deleted entity" verify_delta_includes_deletion

print_summary
