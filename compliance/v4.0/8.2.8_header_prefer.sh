#!/bin/bash

# OData v4 Compliance Test: 8.2.8 Header Prefer
# Tests Prefer header and Preference-Applied response header
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.8 Header Prefer"
echo "======================================"
echo ""
echo "Description: Validates Prefer header support for controlling response behavior"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer"
echo ""

# Test 1: POST without Prefer header returns representation by default
test_post_default_returns_representation() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Test Product","Price":99.99,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    local BODY=$(echo "$RESPONSE" | head -n -1)
    
    # Should return 201 Created with body
    if ! check_status "$HTTP_CODE" "201"; then
        return 1
    fi
    
    # Body should contain the created entity
    if ! check_contains "$BODY" '"Name"'; then
        echo "  Details: Response body should contain created entity"
        return 1
    fi
    
    return 0
}

# Test 2: POST with Prefer: return=minimal returns 204 No Content
test_post_prefer_minimal() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Name":"Minimal Product","Price":49.99,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should return 204 No Content
    if ! check_status "$HTTP_CODE" "204"; then
        return 1
    fi
    
    # Preference-Applied header should be present
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:.*return=minimal"; then
        echo "  Details: Preference-Applied header should be 'return=minimal'"
        return 1
    fi
    
    # Location header should still be present
    if ! echo "$RESPONSE" | grep -qi "Location:"; then
        echo "  Details: Location header should be present even with return=minimal"
        return 1
    fi
    
    return 0
}

# Test 3: POST with explicit Prefer: return=representation
test_post_prefer_representation() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Full Product","Price":149.99,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
    
    # Should return 201 Created
    if ! check_status "$HTTP_CODE" "201"; then
        return 1
    fi
    
    # Preference-Applied header should be present
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:.*return=representation"; then
        echo "  Details: Preference-Applied header should be 'return=representation'"
        return 1
    fi
    
    # Body should contain the created entity
    if ! check_contains "$BODY" '"Name"'; then
        echo "  Details: Response body should contain created entity"
        return 1
    fi
    
    return 0
}

# Test 4: PATCH without Prefer header returns 204 No Content by default
test_patch_default_no_content() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$SERVER_URL/Products(1)" \
        -H "Content-Type: application/json" \
        -d '{"Price":199.99}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 204 No Content by default
    check_status "$HTTP_CODE" "204"
}

# Test 5: PATCH with Prefer: return=representation returns updated entity
test_patch_prefer_representation() {
    local RESPONSE=$(curl -s -i -X PATCH "$SERVER_URL/Products(2)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Price":299.99}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
    
    # Should return 200 OK with body
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    # Preference-Applied header should be present
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:.*return=representation"; then
        echo "  Details: Preference-Applied header should be 'return=representation'"
        return 1
    fi
    
    # Body should contain the updated entity with new price
    if ! check_contains "$BODY" '"Price"'; then
        echo "  Details: Response body should contain updated entity"
        return 1
    fi
    
    return 0
}

# Test 6: PATCH with explicit Prefer: return=minimal
test_patch_prefer_minimal() {
    local RESPONSE=$(curl -s -i -X PATCH "$SERVER_URL/Products(3)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal" \
        -d '{"Price":99.99}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should return 204 No Content
    if ! check_status "$HTTP_CODE" "204"; then
        return 1
    fi
    
    # Preference-Applied header should be present
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:.*return=minimal"; then
        echo "  Details: Preference-Applied header should be 'return=minimal'"
        return 1
    fi
    
    return 0
}

# Test 7: PUT without Prefer header returns 204 No Content by default
test_put_default_no_content() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$SERVER_URL/Products(4)" \
        -H "Content-Type: application/json" \
        -d '{"Name":"Updated Product","Price":499.99,"CategoryID":2,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 204 No Content by default
    check_status "$HTTP_CODE" "204"
}

# Test 8: PUT with Prefer: return=representation returns updated entity
test_put_prefer_representation() {
    local RESPONSE=$(curl -s -i -X PUT "$SERVER_URL/Products(5)" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=representation" \
        -d '{"Name":"Full Update","Price":599.99,"CategoryID":2,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    local BODY=$(echo "$RESPONSE" | sed -n '/^{/,$p')
    
    # Should return 200 OK with body
    if ! check_status "$HTTP_CODE" "200"; then
        return 1
    fi
    
    # Preference-Applied header should be present
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:.*return=representation"; then
        echo "  Details: Preference-Applied header should be 'return=representation'"
        return 1
    fi
    
    # Body should contain the updated entity
    if ! check_contains "$BODY" '"Name"'; then
        echo "  Details: Response body should contain updated entity"
        return 1
    fi
    
    return 0
}

# Test 9: Prefer header is case-insensitive
test_prefer_case_insensitive() {
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: RETURN=MINIMAL" \
        -d '{"Name":"Case Test","Price":79.99,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    # Should return 204 No Content (case-insensitive matching)
    check_status "$HTTP_CODE" "204"
}

# Test 10: Multiple preferences in header (comma-separated)
test_multiple_preferences() {
    local RESPONSE=$(curl -s -i -X POST "$SERVER_URL/Products" \
        -H "Content-Type: application/json" \
        -H "Prefer: return=minimal, respond-async" \
        -d '{"Name":"Multi Pref","Price":129.99,"CategoryID":1,"Status":1}' 2>&1)
    
    local HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP/" | tail -1 | awk '{print $2}')
    
    # Should honor return=minimal
    if ! check_status "$HTTP_CODE" "204"; then
        return 1
    fi
    
    # Preference-Applied should indicate which preferences were applied
    if ! echo "$RESPONSE" | grep -qi "Preference-Applied:"; then
        echo "  Details: Preference-Applied header should be present"
        return 1
    fi
    
    return 0
}

# Cleanup function to remove test data
cleanup() {
    # Test data will be cleaned by next reseed
    return 0
}

# Register cleanup
register_cleanup

# Run tests
echo "  Request: POST /Products (no Prefer header)"
run_test "POST without Prefer returns representation by default" test_post_default_returns_representation

echo "  Request: POST /Products (Prefer: return=minimal)"
run_test "POST with Prefer: return=minimal returns 204 No Content" test_post_prefer_minimal

echo "  Request: POST /Products (Prefer: return=representation)"
run_test "POST with explicit Prefer: return=representation" test_post_prefer_representation

echo "  Request: PATCH /Products(1) (no Prefer header)"
run_test "PATCH without Prefer returns 204 No Content by default" test_patch_default_no_content

echo "  Request: PATCH /Products(2) (Prefer: return=representation)"
run_test "PATCH with Prefer: return=representation returns entity" test_patch_prefer_representation

echo "  Request: PATCH /Products(3) (Prefer: return=minimal)"
run_test "PATCH with explicit Prefer: return=minimal" test_patch_prefer_minimal

echo "  Request: PUT /Products(4) (no Prefer header)"
run_test "PUT without Prefer returns 204 No Content by default" test_put_default_no_content

echo "  Request: PUT /Products(5) (Prefer: return=representation)"
run_test "PUT with Prefer: return=representation returns entity" test_put_prefer_representation

echo "  Request: POST with case-insensitive Prefer"
run_test "Prefer header is case-insensitive" test_prefer_case_insensitive

echo "  Request: POST with multiple preferences"
run_test "Multiple preferences in header" test_multiple_preferences

print_summary

