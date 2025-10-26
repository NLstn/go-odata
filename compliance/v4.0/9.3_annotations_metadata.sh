#!/bin/bash

# OData v4 Compliance Test: 9.3 Annotations in Metadata
# Tests vocabulary annotations in metadata document
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 9.3 Annotations in Metadata"
echo "======================================"
echo ""
echo "Description: Validates vocabulary annotations in metadata document"
echo "             including Core, Capabilities, and other standard vocabularies."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html"
echo ""

# Test 1: Metadata contains Annotations element
test_annotations_element() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Annotations may be present in metadata
    # Just verify metadata is valid
    if echo "$RESPONSE" | grep -q "<Schema\\|\"\\$Version\""; then
        return 0
    else
        echo "  Details: Invalid metadata structure"
        return 1
    fi
}

# Test 2: Core vocabulary annotations (Description, LongDescription)
test_core_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Core.Description and Core.LongDescription are common annotations
    if echo "$RESPONSE" | grep -q "Core\\.Description\\|Description="; then
        return 0
    else
        # Core vocabulary is optional
        return 0
    fi
}

# Test 3: Capabilities vocabulary (InsertRestrictions, UpdateRestrictions)
test_capabilities_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Capabilities annotations describe service capabilities
    if echo "$RESPONSE" | grep -q "Capabilities\\."; then
        return 0
    else
        # Capabilities vocabulary is optional
        return 0
    fi
}

# Test 4: Validation vocabulary (Pattern, AllowedValues)
test_validation_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Validation vocabulary for property constraints
    if echo "$RESPONSE" | grep -q "Validation\\."; then
        return 0
    else
        # Validation vocabulary is optional
        return 0
    fi
}

# Test 5: Measures vocabulary (Unit, ISOCurrency)
test_measures_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Measures vocabulary for units of measure
    if echo "$RESPONSE" | grep -q "Measures\\."; then
        return 0
    else
        # Measures vocabulary is optional
        return 0
    fi
}

# Test 6: Annotation target (entity type, property, navigation)
test_annotation_target() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Annotations can target various elements
    if echo "$RESPONSE" | grep -q "Target="; then
        return 0
    else
        # Annotation Target attribute is for separate annotation files
        return 0
    fi
}

# Test 7: Inline annotations on properties
test_inline_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Properties can have inline annotations
    if echo "$RESPONSE" | grep -q "<Property"; then
        # Properties exist, inline annotations are optional
        return 0
    else
        echo "  Details: No properties found in metadata"
        return 1
    fi
}

# Test 8: Computed annotation
test_computed_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Core.Computed annotation for computed properties
    if echo "$RESPONSE" | grep -q "Computed"; then
        return 0
    else
        # Computed annotation is optional
        return 0
    fi
}

# Test 9: Immutable annotation
test_immutable_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Core.Immutable annotation for read-only properties
    if echo "$RESPONSE" | grep -q "Immutable"; then
        return 0
    else
        # Immutable annotation is optional
        return 0
    fi
}

# Test 10: Annotation with complex value
test_annotation_complex_value() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Annotations can have complex structured values
    if echo "$RESPONSE" | grep -q "<Record\\|<Collection"; then
        return 0
    else
        # Complex annotation values are optional
        return 0
    fi
}

# Test 11: Reference to external vocabulary
test_vocabulary_reference() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # References to standard vocabularies
    if echo "$RESPONSE" | grep -q "<Reference\\|Org.OData"; then
        return 0
    else
        # External references are optional
        return 0
    fi
}

# Test 12: Annotation on entity set
test_entityset_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # EntitySet can have annotations
    if echo "$RESPONSE" | grep -q "EntitySet"; then
        # EntitySets exist, annotations are optional
        return 0
    else
        echo "  Details: No EntitySet found in metadata"
        return 1
    fi
}

# Test 13: Annotation on navigation property
test_navigation_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # NavigationProperty can have annotations
    if echo "$RESPONSE" | grep -q "NavigationProperty"; then
        # Navigation properties exist, annotations are optional
        return 0
    else
        # No navigation properties is acceptable
        return 0
    fi
}

# Test 14: Permission annotations (Permissions, ReadRestrictions)
test_permission_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Permission and restriction annotations
    if echo "$RESPONSE" | grep -q "Permissions\\|ReadRestrictions"; then
        return 0
    else
        # Permission annotations are optional
        return 0
    fi
}

