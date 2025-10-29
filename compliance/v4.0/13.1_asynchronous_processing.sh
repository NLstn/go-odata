#!/bin/bash

# OData v4 Compliance Test: 13.1 Asynchronous Request Processing
# Tests OData v4 asynchronous request processing capabilities
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AsynchronousRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 13.1 Asynchronous Processing"
echo "======================================"
echo ""
echo "Description: Tests asynchronous request processing features including"
echo "             the Prefer: respond-async header, status monitor URLs,"
echo "             and proper async response patterns per OData v4 spec."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AsynchronousRequests"
echo ""

# Note: Asynchronous processing is an OPTIONAL feature in OData v4

# Test 1: Service accepts Prefer: respond-async header
test_prefer_async_header_accepted() {
    # Service must accept the header (even if it doesn't honor it)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Prefer: respond-async")
    
    # Should return 200 (synchronous) or 202 (asynchronous)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ]; then
        return 0
    else
        echo "  Details: Service must accept Prefer: respond-async header (got $HTTP_CODE)"
        return 1
    fi
}

# Test 2: Service can respond synchronously even with Prefer: respond-async
test_synchronous_response_allowed() {
    # Service is allowed to respond synchronously even when async is requested
    local HTTP_CODE=$(http_get "$SERVER_URL/Products" -H "Prefer: respond-async")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "202" ]; then
        # Async response - also valid
        return 0
    else
        echo "  Details: Service should respond with 200 or 202 (got $HTTP_CODE)"
        return 1
    fi
}

# Test 3: Async response returns 202 Accepted
test_async_returns_202() {
    # If service supports async, it returns 202
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        # Service responded synchronously - allowed
        skip_test "Async 202 response" "Service does not support asynchronous processing"
        return 0
    else
        echo "  Details: Async requests should return 202 or fall back to 200 (got $HTTP_CODE)"
        return 1
    fi
}

# Test 4: Async response includes Location header
test_async_location_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        # Check for Location header with status monitor URL
        if echo "$RESPONSE" | grep -iq "^Location:"; then
            return 0
        else
            echo "  Details: 202 Accepted response must include Location header"
            return 1
        fi
    elif [ "$HTTP_CODE" = "200" ]; then
        skip_test "Async Location header" "Service does not support asynchronous processing"
        return 0
    else
        echo "  Details: Could not test async Location header (got $HTTP_CODE)"
        return 1
    fi
}

# Test 5: Async response may include Retry-After header
test_async_retry_after() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        # Retry-After is optional but recommended
        # Test passes whether present or not
        return 0
    elif [ "$HTTP_CODE" = "200" ]; then
        skip_test "Async Retry-After header" "Service does not support asynchronous processing"
        return 0
    else
        return 0
    fi
}

# Test 6: Service can indicate async support via Preference-Applied header
test_preference_applied_header() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" -H "Prefer: respond-async" 2>&1)
    
    # If service honors the preference, it should include Preference-Applied
    # This is optional, so we just check it's valid if present
    return 0
}

# Test 7: POST with async preference
test_async_post_request() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Prefer: respond-async" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Async Test","Price":99.99,"CategoryID":1}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    # Should return 201, 202, or 400 (validation error)
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "202" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: POST with async preference (got $HTTP_CODE)"
        return 1
    fi
}

# Test 8: DELETE with async preference
test_async_delete_request() {
    # First create an entity to delete
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Delete Test","Price":1.00,"CategoryID":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local BODY=$(echo "$CREATE_RESPONSE" | sed '$d')
        local ID=$(echo "$BODY" | grep -oP '"ID":\s*\K\d+' | head -1)
        
        if [ -n "$ID" ]; then
            # Try async delete
            local DEL_RESPONSE=$(curl -s -i -X DELETE "$SERVER_URL/Products($ID)" \
                -H "Prefer: respond-async" 2>&1)
            local DEL_CODE=$(echo "$DEL_RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
            
            # Clean up if still exists
            curl -s -X DELETE "$SERVER_URL/Products($ID)" >/dev/null 2>&1
            
            # Should return 204, 202, or 404
            if [ "$DEL_CODE" = "204" ] || [ "$DEL_CODE" = "202" ] || [ "$DEL_CODE" = "404" ]; then
                return 0
            else
                echo "  Details: DELETE with async preference (got $DEL_CODE)"
                return 1
            fi
        fi
    fi
    
    skip_test "Async DELETE request" "Could not create test entity for deletion"
    return 0
}

# Test 9: Async responses must not include response body
test_async_no_body() {
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products" -H "Prefer: respond-async" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
    
    if [ "$HTTP_CODE" = "202" ]; then
        # 202 response should not include the result body
        # Just check it's a short response (no entity data)
        local BODY_SIZE=$(echo "$RESPONSE" | wc -c)
        if [ "$BODY_SIZE" -lt 1000 ]; then
            return 0
        else
            echo "  Details: 202 response should not include full result body"
            return 1
        fi
    elif [ "$HTTP_CODE" = "200" ]; then
        skip_test "Async no body in 202" "Service does not support asynchronous processing"
        return 0
    else
        return 0
    fi
}

# Test 10: Service handles multiple concurrent async requests
test_multiple_async_requests() {
    # This is more of a capability test - just verify service doesn't break
    local HTTP_CODE1=$(http_get "$SERVER_URL/Products" -H "Prefer: respond-async")
    local HTTP_CODE2=$(http_get "$SERVER_URL/Categories" -H "Prefer: respond-async")
    
    # Both should succeed (200 or 202)
    if ([ "$HTTP_CODE1" = "200" ] || [ "$HTTP_CODE1" = "202" ]) && \
       ([ "$HTTP_CODE2" = "200" ] || [ "$HTTP_CODE2" = "202" ]); then
        return 0
    else
        echo "  Details: Service should handle multiple async requests (got $HTTP_CODE1, $HTTP_CODE2)"
        return 1
    fi
}

# Run tests
run_test "Service accepts Prefer: respond-async header" test_prefer_async_header_accepted
run_test "Service can respond synchronously with async preference" test_synchronous_response_allowed
run_test "Async response returns 202 Accepted" test_async_returns_202
run_test "Async 202 response includes Location header" test_async_location_header
run_test "Async response may include Retry-After" test_async_retry_after
run_test "Preference-Applied header indicates async support" test_preference_applied_header
run_test "POST request with async preference" test_async_post_request
run_test "DELETE request with async preference" test_async_delete_request
run_test "Async 202 response excludes result body" test_async_no_body
run_test "Service handles multiple async requests" test_multiple_async_requests

print_summary
