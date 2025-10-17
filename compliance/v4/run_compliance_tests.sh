#!/bin/bash

# OData v4 Compliance Test Suite - Master Runner
# Runs all or selected compliance tests and generates a comprehensive report

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
REPORT_FILE="${REPORT_FILE:-compliance-report.md}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Arrays to track test results
declare -a TEST_SCRIPTS
declare -a TEST_NAMES
declare -a TEST_RESULTS
declare -a TEST_PASSED
declare -a TEST_TOTAL

# Function to print usage
usage() {
    echo "Usage: $0 [options] [test_script_pattern]"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  -s, --server URL     Set server URL (default: http://localhost:8080)"
    echo "  -o, --output FILE    Set report output file (default: compliance-report.md)"
    echo "  -v, --verbose        Show detailed test output"
    echo "  -f, --failures-only  Only show output for failing tests"
    echo ""
    echo "Examples:"
    echo "  $0                   # Run all tests and generate compliance report"
    echo "  $0 8.1.1            # Run specific test (no report generated)"
    echo "  $0 10.1             # Run specific test with detailed output (no report)"
    echo "  $0 header           # Run all tests containing 'header' (no report)"
    echo "  $0 -f               # Run all tests, show only failures, generate report"
    echo "  $0 -v 10.1          # Run specific test with full verbose output"
    echo "  $0 -s http://localhost:9090 -o report.md"
    echo ""
    echo "Note: When running individual tests (with pattern), the compliance report"
    echo "      is NOT updated. Only full test runs (no pattern) generate reports."
    echo ""
    exit 0
}

# Parse command line arguments
VERBOSE=0
FAILURES_ONLY=0
PATTERN=""
SKIP_REPORT=0
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        -s|--server)
            SERVER_URL="$2"
            shift 2
            ;;
        -o|--output)
            REPORT_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        -f|--failures-only)
            FAILURES_ONLY=1
            shift
            ;;
        *)
            PATTERN="$1"
            SKIP_REPORT=1  # Don't generate report for individual tests
            shift
            ;;
    esac
done

# Export SERVER_URL for child scripts
export SERVER_URL

echo ""
echo "╔════════════════════════════════════════════════════════╗"
echo "║     OData v4 Compliance Test Suite                    ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""
echo "Server URL: $SERVER_URL"
echo "Report File: $REPORT_FILE"
echo ""

