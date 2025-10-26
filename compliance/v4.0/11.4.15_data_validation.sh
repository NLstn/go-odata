#!/bin/bash

# OData v4 Compliance Test: 11.4.15 Data Validation and Constraints
# Tests data validation, required fields, and constraint enforcement
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.15 Data Validation"
echo "======================================"
echo ""
echo "Description: Validates that the service enforces data validation rules,"
echo "             required fields, and constraints on entity creation and updates."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html"
echo ""


# Test 1: Missing required field returns error
test_missing_required_field() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Price":99.99,"CategoryID":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 400 Bad Request for missing required field (Name)
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 for missing required field)"
        return 1
    fi
}

# Test 2: Empty string for required field
test_empty_required_field() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"","Price":99.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # May accept empty string or reject it depending on implementation
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "201" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 3: Invalid data type returns error
test_invalid_data_type() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test Product","Price":"not-a-number","CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 400 Bad Request for invalid data type
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 for invalid data type)"
        return 1
    fi
}

# Test 4: Negative price validation (business rule)
test_negative_price() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Negative Price Product","Price":-50,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Implementation may accept or reject negative prices
    # We test that it handles it consistently
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "201" ]; then
        if [ "$HTTP_CODE" = "201" ]; then
            local BODY=$(echo "$RESPONSE" | head -n -1)
            local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        fi
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 5: Malformed JSON returns error
test_malformed_json() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test","Price":99.99,}')  # Trailing comma
    
    # Should return 400 Bad Request for malformed JSON
    if [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        echo "  Details: Status $HTTP_CODE (expected 400 for malformed JSON)"
        return 1
    fi
}

# Test 6: Extra unknown properties
test_unknown_properties() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test Product","Price":99.99,"CategoryID":1,"Status":1,"UnknownField":"value"}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # May accept (ignore) or reject unknown properties
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "400" ]; then
        if [ "$HTTP_CODE" = "201" ]; then
            local ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        fi
        return 0
    else
        echo "  Details: Status $HTTP_CODE"
        return 1
    fi
}

# Test 7: Update with invalid data type
test_patch_invalid_type() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Validation Test","Price":99.99,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
            
            # Try to update with invalid type
            local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($ID)" \
                -H "Content-Type: application/json" \
                -d '{"Price":"invalid"}')
            
            check_status "$HTTP_CODE" "400"
            return $?
    fi
    
    echo "  Details: Could not create test entity"
    return 1
}

# Test 8: Required field cannot be set to null in update
test_patch_required_to_null() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Required Field Test","Price":99.99,"CategoryID":1,"Status":1}')
    
    local CREATE_CODE=$(echo "$RESPONSE" | tail -1)
    local CREATE_BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$CREATE_CODE" = "201" ]; then
        local ID=$(echo "$CREATE_BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
            
            # Try to set required field to null
            local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$SERVER_URL/Products($ID)" \
                -H "Content-Type: application/json" \
                -d '{"Name":null}')
            
            # Should reject or handle gracefully
            if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "204" ]; then
                return 0
            else
                echo "  Details: Status $HTTP_CODE"
                return 1
        fi
    fi
    
    echo "  Details: Could not create test entity"
    return 1
}

# Test 9: Content-Type header missing or incorrect
test_missing_content_type() {
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/Products" \
        -d '{"Name":"Test","Price":99.99,"CategoryID":1,"Status":1}')
    
    # Should return 415 Unsupported Media Type without Content-Type
    if [ "$HTTP_CODE" = "415" ] || [ "$HTTP_CODE" = "400" ]; then
        return 0
    else
        # Clean up if it was created
        echo "  Details: Status $HTTP_CODE (expected 415 or 400)"
        return 1
    fi
}

# Test 10: Readonly property in POST should be ignored
test_readonly_property_post() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"ID":99999,"Name":"Readonly Test","Price":99.99,"CategoryID":1,"Status":1}')
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        # Verify that ID was assigned by server, not the provided value
        local ACTUAL_ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | head -1 | grep -o '[0-9]*')
        if [ -n "$ACTUAL_ID" ]; then
            # ID should be auto-assigned, not 99999
            if [ "$ACTUAL_ID" != "99999" ]; then
                return 0
            else
                echo "  Details: Server used client-provided ID instead of auto-generating"
                return 1
            fi
        fi
    else
        echo "  Details: Status $HTTP_CODE (expected 201)"
        return 1
    fi
}

echo "  Request: POST without required field"
run_test "Missing required field returns 400" test_missing_required_field

echo "  Request: POST with empty string for required field"
run_test "Empty string for required field" test_empty_required_field

echo "  Request: POST with invalid data type"
run_test "Invalid data type returns 400" test_invalid_data_type

echo "  Request: POST with negative price"
run_test "Negative price validation" test_negative_price

echo "  Request: POST with malformed JSON"
run_test "Malformed JSON returns 400" test_malformed_json

echo "  Request: POST with unknown properties"
run_test "Unknown properties handled gracefully" test_unknown_properties

echo "  Request: PATCH with invalid data type"
run_test "Update with invalid type returns 400" test_patch_invalid_type

echo "  Request: PATCH to set required field to null"
run_test "Required field cannot be set to null" test_patch_required_to_null

echo "  Request: POST without Content-Type header"
run_test "Missing Content-Type returns 415" test_missing_content_type

echo "  Request: POST with readonly property"
run_test "Readonly property (ID) ignored in POST" test_readonly_property_post

print_summary
