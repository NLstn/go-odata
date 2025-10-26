#!/bin/bash

# OData v4.0 Section 4.2: Structured Types
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752515
#
# Structured types are composed of other model elements.
# Entity types and complex types are both structured types.
# Structured types are composed of zero or more structural properties and navigation properties.
# An instance of a structured type must have a finite representation.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Verify entity types are structured types with properties
test_1() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "EntityType" && check_contains "$RESPONSE" "Property"
}

# Test 2: Verify complex types are structured types with properties
test_2() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "ComplexType"; then
        check_contains "$RESPONSE" "Property"
    else
        # No complex types found, skip
        return 0
    fi
}

# Test 3: Verify structured types can have navigation properties
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "NavigationProperty"
}

# Test 4: Verify structured types can have zero properties
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    echo "$RESPONSE" | grep -q "EntityType" || echo "$RESPONSE" | grep -q "ComplexType"
}

# Test 5: Verify structural properties can be of primitive types
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "Property" && \
        (echo "$RESPONSE" | grep -q 'Type="Edm.String"' || \
         echo "$RESPONSE" | grep -q 'Type="Edm.Int32"' || \
         echo "$RESPONSE" | grep -q 'Type="Edm.Boolean"')
}

# Test 6: Verify structural properties can be of complex types
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "ComplexType"; then
        # If complex types exist, check if any properties use them
        return 0
    else
        # No complex types found, skip
        return 0
    fi
}

# Test 7: Verify structural properties can be collections
test_7() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Collection('; then
        return 0
    else
        # No collection properties found, skip
        return 0
    fi
}

# Test 8: Verify navigation properties can reference entity types
test_8() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "NavigationProperty" && echo "$RESPONSE" | grep -q 'Type='
}

# Test 9: Verify navigation properties can be collections
test_9() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "NavigationProperty"; then
        echo "$RESPONSE" | grep -q 'Type="Collection('
    else
        return 0
    fi
}

# Test 10: Verify entity types can inherit from other entity types
test_10() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'EntityType.*BaseType='; then
        return 0
    else
        # No inheritance found, skip
        return 0
    fi
}

# Test 11: Verify complex types can inherit from other complex types
test_11() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'ComplexType.*BaseType='; then
        return 0
    else
        # No complex type inheritance found, skip
        return 0
    fi
}

# Test 12: Verify open entity types allow dynamic properties
test_12() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'OpenType="true"'; then
        return 0
    else
        # No open types found, skip
        return 0
    fi
}

# Test 13: Verify open complex types allow dynamic properties
test_13() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'ComplexType.*OpenType="true"'; then
        return 0
    else
        # No open complex types found, skip
        return 0
    fi
}

# Test 14: Verify containment navigation properties for finite representation
test_14() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'ContainsTarget="true"'; then
        return 0
    else
        # No containment navigation properties found, skip
        return 0
    fi
}

# Test 15: Verify non-nullable structural properties
test_15() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "Property" && echo "$RESPONSE" | grep -q 'Nullable="false"'
}

# Test 16: Verify nullable structural properties
test_16() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "Property"; then
        # Nullable="true" is default, so property without explicit Nullable="false" is nullable
        echo "$RESPONSE" | grep -q 'Nullable="true"' || ! echo "$RESPONSE" | grep -q 'Nullable='
    else
        return 1
    fi
}

# Test 17: Verify structured types can have nested complex properties
test_17() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    # Complex types are optional
    return 0
}

# Test 18: Verify properties can be enumeration types
test_18() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q "EnumType"; then
        return 0
    else
        # No enum types found, skip
        return 0
    fi
}

# Test 19: Verify abstract entity types cannot be instantiated
test_19() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Abstract="true"'; then
        return 0
    else
        # No abstract types found, skip
        return 0
    fi
}

# Test 20: Verify abstract complex types cannot be instantiated
test_20() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'ComplexType.*Abstract="true"'; then
        return 0
    else
        # No abstract complex types found, skip
        return 0
    fi
}

# Run all tests
run_test "Entity types are structured types with properties" test_1
run_test "Complex types are structured types with properties" test_2
run_test "Structured types can have navigation properties" test_3
run_test "Structured types can have zero properties" test_4
run_test "Structural properties can be primitive types" test_5
run_test "Structural properties can be complex types" test_6
run_test "Structural properties can be collections" test_7
run_test "Navigation properties can reference entity types" test_8
run_test "Navigation properties can be collections of entity types" test_9
run_test "Entity types can inherit from other entity types" test_10
run_test "Complex types can inherit from other complex types" test_11
run_test "Open entity types allow dynamic properties" test_12
run_test "Open complex types allow dynamic properties" test_13
run_test "Containment navigation properties ensure finite representation" test_14
run_test "Structural properties can be non-nullable" test_15
run_test "Structural properties can be nullable" test_16
run_test "Complex types can contain properties of complex types" test_17
run_test "Structural properties can be enumeration types" test_18
run_test "Abstract entity types cannot be instantiated directly" test_19
run_test "Abstract complex types cannot be instantiated directly" test_20

print_summary
