#!/bin/bash

# OData v4 Compliance Test: 11.2.5.7 $skiptoken Query Option
# Tests server-driven paging with $skiptoken
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionskiptoken

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.5.7 \$skiptoken"
echo "======================================"
echo ""
echo "Description: Validates server-driven paging with \$skiptoken for"
echo "             continuation of result sets."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionskiptoken"
echo ""

# Test 1: Response with @odata.nextLink includes skiptoken
test_nextlink_has_skiptoken() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$top=2")
    
    # Check if there's a nextLink
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        local NEXTLINK=$(echo "$RESPONSE" | grep -o '"@odata.nextLink":"[^"]*"' | head -1)
        
        # NextLink should contain skiptoken parameter
        if echo "$NEXTLINK" | grep -q "skiptoken"; then
            return 0
        else
            echo "  Details: @odata.nextLink present but missing \$skiptoken parameter"
            return 1
        fi
    else
        # If there's no nextLink, it means the result fits in one page
        # This is acceptable if there are few entities
        echo "  Details: No @odata.nextLink (result set fits in one page)"
        return 0
    fi
}

# Test 2: Following nextLink returns next page
test_follow_nextlink() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$top=2")
    
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        # Extract the nextLink URL
        local NEXTLINK=$(echo "$RESPONSE" | grep -o '"@odata.nextLink":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        # Follow the nextLink
        local HTTP_CODE=$(http_get "$NEXTLINK")
        
        check_status "$HTTP_CODE" "200"
    else
        echo "  Details: No @odata.nextLink to follow"
        return 0
    fi
}

# Test 3: skiptoken is opaque (service-generated)
test_skiptoken_opaque() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$top=2")
    
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        # The skiptoken value should be URL-encoded and opaque
        # We just verify it exists and is not empty
        local NEXTLINK=$(echo "$RESPONSE" | grep -o '"@odata.nextLink":"[^"]*"' | head -1)
        
        if echo "$NEXTLINK" | grep -q 'skiptoken=[^&"]\+'; then
            return 0
        else
            echo "  Details: \$skiptoken parameter is empty"
            return 1
        fi
    else
        echo "  Details: No @odata.nextLink to examine"
        return 0
    fi
}

# Test 4: Invalid skiptoken returns error
test_invalid_skiptoken() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$skiptoken=invalid_token_xyz")
    
    # Should return 400 for invalid skiptoken
    check_status "$HTTP_CODE" "400"
}

# Test 5: Combine skiptoken with other query options
test_skiptoken_with_filter() {
    # First get a page with filter
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price%20gt%200&\$top=2")
    
    if echo "$RESPONSE" | grep -q "@odata.nextLink"; then
        local NEXTLINK=$(echo "$RESPONSE" | grep -o '"@odata.nextLink":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        # The nextLink should preserve the filter
        if echo "$NEXTLINK" | grep -q 'filter'; then
            return 0
        else
            echo "  Details: \$filter not preserved in @odata.nextLink"
            return 1
        fi
    else
        echo "  Details: No @odata.nextLink (result set fits in one page)"
        return 0
    fi
}

# Test 6: Server-driven paging with Prefer maxpagesize
test_prefer_maxpagesize() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products" \
        -H "Prefer: odata.maxpagesize=2")
    
    # With maxpagesize=2, if there are more than 2 products, should have nextLink
    local VALUE_COUNT=$(echo "$RESPONSE" | grep -o '"ID"' | wc -l)
    
    if [ "$VALUE_COUNT" -le 2 ]; then
        # Either respects maxpagesize or has fewer entities
        return 0
    else
        echo "  Details: Server returned $VALUE_COUNT entities, expected max 2"
        return 1
    fi
}

echo "  Request: GET $SERVER_URL/Products?\$top=2"
run_test "@odata.nextLink contains \$skiptoken parameter" test_nextlink_has_skiptoken

echo "  Request: Follow @odata.nextLink"
run_test "Following @odata.nextLink returns next page (200 OK)" test_follow_nextlink

echo "  Request: GET $SERVER_URL/Products?\$top=2"
run_test "\$skiptoken value is opaque and non-empty" test_skiptoken_opaque

echo "  Request: GET $SERVER_URL/Products?\$skiptoken=invalid_token_xyz"
run_test "Invalid \$skiptoken returns 400 Bad Request" test_invalid_skiptoken

echo "  Request: GET $SERVER_URL/Products?\$filter=Price gt 0&\$top=2"
run_test "\$skiptoken preserves other query options like \$filter" test_skiptoken_with_filter

echo "  Request: GET $SERVER_URL/Products with Prefer: odata.maxpagesize=2"
run_test "Server-driven paging respects Prefer: odata.maxpagesize" test_prefer_maxpagesize

print_summary