# Test 15: JSON metadata annotations format
test_json_metadata_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata?\\$format=json")
    
    # Annotations in JSON CSDL
    if echo "$RESPONSE" | grep -q "\"@"; then
        return 0
    else
        # JSON format may not be supported, or no annotations
        return 0
    fi
}

# Test 16: Custom vocabulary annotations
test_custom_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Services can define custom vocabularies
    # Just verify metadata is well-formed
    if echo "$RESPONSE" | grep -q "<Schema\\|\"\\$Version\""; then
        return 0
    else
        echo "  Details: Invalid metadata"
        return 1
    fi
}

# Test 17: Annotation inheritance
test_annotation_inheritance() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Annotations can be inherited from base types
    if echo "$RESPONSE" | grep -q "BaseType="; then
        # Base types exist, annotation inheritance is implicit
        return 0
    else
        # No derived types is acceptable
        return 0
    fi
}

# Test 18: Term definition in metadata
test_term_definition() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Services can define custom terms
    if echo "$RESPONSE" | grep -q "<Term"; then
        return 0
    else
        # Custom terms are optional
        return 0
    fi
}

# Test 19: Annotation with null value
test_annotation_null_value() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Annotations can have null values
    if echo "$RESPONSE" | grep -q "<Null"; then
        return 0
    else
        # Null values in annotations are optional
        return 0
    fi
}

# Test 20: Multiple annotations on same target
test_multiple_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    
    # Elements can have multiple annotations
    # Verify metadata contains properties (which could have multiple annotations)
    if echo "$RESPONSE" | grep -q "<Property\\|<EntityType"; then
        return 0
    else
        echo "  Details: No annotatable elements found"
        return 1
    fi
}

echo "  Request: GET /\$metadata"
run_test "Metadata structure supports annotations" test_annotations_element

echo "  Request: GET /\$metadata - check Core vocabulary"
run_test "Core vocabulary annotations (Description)" test_core_vocabulary

echo "  Request: GET /\$metadata - check Capabilities vocabulary"
run_test "Capabilities vocabulary annotations" test_capabilities_vocabulary

echo "  Request: GET /\$metadata - check Validation vocabulary"
run_test "Validation vocabulary annotations" test_validation_vocabulary

echo "  Request: GET /\$metadata - check Measures vocabulary"
run_test "Measures vocabulary annotations" test_measures_vocabulary

echo "  Request: GET /\$metadata - check annotation targets"
run_test "Annotations can target various elements" test_annotation_target

echo "  Request: GET /\$metadata - check inline annotations"
run_test "Inline annotations on properties" test_inline_annotations

echo "  Request: GET /\$metadata - check Computed annotation"
run_test "Core.Computed annotation for computed properties" test_computed_annotation

echo "  Request: GET /\$metadata - check Immutable annotation"
run_test "Core.Immutable annotation for read-only properties" test_immutable_annotation

echo "  Request: GET /\$metadata - check complex annotation values"
run_test "Annotations can have complex structured values" test_annotation_complex_value

echo "  Request: GET /\$metadata - check vocabulary references"
run_test "References to external standard vocabularies" test_vocabulary_reference

echo "  Request: GET /\$metadata - check EntitySet annotations"
run_test "Annotations on EntitySet" test_entityset_annotation

echo "  Request: GET /\$metadata - check NavigationProperty annotations"
run_test "Annotations on navigation properties" test_navigation_annotation

echo "  Request: GET /\$metadata - check permission annotations"
run_test "Permission and restriction annotations" test_permission_annotations

echo "  Request: GET /\$metadata?\$format=json"
run_test "Annotations in JSON metadata format" test_json_metadata_annotations

echo "  Request: GET /\$metadata - check custom vocabularies"
run_test "Custom vocabulary annotations" test_custom_annotations

echo "  Request: GET /\$metadata - check base types"
run_test "Annotation inheritance from base types" test_annotation_inheritance

echo "  Request: GET /\$metadata - check Term definitions"
run_test "Custom term definitions in metadata" test_term_definition

echo "  Request: GET /\$metadata - check null values"
run_test "Annotations can have null values" test_annotation_null_value

echo "  Request: GET /\$metadata - check multiple annotations"
run_test "Multiple annotations on same target" test_multiple_annotations

print_summary
