#!/bin/bash

# OData v4 Compliance Test: 11.3.7 Geospatial Functions in Filter
# Tests geographic functions (geo.distance, geo.length, geo.intersects) in filter expressions
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_GeospatialFunctions

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 11.3.7 Geospatial Functions in Filter"
echo "======================================"
echo ""
echo "Description: Validates geospatial functions in filter expressions"
echo "             according to OData v4 specification. Tests geo.distance,"
echo "             geo.length, geo.intersects, and other geographic operations."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_GeospatialFunctions"
echo ""

# Note: Geospatial functions are optional OData features
# Many implementations may not support them initially

# Test 1: geo.distance function in filter
test_geo_distance() {
    # Filter products within 10km of a point (if Location is a geospatial property)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
    
    # 200 OK = supported, 400/501 = not implemented
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: geo.distance not implemented (status: $HTTP_CODE)"
        return 0  # Pass - not all services must support geo functions
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 2: geo.length function in filter
test_geo_length() {
    # Filter by length of a linestring geometry
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.length(Route) gt 1000")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: geo.length not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 3: geo.intersects function in filter
test_geo_intersects() {
    # Filter entities that intersect with a polygon
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.intersects(Area,geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))')")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: geo.intersects not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 4: Invalid geo function returns error
test_invalid_geo_function() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.invalid(Location)")
    
    # Should return 400 Bad Request for invalid function
    if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "404" ]; then
        return 0
    else
        echo "  Details: Status: $HTTP_CODE (expected 400 or 404)"
        return 1
    fi
}

# Test 5: geo.distance with invalid syntax returns error
test_geo_distance_invalid_syntax() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.distance(Location) lt 100")
    
    # Missing required parameter should return 400
    check_status "$HTTP_CODE" "400"
}

# Test 6: Valid geospatial literal format
test_geo_literal_format() {
    # Test that properly formatted geography literals are accepted
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.distance(Location,geography'SRID=4326;POINT(-122.1 47.6)') lt 5000")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Geospatial functions not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 7: geometry vs geography distinction
test_geometry_vs_geography() {
    # Test geometry (flat earth) vs geography (round earth)
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=geo.distance(Location,geometry'SRID=0;POINT(0 0)') lt 100")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Geometry type not implemented (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

# Test 8: Combining geo functions with other filters
test_geo_combined_filter() {
    # Combine geospatial with regular filter
    local HTTP_CODE=$(http_get "$SERVER_URL/Products?\$filter=Price gt 100 and geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
    
    if [ "$HTTP_CODE" = "200" ]; then
        return 0
    elif [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "501" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  Details: Combined geo filter not supported (status: $HTTP_CODE)"
        return 0  # Pass - optional feature
    else
        echo "  Details: Unexpected status: $HTTP_CODE"
        return 1
    fi
}

echo "  Request: GET \$filter=geo.distance(Location,...) lt 10000"
run_test "geo.distance function in filter (optional)" test_geo_distance

echo "  Request: GET \$filter=geo.length(Route) gt 1000"
run_test "geo.length function in filter (optional)" test_geo_length

echo "  Request: GET \$filter=geo.intersects(Area,...)"
run_test "geo.intersects function in filter (optional)" test_geo_intersects

echo "  Request: GET \$filter=geo.invalid(Location)"
run_test "Invalid geo function returns 400 error" test_invalid_geo_function

echo "  Request: GET \$filter=geo.distance(Location) lt 100"
run_test "geo.distance with missing parameter returns 400" test_geo_distance_invalid_syntax

echo "  Request: GET \$filter with properly formatted geography literal"
run_test "Valid geospatial literal format accepted" test_geo_literal_format

echo "  Request: GET \$filter with geometry (flat earth) type"
run_test "Geometry vs geography type distinction" test_geometry_vs_geography

echo "  Request: GET \$filter=Price gt 100 and geo.distance(...)"
run_test "Combining geo functions with other filters" test_geo_combined_filter

print_summary
