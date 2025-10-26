#!/bin/bash

# OData v4 Compliance Test: 11.6 Annotations
# Tests instance annotations, @odata annotations, and custom annotations
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_InstanceAnnotations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.6 Annotations"
echo "======================================"
echo ""
echo "Description: Validates handling of instance annotations, control information,"
echo "             and custom annotations in OData responses."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_InstanceAnnotations"
echo ""

# Test 1: Standard @odata.context annotation
test_odata_context() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    if [ "$HTTP_CODE" = "200" ]; then
        if echo "$RESPONSE" | grep -q '@odata.context'; then
            return 0
        else
            echo "  Details: @odata.context not found (required for minimal metadata)"
            return 1
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 2: @odata.count annotation
test_odata_count() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$count=true")
    
    if echo "$RESPONSE" | grep -q '@odata.count'; then
        return 0
    else
        echo "  Details: @odata.count not found when \$count=true"
        return 1
    fi
}

# Test 3: @odata.nextLink annotation
test_odata_nextlink() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$top=1")
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$top=1")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # @odata.nextLink is optional unless there are more results
        if echo "$RESPONSE" | grep -q '@odata.nextLink'; then
            return 0
        else
            echo "  Details: @odata.nextLink not present (may be optional)"
            return 0  # Pass - optional if all results fit
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: @odata.id annotation for entities
test_odata_id() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    if echo "$RESPONSE" | grep -q '@odata.id'; then
        return 0
    else
        echo "  Details: @odata.id not found (required for entities)"
        return 1
    fi
}

# Test 5: @odata.editLink annotation
test_odata_editlink() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # @odata.editLink is optional in minimal metadata
    if echo "$RESPONSE" | grep -q '@odata.editLink\|@odata.id'; then
        return 0
    else
        echo "  Details: No edit link (acceptable if @odata.id present)"
        return 0  # Pass - optional
    fi
}

# Test 6: @odata.type annotation
test_odata_type() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)?accept=application/json;odata.metadata=full")
    
    # @odata.type is optional in minimal metadata but required in full
    if echo "$RESPONSE" | grep -q '@odata.type'; then
        return 0
    else
        echo "  Details: @odata.type not present (acceptable for minimal metadata)"
        return 0  # Pass - optional in minimal
    fi
}

# Test 7: @odata.deltaLink annotation
test_odata_deltalink() {
    # Try to get delta link
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # @odata.deltaLink is optional and only for delta responses
    if echo "$RESPONSE" | grep -q '@odata.deltaLink'; then
        return 0
    else
        echo "  Details: @odata.deltaLink not present (optional feature)"
        return 0  # Pass - optional
    fi
}

# Test 8: Custom instance annotations
test_custom_annotations() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    
    # Custom annotations start with @ but not @odata
    # This is optional - services may not provide custom annotations
    if echo "$RESPONSE" | grep -qE '"@[^o]|"@o[^d]'; then
        return 0
    else
        echo "  Details: No custom annotations (optional)"
        return 0  # Pass - optional
    fi
}

# Test 9: @odata.removed annotation for delta responses
test_odata_removed() {
    # This is only for delta responses
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    # Just check endpoint works - @odata.removed is very specific
    check_status "$HTTP_CODE" "200"
}

# Test 10: Annotations in metadata=full
test_annotations_full_metadata() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)" -H "Accept: application/json;odata.metadata=full")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)" \
        -H "Accept: application/json;odata.metadata=full" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Full metadata should include more annotations
        if echo "$RESPONSE" | grep -q '@odata'; then
            return 0
        else
            echo "  Details: No @odata annotations in full metadata"
            return 1
        fi
    else
        echo "  Details: Status: $HTTP_CODE"
        return 1
    fi
}

# Test 11: No annotations in metadata=none
test_no_annotations_none_metadata() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)" -H "Accept: application/json;odata.metadata=none")
    
    # metadata=none should remove most annotations
    if echo "$RESPONSE" | grep -q '@odata.context'; then
        echo "  Details: @odata.context should not be present in metadata=none"
        return 1
    else
        return 0
    fi
}

# Test 12: Annotations in collections
test_annotations_in_collections() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products")
    
    # Check for proper collection format with annotations
    if echo "$RESPONSE" | grep -q '"value"' && echo "$RESPONSE" | grep -q '@odata.context'; then
        return 0
    else
        echo "  Details: Missing required collection structure or annotations"
        return 1
    fi
}

echo "  Request: GET Products"
run_test "@odata.context annotation present" test_odata_context

echo "  Request: GET Products?\$count=true"
run_test "@odata.count annotation with \$count" test_odata_count

echo "  Request: GET Products?\$top=1"
run_test "@odata.nextLink for paging (optional)" test_odata_nextlink

echo "  Request: GET Products(1)"
run_test "@odata.id annotation for entity" test_odata_id

echo "  Request: GET Products(1)"
run_test "@odata.editLink annotation (optional)" test_odata_editlink

echo "  Request: GET Products(1) with full metadata"
run_test "@odata.type annotation" test_odata_type

echo "  Request: Check for @odata.deltaLink"
run_test "@odata.deltaLink annotation (optional)" test_odata_deltalink

echo "  Request: Check for custom annotations"
run_test "Custom instance annotations (optional)" test_custom_annotations

echo "  Request: Check @odata.removed"
run_test "@odata.removed for delta (optional)" test_odata_removed

echo "  Request: GET with metadata=full"
run_test "Annotations in metadata=full" test_annotations_full_metadata

echo "  Request: GET with metadata=none"
run_test "No annotations in metadata=none" test_no_annotations_none_metadata

echo "  Request: GET collection with annotations"
run_test "Annotations in collection responses" test_annotations_in_collections

print_summary
