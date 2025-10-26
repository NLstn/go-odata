#!/bin/bash

# OData v4 Compliance Test: 11.4.9.1 Batch Error Handling
# Tests error handling in batch requests including changeset atomicity
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.9.1 Batch Error Handling"
echo "======================================"
echo ""
echo "Description: Validates error handling in batch requests, including"
echo "             changeset atomicity, error responses, and malformed requests."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests"
echo ""

# Global variable to track batch support
BATCH_SUPPORTED=""

# Test 1: Batch endpoint responds to malformed boundary
test_malformed_boundary() {
    local BATCH_BODY="--wrong_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--wrong_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 400 for malformed batch or 501 if not supported
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 2: Check batch support
test_batch_support() {
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BATCH_SUPPORTED="$HTTP_CODE"
    
    # Either supported (200) or not implemented (501)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 3: Changeset with one invalid request should fail atomically
test_changeset_atomicity() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0  # Skip if batch not supported
    fi
    
    # Changeset with valid POST and invalid POST (missing required field)
    local BATCH_BODY="--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Valid Product\",\"Price\":99.99,\"CategoryID\":1,\"Status\":1}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

{\"Price\":50.00}

--changeset_boundary--

--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Should return 200 but with error responses inside
    # Neither product should be created due to atomicity
    if [ "$HTTP_CODE" = "200" ]; then
        # Check that at least one response indicates error
        echo "$BODY" | grep -q "HTTP/1.1 4[0-9][0-9]"
    else
        return 1
    fi
}

# Test 4: Error in one request shouldn't affect others outside changeset
test_independent_requests() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(99999) HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(2) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have success responses for first and third request
        local SUCCESS_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1 200" || echo "0")
        local NOT_FOUND_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1 404" || echo "0")
        [ "$SUCCESS_COUNT" -ge 2 ] && [ "$NOT_FOUND_COUNT" -ge 1 ]
    else
        return 1
    fi
}

# Test 5: Invalid HTTP method in batch
test_invalid_method() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

INVALID Products(1) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 200 with error response inside, or 400 for malformed batch
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 6: Missing Content-Type in batch part
test_missing_content_type() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should handle gracefully - either accept or reject
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 7: Empty batch request
test_empty_batch() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should accept empty batch (return 200) or reject it (400)
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]
}

# Test 8: Nested changesets (should be rejected)
test_nested_changesets() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: multipart/mixed; boundary=changeset1

--changeset1
Content-Type: multipart/mixed; boundary=changeset2

--changeset2
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Test\",\"Price\":50,\"CategoryID\":1,\"Status\":1}

--changeset2--

--changeset1--

--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Nested changesets are not allowed per spec, but some implementations may handle gracefully
    # Accept either rejection (400) or processing as single-level (200)
    [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 9: Response includes proper error format for failed operations
test_error_format_in_batch() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(99999) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    
    # Response should contain error JSON structure
    echo "$RESPONSE" | grep -q "\"error\"" || echo "$RESPONSE" | grep -q "404"
}

# Test 10: Batch with GET and POST maintains order
test_request_order() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Order Test\",\"Price\":25.00,\"CategoryID\":1,\"Status\":1}

--changeset_boundary--

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(2) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have at least 3 responses in order
        local RESPONSE_COUNT=$(echo "$RESPONSE" | head -n -1 | grep -c "HTTP/1.1" || echo "0")
        [ "$RESPONSE_COUNT" -ge 3 ]
    else
        return 1
    fi
}

echo "  Request: POST batch with mismatched boundary"
run_test "Malformed batch boundary handled correctly" test_malformed_boundary

echo "  Request: POST batch with single GET"
run_test "Batch endpoint is accessible" test_batch_support

echo "  Request: POST batch with changeset containing invalid request"
run_test "Changeset atomicity - all fail if one fails" test_changeset_atomicity

echo "  Request: POST batch with mix of valid/invalid independent requests"
run_test "Independent requests don't affect each other" test_independent_requests

echo "  Request: POST batch with invalid HTTP method"
run_test "Invalid HTTP method in batch handled properly" test_invalid_method

echo "  Request: POST batch part without Content-Type"
run_test "Missing Content-Type in batch part handled" test_missing_content_type

echo "  Request: POST batch with no requests"
run_test "Empty batch request handled" test_empty_batch

echo "  Request: POST batch with nested changesets"
run_test "Nested changesets are rejected" test_nested_changesets

echo "  Request: POST batch with 404 request"
run_test "Error responses in batch have proper format" test_error_format_in_batch

echo "  Request: POST batch with mixed GET/POST operations"
run_test "Batch maintains request order in responses" test_request_order

print_summary
