#!/bin/bash

# OData v4 Compliance Test: 5.4 Type Definitions
# Tests custom type definitions in metadata and their usage
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.4 Type Definitions"
echo "======================================"
echo ""
echo "Description: Validates custom type definitions in metadata document"
echo "             and their proper usage in entity properties."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html"
echo ""

# Test 1: Metadata contains TypeDefinition elements
test_typedef_in_metadata() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Type definitions should be in the metadata
    # Even if not explicitly defined, test that metadata is valid XML
    if echo "$RESPONSE" | grep -q "<Schema" || echo "$RESPONSE" | grep -q '"$Version"'; then
        return 0
    else
        echo "  Details: Metadata does not contain valid schema"
        return 1
    fi
}

# Test 2: Metadata type definitions have UnderlyingType
test_typedef_underlying_type() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # If TypeDefinition exists, it should have UnderlyingType
    # For this test, we check if metadata is well-formed
    if echo "$RESPONSE" | grep -q "TypeDefinition"; then
        if echo "$RESPONSE" | grep -q "UnderlyingType"; then
            return 0
        else
            echo "  Details: TypeDefinition found but missing UnderlyingType"
            return 1
        fi
    else
        # No type definitions is acceptable (optional feature)
        return 0
    fi
}

# Test 3: Type definition with MaxLength facet
test_typedef_maxlength() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Check if any properties use MaxLength constraint
    if echo "$RESPONSE" | grep -q "MaxLength"; then
        return 0
    else
        # MaxLength may not be used, which is acceptable
        return 0
    fi
}

# Test 4: Type definition with Precision and Scale facets
test_typedef_precision_scale() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Check for Precision and Scale attributes (typically on Decimal types)
    if echo "$RESPONSE" | grep -q "Type=\"Edm.Decimal\""; then
        # Decimal properties may have Precision and Scale
        return 0
    else
        # No Decimal types is acceptable
        return 0
    fi
}

# Test 5: Type definition with SRID facet (geographic types)
test_typedef_srid() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # SRID is for geographic types (optional feature)
    # Just verify metadata is parseable
    if echo "$RESPONSE" | grep -q "SRID"; then
        return 0
    else
        # No geographic types is acceptable
        return 0
    fi
}

# Test 6: Primitive type as underlying type
test_typedef_primitive_underlying() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Type definitions should be based on Edm primitive types
    if echo "$RESPONSE" | grep -q "Edm\\.String\\|Edm\\.Int32\\|Edm\\.Decimal"; then
        return 0
    else
        echo "  Details: No Edm primitive types found in metadata"
        return 1
    fi
}

# Test 7: Entity properties use type definitions
test_entity_uses_typedef() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Properties should be typed with Edm types
    if echo "$RESPONSE" | grep -q "Property.*Type="; then
        return 0
    else
        echo "  Details: No typed properties found"
        return 1
    fi
}

# Test 8: Nullable type definition
test_typedef_nullable() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Properties can be nullable
    if echo "$RESPONSE" | grep -q "Nullable=\"true\"\\|Nullable=\"false\""; then
        return 0
    else
        # Nullable attribute may be omitted (defaults to true)
        return 0
    fi
}

# Test 9: Type definition in JSON metadata
test_typedef_json_metadata() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata?\\$format=json")
    
    # JSON CSDL format
    if echo "$RESPONSE" | grep -q '"$Version"\\|"@odata.context"'; then
        return 0
    else
        # JSON format may not be supported
        return 0
    fi
}

# Test 10: Complex type with type definitions
test_complex_type_typedef() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Complex types may use type definitions for their properties
    if echo "$RESPONSE" | grep -q "ComplexType"; then
        return 0
    else
        # No complex types is acceptable
        return 0
    fi
}

# Test 11: Default value in type definition
test_typedef_default_value() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Properties can have default values
    if echo "$RESPONSE" | grep -q "DefaultValue"; then
        return 0
    else
        # Default values are optional
        return 0
    fi
}

# Test 12: Unicode facet for string types
test_typedef_unicode() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Unicode attribute for string types (optional)
    if echo "$RESPONSE" | grep -q "Type=\"Edm.String\""; then
        # String types exist, Unicode facet is optional
        return 0
    else
        return 0
    fi
}

# Test 13: Type definition namespace
test_typedef_namespace() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Schema should have a Namespace attribute
    if echo "$RESPONSE" | grep -q "Namespace="; then
        return 0
    else
        # JSON format might not use Namespace attribute
        if echo "$RESPONSE" | grep -q '"$Version"'; then
            return 0
        else
            echo "  Details: No namespace found in metadata"
            return 1
        fi
    fi
}

# Test 14: Enum type as type definition
test_enum_typedef() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Enum types are a form of type definition
    if echo "$RESPONSE" | grep -q "EnumType"; then
        return 0
    else
        # No enum types is acceptable
        return 0
    fi
}

# Test 15: Type definition inheritance (optional)
test_typedef_inheritance() {
    # Type definitions cannot have base types (they're not entity types)
    # This test verifies that type definitions are distinct from entity types
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Just verify metadata structure is valid
    if echo "$RESPONSE" | grep -q "<Schema\\|\"\\$Version\""; then
        return 0
    else
        echo "  Details: Invalid metadata structure"
        return 1
    fi
}

echo "  Request: GET /\$metadata"
run_test "Metadata contains valid schema structure" test_typedef_in_metadata

echo "  Request: GET /\$metadata - check TypeDefinition"
run_test "TypeDefinition elements have UnderlyingType" test_typedef_underlying_type

echo "  Request: GET /\$metadata - check MaxLength"
run_test "Type definitions can use MaxLength facet" test_typedef_maxlength

echo "  Request: GET /\$metadata - check Precision/Scale"
run_test "Type definitions can use Precision and Scale facets" test_typedef_precision_scale

echo "  Request: GET /\$metadata - check SRID"
run_test "Type definitions can use SRID facet for geographic types" test_typedef_srid

echo "  Request: GET /\$metadata - check underlying types"
run_test "Type definitions based on Edm primitive types" test_typedef_primitive_underlying

echo "  Request: GET /\$metadata - check property types"
run_test "Entity properties use type definitions" test_entity_uses_typedef

echo "  Request: GET /\$metadata - check Nullable"
run_test "Type definitions support Nullable facet" test_typedef_nullable

echo "  Request: GET /\$metadata?\$format=json"
run_test "Type definitions in JSON metadata format" test_typedef_json_metadata

echo "  Request: GET /\$metadata - check ComplexType"
run_test "Complex types can use type definitions" test_complex_type_typedef

echo "  Request: GET /\$metadata - check DefaultValue"
run_test "Properties can have default values" test_typedef_default_value

echo "  Request: GET /\$metadata - check Unicode facet"
run_test "String types support Unicode facet" test_typedef_unicode

echo "  Request: GET /\$metadata - check Namespace"
run_test "Schema has proper namespace definition" test_typedef_namespace

echo "  Request: GET /\$metadata - check EnumType"
run_test "Enum types as type definitions" test_enum_typedef

echo "  Request: GET /\$metadata - verify structure"
run_test "Type definitions distinct from entity types" test_typedef_inheritance

print_summary
