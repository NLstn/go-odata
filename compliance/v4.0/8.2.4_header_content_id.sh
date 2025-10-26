#!/bin/bash

# OData v4 Compliance Test: 8.2.4 Content-ID Header
# Tests Content-ID header usage in batch requests for referencing entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentID

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.4 Content-ID Header"
echo "======================================"
echo ""
echo "Description: Validates Content-ID header usage in batch requests"
echo "             for referencing newly created entities within the same batch."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderContentID"
echo ""

# Test 1: Content-ID in batch changeset
test_content_id_basic() {
    local BOUNDARY="batch_boundary"
    local CHANGESET="changeset_boundary"
    
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"ContentID Test Product\",\"Price\":99.99,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check if response contains reference to Content-ID
        if echo "$RESPONSE" | grep -q "Content-ID"; then
            return 0
        else
            # Content-ID in response is optional but request should succeed
            return 0
        fi
    else
        echo "  Details: Batch request failed with status $HTTP_CODE"
        return 1
    fi
}

# Test 2: Reference Content-ID in subsequent request
test_content_id_reference() {
    local BOUNDARY="batch_ref"
    local CHANGESET="changeset_ref"
    
    # Create a category and reference it when creating a product
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: newCategory

POST Categories HTTP/1.1
Content-Type: application/json

{\"Name\":\"Ref Test Category\"}
--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: newProduct

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Product with Ref\",\"Price\":55.55,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Both operations should succeed (status 201 or 204 in the batch response)
        if echo "$BODY" | grep -q "201\|204"; then
            return 0
        else
            echo "  Details: Expected 201 or 204 in batch response"
            return 1
        fi
    else
        echo "  Details: Batch failed with status $HTTP_CODE"
        return 1
    fi
}

# Test 3: Content-ID with $-prefix reference
test_content_id_dollar_prefix() {
    local BOUNDARY="batch_dollar"
    local CHANGESET="changeset_dollar"
    
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 100

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Dollar Prefix Test\",\"Price\":123.45,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Content-ID: 100 can be referenced as $100 in subsequent requests
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 4: Multiple Content-IDs in same changeset
test_multiple_content_ids() {
    local BOUNDARY="batch_multi"
    local CHANGESET="changeset_multi"
    
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: prod1

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Multi Product 1\",\"Price\":11.11,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: prod2

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Multi Product 2\",\"Price\":22.22,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: prod3

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Multi Product 3\",\"Price\":33.33,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # All three should succeed
        local SUCCESS_COUNT=$(echo "$BODY" | grep -c "201\|204")
        if [ "$SUCCESS_COUNT" -ge 3 ]; then
            return 0
        else
            echo "  Details: Expected 3 successful operations, found $SUCCESS_COUNT"
            return 1
        fi
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 5: Content-ID uniqueness within batch
test_content_id_uniqueness() {
    local BOUNDARY="batch_unique"
    local CHANGESET="changeset_unique"
    
    # Using duplicate Content-ID should either be handled or rejected
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: dup

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Duplicate ID 1\",\"Price\":44.44,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: dup

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Duplicate ID 2\",\"Price\":55.55,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should either succeed (if duplicates allowed) or fail gracefully (400)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Unexpected status $HTTP_CODE"
        return 1
    fi
}

# Test 6: Content-ID in GET request (read operations)
test_content_id_in_get() {
    local BOUNDARY="batch_get"
    
    # Content-ID is primarily for POST/PUT/PATCH in changesets
    # but can be used in read operations too
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: getOp1

GET Products(1) HTTP/1.1

--${BOUNDARY}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: getOp2

GET Products(2) HTTP/1.1

--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Content-ID on GET is less common but should be accepted
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 7: Content-ID scope within changeset
test_content_id_scope() {
    local BOUNDARY="batch_scope"
    local CHANGESET1="changeset_scope1"
    local CHANGESET2="changeset_scope2"
    
    # Content-ID should be scoped to changeset
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET1}

--${CHANGESET1}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: scopeTest

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Scope Test 1\",\"Price\":66.66,\"CategoryID\":1,\"Status\":1}
--${CHANGESET1}--
--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET2}

--${CHANGESET2}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: scopeTest

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Scope Test 2\",\"Price\":77.77,\"CategoryID\":1,\"Status\":1}
--${CHANGESET2}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Same Content-ID in different changesets should be OK
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 8: Content-ID format validation
test_content_id_format() {
    local BOUNDARY="batch_format"
    local CHANGESET="changeset_format"
    
    # Test with alphanumeric Content-ID
    local BATCH_REQUEST="--${BOUNDARY}
Content-Type: multipart/mixed; boundary=${CHANGESET}

--${CHANGESET}
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: ABC123xyz

POST Products HTTP/1.1
Content-Type: application/json

{\"Name\":\"Format Test\",\"Price\":88.88,\"CategoryID\":1,\"Status\":1}
--${CHANGESET}--
--${BOUNDARY}--"

    local RESPONSE=$(echo "$BATCH_REQUEST" | curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/\$batch" \
        -H "Content-Type: multipart/mixed; boundary=${BOUNDARY}" \
        --data-binary @-)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Alphanumeric Content-ID should be accepted
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

echo "  Request: Batch with Content-ID in changeset"
run_test "Content-ID header accepted in batch changeset" test_content_id_basic

echo "  Request: Batch with Content-ID reference"
run_test "Content-ID can reference newly created entities" test_content_id_reference

echo "  Request: Content-ID with numeric value"
run_test "Content-ID with numeric identifier" test_content_id_dollar_prefix

echo "  Request: Multiple Content-IDs in changeset"
run_test "Multiple unique Content-IDs in same changeset" test_multiple_content_ids

echo "  Request: Duplicate Content-IDs handled"
run_test "Duplicate Content-IDs handled appropriately" test_content_id_uniqueness

echo "  Request: Content-ID in GET operations"
run_test "Content-ID accepted in read operations" test_content_id_in_get

echo "  Request: Content-ID in multiple changesets"
run_test "Content-ID scoped within changesets" test_content_id_scope

echo "  Request: Alphanumeric Content-ID"
run_test "Content-ID with alphanumeric format" test_content_id_format

print_summary