# Check if server is accessible
echo -n "Checking server connectivity... "
if curl -s -f -o /dev/null -w "%{http_code}" "$SERVER_URL/" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Connected${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
    echo ""
    echo "Error: Cannot connect to server at $SERVER_URL"
    echo "Please ensure the development server is running:"
    echo "  cd cmd/devserver"
    echo "  go run ."
    exit 1
fi
echo ""

# Find all test scripts
if [ -n "$PATTERN" ]; then
    echo "Running tests matching pattern: $PATTERN"
    SCRIPTS=$(find "$SCRIPT_DIR" -name "*${PATTERN}*.sh" -type f ! -name "run_compliance_tests.sh" ! -name "test_framework.sh" ! -name "convert_to_framework.sh" | sort)
else
    echo "Running all compliance tests..."
    SCRIPTS=$(find "$SCRIPT_DIR" -name "*.sh" -type f ! -name "run_compliance_tests.sh" ! -name "test_framework.sh" ! -name "convert_to_framework.sh" | sort)
fi

if [ -z "$SCRIPTS" ]; then
    echo "No test scripts found."
    exit 1
fi

echo ""
echo "═════════════════════════════════════════════════════════"
echo ""

# Run each test script
TEST_INDEX=0
for script in $SCRIPTS; do
    TEST_NAME=$(basename "$script" .sh)
    TEST_SCRIPTS[$TEST_INDEX]="$script"
    TEST_NAMES[$TEST_INDEX]="$TEST_NAME"
    
    # Capture output
    OUTPUT=$(bash "$script" 2>&1)
    EXIT_CODE=$?
    
    # Determine pass/fail
    if [ $EXIT_CODE -eq 0 ]; then
        TEST_RESULTS[$TEST_INDEX]="PASS"
        RESULT_MSG="${GREEN}✓ PASSED${NC}"
    else
        TEST_RESULTS[$TEST_INDEX]="FAIL"
        RESULT_MSG="${RED}✗ FAILED${NC}"
    fi
    
    # Decide whether to show output based on flags
    SHOW_OUTPUT=0
    if [ $FAILURES_ONLY -eq 1 ]; then
        # Only show output for failing tests
        if [ $EXIT_CODE -ne 0 ]; then
            SHOW_OUTPUT=1
        fi
    else
        # Always show for normal runs
        SHOW_OUTPUT=1
    fi
    
    if [ $SHOW_OUTPUT -eq 1 ]; then
        echo -e "${BLUE}Running: $TEST_NAME${NC}"
        echo "─────────────────────────────────────────────────────────"
        
        if [ $VERBOSE -eq 1 ] || [ $SKIP_REPORT -eq 1 ]; then
            # Show full output for verbose mode OR when running individual tests
            echo "$OUTPUT"
        else
            # Show summary for full test suite runs
            echo "$OUTPUT" | grep -E "(Test [0-9]+:|✓ PASS:|✗ FAIL:|Summary:|Status:)"
        fi
        
        echo -e "$RESULT_MSG"
        echo ""
    fi
    
    # Extract test counts from standardized output format
    # Format: COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:TOTAL=Z
    RESULT_LINE=$(echo "$OUTPUT" | grep "COMPLIANCE_TEST_RESULT:")
    
    if [ -n "$RESULT_LINE" ]; then
        PASSED=$(echo "$RESULT_LINE" | grep -oP 'PASSED=\K\d+' || echo "0")
        FAILED=$(echo "$RESULT_LINE" | grep -oP 'FAILED=\K\d+' || echo "0")
        TOTAL=$(echo "$RESULT_LINE" | grep -oP 'TOTAL=\K\d+' || echo "0")
        
        TEST_PASSED[$TEST_INDEX]=${PASSED:-0}
        TEST_TOTAL[$TEST_INDEX]=${TOTAL:-0}
    else
        # ERROR: Test script does not use the standardized test framework
        echo ""
        echo -e "${RED}ERROR${NC}: Test script '$TEST_NAME' does not output the required COMPLIANCE_TEST_RESULT line"
        echo "  All compliance tests MUST use the test framework and call print_summary() at the end"
        echo "  Expected format: COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:TOTAL=Z"
        echo ""
        echo "  To fix this test:"
        echo "    1. Source the test framework at the top: source \"\$(dirname \"\$0\")/test_framework.sh\""
        echo "    2. Use the test_result() function for each test"
        echo "    3. Call print_summary() at the end instead of custom summary output"
        echo ""
        
        # Mark this test as failed with 0 tests run
        TEST_PASSED[$TEST_INDEX]=0
        TEST_TOTAL[$TEST_INDEX]=0
        TEST_RESULTS[$TEST_INDEX]="FAIL"
    fi
    
    TEST_INDEX=$((TEST_INDEX + 1))
done

# Calculate overall statistics
TOTAL_PASSED=0
TOTAL_TESTS=0
TOTAL_FAILED=0
SCRIPTS_PASSED=0
SCRIPTS_TOTAL=${#TEST_SCRIPTS[@]}

for i in "${!TEST_SCRIPTS[@]}"; do
    PASSED_VAL=${TEST_PASSED[$i]:-0}
    TOTAL_VAL=${TEST_TOTAL[$i]:-0}
    TOTAL_PASSED=$((TOTAL_PASSED + PASSED_VAL))
    TOTAL_TESTS=$((TOTAL_TESTS + TOTAL_VAL))
    TOTAL_FAILED=$((TOTAL_FAILED + TOTAL_VAL - PASSED_VAL))
    if [ "${TEST_RESULTS[$i]}" = "PASS" ]; then
        SCRIPTS_PASSED=$((SCRIPTS_PASSED + 1))
    fi
done

# Calculate percentages
if [ $TOTAL_TESTS -gt 0 ]; then
    PERCENTAGE=$((TOTAL_PASSED * 100 / TOTAL_TESTS))
else
    PERCENTAGE=0
fi

if [ $SCRIPTS_TOTAL -gt 0 ]; then
    SCRIPT_PERCENTAGE=$((SCRIPTS_PASSED * 100 / SCRIPTS_TOTAL))
else
    SCRIPT_PERCENTAGE=0
fi

# Print console summary
echo "═════════════════════════════════════════════════════════"
echo ""
echo "╔════════════════════════════════════════════════════════╗"
echo "║                  OVERALL SUMMARY                       ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""
echo "Test Scripts: $SCRIPTS_PASSED/$SCRIPTS_TOTAL passed ($SCRIPT_PERCENTAGE%)"
echo "Individual Tests:"
echo "  - Total: $TOTAL_TESTS"
echo "  - Passing: $TOTAL_PASSED"
echo "  - Failing: $TOTAL_FAILED"
if [ $TOTAL_TESTS -gt 0 ]; then
    echo "  - Pass Rate: $PERCENTAGE%"
fi
echo ""

if [ $SCRIPTS_PASSED -eq $SCRIPTS_TOTAL ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    OVERALL_STATUS="PASSING"
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    OVERALL_STATUS="FAILING"
fi
echo ""

# Only generate report when running all tests (not individual tests)
if [ $SKIP_REPORT -eq 1 ]; then
    echo "Skipping report generation for individual test run."
    echo "Run without pattern to generate full compliance report."
    echo ""
    
    # Exit with appropriate code
    if [ "$OVERALL_STATUS" = "PASSING" ]; then
        exit 0
    else
        exit 1
    fi
fi

# Generate markdown report
cat > "$REPORT_FILE" << EOF
# OData v4 Compliance Test Report

**Generated:** $(date)  
**Server:** $SERVER_URL  
**Overall Status:** $OVERALL_STATUS

## Summary

- **Test Scripts:** $SCRIPTS_PASSED/$SCRIPTS_TOTAL passed ($SCRIPT_PERCENTAGE%)
- **Individual Tests:** $TOTAL_TESTS total

| Metric | Count |
|--------|-------|
| Passing | $TOTAL_PASSED |
| Failing | $TOTAL_FAILED |
| Total | $TOTAL_TESTS |

## Test Results

| Test Section | Status | Passed | Failed | Total | Details |
|-------------|--------|--------|--------|-------|---------|
EOF

# Create a temporary file for sorting
TEMP_REPORT=$(mktemp)

# Add each test to a temporary file for sorting
for i in "${!TEST_SCRIPTS[@]}"; do
    TEST_NAME="${TEST_NAMES[$i]}"
    STATUS="${TEST_RESULTS[$i]}"
    PASSED="${TEST_PASSED[$i]}"
    TOTAL="${TEST_TOTAL[$i]}"
    FAILED=$((TOTAL - PASSED))
    
    if [ "$STATUS" = "PASS" ]; then
        STATUS_EMOJI="✅"
    else
        STATUS_EMOJI="❌"
    fi
    
    # Extract description from test script
    DESCRIPTION=$(grep -A1 "^# OData v4 Compliance Test:" "${TEST_SCRIPTS[$i]}" | tail -1 | sed 's/^# //')
    
    echo "$TEST_NAME|$STATUS_EMOJI $STATUS|$PASSED|$FAILED|$TOTAL|$DESCRIPTION" >> "$TEMP_REPORT"
done

# Sort by section number (numeric sort on the first field before underscore)
# Then format and append to report
sort -t'.' -k1,1n -k2,2n "$TEMP_REPORT" | while IFS='|' read -r name status passed failed total desc; do
    echo "| $name | $status | $passed | $failed | $total | $desc |" >> "$REPORT_FILE"
done

# Clean up temp file
rm -f "$TEMP_REPORT"

# Add footer to report
cat >> "$REPORT_FILE" << EOF

## Test Categories

### Headers (8.x)
Tests for HTTP headers according to OData v4 specification, including Content-Type, OData-Version, and other protocol headers.

### Service Document (9.1)
Tests for the service document format and structure.

### Query Options (11.2.5.x)
Tests for system query options like \$filter, \$select, \$orderby, \$top, \$skip, etc.

### CRUD Operations (11.4.x)
Tests for Create, Read, Update, and Delete operations on entities.

## Running Tests

To run all tests:
\`\`\`bash
cd compliance/v4
./run_compliance_tests.sh
\`\`\`

To run specific tests:
\`\`\`bash
./run_compliance_tests.sh 8.1.1          # Run specific section
./run_compliance_tests.sh header        # Run tests matching pattern
\`\`\`

## Notes

- Tests are designed to be non-destructive and clean up any test data they create
- Each test validates specific aspects of the OData v4 specification
- Tests should be run against a clean development server instance

EOF

echo "Report generated: $REPORT_FILE"
echo ""

# Exit with appropriate code
if [ "$OVERALL_STATUS" = "PASSING" ]; then
    exit 0
else
    exit 1
fi
