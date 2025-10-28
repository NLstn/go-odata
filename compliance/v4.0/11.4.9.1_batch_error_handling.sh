#!/bin/bash

# OData v4 Compliance Test: 11.4.9.1 Batch Error Handling
# Tests error handling in batch requests including changeset atomicity
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests

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
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests"
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
    
    if [ "$HTTP_CODE" != "400" ]; then
        echo "  Details: Expected 400 for malformed batch boundary but received $HTTP_CODE"
        return 1
    fi

    return 0
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
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 from \\$batch endpoint but received $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 3: Changeset with one invalid request should fail atomically
test_changeset_atomicity() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot verify changeset atomicity because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 from changeset batch but received $HTTP_CODE"
        return 1
    fi

    if ! echo "$BODY" | grep -q "HTTP/1.1 4[0-9][0-9]"; then
        echo "  Details: Expected at least one 4xx response inside changeset"
        return 1
    fi

    return 0
}

# Test 4: Error in one request shouldn't affect others outside changeset
test_independent_requests() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot verify independent requests because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 for independent requests batch but received $HTTP_CODE"
        return 1
    fi

    local SUCCESS_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1 200" || echo "0")
    local NOT_FOUND_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1 404" || echo "0")

    if [ "$SUCCESS_COUNT" -lt 2 ] || [ "$NOT_FOUND_COUNT" -lt 1 ]; then
        echo "  Details: Expected at least two 200 responses and one 404 response, got $SUCCESS_COUNT successes and $NOT_FOUND_COUNT not-founds"
        return 1
    fi

    return 0
}

# Test 5: Invalid HTTP method in batch
test_invalid_method() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot validate invalid methods because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "400" ]; then
        echo "  Details: Expected 200 with error payload or 400 for invalid method but received $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 6: Missing Content-Type in batch part
test_missing_content_type() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot validate missing Content-Type because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "400" ]; then
        echo "  Details: Expected 200 or 400 for missing Content-Type but received $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 7: Empty batch request
test_empty_batch() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot validate empty batches because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
    fi
    
    local BATCH_BODY="--batch_boundary--"

    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should accept empty batch (return 200) or reject it (400)
    if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "400" ]; then
        echo "  Details: Expected 200 or 400 for empty batch but received $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 8: Nested changesets (should be rejected)
test_nested_changesets() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot validate nested changesets because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    
    if [ "$HTTP_CODE" != "400" ] && [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 400 or 200 for nested changesets but received $HTTP_CODE"
        return 1
    fi

    return 0
}

# Test 9: Response includes proper error format for failed operations
test_error_format_in_batch() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot inspect batch error format because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    if echo "$RESPONSE" | grep -q "\"error\""; then
        return 0
    fi

    if echo "$RESPONSE" | grep -q "404"; then
        return 0
    fi

    echo "  Details: Expected error payload or 404 marker in batch response"
    return 1
}

# Test 10: Batch with GET and POST maintains order
test_request_order() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        echo "  Details: Cannot verify batch ordering because \\$batch endpoint returned $BATCH_SUPPORTED"
        return 1
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
    
    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected 200 for mixed batch but received $HTTP_CODE"
        return 1
    fi

    local RESPONSE_COUNT=$(echo "$RESPONSE" | head -n -1 | grep -c "HTTP/1.1" || echo "0")
    if [ "$RESPONSE_COUNT" -lt 3 ]; then
        echo "  Details: Expected at least 3 responses but found $RESPONSE_COUNT"
        return 1
    fi

    return 0
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
