#!/bin/bash

# OData v4 Compliance Test Framework
# Common functions and utilities for compliance tests
# Source this file in your test scripts: source "$(dirname "$0")/test_framework.sh"

# Test counters
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Server URL (can be overridden by environment variable)
SERVER_URL="${SERVER_URL:-http://localhost:9090}"

# Debug mode (can be enabled via DEBUG environment variable or --debug flag)
DEBUG="${DEBUG:-0}"

# Note: Database reseeding is handled centrally by run_compliance_tests.sh
# before each test script runs when running the full suite.
# For individual test runs, call reseed_database() at the start of your test script.

# Function to print debug information for HTTP requests/responses
# Usage: debug_log_request METHOD URL [HEADERS...] [BODY]
debug_log_request() {
    if [ "$DEBUG" != "1" ]; then
        return
    fi
    
    echo "" >&2
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}" >&2
    echo -e "${BLUE}║ DEBUG: HTTP Request                                  ║${NC}" >&2
    echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}" >&2
    echo "" >&2
    
    local method="$1"
    local url="$2"
    shift 2
    
    echo -e "${YELLOW}Method:${NC} $method" >&2
    echo -e "${YELLOW}URL:${NC} $url" >&2
    
    # Parse headers and body from remaining arguments
    local headers=""
    local body=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -H|--header)
                headers="${headers}  $2\n"
                shift 2
                ;;
            -d|--data)
                body="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [ -n "$headers" ]; then
        echo -e "${YELLOW}Headers:${NC}" >&2
        echo -e "$headers" >&2
    fi
    
    if [ -n "$body" ]; then
        echo -e "${YELLOW}Body:${NC}" >&2
        if echo "$body" | python3 -m json.tool >/dev/null 2>&1; then
            echo "$body" | python3 -m json.tool >&2
        else
            echo "$body" >&2
        fi
    fi
    echo "" >&2
}

# Function to print debug information for HTTP responses
# Usage: debug_log_response STATUS_CODE BODY
debug_log_response() {
    if [ "$DEBUG" != "1" ]; then
        return
    fi
    
    local status_code="$1"
    local body="$2"
    
    echo "" >&2
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}" >&2
    echo -e "${BLUE}║ DEBUG: HTTP Response                                 ║${NC}" >&2
    echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}" >&2
    echo "" >&2
    echo -e "${YELLOW}Status Code:${NC} $status_code" >&2
    
    if [ -n "$body" ]; then
        echo -e "${YELLOW}Body:${NC}" >&2
        # Try to pretty-print JSON, fall back to raw output if not JSON
        if echo "$body" | python3 -m json.tool >/dev/null 2>&1; then
            echo "$body" | python3 -m json.tool >&2
        else
            echo "$body" >&2
        fi
    fi
    echo "" >&2
}

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

# Test filtering support
# Set TEST_FILTER environment variable to run only specific tests
# TEST_FILTER can be:
#   - Empty: run all tests (default)
#   - "test_name": run only test with function name matching exactly
#   - "pattern": run only tests with function name containing pattern
TEST_FILTER="${TEST_FILTER:-}"

# Function to check if a test should run based on filter
# Usage: should_run_test "test_func_name"
# Returns: 0 if test should run, 1 if it should be skipped
should_run_test() {
    local test_func="$1"
    
    # No filter means run all tests
    if [ -z "$TEST_FILTER" ]; then
        return 0
    fi
    
    # Check if test function name matches filter (exact or pattern)
    if [[ "$test_func" == *"$TEST_FILTER"* ]]; then
        return 0
    fi
    
    return 1
}

