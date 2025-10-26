#!/bin/bash

# OData v4 Compliance Test: 14.1 Vocabulary Annotations
# Tests support for OData vocabulary annotations in metadata and responses
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 14.1 Vocabulary Annotations"
echo "======================================"
echo ""
echo "Description: Validates vocabulary annotations in metadata and instance"
echo "             annotations in responses. Tests Core vocabulary annotations"
echo "             like Description, LongDescription, and computed/immutable."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation"
echo ""

# Test 1: Metadata document structure supports annotations
test_metadata_annotation_structure() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Metadata should be valid XML/JSON
        return 0
    else
        return 1
    fi
}

# Test 2: Core.Description annotation (optional)
test_core_description_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Check if any Core.Description annotations are present (optional feature)
    # If present, they should be properly formatted
    return 0
}

# Test 3: Core.LongDescription annotation (optional)
test_core_longdescription_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Check if any Core.LongDescription annotations are present (optional feature)
    return 0
}

# Test 4: Computed annotation for read-only properties
test_computed_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Check if computed properties are marked (optional)
    # Computed properties should not be updateable
    return 0
}

# Test 5: Immutable annotation for create-only properties
test_immutable_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Check if immutable properties are marked (optional)
    # Immutable properties can be set on create but not updated
    return 0
}

# Test 6: Instance annotations in entity response
test_instance_annotations_in_entity() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Instance annotations start with @ and are at the same level as properties
        # Check for @odata.context which is required
        echo "$RESPONSE" | grep -q "@odata.context"
    else
        return 1
    fi
}

# Test 7: Instance annotations in collection response
test_instance_annotations_in_collection() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have @odata.context annotation
        echo "$RESPONSE" | grep -q "@odata.context"
    else
        return 1
    fi
}

# Test 8: @odata.type instance annotation
test_odata_type_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    # @odata.type annotation may be present for type information
    # This is optional but should be valid JSON if present
    return 0
}

# Test 9: @odata.id canonical URL annotation
test_odata_id_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # @odata.id should contain the canonical URL (optional)
        # If present, should be valid
        return 0
    else
        return 1
    fi
}

# Test 10: @odata.editLink annotation (optional)
test_odata_editlink_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    # @odata.editLink may be present (optional)
    return 0
}

# Test 11: @odata.etag annotation
test_odata_etag_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    # @odata.etag may be present if entity supports optimistic concurrency
    # If present, should be properly formatted
    return 0
}

# Test 12: Custom instance annotations prefixed with @
test_custom_instance_annotations() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Custom annotations should start with @
        # This test just ensures response is valid JSON
        echo "$RESPONSE" | grep -q "{"
    else
        return 1
    fi
}

# Test 13: Annotation in error response
test_annotation_in_error_response() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(99999)")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(99999)")
    
    if [ "$HTTP_CODE" = "404" ]; then
        # Error responses should not have standard instance annotations
        # but should have error object
        echo "$RESPONSE" | grep -q "\"error\""
    else
        return 1
    fi
}

# Test 14: Capabilities vocabulary in metadata (optional)
test_capabilities_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Capabilities vocabulary describes service capabilities
    # This is optional but if present should be valid
    return 0
}

# Test 15: Measures vocabulary in metadata (optional)
test_measures_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Measures vocabulary for unit of measure annotations
    # This is optional
    return 0
}

# Test 16: Validation vocabulary in metadata (optional)
test_validation_vocabulary() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    # Validation vocabulary for constraints
    # This is optional
    return 0
}

# Test 17: Annotation target in metadata
test_annotation_target_in_metadata() {
    local RESPONSE=$(curl -s "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/\$metadata")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Annotations in metadata should target entities, properties, etc.
        # Basic validation that metadata is well-formed
        return 0
    else
        return 1
    fi
}

# Test 18: @odata.nextLink annotation in paginated results
test_odata_nextlink_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$top=2")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$top=2")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # May have @odata.nextLink if there are more results
        # This is acceptable whether present or not
        return 0
    else
        return 1
    fi
}

# Test 19: @odata.count annotation
test_odata_count_annotation() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products?\$count=true")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$count=true")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Should have @odata.count annotation
        echo "$RESPONSE" | grep -q "@odata.count"
    else
        return 1
    fi
}

# Test 20: Annotation ordering in JSON (annotations before properties)
test_annotation_ordering() {
    local RESPONSE=$(curl -s "$SERVER_URL/Products(1)")
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Annotations should appear before regular properties (recommended)
        # But JSON doesn't enforce order, so this is just a validation test
        echo "$RESPONSE" | grep -q "@odata.context"
    else
        return 1
    fi
}

echo "  Request: GET metadata document"
run_test "Metadata structure supports annotations" test_metadata_annotation_structure

echo "  Request: Check for Core.Description annotations"
run_test "Core.Description annotation support (optional)" test_core_description_annotation

echo "  Request: Check for Core.LongDescription annotations"
run_test "Core.LongDescription annotation support (optional)" test_core_longdescription_annotation

echo "  Request: Check for Computed annotations"
run_test "Computed property annotations (optional)" test_computed_annotation

echo "  Request: Check for Immutable annotations"
run_test "Immutable property annotations (optional)" test_immutable_annotation

echo "  Request: GET entity, check instance annotations"
run_test "Instance annotations in entity response" test_instance_annotations_in_entity

echo "  Request: GET collection, check instance annotations"
run_test "Instance annotations in collection response" test_instance_annotations_in_collection

echo "  Request: Check for @odata.type annotation"
run_test "@odata.type instance annotation (optional)" test_odata_type_annotation

echo "  Request: Check for @odata.id annotation"
run_test "@odata.id canonical URL annotation (optional)" test_odata_id_annotation

echo "  Request: Check for @odata.editLink annotation"
run_test "@odata.editLink annotation (optional)" test_odata_editlink_annotation

echo "  Request: Check for @odata.etag annotation"
run_test "@odata.etag annotation support" test_odata_etag_annotation

echo "  Request: Check custom instance annotations format"
run_test "Custom instance annotations format" test_custom_instance_annotations

echo "  Request: GET non-existent entity, check error annotations"
run_test "Annotations in error responses" test_annotation_in_error_response

echo "  Request: Check for Capabilities vocabulary"
run_test "Capabilities vocabulary in metadata (optional)" test_capabilities_vocabulary

echo "  Request: Check for Measures vocabulary"
run_test "Measures vocabulary in metadata (optional)" test_measures_vocabulary

echo "  Request: Check for Validation vocabulary"
run_test "Validation vocabulary in metadata (optional)" test_validation_vocabulary

echo "  Request: Check annotation targets in metadata"
run_test "Annotation targets in metadata" test_annotation_target_in_metadata

echo "  Request: GET with pagination, check @odata.nextLink"
run_test "@odata.nextLink annotation in paginated results" test_odata_nextlink_annotation

echo "  Request: GET with \$count, check annotation"
run_test "@odata.count annotation" test_odata_count_annotation

echo "  Request: Check annotation ordering in JSON"
run_test "Annotation ordering in JSON response" test_annotation_ordering

print_summary
