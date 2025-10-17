#!/bin/bash

# OData v4 Compliance Test: 8.2.7 Header Accept
# Tests Accept header content negotiation according to OData v4 specification
# Spec: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderAccept

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test_framework.sh"
source "$SCRIPT_DIR/test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: 8.2.7 Header Accept"
echo "======================================"
echo ""
echo "Description: Validates Accept header content negotiation and handling"
echo "             of different media types according to OData v4 specification."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderAccept"
echo ""



# Test 1: Accept: application/json should return JSON
echo "Test 1: Accept: application/json returns JSON"
echo "  Request: GET $SERVER_URL/Products(1) with Accept: application/json"
RESPONSE=$(curl -s -i -H "Accept: application/json" "$SERVER_URL/Products(1)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        test_result "Accept application/json" "PASS"
    else
        test_result "Accept application/json" "FAIL" "Content-Type is $CONTENT_TYPE"
    fi
else
    test_result "Accept application/json" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 2: Accept: */* should return JSON (default)
echo ""
echo "Test 2: Accept: */* returns JSON (default)"
echo "  Request: GET $SERVER_URL/Products(1) with Accept: */*"
RESPONSE=$(curl -s -i -H "Accept: */*" "$SERVER_URL/Products(1)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        test_result "Accept */* returns JSON" "PASS"
    else
        test_result "Accept */* returns JSON" "FAIL" "Content-Type is $CONTENT_TYPE"
    fi
else
    test_result "Accept */* returns JSON" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 3: Unsupported media type should return 406
echo ""
echo "Test 3: Unsupported Accept media type returns 406"
echo "  Request: GET $SERVER_URL/Products(1) with Accept: application/xml"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "Accept: application/xml" "$SERVER_URL/Products(1)" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "$HTTP_CODE" = "406" ]; then
    test_result "Unsupported Accept returns 406" "PASS"
elif [ "$HTTP_CODE" = "200" ]; then
    # Server might support XML, check Content-Type
    FULL_RESPONSE=$(curl -s -i -H "Accept: application/xml" "$SERVER_URL/Products(1)" 2>&1)
    CONTENT_TYPE=$(echo "$FULL_RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
    if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
        test_result "Unsupported Accept returns 406" "PASS" "Server supports XML (optional)"
    else
        test_result "Unsupported Accept returns 406" "FAIL" "Expected 406 or XML response, got JSON with HTTP 200"
    fi
else
    test_result "Unsupported Accept returns 406" "FAIL" "Expected HTTP 406, got $HTTP_CODE"
fi

# Test 4: Accept with parameters
echo ""
echo "Test 4: Accept: application/json;odata.metadata=minimal"
echo "  Request: GET $SERVER_URL/Products(1) with Accept: application/json;odata.metadata=minimal"
RESPONSE=$(curl -s -i -H "Accept: application/json;odata.metadata=minimal" "$SERVER_URL/Products(1)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        test_result "Accept with odata.metadata parameter" "PASS"
    else
        test_result "Accept with odata.metadata parameter" "FAIL" "Content-Type is $CONTENT_TYPE"
    fi
else
    test_result "Accept with odata.metadata parameter" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 5: Accept with quality values
echo ""
echo "Test 5: Accept with quality values prioritizes correctly"
echo "  Request: GET $SERVER_URL/Products(1) with Accept: text/plain;q=0.5, application/json;q=1.0"
RESPONSE=$(curl -s -i -H "Accept: text/plain;q=0.5, application/json;q=1.0" "$SERVER_URL/Products(1)" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$CONTENT_TYPE" | grep -q "application/json"; then
        test_result "Accept quality values respected" "PASS"
    else
        test_result "Accept quality values respected" "FAIL" "Expected JSON, got $CONTENT_TYPE"
    fi
else
    test_result "Accept quality values respected" "FAIL" "HTTP $HTTP_CODE"
fi

# Test 6: Metadata document should support application/xml
echo ""
echo "Test 6: Metadata document with Accept: application/xml"
echo "  Request: GET $SERVER_URL/\$metadata with Accept: application/xml"
RESPONSE=$(curl -s -i -H "Accept: application/xml" "$SERVER_URL/\$metadata" 2>&1)
CONTENT_TYPE=$(echo "$RESPONSE" | grep -i "^Content-Type:" | head -1 | sed 's/Content-Type: //i' | tr -d '\r')
HTTP_CODE=$(echo "$RESPONSE" | grep "^HTTP" | tail -1 | awk '{print $2}')

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$CONTENT_TYPE" | grep -q "application/xml"; then
        test_result "Metadata accepts application/xml" "PASS"
    else
        test_result "Metadata accepts application/xml" "FAIL" "Content-Type is $CONTENT_TYPE"
    fi
else
    test_result "Metadata accepts application/xml" "FAIL" "HTTP $HTTP_CODE"
fi


print_summary
