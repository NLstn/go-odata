#!/bin/bash

# OData v4 Compliance Test: 11.4.8 Modify Relationship References
# Tests modifying relationships using $ref
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ManagingRelationships

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.8 Modify Relationships"
echo "======================================"
echo ""
echo "Description: Validates modifying relationships between entities"
echo "             using \$ref endpoints."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ManagingRelationships"
echo ""

# Test 1: GET $ref returns reference URL
test_get_ref() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Category/\$ref")

    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Expected status 200 for GET $ref but received $HTTP_CODE"
        return 1
    fi
}

# Test 2: PUT $ref creates/updates single-valued relationship
test_put_ref_single() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$SERVER_URL/Products(1)/Category/\$ref" \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'"$SERVER_URL"'/Categories(1)"}' 2>&1)
    
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Expected status 204 (or 200 if a representation is returned) for PUT $ref but received $HTTP_CODE"
        return 1
    fi
}

# Test 3: POST $ref adds to collection-valued relationship
test_post_ref_collection() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products(1)/RelatedProducts/\$ref" \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'"$SERVER_URL"'/Products(2)"}' 2>&1)
    
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "201" ]; then
        return 0
    else
        echo "  Details: Expected status 204 (or 201 if a new resource is created) for POST $ref but received $HTTP_CODE"
        return 1
    fi
}

# Test 4: DELETE $ref removes relationship
test_delete_ref() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/Products(1)/Category/\$ref" 2>&1)
    
    if [ "$HTTP_CODE" = "204" ]; then
        return 0
    else
        echo "  Details: Expected status 204 for DELETE $ref but received $HTTP_CODE"
        return 1
    fi
}

# Test 5: Invalid $ref URL returns error
test_invalid_ref_url() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$SERVER_URL/Products(1)/Category/\$ref" \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"invalid-url"}' 2>&1)
    
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Expected status 400 for invalid $ref payload but received $HTTP_CODE"
        return 1
    fi
}

# Test 6: $ref to non-existent navigation property returns error
test_ref_nonexistent_property() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/NonExistentProperty/\$ref")
    
    # Should return 404
    check_status "$HTTP_CODE" "404"
}

echo "  Request: GET $SERVER_URL/Products(1)/Category/\$ref"
run_test "GET \$ref returns reference URL" test_get_ref

echo "  Request: PUT $SERVER_URL/Products(1)/Category/\$ref"
run_test "PUT \$ref creates/updates single-valued relationship" test_put_ref_single

echo "  Request: POST $SERVER_URL/Products(1)/RelatedProducts/\$ref"
run_test "POST \$ref adds to collection-valued relationship" test_post_ref_collection

echo "  Request: DELETE $SERVER_URL/Products(1)/Category/\$ref"
run_test "DELETE \$ref removes relationship" test_delete_ref

echo "  Request: PUT $SERVER_URL/Products(1)/Category/\$ref with invalid URL"
run_test "Invalid \$ref URL returns error" test_invalid_ref_url

echo "  Request: GET $SERVER_URL/Products(1)/NonExistentProperty/\$ref"
run_test "\$ref to non-existent navigation property returns 404" test_ref_nonexistent_property

print_summary
