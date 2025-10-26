#!/bin/bash

# OData v4 Compliance Test: 11.2.15 Entity References ($ref)
# Tests $ref for working with entity references
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingReferences

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.15 Entity References"
echo "======================================"
echo ""
echo "Description: Validates \$ref for retrieving and manipulating entity references"
echo "             instead of the full entity representation."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingReferences"
echo ""

# Test 1: Get reference to single entity
test_entity_ref_single() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)/\$ref")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should return @odata.id with the entity reference
        if echo "$BODY" | grep -q "@odata.id"; then
            return 0
        else
            echo "  Details: Response missing @odata.id"
            return 1
        fi
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: Get reference to collection
test_entity_ref_collection() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products/\$ref")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should return collection of references
        if echo "$BODY" | grep -q "value"; then
            return 0
        else
            echo "  Details: Response missing value array"
            return 1
        fi
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 3: Reference should contain @odata.context
test_ref_has_context() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)/\$ref")
    
    if echo "$RESPONSE" | grep -q "@odata.context"; then
        return 0
    else
        echo "  Details: Reference missing @odata.context"
        return 1
    fi
}

# Test 4: Reference should NOT contain entity properties
test_ref_no_properties() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)/\$ref")
    
    # Should not contain entity properties like Name, Price
    if ! echo "$RESPONSE" | grep -q '"Name"' && ! echo "$RESPONSE" | grep -q '"Price"'; then
        return 0
    else
        echo "  Details: Reference contains entity properties"
        return 1
    fi
}

# Test 5: \$ref with \$filter on collection
test_ref_with_filter() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$filter=Price%20gt%2050")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 6: \$ref with \$top on collection
test_ref_with_top() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$top=3")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 7: \$ref with \$skip on collection
test_ref_with_skip() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$skip=2")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 8: \$ref with \$orderby on collection
test_ref_with_orderby() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$orderby=ID")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 9: \$ref should not support \$expand
test_ref_no_expand() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$expand=Category")
    
    # Should return 400 Bad Request as $expand is not valid with $ref
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 for invalid query option)"
        return 1
    fi
}

# Test 10: \$ref should not support \$select
test_ref_no_select() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products/\$ref?\$select=Name")
    
    # Should return 400 Bad Request as $select is not valid with $ref
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 for invalid query option)"
        return 1
    fi
}

# Test 11: \$ref on non-existent entity returns 404
test_ref_not_found() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(999999)/\$ref")
    check_status "$HTTP_CODE" "404"
}

# Test 12: \$ref with \$count
test_ref_with_count() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products/\$ref?\$count=true")
    
    if echo "$RESPONSE" | grep -q "@odata.count"; then
        return 0
    else
        echo "  Details: Response missing @odata.count"
        return 1
    fi
}

echo "  Request: GET /Products(1)/\$ref"
run_test "Get reference to single entity" test_entity_ref_single

echo "  Request: GET /Products/\$ref"
run_test "Get references to entity collection" test_entity_ref_collection

echo "  Request: Verify @odata.context in reference"
run_test "Reference contains @odata.context" test_ref_has_context

echo "  Request: Verify no entity properties in reference"
run_test "Reference does not contain entity properties" test_ref_no_properties

echo "  Request: GET /Products/\$ref?\$filter=..."
run_test "\$ref with \$filter" test_ref_with_filter

echo "  Request: GET /Products/\$ref?\$top=3"
run_test "\$ref with \$top" test_ref_with_top

echo "  Request: GET /Products/\$ref?\$skip=2"
run_test "\$ref with \$skip" test_ref_with_skip

echo "  Request: GET /Products/\$ref?\$orderby=ID"
run_test "\$ref with \$orderby" test_ref_with_orderby

echo "  Request: GET /Products/\$ref?\$expand=Category"
run_test "\$ref rejects \$expand (should return 400)" test_ref_no_expand

echo "  Request: GET /Products/\$ref?\$select=Name"
run_test "\$ref rejects \$select (should return 400)" test_ref_no_select

echo "  Request: GET /Products(999999)/\$ref"
run_test "\$ref on non-existent entity returns 404" test_ref_not_found

echo "  Request: GET /Products/\$ref?\$count=true"
run_test "\$ref with \$count" test_ref_with_count

print_summary
