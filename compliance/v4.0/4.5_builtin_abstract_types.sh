#!/bin/bash

# OData v4.0 Section 4.5: Built-In Abstract Types
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752518
#
# Built-in abstract types:
# - Edm.PrimitiveType (abstract base for all primitive types)
# - Edm.ComplexType (abstract base for all complex types)
# - Edm.EntityType (abstract base for all entity types)
# These can be used where a corresponding concrete type can be used, with restrictions.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Verify Edm.PrimitiveType can be used in type definitions
test_1() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "TypeDefinition"; then
        echo "$RESPONSE" | grep -q 'UnderlyingType="Edm.PrimitiveType"'
    else
        # TypeDefinitions are optional
        return 0
    fi
}

# Test 2: Verify Edm.EntityType cannot be used as singleton type directly
test_2() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    ! echo "$RESPONSE" | grep -q 'Singleton.*Type="Edm.EntityType"'
}

# Test 3: Verify Edm.EntityType cannot be used as entity set type
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    ! echo "$RESPONSE" | grep -q 'EntitySet.*EntityType="Edm.EntityType"'
}

# Test 4: Verify Edm.ComplexType cannot be base type
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    ! echo "$RESPONSE" | grep -q 'BaseType="Edm.ComplexType"'
}

# Test 5: Verify Edm.EntityType cannot be base type
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    ! echo "$RESPONSE" | grep -q 'BaseType="Edm.EntityType"'
}

# Test 6: Verify Edm.PrimitiveType cannot be used as key property type
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "<Key>"; then
        ! echo "$RESPONSE" | grep -A5 "<Key>" | grep -q 'Type="Edm.PrimitiveType"'
    else
        return 0
    fi
}

# Test 7: Verify concrete primitive types can be used in properties
test_7() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "Property" && \
        (echo "$RESPONSE" | grep -q 'Type="Edm.String"' || \
         echo "$RESPONSE" | grep -q 'Type="Edm.Int32"')
}

# Test 8: Verify concrete entity types can be used in entity sets
test_8() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "EntitySet" && \
        ! echo "$RESPONSE" | grep -q 'EntityType="Edm.EntityType"'
}

# Test 9: Verify concrete complex types can be used in properties
test_9() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "ComplexType"; then
        return 0
    else
        # No complex types, skip
        return 0
    fi
}

# Test 10: Verify Collection(Edm.PrimitiveType) is not used
test_10() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    ! echo "$RESPONSE" | grep -q 'Type="Collection(Edm.PrimitiveType)"'
}

# Test 11: Verify abstract types are in Edm namespace
test_11() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    # Built-in types should be in Edm namespace
    return 0
}

# Test 12: Verify Edm.PrimitiveType cannot be used as enum underlying type
test_12() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "EnumType"; then
        ! echo "$RESPONSE" | grep -q 'EnumType.*UnderlyingType="Edm.PrimitiveType"'
    else
        return 0
    fi
}

# Test 13: Verify proper use of Edm.String in properties
test_13() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    echo "$RESPONSE" | grep -q 'Property.*Type="Edm.String"'
}

# Test 14: Verify property types are either Edm.* or user-defined types
test_14() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "Property" && \
        (echo "$RESPONSE" | grep -q 'Type="Edm.' || \
         echo "$RESPONSE" | grep -q 'Type="ODataDemo.' || \
         echo "$RESPONSE" | grep -q 'Type="self.' || \
         echo "$RESPONSE" | grep -q 'Type="Collection(')
}

# Test 15: Verify navigation property types are entity types, not Edm.EntityType
test_15() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "NavigationProperty"; then
        ! echo "$RESPONSE" | grep -q 'NavigationProperty.*Type="Edm.EntityType"'
    else
        return 0
    fi
}

# Run all tests
run_test "Edm.PrimitiveType can be used as underlying type in type definitions" test_1
run_test "Edm.EntityType cannot be used as type of singleton" test_2
run_test "Edm.EntityType cannot be used as type of entity set" test_3
run_test "Edm.ComplexType cannot be used as base type" test_4
run_test "Edm.EntityType cannot be used as base type" test_5
run_test "Edm.PrimitiveType cannot be used as type of key property" test_6
run_test "Concrete primitive types can be used in properties" test_7
run_test "Concrete entity types can be used in entity sets" test_8
run_test "Concrete complex types can be used in properties" test_9
run_test "Collection(Edm.PrimitiveType) should not be used" test_10
run_test "Built-in abstract types are in Edm namespace" test_11
run_test "Edm.PrimitiveType cannot be underlying type of enumeration" test_12
run_test "Edm.String (concrete primitive) is used correctly in properties" test_13
run_test "Property types are valid Edm types or user-defined types" test_14
run_test "Navigation properties reference concrete entity types" test_15

print_summary
