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
    echo ""
    echo "Examples:"
    echo "  $0                   # Run all tests"
    echo "  $0 8.1.1            # Run specific test by section number"
    echo "  $0 header           # Run all tests containing 'header' in filename"
    echo "  $0 -s http://localhost:9090 -o report.md"
    echo ""
    exit 0
}

# Parse command line arguments
VERBOSE=0
PATTERN=""
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
        *)
            PATTERN="$1"
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
    SCRIPTS=$(find "$SCRIPT_DIR" -name "*${PATTERN}*.sh" -type f ! -name "run_compliance_tests.sh" | sort)
else
    echo "Running all compliance tests..."
    SCRIPTS=$(find "$SCRIPT_DIR" -name "*.sh" -type f ! -name "run_compliance_tests.sh" | sort)
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
    
    echo -e "${BLUE}Running: $TEST_NAME${NC}"
    echo "─────────────────────────────────────────────────────────"
    
    if [ $VERBOSE -eq 1 ]; then
        # Run with full output
        bash "$script"
        EXIT_CODE=$?
    else
        # Capture output and show summary
        OUTPUT=$(bash "$script" 2>&1)
        EXIT_CODE=$?
        
        # Extract and show only summary
        echo "$OUTPUT" | grep -E "(Test [0-9]+:|✓ PASS:|✗ FAIL:|Summary:|Status:)"
    fi
    
    # Extract test counts from output
    if [ $VERBOSE -eq 1 ]; then
        SUMMARY=$(bash "$script" 2>&1 | grep "Summary:")
    else
        SUMMARY=$(echo "$OUTPUT" | grep "Summary:")
    fi
    
    if [ -n "$SUMMARY" ]; then
        PASSED=$(echo "$SUMMARY" | grep -o '[0-9]*' | head -1)
        TOTAL=$(echo "$SUMMARY" | grep -o '[0-9]*' | tail -1)
        TEST_PASSED[$TEST_INDEX]=$PASSED
        TEST_TOTAL[$TEST_INDEX]=$TOTAL
    else
        TEST_PASSED[$TEST_INDEX]=0
        TEST_TOTAL[$TEST_INDEX]=0
    fi
    
    if [ $EXIT_CODE -eq 0 ]; then
        TEST_RESULTS[$TEST_INDEX]="PASS"
        echo -e "${GREEN}✓ PASSED${NC}"
    else
        TEST_RESULTS[$TEST_INDEX]="FAIL"
        echo -e "${RED}✗ FAILED${NC}"
    fi
    
    echo ""
    TEST_INDEX=$((TEST_INDEX + 1))
done

# Calculate overall statistics
TOTAL_PASSED=0
TOTAL_TESTS=0
SCRIPTS_PASSED=0
SCRIPTS_TOTAL=${#TEST_SCRIPTS[@]}

for i in "${!TEST_SCRIPTS[@]}"; do
    TOTAL_PASSED=$((TOTAL_PASSED + ${TEST_PASSED[$i]}))
    TOTAL_TESTS=$((TOTAL_TESTS + ${TEST_TOTAL[$i]}))
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
echo "Individual Tests: $TOTAL_PASSED/$TOTAL_TESTS passed ($PERCENTAGE%)"
echo ""

if [ $SCRIPTS_PASSED -eq $SCRIPTS_TOTAL ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    OVERALL_STATUS="PASSING"
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    OVERALL_STATUS="FAILING"
fi
echo ""

# Generate markdown report
cat > "$REPORT_FILE" << EOF
# OData v4 Compliance Test Report

**Generated:** $(date)  
**Server:** $SERVER_URL  
**Overall Status:** $OVERALL_STATUS

## Summary

- **Test Scripts:** $SCRIPTS_PASSED/$SCRIPTS_TOTAL passed ($SCRIPT_PERCENTAGE%)
- **Individual Tests:** $TOTAL_PASSED/$TOTAL_TESTS passed ($PERCENTAGE%)

## Test Results

| Test Section | Status | Passed | Total | Details |
|-------------|--------|--------|-------|---------|
EOF

# Add each test to the report
for i in "${!TEST_SCRIPTS[@]}"; do
    TEST_NAME="${TEST_NAMES[$i]}"
    STATUS="${TEST_RESULTS[$i]}"
    PASSED="${TEST_PASSED[$i]}"
    TOTAL="${TEST_TOTAL[$i]}"
    
    if [ "$STATUS" = "PASS" ]; then
        STATUS_EMOJI="✅"
    else
        STATUS_EMOJI="❌"
    fi
    
    # Extract description from test script
    DESCRIPTION=$(grep -A1 "^# OData v4 Compliance Test:" "${TEST_SCRIPTS[$i]}" | tail -1 | sed 's/^# //')
    
    echo "| $TEST_NAME | $STATUS_EMOJI $STATUS | $PASSED | $TOTAL | $DESCRIPTION |" >> "$REPORT_FILE"
done

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
