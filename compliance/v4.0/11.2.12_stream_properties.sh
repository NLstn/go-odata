#!/bin/bash

# OData v4 Compliance Test: 11.2.12 Stream Properties and Media Entities
# Tests media entities, stream properties, and $value for streams
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_MediaEntities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.2.12 Stream Properties and Media Entities"
echo "======================================"
echo ""
echo "Description: Validates handling of media entities and stream properties"
echo "             including $value access for binary content."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_MediaEntities"
echo ""

# Note: Stream properties and media entities are advanced OData features
# Many implementations may not support them initially

# Test 1: Request media entity
test_media_entity() {
    # Try to access a media entity (e.g., an image or document)
    local HTTP_CODE=$(http_get "$SERVER_URL/MediaItems(1)")

    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity missing (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 2: Request media entity $value (binary content)
test_media_entity_value() {
    # Access binary content of media entity using $value
    local HTTP_CODE=$(http_get "$SERVER_URL/MediaItems(1)/\$value")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity \$value missing (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 3: Request stream property
test_stream_property() {
    # Access named stream property (e.g., Product(1)/Photo)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Photo")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - stream property missing (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 4: Media entity with content type
test_media_content_type() {
    # Check Content-Type header for media entity
    local HEADERS=$(curl -s -I "$SERVER_URL/MediaItems(1)/\$value" 2>&1)
    local HTTP_CODE=$(echo "$HEADERS" | head -1 | grep -o '[0-9]\{3\}' | head -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        # Check for appropriate Content-Type (image/*, application/*, etc.)
        if echo "$HEADERS" | grep -qi "Content-Type:"; then
            return 0
        fi

        echo "  Details: Specification violation - missing Content-Type header"
        return 1
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media content unavailable (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 5: POST media entity (upload)
test_post_media_entity() {
    # Try to create media entity by POSTing binary content
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER_URL/MediaItems" \
        -H "Content-Type: image/png" \
        -d "fake-binary-data" 2>&1)
    
    if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity creation unsupported (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 6: PUT to update media entity content
test_put_media_value() {
    # Update media entity binary content using PUT to $value
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$SERVER_URL/MediaItems(1)/\$value" \
        -H "Content-Type: image/png" \
        -d "updated-binary-data" 2>&1)
    
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity update unsupported (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 7: Media entity metadata
test_media_metadata() {
    # Access media entity metadata (not the binary content)
    local HTTP_CODE=$(http_get "$SERVER_URL/MediaItems(1)")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity metadata missing (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 8: Stream property in metadata
test_stream_in_metadata() {
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")
    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")

        if echo "$RESPONSE" | grep -q 'HasStream\|Stream'; then
            return 0
        fi

        echo "  Details: Specification violation - metadata missing stream annotations"
        return 1
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - metadata endpoint unavailable (status: $HTTP_CODE)"
    else
        echo "  Details: Status: $HTTP_CODE"
    fi

    return 1
}

# Test 9: Accept header for media content
test_media_accept_header() {
    # Request with specific Accept header for media content
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/MediaItems(1)/\$value" \
        -H "Accept: image/png" 2>&1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media stream negotiation failed (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 10: DELETE media entity
test_delete_media() {
    # Try to delete media entity
    local HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER_URL/MediaItems(999999)" 2>&1)
    
    # 404 for not found, 204/200 for deleted, 405 for not allowed
    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity deletion failed (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 11: Media link entry
test_media_link_entry() {
    # Check for @odata.mediaReadLink or @odata.mediaEditLink in response
    local HTTP_CODE=$(http_get "$SERVER_URL/MediaItems(1)")

    if [ "$HTTP_CODE" = "200" ]; then
        local RESPONSE=$(http_get_body "$SERVER_URL/MediaItems(1)")

        if echo "$RESPONSE" | grep -q '@odata.media'; then
            return 0
        fi

        echo "  Details: Specification violation - missing media link annotations"
        return 1
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - media entity endpoint unavailable (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

# Test 12: Stream property $value
test_stream_property_value() {
    # Access stream property content using $value
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)/Photo/\$value")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    fi

    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "405" ] || [ "$HTTP_CODE" = "501" ]; then
        echo "  Details: Specification violation - stream property \$value missing (status: $HTTP_CODE)"
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
    fi

    return 1
}

echo "  Request: GET MediaItems(1)"
run_test "Request media entity (optional)" test_media_entity

echo "  Request: GET MediaItems(1)/\$value"
run_test "Request media entity binary content" test_media_entity_value

echo "  Request: GET Products(1)/Photo"
run_test "Request stream property (optional)" test_stream_property

echo "  Request: Check Content-Type for media content"
run_test "Media entity Content-Type header" test_media_content_type

echo "  Request: POST MediaItems with binary data"
run_test "Create media entity (upload)" test_post_media_entity

echo "  Request: PUT MediaItems(1)/\$value"
run_test "Update media entity content" test_put_media_value

echo "  Request: GET MediaItems(1) metadata"
run_test "Access media entity metadata" test_media_metadata

echo "  Request: GET \$metadata"
run_test "Stream properties in metadata" test_stream_in_metadata

echo "  Request: GET with Accept header for media"
run_test "Accept header for media content" test_media_accept_header

echo "  Request: DELETE MediaItems(999999)"
run_test "Delete media entity" test_delete_media

echo "  Request: Check for media link annotations"
run_test "Media link entry annotations" test_media_link_entry

echo "  Request: GET Products(1)/Photo/\$value"
run_test "Stream property \$value access" test_stream_property_value

print_summary
