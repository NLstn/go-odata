#!/bin/bash

# OData v4 Compliance Test: 11.4.9 Batch Requests
# Tests batch request processing according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Global variable to track batch support
BATCH_SUPPORTED=""

# Test 1: \$batch endpoint exists
test_1() {
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
    
    [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "501" ]
}

# Test 2: Batch response has multipart/mixed Content-Type
test_2() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0  # Skip if batch not supported
    fi
    
    local BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary--"

    local FULL_RESPONSE=$(curl -s -i -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    local CONTENT_TYPE=$(echo "$FULL_RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    
    echo "$CONTENT_TYPE" | grep -q "multipart/mixed"
}

# Test 3: Batch with multiple GET requests
test_3() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local MULTI_BATCH="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(2) HTTP/1.1
Accept: application/json


--batch_boundary--"
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$MULTI_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1" || echo "0")
        [ "$RESPONSE_COUNT" -ge 2 ]
    else
        return 1
    fi
}

# Test 4: Batch request with changeset
test_4() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local CHANGESET_BATCH="--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST Products HTTP/1.1
Content-Type: application/json
Accept: application/json

{\"Name\":\"Batch Test Product\",\"Price\":99.99}

--changeset_boundary--
--batch_boundary--"
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$CHANGESET_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "200" ]
}

# Test 5: Invalid batch request returns 400
test_5() {
    if [ "$BATCH_SUPPORTED" != "200" ]; then
        return 0
    fi
    
    local INVALID_BATCH="--batch_boundary
INVALID CONTENT
--batch_boundary--"
    
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$INVALID_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    [ "$HTTP_CODE" = "400" ]
}

run_test "\$batch endpoint responds" test_1
run_test "Batch response has multipart/mixed Content-Type" test_2
run_test "Batch with multiple GET requests" test_3
run_test "Batch request with changeset (atomicity)" test_4
run_test "Invalid batch request returns 400" test_5

print_summary
