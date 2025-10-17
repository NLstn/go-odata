#!/bin/bash

# OData v4 Compliance Test: 9.2 Metadata Document
# Tests metadata document structure and format
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_MetadataDocumentRequest

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 9.2 Metadata Document"
echo "======================================"
echo ""
echo "Description: Validates metadata document structure, format, and content"
echo "             according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_MetadataDocumentRequest"
echo ""



# Test 1: Metadata document is accessible at $metadata
echo "Test 1: Metadata document accessible at \$metadata"
echo "  Request: GET $SERVER_URL/\$metadata"
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/\$metadata" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    test_result "Metadata document returns 200 OK" "PASS"
else
    test_result "Metadata document returns 200 OK" "FAIL" "Status code: $HTTP_CODE"
fi
echo ""

# Test 2: Metadata Content-Type is application/xml
echo "Test 2: Metadata Content-Type is application/xml"
echo "  Request: GET $SERVER_URL/\$metadata"
RESPONSE=$(curl -s -i "$SERVER_URL/\$metadata" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')

if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
    test_result "Metadata Content-Type is application/xml" "PASS"
else
    test_result "Metadata Content-Type is application/xml" "FAIL" "Content-Type: $CONTENT_TYPE"
fi
echo ""

# Test 3: Metadata contains Edmx element
echo "Test 3: Metadata contains Edmx root element"
echo "  Request: GET $SERVER_URL/\$metadata"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

if echo "$RESPONSE" | grep -q "<edmx:Edmx"; then
    test_result "Metadata contains <edmx:Edmx> element" "PASS"
elif echo "$RESPONSE" | grep -q "<Edmx"; then
    test_result "Metadata contains <Edmx> element" "PASS"
else
    test_result "Metadata contains Edmx element" "FAIL" "No Edmx element found"
fi
echo ""

# Test 4: Metadata contains DataServices element
echo "Test 4: Metadata contains DataServices element"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

if echo "$RESPONSE" | grep -q "<edmx:DataServices"; then
    test_result "Metadata contains <edmx:DataServices> element" "PASS"
elif echo "$RESPONSE" | grep -q "<DataServices"; then
    test_result "Metadata contains <DataServices> element" "PASS"
else
    test_result "Metadata contains DataServices element" "FAIL" "No DataServices element found"
fi
echo ""

# Test 5: Metadata contains Schema element
echo "Test 5: Metadata contains Schema element"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

if echo "$RESPONSE" | grep -q "<Schema"; then
    test_result "Metadata contains <Schema> element" "PASS"
else
    test_result "Metadata contains Schema element" "FAIL" "No Schema element found"
fi
echo ""

# Test 6: Metadata contains EntityType definitions
echo "Test 6: Metadata contains EntityType definitions"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

if echo "$RESPONSE" | grep -q "<EntityType"; then
    test_result "Metadata contains <EntityType> definitions" "PASS"
else
    test_result "Metadata contains EntityType definitions" "FAIL" "No EntityType elements found"
fi
echo ""

# Test 7: Metadata contains EntityContainer
echo "Test 7: Metadata contains EntityContainer"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

if echo "$RESPONSE" | grep -q "<EntityContainer"; then
    test_result "Metadata contains <EntityContainer> element" "PASS"
else
    test_result "Metadata contains EntityContainer element" "FAIL" "No EntityContainer found"
fi
echo ""

# Test 8: Metadata is valid XML
echo "Test 8: Metadata document is valid XML"
RESPONSE=$(curl -s "$SERVER_URL/\$metadata" 2>&1)

# Try to validate with xmllint if available
if command -v xmllint &> /dev/null; then
    if echo "$RESPONSE" | xmllint --noout - 2>&1; then
        test_result "Metadata is valid XML (xmllint)" "PASS"
    else
        test_result "Metadata is valid XML (xmllint)" "FAIL" "XML validation failed"
    fi
else
    # Basic check: look for matching tags
    if echo "$RESPONSE" | grep -q "</edmx:Edmx>"; then
        test_result "Metadata appears to be valid XML (basic check)" "PASS"
    elif echo "$RESPONSE" | grep -q "</Edmx>"; then
        test_result "Metadata appears to be valid XML (basic check)" "PASS"
    else
        test_result "Metadata appears to be valid XML (basic check)" "FAIL" "No closing Edmx tag"
    fi
fi
echo ""


print_summary