# Function to run a test and track results
# Usage: run_test "Test description" command [expected_output]
run_test() {
    local description="$1"
    local test_func="$2"
    
    # Check if this test should run based on filter
    if ! should_run_test "$test_func"; then
        # Silently skip tests that don't match filter
        return 0
    fi
    
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

# Function to skip a test and track it as skipped
# Usage: skip_test "Test description" "Reason for skipping"
skip_test() {
    local description="$1"
    local reason="${2:-Feature not yet implemented}"
    
    # Automatically reseed before first test when running individual scripts
    if [ $FIRST_TEST -eq 1 ]; then
        FIRST_TEST=0
        reseed_database
        echo ""
    fi
    
    TOTAL=$((TOTAL + 1))
    SKIPPED=$((SKIPPED + 1))
    echo ""
    echo "Test $TOTAL: $description"
    echo -e "${YELLOW}⊘ SKIP${NC}: $description"
    echo "  Reason: $reason"
    return 0
}

# Function to URL encode a string (spaces and special characters)
# This ensures curl can properly handle OData query parameters
url_encode() {
    local string="$1"
    # Use printf and xxd to properly encode, but for simple case just encode spaces
    # This is a basic implementation that handles the most common case
    echo "$string" | sed 's/ /%20/g'
}

# Internal function to make HTTP request and return unified response object
# Returns a response object containing HTTP code, headers, and body separated by special markers
# Usage: response=$(_http_request_internal METHOD URL [data] [curl_options...])
#        code=$(http_response_code "$response")
#        headers=$(http_response_headers "$response")
#        body=$(http_response_body "$response")
_http_request_internal() {
    local method="$1"
    local url="$2"
    shift 2
    
    local data=""
    local curl_method_args=()
    
    # For POST, PATCH, PUT, the third argument is data
    if [ "$method" = "POST" ] || [ "$method" = "PATCH" ] || [ "$method" = "PUT" ]; then
        data="$1"
        shift
        curl_method_args=(-X "$method" -d "$data")
    elif [ "$method" = "DELETE" ]; then
        curl_method_args=(-X DELETE)
    fi
    
    # URL encode spaces in the query string part
    if [[ "$url" == *"?"* ]]; then
        local base="${url%%\?*}"
        local query="${url#*\?}"
        query=$(url_encode "$query")
        url="${base}?${query}"
    fi
    
    # Log request in debug mode
    if [ -n "$data" ]; then
        debug_log_request "$method" "$url" "$@" -d "$data"
    else
        debug_log_request "$method" "$url" "$@"
    fi
    
    # Execute the request and capture headers, body, and status code
    # We use a temporary file for headers to avoid complex delimiter issues
    local temp_headers=$(mktemp)
    local response=$(curl -g -s -D "$temp_headers" -w "\n---HTTP_STATUS_CODE---\n%{http_code}\n---END_HTTP_STATUS_CODE---" "${curl_method_args[@]}" "$@" "$url")
    local http_code=$(echo "$response" | sed -n '/---HTTP_STATUS_CODE---/,/---END_HTTP_STATUS_CODE---/p' | sed '1d;$d' | tr -d '\n')
    local body=$(echo "$response" | sed '/---HTTP_STATUS_CODE---/,/---END_HTTP_STATUS_CODE---/d')
    local headers=$(cat "$temp_headers")
    rm -f "$temp_headers"
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return unified response with markers for parsing
    echo "---HTTP_RESPONSE_START---"
    echo "---HTTP_CODE---"
    echo "$http_code"
    echo "---HTTP_HEADERS---"
    echo "$headers"
    echo "---HTTP_BODY---"
    echo "$body"
    echo "---HTTP_RESPONSE_END---"
}

# Unified HTTP request functions that return response object with code, headers, and body
# Usage: response=$(http_request_get URL [curl_options...])
#        code=$(http_response_code "$response")
#        headers=$(http_response_headers "$response")
#        body=$(http_response_body "$response")
http_request_get() {
    _http_request_internal "GET" "$@"
}

http_request_post() {
    _http_request_internal "POST" "$@"
}

http_request_patch() {
    _http_request_internal "PATCH" "$@"
}

http_request_put() {
    _http_request_internal "PUT" "$@"
}

http_request_delete() {
    _http_request_internal "DELETE" "$@"
}

# Function to extract HTTP status code from unified response
# Usage: code=$(http_response_code "$response")
http_response_code() {
    local response="$1"
    echo "$response" | sed -n '/---HTTP_CODE---/,/---HTTP_HEADERS---/p' | sed '1d;$d'
}

# Function to extract HTTP headers from unified response
# Usage: headers=$(http_response_headers "$response")
http_response_headers() {
    local response="$1"
    echo "$response" | sed -n '/---HTTP_HEADERS---/,/---HTTP_BODY---/p' | sed '1d;$d'
}

# Function to extract HTTP body from unified response
# Usage: body=$(http_response_body "$response")
http_response_body() {
    local response="$1"
    echo "$response" | sed -n '/---HTTP_BODY---/,/---HTTP_RESPONSE_END---/p' | sed '1d;$d'
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
    
    # Log request in debug mode
    debug_log_request "GET" "$url" "$@"
    
    # Execute the request and capture both status and body for debug logging
    local response=$(curl -g -s -w "\n%{http_code}" "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return just the status code
    echo "$http_code"
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
    
    # Log request in debug mode
    debug_log_request "GET" "$url" "$@"
    
    # Execute the request and capture both status and body
    local response=$(curl -g -s -w "\n%{http_code}" "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return just the body
    echo "$body"
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
    
    # Log request in debug mode
    debug_log_request "POST" "$url" "$@" -d "$data"
    
    # Execute the request and capture both status and body
    local response=$(curl -g -s -w "\n%{http_code}" -X POST -d "$data" "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return full response (for backward compatibility)
    echo "$response"
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
    
    # Log request in debug mode
    debug_log_request "PATCH" "$url" "$@" -d "$data"
    
    # Execute the request and capture both status and body
    local response=$(curl -g -s -w "\n%{http_code}" -X PATCH -d "$data" "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return full response (for backward compatibility)
    echo "$response"
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
    
    # Log request in debug mode
    debug_log_request "PUT" "$url" "$@" -d "$data"
    
    # Execute the request and capture both status and body
    local response=$(curl -g -s -w "\n%{http_code}" -X PUT -d "$data" "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return full response (for backward compatibility)
    echo "$response"
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
    
    # Log request in debug mode
    debug_log_request "DELETE" "$url" "$@"
    
    # Execute the request and capture both status and body
    local response=$(curl -g -s -w "\n%{http_code}" -X DELETE "$@" "$url")
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')
    
    # Log response in debug mode
    debug_log_response "$http_code" "$body"
    
    # Return full response (for backward compatibility)
    echo "$response"
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
    echo "COMPLIANCE_TEST_RESULT:PASSED=$PASSED:FAILED=$FAILED:SKIPPED=$SKIPPED:TOTAL=$TOTAL"
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
