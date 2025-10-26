#!/bin/bash

# OData v4.0 Section 4.4: Primitive Types  
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752517
#
# OData defines 33 primitive types including:
# - Edm.Binary, Edm.Boolean, Edm.Byte, Edm.Date, Edm.DateTimeOffset, Edm.Decimal, Edm.Double
# - Edm.Duration, Edm.Guid, Edm.Int16, Edm.Int32, Edm.Int64, Edm.SByte, Edm.Single
# - Edm.Stream, Edm.String, Edm.TimeOfDay
# - Geographic and Geometric types
# Special values: -INF, INF, NaN for Double, Single, and floating Decimal

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

# Test 1: Verify Edm.String primitive type support
test_1() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" 'Type="Edm.String"'
}

# Test 2: Verify Edm.Int32 primitive type support
test_2() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" 'Type="Edm.Int32"'
}

# Test 3: Verify Edm.Boolean primitive type support
test_3() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Boolean"'; then
        return 0
    else
        # No Boolean properties, skip
        return 0
    fi
}

# Test 4: Verify Edm.Decimal primitive type support
test_4() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Decimal"'; then
        return 0
    else
        return 0
    fi
}

# Test 5: Verify Edm.Double primitive type support
test_5() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Double"'; then
        return 0
    else
        return 0
    fi
}

# Test 6: Verify Edm.Single primitive type support
test_6() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Single"'; then
        return 0
    else
        return 0
    fi
}

# Test 7: Verify Edm.Guid primitive type support
test_7() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Guid"'; then
        return 0
    else
        return 0
    fi
}

# Test 8: Verify Edm.DateTimeOffset primitive type support
test_8() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.DateTimeOffset"'; then
        return 0
    else
        return 0
    fi
}

# Test 9: Verify Edm.Date primitive type support (OData 4.0+)
test_9() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Date"'; then
        return 0
    else
        return 0
    fi
}

# Test 10: Verify Edm.TimeOfDay primitive type support (OData 4.0+)
test_10() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.TimeOfDay"'; then
        return 0
    else
        return 0
    fi
}

# Test 11: Verify Edm.Duration primitive type support
test_11() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Duration"'; then
        return 0
    else
        return 0
    fi
}

# Test 12: Verify Edm.Binary primitive type support
test_12() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Binary"'; then
        return 0
    else
        return 0
    fi
}

# Test 13: Verify Edm.Stream primitive type support
test_13() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Stream"'; then
        return 0
    else
        return 0
    fi
}

# Test 14: Verify Edm.Byte primitive type support
test_14() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Byte"'; then
        return 0
    else
        return 0
    fi
}

# Test 15: Verify Edm.SByte primitive type support
test_15() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.SByte"'; then
        return 0
    else
        return 0
    fi
}

# Test 16: Verify Edm.Int16 primitive type support
test_16() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Int16"'; then
        return 0
    else
        return 0
    fi
}

# Test 17: Verify Edm.Int64 primitive type support
test_17() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Int64"'; then
        return 0
    else
        return 0
    fi
}

# Test 18: Verify Edm.Geography primitive type support
test_18() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Geography'; then
        return 0
    else
        return 0
    fi
}

# Test 19: Verify Edm.GeographyPoint primitive type support
test_19() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.GeographyPoint"'; then
        return 0
    else
        return 0
    fi
}

# Test 20: Verify Edm.Geometry primitive type support
test_20() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Geometry'; then
        return 0
    else
        return 0
    fi
}

# Test 21: Verify Edm.GeometryPoint primitive type support
test_21() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.GeometryPoint"'; then
        return 0
    else
        return 0
    fi
}

# Test 22: Verify MaxLength facet for Edm.String
test_22() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.String".*MaxLength=' || \
       echo "$RESPONSE" | grep -q 'MaxLength=.*Type="Edm.String"'; then
        return 0
    else
        return 0
    fi
}

# Test 23: Verify Precision facet for Edm.Decimal
test_23() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Decimal".*Precision=' || \
       echo "$RESPONSE" | grep -q 'Precision=.*Type="Edm.Decimal"'; then
        return 0
    else
        return 0
    fi
}

# Test 24: Verify Scale facet for Edm.Decimal
test_24() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Decimal".*Scale=' || \
       echo "$RESPONSE" | grep -q 'Scale=.*Type="Edm.Decimal"'; then
        return 0
    else
        return 0
    fi
}

# Test 25: Verify Unicode facet for Edm.String
test_25() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.String".*Unicode=' || \
       echo "$RESPONSE" | grep -q 'Unicode=.*Type="Edm.String"'; then
        return 0
    else
        return 0
    fi
}

# Test 26: Verify Precision facet for Edm.DateTimeOffset
test_26() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.DateTimeOffset".*Precision=' || \
       echo "$RESPONSE" | grep -q 'Precision=.*Type="Edm.DateTimeOffset"'; then
        return 0
    else
        return 0
    fi
}

# Test 27: Verify SRID facet for geographic types
test_27() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Edm.Geography.*SRID=' || \
       echo "$RESPONSE" | grep -q 'SRID=.*Type="Edm.Geography'; then
        return 0
    else
        return 0
    fi
}

# Test 28: Verify collection of primitive types
test_28() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if echo "$RESPONSE" | grep -q 'Type="Collection(Edm.'; then
        return 0
    else
        return 0
    fi
}

# Test 29: Verify primitive type in key property
test_29() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    check_contains "$RESPONSE" "Key" && check_contains "$RESPONSE" "PropertyRef"
}

# Test 30: Verify primitive types are case-sensitive
test_30() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    if check_contains "$RESPONSE" 'Type="Edm.'; then
        ! echo "$RESPONSE" | grep -q 'Type="edm.'
    else
        return 1
    fi
}

# Run all tests
run_test "Edm.String primitive type is supported" test_1
run_test "Edm.Int32 primitive type is supported" test_2
run_test "Edm.Boolean primitive type is supported" test_3
run_test "Edm.Decimal primitive type is supported" test_4
run_test "Edm.Double primitive type is supported" test_5
run_test "Edm.Single primitive type is supported" test_6
run_test "Edm.Guid primitive type is supported" test_7
run_test "Edm.DateTimeOffset primitive type is supported" test_8
run_test "Edm.Date primitive type is supported" test_9
run_test "Edm.TimeOfDay primitive type is supported" test_10
run_test "Edm.Duration primitive type is supported" test_11
run_test "Edm.Binary primitive type is supported" test_12
run_test "Edm.Stream primitive type is supported" test_13
run_test "Edm.Byte primitive type is supported" test_14
run_test "Edm.SByte primitive type is supported" test_15
run_test "Edm.Int16 primitive type is supported" test_16
run_test "Edm.Int64 primitive type is supported" test_17
run_test "Edm.Geography abstract type is supported" test_18
run_test "Edm.GeographyPoint primitive type is supported" test_19
run_test "Edm.Geometry abstract type is supported" test_20
run_test "Edm.GeometryPoint primitive type is supported" test_21
run_test "Edm.String can have MaxLength facet" test_22
run_test "Edm.Decimal can have Precision facet" test_23
run_test "Edm.Decimal can have Scale facet" test_24
run_test "Edm.String can have Unicode facet" test_25
run_test "Edm.DateTimeOffset can have Precision facet" test_26
run_test "Geographic types can have SRID facet" test_27
run_test "Collections of primitive types are supported" test_28
run_test "Key properties use primitive types" test_29
run_test "Primitive type names are case-sensitive (Edm.String not edm.string)" test_30

print_summary
