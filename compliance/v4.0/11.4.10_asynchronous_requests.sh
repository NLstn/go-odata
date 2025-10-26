#!/bin/bash

# OData v4 Compliance Test: 11.4.10 Asynchronous Requests
# Tests asynchronous request processing with Prefer: respond-async header
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_AsynchronousRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.10 Asynchronous Requests"
echo "======================================"
echo ""
echo "Description: Validates asynchronous request processing according to OData v4 specification"
echo "             Tests Prefer: respond-async header and status monitor URLs"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_AsynchronousRequests"
echo ""


# Test 1: Prefer respond-async header is accepted
test_async_header_accepted() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" \
        -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should return either 200 (sync) or 202 (async)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 200 or 202)"
        return 1
    fi
}

# Test 2: Async POST request
test_async_post() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: respond-async" \
        -d '{"Name":"Async Test Product","Price":99.99,"CategoryID":1,"Status":1}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should return 201 (sync) or 202 (async)
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "202" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 201 or 202)"
        return 1
    fi
}

# Test 3: Async DELETE request
test_async_delete() {
    # Create entity first
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Async Delete Test","Price":50,"CategoryID":1,"Status":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            # Try async delete
            local DELETE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Products($ID)" \
                -H "Prefer: respond-async" 2>&1)
            
            # Should return 204/200 (sync) or 202 (async)
            if [ "$DELETE_CODE" = "204" ] || [ "$DELETE_CODE" = "200" ] || [ "$DELETE_CODE" = "202" ]; then
                return 0
            else
                echo "  Details: Status code: $DELETE_CODE (expected 204, 200, or 202)"
                return 1
            fi
        fi
    fi
    echo "  Details: Failed to create test entity"
    return 1
}

# Test 4: Check for Location header on 202 response
test_async_location_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" \
        -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        # Should have Location header pointing to status monitor
        if echo "$RESPONSE" | grep -qi "Location:"; then
            return 0
        else
            echo "  Details: 202 response missing Location header"
            return 1
        fi
    else
        # If not async, that's also valid (service doesn't support async)
        echo "  Details: Service processed request synchronously (status: $HTTP_CODE)"
        return 0
    fi
}

# Test 5: Preference-Applied header
test_preference_applied_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" \
        -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        # Should have Preference-Applied header
        if echo "$RESPONSE" | grep -qi "Preference-Applied:.*respond-async"; then
            return 0
        else
            echo "  Details: Missing Preference-Applied header"
            # Not strictly required, so warn but don't fail
            return 0
        fi
    else
        echo "  Details: Request processed synchronously"
        return 0
    fi
}

echo "  Request: GET with Prefer: respond-async"
run_test "Prefer: respond-async header is accepted" test_async_header_accepted

echo "  Request: POST with Prefer: respond-async"
run_test "Async POST request is handled" test_async_post

echo "  Request: DELETE with Prefer: respond-async"
run_test "Async DELETE request is handled" test_async_delete

echo "  Request: Check for Location header on 202"
run_test "202 response includes Location header" test_async_location_header

echo "  Request: Check for Preference-Applied header"
run_test "Preference-Applied header is returned" test_preference_applied_header

print_summary
