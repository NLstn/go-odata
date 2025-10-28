#!/bin/bash

# OData v4 Compliance Test: 5.3 Enumeration Types - Metadata Members
# Verifies that enum metadata reflects actual Go enum values and configured namespace.
# Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_EnumerationType

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 5.3 Enumeration Types (Metadata)"
echo "======================================"

test_enum_metadata_xml() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata")

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected status 200, got $HTTP_CODE"
        return 1
    fi

    if ! echo "$RESPONSE" | grep -q '<Schema[^>]*Namespace="ComplianceService"'; then
        echo "  Details: Schema namespace is not ComplianceService"
        return 1
    fi

    if ! echo "$RESPONSE" | grep -q '<EnumType Name="ProductStatus" UnderlyingType="Edm.Int32" IsFlags="true">'; then
        echo "  Details: ProductStatus enum definition missing or incorrect"
        return 1
    fi

    for MEMBER in "None" "InStock" "OnSale" "Discontinued" "Featured"; do
        if ! echo "$RESPONSE" | grep -q "<Member Name=\"$MEMBER\""; then
            echo "  Details: Enum member $MEMBER missing in XML metadata"
            return 1
        fi
    done

    if ! echo "$RESPONSE" | grep -q '<Member Name="Featured" Value="8" />'; then
        echo "  Details: Featured member has incorrect value"
        return 1
    fi

    return 0
}

test_enum_metadata_json() {
    local RESPONSE=$(http_get_body "$SERVER_URL/\$metadata?\$format=json")
    local HTTP_CODE=$(http_get "$SERVER_URL/\$metadata?\$format=json")

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  Details: Expected status 200, got $HTTP_CODE"
        return 1
    fi

    export RESPONSE
    python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["RESPONSE"])
schema = data.get("ComplianceService")
if not isinstance(schema, dict):
    print("  Details: JSON metadata missing ComplianceService schema")
    sys.exit(1)

enum = schema.get("ProductStatus")
if not isinstance(enum, dict):
    print("  Details: JSON metadata missing ProductStatus enum")
    sys.exit(1)

if enum.get("$UnderlyingType") != "Edm.Int32":
    print("  Details: Unexpected underlying type", enum.get("$UnderlyingType"))
    sys.exit(1)

expected = {
    "None": 0,
    "InStock": 1,
    "OnSale": 2,
    "Discontinued": 4,
    "Featured": 8,
}

for name, value in expected.items():
    if enum.get(name) != value:
        print(f"  Details: Enum member {name} expected {value}, got {enum.get(name)}")
        sys.exit(1)

if data.get("$EntityContainer") != "ComplianceService.Container":
    print("  Details: Unexpected $EntityContainer value", data.get("$EntityContainer"))
    sys.exit(1)
PY
    local PY_RESULT=$?
    unset RESPONSE
    return $PY_RESULT
}

run_test "XML metadata enumerates ProductStatus members" test_enum_metadata_xml
run_test "JSON metadata enumerates ProductStatus members" test_enum_metadata_json

print_summary

exit $EXIT_CODE
