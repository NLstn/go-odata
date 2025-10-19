#!/bin/bash

# OData v4 Compliance Test Framework
# Common functions and utilities for compliance tests
# Source this file in your test scripts: source "$(dirname "$0")/test_framework.sh"

# Test counters
TOTAL=0
PASSED=0
FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Server URL (can be overridden by environment variable)
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

# Note: Database reseeding is handled centrally by run_compliance_tests.sh
# before each test script runs when running the full suite.
# For individual test runs, call reseed_database() at the start of your test script.

# Function to reseed the database to default state
# Call this at the beginning of a test script if running it individually
reseed_database() {
    echo -n "Reseeding database... "
    local RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/Reseed" 2>&1)
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ Failed (status: $HTTP_CODE)${NC}"
        return 1
    fi
}

# Automatically reseed before the first test in each script
# This ensures individual test runs start with clean data
FIRST_TEST=1

# Function to run a test and track results
# Usage: run_test "Test description" command [expected_output]
run_test() {
    local description="$1"
    local test_func="$2"
    
    # Automatically reseed before first test when running individual scripts
    if [ $FIRST_TEST -eq 1 ]; then
        FIRST_TEST=0
        reseed_database
        echo ""
    fi
    
    TOTAL=$((TOTAL + 1))
    echo ""
    echo "Test $TOTAL: $description"
    
    # Run the test function
    if $test_func; then
        PASSED=$((PASSED + 1))
        echo -e "${GREEN}✓ PASS${NC}: $description"
        return 0
    else
        FAILED=$((FAILED + 1))
        echo -e "${RED}✗ FAIL${NC}: $description"
        return 1
    fi
}

# Function to URL encode a string (spaces and special characters)
# This ensures curl can properly handle OData query parameters
url_encode() {
    local string="$1"
    # Use printf and xxd to properly encode, but for simple case just encode spaces
    # This is a basic implementation that handles the most common case
    echo "$string" | sed 's/ /%20/g'
}

# Function to make HTTP request and return status code
# Usage: http_get URL [headers...]
http_get() {
    local url="$1"
    shift
    # Use --globoff to prevent curl from interpreting [] and {} as patterns
    # URL encode spaces in the query string part
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s -o /dev/null -w "%{http_code}" "$@" "$url"
}

# Function to make HTTP request and return response body
# Usage: http_get_body URL [headers...]
http_get_body() {
    local url="$1"
    shift
    # Use --globoff to prevent curl from interpreting [] and {} as patterns
    # URL encode spaces in the query string part
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s "$@" "$url"
}

# Function to make HTTP POST request
# Usage: http_post URL data [headers...]
http_post() {
    local url="$1"
    local data="$2"
    shift 2
    # URL encode if needed
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s -X POST -d "$data" "$@" "$url"
}

# Function to make HTTP PATCH request
# Usage: http_patch URL data [headers...]
http_patch() {
    local url="$1"
    local data="$2"
    shift 2
    # URL encode if needed
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s -X PATCH -d "$data" "$@" "$url"
}

# Function to make HTTP PUT request
# Usage: http_put URL data [headers...]
http_put() {
    local url="$1"
    local data="$2"
    shift 2
    # URL encode if needed
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s -X PUT -d "$data" "$@" "$url"
}

# Function to make HTTP DELETE request
# Usage: http_delete URL [headers...]
http_delete() {
    local url="$1"
    shift
    # URL encode if needed
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    curl -g -s -X DELETE "$@" "$url"
}

# Function to check if response contains expected value
# Usage: check_contains "$response" "expected_value"
check_contains() {
    local response="$1"
    local expected="$2"
    
    if echo "$response" | grep -q "$expected"; then
        return 0
    else
        echo "  Details: Expected '$expected' not found in response"
        return 1
    fi
}

# Function to check HTTP status code
# Usage: check_status actual expected
check_status() {
    local actual="$1"
    local expected="$2"
    
    if [ "$actual" = "$expected" ]; then
        return 0
    else
        echo "  Details: Status code: $actual (expected $expected)"
        return 1
    fi
}

# Function to check if JSON response has a field
# Usage: check_json_field "$response" "field_name"
check_json_field() {
    local response="$1"
    local field="$2"
    
    if echo "$response" | grep -q "\"$field\""; then
        return 0
    else
        echo "  Details: Field '$field' not found in JSON response"
        return 1
    fi
}

# Function to print test summary in standardized format
# This MUST be called at the end of every test script
print_summary() {
    echo ""
    echo "======================================"
    echo "COMPLIANCE_TEST_RESULT:PASSED=$PASSED:FAILED=$FAILED:TOTAL=$TOTAL"
    echo "======================================"
    
    if [ $FAILED -eq 0 ]; then
        echo "Status: PASSING"
        exit 0
    else
        echo "Status: FAILING"
        exit 1
    fi
}

# Trap to ensure cleanup happens on exit
cleanup_registered=0
register_cleanup() {
    if [ $cleanup_registered -eq 0 ]; then
        trap cleanup_and_exit EXIT
        cleanup_registered=1
    fi
}

cleanup_and_exit() {
    if [ "$(type -t cleanup)" = "function" ]; then
        echo ""
        echo "Cleaning up test data..."
        cleanup
    fi
}
