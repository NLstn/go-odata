#!/bin/bash

# OData v4 Compliance Test: 11.2.2 Canonical URL
# Tests canonical URL representation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CanonicalURL

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.2 Canonical URL"
echo "======================================"
echo ""
echo "Description: Validates that entities have canonical URLs in @odata.id"
echo "             and that these URLs can be used to access the resource."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CanonicalURL"
echo ""



# Test 1: Entity should have @odata.id with canonical URL
echo "Test 1: Entity has @odata.id with canonical URL"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"@odata.id"'; then
        ODATA_ID=$(echo "$BODY" | grep -o '"@odata.id":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -n "$ODATA_ID" ]; then
            test_result "Entity contains @odata.id" "PASS"
        else
            test_result "Entity contains @odata.id" "FAIL" "@odata.id field is empty"
        fi
    else
        test_result "Entity contains @odata.id" "FAIL" "No @odata.id found in response"
    fi
else
    test_result "Entity contains @odata.id" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Canonical URL should be dereferenceable
echo ""
echo "Test 2: @odata.id URL is dereferenceable"
if [ -n "$ODATA_ID" ]; then
    echo "  Request: GET $ODATA_ID"
    # Extract relative path from @odata.id if it's a full URL
    if echo "$ODATA_ID" | grep -q "^http"; then
        # Full URL
        CANONICAL_RESPONSE=$(curl -s -w "\n%{http_code}" "$ODATA_ID" 2>&1)
    else
        # Relative URL
        CANONICAL_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/$ODATA_ID" 2>&1)
    fi
    
    CANONICAL_HTTP_CODE=$(echo "$CANONICAL_RESPONSE" | tail -1)
    
    if [ "$CANONICAL_HTTP_CODE" = "200" ]; then
        test_result "Canonical URL is dereferenceable" "PASS"
    else
        test_result "Canonical URL is dereferenceable" "FAIL" "HTTP $CANONICAL_HTTP_CODE when accessing @odata.id URL"
    fi
else
    test_result "Canonical URL is dereferenceable" "FAIL" "No @odata.id to test"
fi

# Test 3: Collection should have @odata.id for each entity
echo ""
echo "Test 3: Each entity in collection has @odata.id"
echo "  Request: GET $SERVER_URL/Products?\$top=3"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$top=3" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    # Count entities in value array
    ENTITY_COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)
    # Count @odata.id occurrences
    ODATA_ID_COUNT=$(echo "$BODY" | grep -o '"@odata.id"' | wc -l)
    
    if [ "$ENTITY_COUNT" -gt 0 ] && [ "$ODATA_ID_COUNT" -eq "$ENTITY_COUNT" ]; then
        test_result "Each entity in collection has @odata.id" "PASS"
    else
        test_result "Each entity in collection has @odata.id" "FAIL" "Found $ENTITY_COUNT entities but $ODATA_ID_COUNT @odata.id fields"
    fi
else
    test_result "Each entity in collection has @odata.id" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 4: Canonical URL format should match entity set and key
echo ""
echo "Test 4: Canonical URL format matches entity set and key pattern"
echo "  Request: GET $SERVER_URL/Products(1)"
RESPONSE=$(curl -s "$SERVER_URL/Products(1)" 2>&1)

if echo "$RESPONSE" | grep -q '"@odata.id"'; then
    ODATA_ID=$(echo "$RESPONSE" | grep -o '"@odata.id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    # Check if URL matches pattern: .../Products(1) or similar
    if echo "$ODATA_ID" | grep -qE 'Products\([0-9]+\)'; then
        test_result "Canonical URL format is correct" "PASS"
    else
        test_result "Canonical URL format is correct" "FAIL" "URL format does not match expected pattern: $ODATA_ID"
    fi
else
    test_result "Canonical URL format is correct" "FAIL" "No @odata.id found"
fi


print_summary
