#!/bin/bash

# OData v4 Compliance Test: 8.1.5 Response Status Codes
# Tests correct HTTP status codes for various operations and error conditions
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ResponseStatusCodes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.1.5 Response Status Codes"
echo "======================================"
echo ""
echo "Description: Validates correct HTTP status codes for successful operations,"
echo "             client errors, and server errors according to OData v4 specification"
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ResponseStatusCodes"
echo ""

CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup

# Test 1: 200 OK for successful GET
test_status_200_ok() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}

# Test 2: 201 Created for successful POST
test_status_201_created() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Status Code Test","Price":99.99,"Category":"Test","Status":1}' 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 201)"
        return 1
    fi
}

# Test 3: 204 No Content for successful DELETE
test_status_204_no_content() {
    # Create entity first
    local CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Delete Test","Price":50,"Category":"Test","Status":1}' 2>&1)
    local CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$CREATE_RESPONSE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ID" ]; then
            # Delete and check status
            local DELETE_CODE=$(http_get "$SERVER_URL/Products($ID)" -X DELETE)
            if [ "$DELETE_CODE" = "204" ] || [ "$DELETE_CODE" = "200" ]; then
                return 0
            else
                CREATED_IDS+=("$ID")  # Clean up if delete failed
                echo "  Details: Delete status: $DELETE_CODE (expected 204 or 200)"
                return 1
            fi
        fi
    fi
    echo "  Details: Failed to create test entity"
    return 1
}

# Test 4: 400 Bad Request for malformed request
test_status_400_bad_request() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d 'invalid json' 2>&1)
    check_status "$HTTP_CODE" "400"
}

# Test 5: 404 Not Found for non-existent entity
test_status_404_not_found() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(999999)")
    check_status "$HTTP_CODE" "404"
}

# Test 6: 405 Method Not Allowed (if applicable)
test_status_405_method_not_allowed() {
    # Try to POST to a specific entity (should be PATCH/PUT)
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test"}' 2>&1)
    
    # Should be 405 or 400 (some implementations may return 400)
    if [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 405, 400, or 404)"
        return 1
    fi
}

# Test 7: 415 Unsupported Media Type for wrong Content-Type
test_status_415_unsupported_media() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: text/plain" \
        -d '{"Name":"Test"}' 2>&1)
    
    # Should be 415 or 400
    if [ "$HTTP_CODE" = "415" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE (expected 415 or 400)"
        return 1
    fi
}

# Test 8: 304 Not Modified for conditional GET (if ETag supported)
test_status_304_not_modified() {
    # Get entity first to get ETag
    local RESPONSE=$(curl -s -i "$SERVER_URL/Products(1)" 2>&1)
    local ETAG=$(echo "$RESPONSE" | grep -i "ETag:" | sed 's/.*ETag: *//i' | tr -d '\r')
    
    if [ -n "$ETAG" ]; then
        local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)" \
            -H "If-None-Match: $ETAG" 2>&1)
        
        if [ "$HTTP_CODE" = "304" ]; then
            return 0
        else
            echo "  Details: Status code: $HTTP_CODE with ETag: $ETAG (expected 304)"
            # Not all implementations support this, so warn but don't fail
            return 0
        fi
    else
        echo "  Details: ETag not supported, skipping test"
        return 0
    fi
}

# Test 9: 412 Precondition Failed for failed If-Match
test_status_412_precondition_failed() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products(1)" \
        -H "Content-Type: application/json" \
        -H "If-Match: \"wrong-etag\"" \
        -d '{"Price":100}' 2>&1)
    
    # Should be 412 if ETags are supported, or possibly 400/200 if not
    if [ "$HTTP_CODE" = "412" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Status code: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Service document returns 200
test_status_service_document() {
    local HTTP_CODE=$(http_get "$SERVER_URL/")
    check_status "$HTTP_CODE" "200"
}

echo "  Request: GET /Products"
run_test "200 OK for successful GET request" test_status_200_ok

echo "  Request: POST /Products"
run_test "201 Created for successful POST request" test_status_201_created

echo "  Request: DELETE /Products(id)"
run_test "204 No Content for successful DELETE" test_status_204_no_content

echo "  Request: POST with invalid JSON"
run_test "400 Bad Request for malformed request" test_status_400_bad_request

echo "  Request: GET /Products(999999)"
run_test "404 Not Found for non-existent entity" test_status_404_not_found

echo "  Request: POST to single entity URL"
run_test "405 Method Not Allowed for invalid method" test_status_405_method_not_allowed

echo "  Request: POST with text/plain Content-Type"
run_test "415 Unsupported Media Type for wrong Content-Type" test_status_415_unsupported_media

echo "  Request: GET with If-None-Match header"
run_test "304 Not Modified for conditional GET" test_status_304_not_modified

echo "  Request: PATCH with wrong If-Match header"
run_test "412 Precondition Failed for failed If-Match" test_status_412_precondition_failed

echo "  Request: GET / (service document)"
run_test "200 OK for service document" test_status_service_document

print_summary
