#!/bin/bash

# OData v4 Compliance Test: 11.4.9 Batch Requests
# Tests batch request processing according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.9 Batch Requests"
echo "======================================"
echo ""
echo "Description: Validates batch request processing including multiple"
echo "             operations in a single HTTP request."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests"
echo ""



# Test 1: \$batch endpoint exists
echo "Test 1: \$batch endpoint responds"
echo "  Request: POST $SERVER_URL/\$batch"
BATCH_BODY="--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET Products(1) HTTP/1.1
Accept: application/json


--batch_boundary--"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
    -d "$BATCH_BODY" \
    "$SERVER_URL/\$batch" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "200" ]; then
    test_result "\$batch endpoint accepts requests" "PASS"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "\$batch endpoint accepts requests" "PASS" "Batch not implemented (optional feature)"
else
    test_result "\$batch endpoint accepts requests" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Batch response has multipart/mixed Content-Type
if [ "$HTTP_CODE" = "200" ]; then
    echo ""
    echo "Test 2: Batch response has multipart/mixed Content-Type"
    FULL_RESPONSE=$(curl -s -i -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$BATCH_BODY" \
        "$SERVER_URL/\$batch" 2>&1)
    CONTENT_TYPE=$(echo "$FULL_RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    
    if echo "$CONTENT_TYPE" | grep -q "multipart/mixed"; then
        test_result "Batch response is multipart/mixed" "PASS"
    else
        test_result "Batch response is multipart/mixed" "FAIL" "Content-Type is $CONTENT_TYPE"
    fi
    
    # Test 3: Batch with multiple GET requests
    echo ""
    echo "Test 3: Batch with multiple GET requests"
    MULTI_BATCH="--batch_boundary
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
    
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$MULTI_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should contain multiple responses
        RESPONSE_COUNT=$(echo "$BODY" | grep -c "HTTP/1.1" || echo "0")
        if [ "$RESPONSE_COUNT" -ge 2 ]; then
            test_result "Batch processes multiple requests" "PASS"
        else
            test_result "Batch processes multiple requests" "FAIL" "Expected 2+ responses, found $RESPONSE_COUNT"
        fi
    else
        test_result "Batch processes multiple requests" "FAIL" "HTTP $HTTP_CODE"
    fi
    
    # Test 4: Batch request with changeset
    echo ""
    echo "Test 4: Batch request with changeset (atomicity)"
    CHANGESET_BATCH="--batch_boundary
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
    
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$CHANGESET_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        test_result "Batch with changeset processed" "PASS"
    else
        test_result "Batch with changeset processed" "FAIL" "HTTP $HTTP_CODE"
    fi
    
    # Test 5: Invalid batch request returns 400
    echo ""
    echo "Test 5: Invalid batch request returns 400"
    INVALID_BATCH="--batch_boundary
INVALID CONTENT
--batch_boundary--"
    
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
        -d "$INVALID_BATCH" \
        "$SERVER_URL/\$batch" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "400" ]; then
        test_result "Invalid batch returns 400" "PASS"
    else
        test_result "Invalid batch returns 400" "FAIL" "Expected HTTP 400, got $HTTP_CODE"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    echo ""
    echo "Skipping remaining batch tests - feature not implemented (optional)"
fi


print_summary
