#!/bin/bash

# OData v4 Compliance Test: 11.2.4 Addressing Collections vs Single Entities
# Tests addressing collections and understanding the difference from single entities
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.4 Collection Operations"
echo "======================================"
echo ""
echo "Description: Validates addressing entity collections and understanding"
echo "             the difference between collections and single entities."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingEntities"
echo ""

# Test 1: Collection returns array with value property
test_collection_format() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # OData v4 requires collections to be wrapped in "value" property
    if check_json_field "$RESPONSE" "value"; then
        return 0
    else
        return 1
    fi
}

# Test 2: Single entity does not have value wrapper
test_single_entity_format() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Single entity should have direct properties, not wrapped in "value"
    # It should have an ID property
    if check_json_field "$RESPONSE" "ID"; then
        # Make sure it's not wrapped in value array
        if ! echo "$RESPONSE" | grep -q '"value"\s*:\s*\['; then
            return 0
        else
            echo "  Details: Single entity incorrectly wrapped in value array"
            return 1
        fi
    else
        echo "  Details: Single entity missing ID field"
        return 1
    fi
}

# Test 3: Collection has @odata.context
test_collection_context() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    if check_json_field "$RESPONSE" "@odata.context"; then
        return 0
    else
        return 1
    fi
}

# Test 4: Collection returns 200 OK
test_collection_status() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    check_status "$HTTP_CODE" "200"
}

# Test 5: Empty collection returns valid structure
test_empty_collection() {
    # Use a filter that should return no results
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=ID%20eq%20-999999")
    
    # Should still have value array (just empty) and context
    if check_json_field "$RESPONSE" "value" && check_json_field "$RESPONSE" "@odata.context"; then
        return 0
    else
        return 1
    fi
}

# Test 6: Collection supports query options
test_collection_query_options() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=5")
    
    check_status "$HTTP_CODE" "200"
}

# Test 7: Single entity does not support $top
test_single_entity_rejects_top() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)?\$top=5")
    
    # Should return 400 as $top is not applicable to single entities
    check_status "$HTTP_CODE" "400"
}

# Test 8: Single entity does not support $skip
test_single_entity_rejects_skip() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)?\$skip=5")
    
    # Should return 400 as $skip is not applicable to single entities
    check_status "$HTTP_CODE" "400"
}

echo "  Request: GET $SERVER_URL/Products"
run_test "Collection returns array wrapped in 'value' property" test_collection_format

echo "  Request: GET $SERVER_URL/Products(1)"
run_test "Single entity returns object without 'value' wrapper" test_single_entity_format

echo "  Request: GET $SERVER_URL/Products"
run_test "Collection includes @odata.context" test_collection_context

echo "  Request: GET $SERVER_URL/Products"
run_test "Collection request returns 200 OK" test_collection_status

echo "  Request: GET $SERVER_URL/Products?\$filter=ID eq -999999"
run_test "Empty collection returns valid structure with empty array" test_empty_collection

echo "  Request: GET $SERVER_URL/Products?\$top=5"
run_test "Collection supports query options like \$top" test_collection_query_options

echo "  Request: GET $SERVER_URL/Products(1)?\$top=5"
run_test "Single entity rejects \$top query option with 400" test_single_entity_rejects_top

echo "  Request: GET $SERVER_URL/Products(1)?\$skip=5"
run_test "Single entity rejects \$skip query option with 400" test_single_entity_rejects_skip

print_summary
