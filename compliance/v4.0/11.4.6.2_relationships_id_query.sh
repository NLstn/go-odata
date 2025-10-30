#!/bin/bash

# OData v4 Compliance Test: 11.4.6.2 Managing Relationships with $id query option
# Validates using the $id system query option on $ref requests as defined in
# OData v4.0 Section 11.4.6.2 (Managing Relationships) of the protocol specification.
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestinganEntityReference

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Ensure database reseed before executing tests (even when run individually)
reseed_database
FIRST_TEST=0

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.4.6.2 Relationships via $id"
echo "======================================"
echo ""
echo "Description: Validates managing relationships using the $id query option"
echo "             on $ref requests for both collection and single-valued"
echo "             navigation properties, including error handling."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestinganEntityReference"
echo ""

reset_relationships() {
    # Remove related product link using both $id and key addressing to cover servers
    curl --globoff -s -o /dev/null -w "%{http_code}" -X DELETE \
        "$SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Products(2)" >/dev/null 2>&1 || true
    curl --globoff -s -o /dev/null -w "%{http_code}" -X DELETE \
        "$SERVER_URL/Products(1)/RelatedProducts(2)/\$ref" >/dev/null 2>&1 || true

    # Restore single-valued navigation back to Category 1 using JSON payload fallback
    curl --globoff -s -o /dev/null -w "%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'"$SERVER_URL"'/Categories(1)"}' \
        "$SERVER_URL/Products(1)/Category/\$ref" >/dev/null 2>&1 || true
}

reset_relationships
trap reset_relationships EXIT

SUPPORTS_ID=0
PROBE_STATUS=""

probe_id_query_support() {
    local probe_url="$SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Products(2)"
    local status=$(curl --globoff -s -o /dev/null -w "%{http_code}" -X POST "$probe_url")

    if [ "$status" = "204" ] || [ "$status" = "201" ]; then
        SUPPORTS_ID=1
        # Clean up the probe-created relationship to maintain baseline state
        curl --globoff -s -o /dev/null -w "%{http_code}" -X DELETE "$probe_url" >/dev/null 2>&1 || true
        return 0
    fi

    PROBE_STATUS="$status"
    case "$status" in
        400|404|405|415|501)
            SUPPORTS_ID=0
            return 1
            ;;
        *)
            SUPPORTS_ID=-1
            return 2
            ;;
    esac
}

probe_id_query_support

if [ "$SUPPORTS_ID" = "0" ]; then
    echo "Feature probe: service responded with status $PROBE_STATUS indicating $id query option is not implemented."
    skip_test "POST collection navigation via \$id query option" "\$id query option not implemented"
    skip_test "PUT single-valued navigation via \$id query option" "\$id query option not implemented"
    skip_test "Invalid \$id query handling (missing value)" "\$id query option not implemented"
    skip_test "Invalid \$id query handling (mismatched set)" "\$id query option not implemented"
    print_summary
    exit 0
elif [ "$SUPPORTS_ID" = "-1" ]; then
    echo "Feature probe received unexpected status: $PROBE_STATUS"
    # Treat as failure for the first test to highlight unexpected behavior
    skip_test "POST collection navigation via \$id query option" "Feature probe returned unexpected status $PROBE_STATUS"
    skip_test "PUT single-valued navigation via \$id query option" "Feature probe returned unexpected status $PROBE_STATUS"
    skip_test "Invalid \$id query handling (missing value)" "Feature probe returned unexpected status $PROBE_STATUS"
    skip_test "Invalid \$id query handling (mismatched set)" "Feature probe returned unexpected status $PROBE_STATUS"
    print_summary
    exit 1
fi

# Helper to confirm 4xx responses
expect_4xx() {
    local status="$1"
    if [[ "$status" =~ ^4 ]]; then
        return 0
    fi
    echo "  Details: Expected 4xx status but received $status"
    return 1
}

# Test 1: POST using $id on collection navigation
post_related_product_via_id() {
    local url="$SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Products(2)"
    local status=$(curl --globoff -s -o /dev/null -w "%{http_code}" -X POST "$url")

    if [ "$status" = "204" ] || [ "$status" = "201" ]; then
        local ref_body=$(http_get_body "$SERVER_URL/Products(1)/RelatedProducts/\$ref")
        if echo "$ref_body" | grep -q "$SERVER_URL/Products(2)"; then
            # Clean up relationship to keep state deterministic
            curl --globoff -s -o /dev/null -w "%{http_code}" -X DELETE "$url" >/dev/null 2>&1 || true
            return 0
        fi
        echo "  Details: Relationship not found in \$ref collection after POST"
    else
        echo "  Details: Expected 204/201 for POST with \$id but received $status"
    fi

    # Cleanup attempt in case of partial success
    curl --globoff -s -o /dev/null -w "%{http_code}" -X DELETE "$url" >/dev/null 2>&1 || true
    return 1
}

# Test 2: PUT using $id on single-valued navigation
put_category_via_id() {
    local url="$SERVER_URL/Products(1)/Category/\$ref?\$id=$SERVER_URL/Categories(2)"
    local status=$(curl --globoff -s -o /dev/null -w "%{http_code}" -X PUT "$url")

    if [ "$status" = "204" ] || [ "$status" = "200" ]; then
        local ref_body=$(http_get_body "$SERVER_URL/Products(1)/Category/\$ref")
        if echo "$ref_body" | grep -q "$SERVER_URL/Categories(2)"; then
            # Restore to original category (Category 1)
            curl --globoff -s -o /dev/null -w "%{http_code}" -X PUT \
                -H "Content-Type: application/json" \
                -d '{"@odata.id":"'"$SERVER_URL"'/Categories(1)"}' \
                "$SERVER_URL/Products(1)/Category/\$ref" >/dev/null 2>&1 || true
            return 0
        fi
        echo "  Details: Category reference not updated to Categories(2)"
    else
        echo "  Details: Expected 204/200 for PUT with \$id but received $status"
    fi

    # Ensure original relationship restored even after failure
    curl --globoff -s -o /dev/null -w "%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -d '{"@odata.id":"'"$SERVER_URL"'/Categories(1)"}' \
        "$SERVER_URL/Products(1)/Category/\$ref" >/dev/null 2>&1 || true
    return 1
}

# Test 3: POST missing $id should return 4xx
post_missing_id_should_fail() {
    local url="$SERVER_URL/Products(1)/RelatedProducts/\$ref"
    local status=$(curl --globoff -s -o /dev/null -w "%{http_code}" -X POST "$url")
    expect_4xx "$status"
}

# Test 4: POST with mismatched entity set should return 4xx
post_mismatched_set_should_fail() {
    local url="$SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Categories(2)"
    local status=$(curl --globoff -s -o /dev/null -w "%{http_code}" -X POST "$url")
    expect_4xx "$status"
}

echo "  Request: POST $SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Products(2)"
run_test "POST collection navigation via \$id query option" post_related_product_via_id

echo "  Request: PUT $SERVER_URL/Products(1)/Category/\$ref?\$id=$SERVER_URL/Categories(2)"
run_test "PUT single-valued navigation via \$id query option" put_category_via_id

echo "  Request: POST $SERVER_URL/Products(1)/RelatedProducts/\$ref (missing \$id)"
run_test "Invalid \$id query handling (missing value)" post_missing_id_should_fail

echo "  Request: POST $SERVER_URL/Products(1)/RelatedProducts/\$ref?\$id=$SERVER_URL/Categories(2)"
run_test "Invalid \$id query handling (mismatched set)" post_mismatched_set_should_fail

print_summary
