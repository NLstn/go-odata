#!/bin/bash

# OData v4 Compliance Test: 8.2.8 Header Prefer
# Tests Prefer header and Preference-Applied response header
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderPrefer

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.8 Header Prefer"
echo "======================================"
echo ""
echo "Description: Validates Prefer header support for controlling response behavior"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderPrefer"
echo ""

CREATED_IDS=()

cleanup() {
    echo ""
    echo "Cleaning up test data..."
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

trap cleanup EXIT


# Test 1: Prefer: return=minimal returns 204 No Content
echo "Test 1: Prefer: return=minimal on POST returns 204 No Content"
echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"
RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -H "Prefer: return=minimal" \
    -d '{"Name":"Minimal Response Test","Price":50.00,"Category":"Test","Status":1}' 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | sed 's/Location: //i' | tr -d '\r')

if [ "$HTTP_CODE" = "204" ]; then
    test_result "Prefer: return=minimal returns 204 No Content" "PASS"
    
    # Extract ID from Location header for cleanup
    if [ -n "$LOCATION" ]; then
        ID=$(echo "$LOCATION" | grep -o '[0-9]*' | tail -1)
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
elif [ "$HTTP_CODE" = "201" ]; then
    test_result "Prefer: return=minimal returns 204" "FAIL" "Returned 201 instead (preference ignored)"
    
    # Try to extract ID for cleanup
    BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
    ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | grep -o '[0-9]*' | head -1)
    if [ -n "$ID" ]; then
        CREATED_IDS+=("$ID")
    fi
else
    test_result "Prefer: return=minimal returns 204" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Prefer: return=representation returns 201 with body
echo "Test 2: Prefer: return=representation on POST returns 201 with body"
echo "  Request: POST $SERVER_URL/Products with Prefer: return=representation"
RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -H "Prefer: return=representation" \
    -d '{"Name":"Representation Test","Price":75.00,"Category":"Test","Status":1}' 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')

if [ "$HTTP_CODE" = "201" ]; then
    if echo "$BODY" | grep -q '"ID"'; then
        test_result "Prefer: return=representation returns 201 with body" "PASS"
        
        # Extract ID for cleanup
        ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | grep -o '[0-9]*' | head -1)
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    else
        test_result "Prefer: return=representation returns body" "FAIL" "No body in response"
    fi
else
    test_result "Prefer: return=representation returns 201" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 3: Check Preference-Applied header when preference is honored
echo "Test 3: Preference-Applied header indicates honored preference"
echo "  Request: POST $SERVER_URL/Products with Prefer: return=minimal"
RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
    -H "Content-Type: application/json" \
    -H "Prefer: return=minimal" \
    -d '{"Name":"Preference Applied Test","Price":25.00,"Category":"Test","Status":1}' 2>&1)

PREF_APPLIED=$(echo "$RESPONSE" | grep -i "^Preference-Applied:" | sed 's/Preference-Applied: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')

if [ -n "$PREF_APPLIED" ]; then
    if echo "$PREF_APPLIED" | grep -q "return=minimal"; then
        test_result "Preference-Applied header present with return=minimal" "PASS"
    else
        test_result "Preference-Applied header contains correct value" "FAIL" "Value: $PREF_APPLIED"
    fi
    
    # Extract ID for cleanup
    LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | sed 's/Location: //i' | tr -d '\r')
    if [ -n "$LOCATION" ]; then
        ID=$(echo "$LOCATION" | grep -o '[0-9]*' | tail -1)
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
else
    # Preference-Applied is optional, so this is not necessarily a failure
    test_result "Preference-Applied header" "PASS" "Header not present (optional feature)"
    
    # Try to get ID for cleanup anyway
    if [ "$HTTP_CODE" = "204" ]; then
        LOCATION=$(echo "$RESPONSE" | grep -i "^Location:" | sed 's/Location: //i' | tr -d '\r')
        if [ -n "$LOCATION" ]; then
            ID=$(echo "$LOCATION" | grep -o '[0-9]*' | tail -1)
            if [ -n "$ID" ]; then
                CREATED_IDS+=("$ID")
            fi
        fi
    else
        BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
        ID=$(echo "$BODY" | grep -o '"ID":[0-9]*' | grep -o '[0-9]*' | head -1)
        if [ -n "$ID" ]; then
            CREATED_IDS+=("$ID")
        fi
    fi
fi
echo ""

# Test 4: Prefer: odata.maxpagesize limits results
echo "Test 4: Prefer: odata.maxpagesize limits number of results"
echo "  Request: GET $SERVER_URL/Products with Prefer: odata.maxpagesize=2"
RESPONSE=$(curl -s -i "$SERVER_URL/Products" \
    -H "Prefer: odata.maxpagesize=2" 2>&1)

BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
COUNT=$(echo "$BODY" | grep -o '"ID"' | wc -l)

if [ "$COUNT" -le 2 ]; then
    test_result "Prefer: odata.maxpagesize=2 limits results" "PASS"
else
    test_result "Prefer: odata.maxpagesize limits results" "FAIL" "Returned $COUNT items (expected max 2)"
fi
echo ""


print_summary
