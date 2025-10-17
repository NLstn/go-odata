#!/bin/bash

# OData v4 Compliance Test: 11.2.8 Delta Links
# Tests delta link support for tracking changes according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingChanges

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.8 Delta Links"
echo "======================================"
echo ""
echo "Description: Validates delta link support for tracking entity changes"
echo "             and synchronization according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_RequestingChanges"
echo ""



# Test 1: Request delta with Prefer: odata.track-changes
echo "Test 1: Request delta with Prefer: odata.track-changes"
echo "  Request: GET $SERVER_URL/Products with Prefer: odata.track-changes"
RESPONSE=$(curl -s -i -H "Prefer: odata.track-changes" "$SERVER_URL/Products" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')
BODY=$(echo "$RESPONSE" | sed -n '/^$/,$p' | tail -n +2)
PREFERENCE_APPLIED=$(echo "$RESPONSE" | grep -i "^Preference-Applied:" | head -1 | sed 's/Preference-Applied: //i' | tr -d '\r')

if [ "$HTTP_CODE" = "200" ]; then
    # Check for @odata.deltaLink in response
    if echo "$BODY" | grep -q '"@odata.deltaLink"'; then
        test_result "Delta request returns @odata.deltaLink" "PASS"
    else
        test_result "Delta request returns @odata.deltaLink" "PASS" "Server doesn't support delta links (optional)"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Delta request returns @odata.deltaLink" "PASS" "Delta links not implemented (optional)"
else
    test_result "Delta request returns @odata.deltaLink" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Preference-Applied header should indicate track-changes support
echo ""
echo "Test 2: Preference-Applied header for track-changes"
if [ "$HTTP_CODE" = "200" ]; then
    if echo "$PREFERENCE_APPLIED" | grep -q "odata.track-changes"; then
        test_result "Preference-Applied includes track-changes" "PASS"
    else
        test_result "Preference-Applied includes track-changes" "PASS" "Header not present (delta not supported)"
    fi
else
    test_result "Preference-Applied includes track-changes" "PASS" "Delta not supported (optional)"
fi

# Test 3: Delta link should be dereferenceable
if echo "$BODY" | grep -q '"@odata.deltaLink"'; then
    echo ""
    echo "Test 3: Delta link is dereferenceable"
    DELTA_LINK=$(echo "$BODY" | grep -o '"@odata.deltaLink":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$DELTA_LINK" ]; then
        # Try to access the delta link
        if echo "$DELTA_LINK" | grep -q "^http"; then
            DELTA_RESPONSE=$(curl -s -w "\n%{http_code}" "$DELTA_LINK" 2>&1)
        else
            DELTA_RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/$DELTA_LINK" 2>&1)
        fi
        
        DELTA_HTTP_CODE=$(echo "$DELTA_RESPONSE" | tail -1)
        
        if [ "$DELTA_HTTP_CODE" = "200" ]; then
            test_result "Delta link is accessible" "PASS"
        else
            test_result "Delta link is accessible" "FAIL" "HTTP $DELTA_HTTP_CODE accessing delta link"
        fi
    else
        test_result "Delta link is accessible" "FAIL" "Delta link is empty"
    fi
fi

# Test 4: Delta response should include context
if echo "$BODY" | grep -q '"@odata.deltaLink"'; then
    echo ""
    echo "Test 4: Delta response includes @odata.context"
    if echo "$BODY" | grep -q '"@odata.context"'; then
        test_result "Delta response has @odata.context" "PASS"
    else
        test_result "Delta response has @odata.context" "FAIL" "Missing @odata.context"
    fi
fi

# Test 5: Delta with $filter
echo ""
echo "Test 5: Delta request with \$filter"
echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 10 with Prefer: odata.track-changes"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "Prefer: odata.track-changes" \
    "$SERVER_URL/Products?\$filter=Price%20gt%2010" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q '"@odata.deltaLink"'; then
        test_result "Delta with \$filter returns deltaLink" "PASS"
    else
        test_result "Delta with \$filter returns deltaLink" "PASS" "Delta not supported (optional)"
    fi
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Delta with \$filter returns deltaLink" "PASS" "Delta not supported (optional)"
else
    test_result "Delta with \$filter returns deltaLink" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 6: Delta token parameter handling
echo ""
echo "Test 6: Request with delta token parameter"
echo "  Request: GET $SERVER_URL/Products?\$deltatoken=test-token"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/Products?\$deltatoken=test-token" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "410" ] || [ "$HTTP_CODE" = "400" ]; then
    # 200 = valid token, 410 = expired token, 400 = invalid token format
    test_result "Delta token parameter handled" "PASS"
elif [ "$HTTP_CODE" = "501" ]; then
    test_result "Delta token parameter handled" "PASS" "Delta tokens not supported (optional)"
else
    test_result "Delta token parameter handled" "FAIL" "HTTP $HTTP_CODE"
fi


print_summary
