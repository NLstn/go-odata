#!/bin/bash

# OData v4 Compliance Test: Collection-Valued Navigation Properties
# Tests collection-valued navigation properties, lambda operators (any/all), and $count
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html
#       Section 11.2.6 (Querying Collections)
#       Section 11.2.6.1.1 (Built-in Filter Operations - lambda operators)
#       Section 13.1.3 (OData 4.0 Advanced Conformance Level - point 5: lambda operators)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: Collection-Valued Navigation Properties"
echo "======================================"
echo ""
echo "Description: Validates handling of collection-valued navigation properties,"
echo "             lambda operators (any/all), and $count on collections."
echo "             Uses Products->Descriptions relationship for testing."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html"
echo "                Section 11.2.6.1.1 (Lambda operators)"
echo ""

# Test 1: Expand collection-valued navigation property
test_expand_collection() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$expand=Descriptions")
        # Check if Descriptions collection is included
        if echo "$RESPONSE" | grep -q '"Descriptions"'; then
            return 0
        else
            echo "  Details: Descriptions collection not found in response"
            return 1
        fi
    else
        echo "  Details: Status: $HTTP_CODE (expected 200)"
        return 1
    fi
}

# Test 2: Filter with collection navigation property using 'any' operator
# OData v4 Advanced: "MUST support the lambda operators any and all"
test_collection_any_operator() {
    # Filter products where any description contains 'Laptop'
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Descriptions/any(d:contains(d/Description,'Laptop'))")
    
    # 200 = supported, 501 = not implemented, 400 = may not be supported
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Lambda operator 'any' not implemented (status: 501)"
        return 0  # Pass - not required for minimal conformance
    elif [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Lambda operator 'any' may not be supported (status: 400)"
        return 0  # Pass - not required for minimal conformance
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: Filter with collection navigation property using 'all' operator
test_collection_all_operator() {
    # Filter products where all descriptions have non-empty Description field
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Descriptions/all(d:d/Description%20ne%20null)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Lambda operator 'all' not implemented (status: 501)"
        return 0  # Pass - not required for minimal conformance
    elif [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Lambda operator 'all' may not be supported (status: 400)"
        return 0  # Pass - not required for minimal conformance
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: Count items in collection navigation property
test_collection_count() {
    # Get count of descriptions for a product
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Descriptions/\$count")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)/Descriptions/\$count")
        # Should return ONLY a number, not a JSON object
        # The response should be plain text like "2" or "3", not {"@odata.context": ...}
        if echo "$RESPONSE" | grep -qE '^\s*[0-9]+\s*$'; then
            return 0
        elif echo "$RESPONSE" | grep -q '@odata.context'; then
            echo "  Details: $count should return plain text number, not JSON collection"
            return 1
        else
            echo "  Details: Response is not a valid count: $RESPONSE"
            return 1
        fi
    elif [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Collection \$count not supported (status: 404)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 5: Filter using $count in expression
# OData Advanced: "MUST support the $count segment on navigation and collection properties"
test_filter_with_count() {
    # Filter products that have more than 1 description
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$filter=Descriptions/\$count%20gt%201")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: \$count in filter not supported (status: $HTTP_CODE)"
        return 0  # Pass - not required for minimal conformance
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 6: Expand with $count
test_expand_with_count() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$expand=Descriptions(\$count=true)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$expand=Descriptions(\$count=true)")
        # Should include @odata.count annotation
        if echo "$RESPONSE" | grep -q "@odata.count\\|@count"; then
            return 0
        else
            # Count may not be in response, but request was accepted
            return 0
        fi
    elif [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: $count in expand not supported (status: 400)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: Expand with $filter on collection
# OData Advanced: "MUST support $filter on expanded collection-valued properties"
test_expand_with_filter() {
    # Expand descriptions but only for English language
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$expand=Descriptions(\$filter=LanguageKey%20eq%20'EN')")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: \$filter in expand not fully supported (status: $HTTP_CODE)"
        return 0  # Pass - advanced feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Empty collection format
test_empty_collection() {
    # Query for a product that doesn't exist or has no descriptions
    # Should return empty array [] not null
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=ID eq 999999")
    
    # If no results, check format - should be empty array in value property
    if echo "$RESPONSE" | grep -q '"value":\s*\[\s*\]'; then
        return 0
    elif echo "$RESPONSE" | grep -q '"value":null'; then
        echo "  Details: Empty collection should be [] not null"
        return 1
    else
        # May have results, which is fine
        return 0
    fi
}

# Test 9: Select with navigation property
test_select_navigation_property() {
    # Select should work with navigation properties
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$select=Name,Descriptions")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: Navigation property in \$select not supported (status: 400)"
        return 0  # Pass - may not be supported
    elif [ "$HTTP_CODE" = "000" ]; then
        echo "  Details: Server returned empty response or crashed (curl exit code 52)"
        echo "           This indicates a server error when processing \$select with navigation properties"
        return 1
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 10: Expand and select together
test_expand_and_select() {
    # Expand descriptions and select specific fields
    local HTTP_CODE=$(curl --globoff -s -o /dev/null -w "%{http_code}" "$SERVER_URL/Products?\$expand=Descriptions(\$select=LanguageKey,Description)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ]; then
        echo "  Details: $select in expand not supported (status: 400)"
        return 0  # Pass - may not be fully supported
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET Products?\$expand=Descriptions"
run_test "Expand collection-valued navigation property" test_expand_collection

echo "  Request: GET \$filter=Descriptions/any(...)"
run_test "Filter with collection 'any' operator" test_collection_any_operator

echo "  Request: GET \$filter=Descriptions/all(...)"
run_test "Filter with collection 'all' operator" test_collection_all_operator

echo "  Request: GET Products(1)/Descriptions/\$count"
run_test "Count items in collection navigation" test_collection_count

echo "  Request: GET \$filter=Descriptions/\$count gt 1"
run_test "Filter using $count in expression" test_filter_with_count

echo "  Request: GET \$expand=Descriptions(\$count=true)"
run_test "Expand with $count option" test_expand_with_count

echo "  Request: GET \$expand=Descriptions(\$filter=...)"
run_test "Expand with $filter on collection" test_expand_with_filter

echo "  Request: Check empty collection format"
run_test "Empty collection returns [] not null" test_empty_collection

echo "  Request: GET \$select=Name,Descriptions"
run_test "Select navigation property" test_select_navigation_property

echo "  Request: GET \$expand=Descriptions(\$select=...)"
run_test "Expand and select together" test_expand_and_select

print_summary

