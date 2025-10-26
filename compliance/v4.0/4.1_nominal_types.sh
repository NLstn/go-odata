#!/bin/bash

# OData v4.0 Section 4.1: Nominal Types
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752514
#
# A nominal type has a name that MUST be a simple identifier.
# Nominal types are referenced using their qualified name.
# The qualified type name MUST be unique within the model.
# Names are case-sensitive, but service authors SHOULD NOT choose names that differ only in case.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Verify entity types exist in metadata
test_1() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "EntityType"
}

# Test 2: Verify entity types have Name attribute
test_2() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" '<EntityType.*Name='
}

# Test 3: Verify qualified names used in navigation properties
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "NavigationProperty"; then
        echo "$RESPONSE" | grep -q 'Type="[^"]*\.'
    else
        return 0
    fi
}

# Test 4: Verify complex types have Name attribute (if any exist)
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "ComplexType"; then
        echo "$RESPONSE" | grep -q '<ComplexType.*Name='
    else
        return 0
    fi
}

# Test 5: Verify enum types have Name attribute (if any exist)
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "EnumType"; then
        echo "$RESPONSE" | grep -q '<EnumType.*Name='
    else
        return 0
    fi
}

# Test 6: Verify properties use qualified type names
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    echo "$RESPONSE" | grep -q 'Property.*Type="Edm\.' || \
    echo "$RESPONSE" | grep -q 'Property.*Type="[^E][^"]*\.'
}

# Test 7: Verify schema has Namespace attribute
test_7() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" '<Schema.*Namespace='
}

# Test 8: Verify built-in primitive types use Edm namespace
test_8() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    echo "$RESPONSE" | grep -q 'Type="Edm\.(String\|Int32\|Boolean\|Decimal)'
}

# Test 9: Verify entity sets reference entity types with qualified names
test_9() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "EntitySet"; then
        echo "$RESPONSE" | grep -q 'EntityType="[^"]*\.'
    else
        return 0
    fi
}

# Test 10: Verify collection types use qualified element type names
test_10() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Collection('; then
        echo "$RESPONSE" | grep -q 'Type="Collection([^)]*\.'
    else
        return 0
    fi
}

# Run all tests
run_test "Entity types exist in metadata" test_1
run_test "Entity types have Name attribute" test_2
run_test "Navigation properties use qualified type names" test_3
run_test "Complex types have Name attribute (if present)" test_4
run_test "Enum types have Name attribute (if present)" test_5
run_test "Properties use qualified type names" test_6
run_test "Schema has Namespace attribute" test_7
run_test "Built-in primitive types use Edm namespace" test_8
run_test "Entity sets reference types with qualified names" test_9
run_test "Collection types use qualified element type names" test_10

print_summary
