#!/bin/bash

# OData v4 Compliance Test Suite - Master Runner
# Runs all or selected compliance tests and generates a comprehensive report

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_URL="${SERVER_URL:-http://localhost:9090}"
REPORT_FILE="${REPORT_FILE:-compliance-report.md}"
DB_TYPE="sqlite"           # sqlite | postgres
DB_DSN=""                  # Optional; for postgres defaults if empty
PARALLEL_JOBS=0            # Number of parallel jobs (0 = sequential)

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
declare -a TEST_SKIPPED
declare -a TEST_TOTAL

# Variable to track if we started the server
SERVER_PID=""
TMP_SERVER_DIR=""
TMP_OUTPUT_DIR=""
CLEANUP_DONE=0

# Function to print usage
usage() {
    echo "Usage: $0 [options] [test_script_pattern]"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  -s, --server URL     Set server URL (default: http://localhost:9090)"
    echo "  -o, --output FILE    Set report output file (default: compliance-report.md)"
    echo "  --db TYPE            Database type to use for the compliance server: sqlite | postgres (default: sqlite)"
    echo "  --dsn DSN           Database DSN/connection string (required for postgres unless DATABASE_URL is set)"
    echo "  --version VERSION    Run tests for specific OData version: 4.0 | 4.01 | all (default: all)"
    echo "  -v, --verbose        Show detailed test output"
    echo "  -f, --failures-only  Only show output for failing tests"
    echo "  --debug              Enable debug mode - prints full HTTP request/response for each test"
    echo "  --external-server    Use an external server (don't start/stop the compliance server)"
    echo "  -j, --parallel N     Run tests in parallel with N concurrent jobs (default: sequential)"
    echo ""
    echo "Examples:"
    echo "  $0                   # Run all tests (auto-starts compliance server)"
    echo "  $0 --version 4.0    # Run only OData 4.0 tests"
    echo "  $0 --version 4.01   # Run only OData 4.01 tests"
    echo "  $0 8.1.1            # Run specific test (auto-starts compliance server)"
    echo "  $0 10.1             # Run specific test with detailed output"
    echo "  $0 header           # Run all tests containing 'header'"
    echo "  $0 -f               # Run all tests, show only failures"
    echo "  $0 -v 10.1          # Run specific test with full verbose output"
    echo "  $0 --debug 8.1.1    # Run test with debug output (full HTTP details)"
    echo "  $0 --external-server # Use already running server"
    echo "  $0 -s http://localhost:9090 -o report.md"
    echo "  $0 -j 4             # Run tests with 4 parallel jobs"
    echo "  $0 --parallel 8     # Run tests with 8 parallel jobs"
    echo ""
    echo "Note: The script automatically starts and stops the compliance server."
    echo "      Use --external-server if you want to manage the server yourself."
    echo ""
    exit 0
}

# Function to cleanup and stop server
cleanup() {
    if [ $CLEANUP_DONE -eq 1 ]; then
        return
    fi
    CLEANUP_DONE=1
    
    if [ -n "$SERVER_PID" ]; then
        echo ""
        echo "Stopping compliance server (PID: $SERVER_PID)..."
        # Send SIGINT (Ctrl+C) instead of SIGKILL to allow graceful shutdown
        kill -INT $SERVER_PID 2>/dev/null || true
        # Wait a bit for graceful shutdown
        sleep 2
        # Force kill if still running
        kill -9 $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        echo "Server stopped."
    fi
    
    # Clean up temporary server directory
    if [ -n "$TMP_SERVER_DIR" ] && [ -d "$TMP_SERVER_DIR" ]; then
        rm -rf "$TMP_SERVER_DIR"
    fi
    
    # Note: TMP_OUTPUT_DIR is cleaned up after processing results, not here
}

# Register cleanup function to run on exit
trap cleanup EXIT INT TERM

# Parse command line arguments
VERBOSE=0
FAILURES_ONLY=0
PATTERN=""
SKIP_REPORT=0
EXTERNAL_SERVER=0
ODATA_VERSION="all"
# Set DEBUG from environment variable if not already set, default to 0
DEBUG=${DEBUG:-0}
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
        --db)
            DB_TYPE="$2"
            shift 2
            ;;
        --dsn)
            DB_DSN="$2"
            shift 2
            ;;
        --version)
            ODATA_VERSION="$2"
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
        --debug)
            DEBUG=1
            shift
            ;;
        --external-server)
            EXTERNAL_SERVER=1
            shift
            ;;
        -j|--parallel)
            PARALLEL_JOBS="$2"
            # Validate that it's a positive integer
            if ! [[ "$PARALLEL_JOBS" =~ ^[0-9]+$ ]] || [ "$PARALLEL_JOBS" -lt 1 ]; then
                echo "Error: --parallel/-j requires a positive integer (got: '$PARALLEL_JOBS')"
                exit 1
            fi
            shift 2
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
# Export DEBUG for child scripts
export DEBUG

echo ""
echo "╔════════════════════════════════════════════════════════╗"
echo "║     OData v4 Compliance Test Suite                     ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""
echo "Server URL: $SERVER_URL"
echo "Database:   $DB_TYPE${DB_DSN:+ (dsn provided)}"
echo "Version:    $ODATA_VERSION"
echo "Report File: $REPORT_FILE"
if [ $PARALLEL_JOBS -gt 0 ]; then
    echo "Parallel Jobs: $PARALLEL_JOBS"
fi
if [ $DEBUG -eq 1 ]; then
    echo "Debug Mode: ENABLED (full HTTP request/response details will be shown)"
fi
echo ""

# Start compliance server if not using external server
if [ $EXTERNAL_SERVER -eq 0 ]; then
    echo "Starting compliance server..."
    
    # Find the project root (two directories up from compliance/v4)
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    
    # Build the compliance server into /tmp directory
    echo "Building compliance server..."
    cd "$PROJECT_ROOT/cmd/complianceserver"
    TMP_SERVER_DIR="/tmp/complianceserver-$$"
    mkdir -p "$TMP_SERVER_DIR"
    go build -o "$TMP_SERVER_DIR/complianceserver" . > /tmp/compliance-build.log 2>&1
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ Failed to build compliance server${NC}"
        echo ""
        echo "Build log:"
        cat /tmp/compliance-build.log
        exit 1
    fi
    
    # Determine defaults for DB depending on type
    if [ "$DB_TYPE" = "postgres" ]; then
        # If no DSN provided, try DATABASE_URL or fall back to a common local default
        if [ -z "$DB_DSN" ]; then
            if [ -n "$DATABASE_URL" ]; then
                DB_DSN="$DATABASE_URL"
            else
                DB_DSN="postgresql://odata:odata_dev@localhost:5432/odata_test?sslmode=disable"
            fi
        fi
        DB_ARGS=( -db postgres -dsn "$DB_DSN" )
    else
        # sqlite by default; use :memory: unless a DSN was provided
        if [ -z "$DB_DSN" ]; then
            DB_DSN=":memory:"
        fi
        DB_ARGS=( -db sqlite -dsn "$DB_DSN" )
    fi

    echo "Starting compliance server from $TMP_SERVER_DIR/complianceserver (db=$DB_TYPE)"
    "$TMP_SERVER_DIR/complianceserver" "${DB_ARGS[@]}" > /tmp/compliance-server.log 2>&1 &
    SERVER_PID=$!
    
    echo "Compliance server started (PID: $SERVER_PID)"
    echo "Waiting for server to be ready..."
    
    # Wait for server to be ready (up to 60 seconds to account for first-time builds)
    for i in {1..60}; do
        if curl -s -f -o /dev/null -w "%{http_code}" "$SERVER_URL/" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Server is ready!${NC}"
            break
        fi
        if [ $i -eq 60 ]; then
            echo -e "${RED}✗ Server failed to start within 60 seconds${NC}"
            echo ""
            echo "Server log:"
            cat /tmp/compliance-server.log
            exit 1
        fi
        sleep 1
    done
    echo ""
    
    # Return to the compliance/ directory
    cd "$SCRIPT_DIR"
else
    # Check if external server is accessible
    echo -n "Checking external server connectivity... "
    if curl -s -f -o /dev/null -w "%{http_code}" "$SERVER_URL/" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Connected${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
        echo ""
        echo "Error: Cannot connect to server at $SERVER_URL"
        echo "Please ensure the compliance server is running:"
        echo "  cd cmd/complianceserver"
        echo "  go run ."
        exit 1
    fi
    echo ""
fi

# Validate version parameter
case "$ODATA_VERSION" in
    4.0|4.01|all)
        ;;
    *)
        echo "Error: Invalid version '$ODATA_VERSION'. Must be: 4.0, 4.01, or all"
        exit 1
        ;;
esac

# Determine which directories to search based on version
TEST_DIRS=()
case "$ODATA_VERSION" in
    4.0)
        TEST_DIRS=("$SCRIPT_DIR/v4.0")
        ;;
    4.01)
        TEST_DIRS=("$SCRIPT_DIR/v4.01")
        ;;
    all)
        TEST_DIRS=("$SCRIPT_DIR/v4.0" "$SCRIPT_DIR/v4.01")
        ;;
esac

# Find all test scripts
if [ -n "$PATTERN" ]; then
    echo "Running tests matching pattern: $PATTERN"
    SCRIPTS=""
    for dir in "${TEST_DIRS[@]}"; do
        DIR_SCRIPTS=$(find "$dir" -name "*${PATTERN}*.sh" -type f ! -name "run_compliance_tests.sh" ! -name "test_framework.sh" ! -name "convert_to_framework.sh" 2>/dev/null | sort)
        SCRIPTS="$SCRIPTS $DIR_SCRIPTS"
    done
else
    echo "Running all compliance tests for OData $ODATA_VERSION..."
    SCRIPTS=""
    for dir in "${TEST_DIRS[@]}"; do
        DIR_SCRIPTS=$(find "$dir" -name "*.sh" -type f ! -name "run_compliance_tests.sh" ! -name "test_framework.sh" ! -name "convert_to_framework.sh" 2>/dev/null | sort)
        SCRIPTS="$SCRIPTS $DIR_SCRIPTS"
    done
fi

if [ -z "$SCRIPTS" ]; then
    echo "No test scripts found."
    exit 1
fi

echo ""
echo "═════════════════════════════════════════════════════════"
echo ""



# Choose parallel or sequential execution
if [ "$PARALLEL_JOBS" -gt 0 ]; then
    echo "Running tests in parallel with $PARALLEL_JOBS concurrent jobs..."
    echo ""
    
    # Create temporary directory for test outputs
    TMP_OUTPUT_DIR=$(mktemp -d)
    
    # Build array of scripts to run
    SCRIPT_ARRAY=()
    for script in $SCRIPTS; do
        SCRIPT_ARRAY+=("$script")
    done
    
    # Launch tests in parallel with job control
    RUNNING_JOBS=0
    TEST_INDEX=0
    declare -a JOB_PIDS
    
    for script in "${SCRIPT_ARRAY[@]}"; do
        TEST_NAME=$(basename "$script" .sh)
        TEST_SCRIPTS[$TEST_INDEX]="$script"
        TEST_NAMES[$TEST_INDEX]="$TEST_NAME"
        
        # Determine OData version based on directory
        if [[ "$script" == */v4.01/* ]]; then
            VERSION_PREFIX="V4.01"
        else
            VERSION_PREFIX="V4"
        fi
        
        # Create output file for this test
        OUTPUT_FILE="$TMP_OUTPUT_DIR/test_${TEST_INDEX}.out"
        
        # Show that we're starting the test
        if [ $FAILURES_ONLY -eq 0 ]; then
            echo -e "${BLUE}Starting: [$VERSION_PREFIX] $TEST_NAME${NC}"
        fi
        
        # Run test in background (inline to ensure proper job tracking)
        # Use timeout to prevent tests from hanging indefinitely (60 seconds per test)
        (timeout 60 bash "$script" > "$OUTPUT_FILE" 2>&1; EXIT=$?; echo $EXIT > "${OUTPUT_FILE}.exit"; exit $EXIT) &
        JOB_PID=$!
        JOB_PIDS+=($JOB_PID)
        
        RUNNING_JOBS=$((RUNNING_JOBS + 1))
        
        # If we've reached the parallel job limit, wait for one to complete
        if [ $RUNNING_JOBS -ge $PARALLEL_JOBS ]; then
            wait -n  # Wait for any one background job to complete
            RUNNING_JOBS=$((RUNNING_JOBS - 1))
        fi
        
        TEST_INDEX=$((TEST_INDEX + 1))
    done
    
    # Wait for all remaining jobs to complete
    # Note: Some PIDs may have already been reaped by wait -n, so errors are suppressed
    wait 2>/dev/null || true
    
    echo ""
    echo "All tests completed. Processing results..."
    echo ""
    
    # Process results from output files
    for i in "${!TEST_SCRIPTS[@]}"; do
        script="${TEST_SCRIPTS[$i]}"
        TEST_NAME="${TEST_NAMES[$i]}"
        OUTPUT_FILE="$TMP_OUTPUT_DIR/test_${i}.out"
        
        # Determine OData version based on directory
        if [[ "$script" == */v4.01/* ]]; then
            VERSION_PREFIX="V4.01"
        else
            VERSION_PREFIX="V4"
        fi
        
        # Read output and exit code
        if [ -f "$OUTPUT_FILE" ]; then
            OUTPUT=$(cat "$OUTPUT_FILE")
            EXIT_CODE=$(cat "${OUTPUT_FILE}.exit" 2>/dev/null || echo "1")
        else
            OUTPUT="Error: Test output file not found"
            EXIT_CODE=1
        fi
        
        # Determine pass/fail
        if [ "$EXIT_CODE" = "0" ]; then
            TEST_RESULTS[$i]="PASS"
            RESULT_MSG="${GREEN}✓ PASSED${NC}"
        else
            TEST_RESULTS[$i]="FAIL"
            RESULT_MSG="${RED}✗ FAILED${NC}"
        fi
        
        # Decide whether to show output based on flags
        SHOW_OUTPUT=0
        if [ $FAILURES_ONLY -eq 1 ]; then
            # Only show output for failing tests
            if [ "$EXIT_CODE" != "0" ]; then
                SHOW_OUTPUT=1
            fi
        else
            # Always show for normal runs
            SHOW_OUTPUT=1
        fi
        
        if [ $SHOW_OUTPUT -eq 1 ]; then
            echo -e "${BLUE}[$VERSION_PREFIX] $TEST_NAME${NC}"
            
            if [ $VERBOSE -eq 1 ] || [ $SKIP_REPORT -eq 1 ]; then
                # Show full output for verbose mode OR when running individual tests
                echo "─────────────────────────────────────────────────────────"
                echo "$OUTPUT"
            fi
            
            echo -e "$RESULT_MSG"
            echo ""
        fi
        
        # Extract test counts from standardized output format
        RESULT_LINE=$(echo "$OUTPUT" | grep "COMPLIANCE_TEST_RESULT:")
        
        if [ -n "$RESULT_LINE" ]; then
            PASSED=$(echo "$RESULT_LINE" | grep -oP 'PASSED=\K\d+' || echo "0")
            FAILED=$(echo "$RESULT_LINE" | grep -oP 'FAILED=\K\d+' || echo "0")
            SKIPPED=$(echo "$RESULT_LINE" | grep -oP 'SKIPPED=\K\d+' || echo "0")
            TOTAL=$(echo "$RESULT_LINE" | grep -oP 'TOTAL=\K\d+' || echo "0")
            
            TEST_PASSED[$i]=${PASSED:-0}
            TEST_SKIPPED[$i]=${SKIPPED:-0}
            TEST_TOTAL[$i]=${TOTAL:-0}
        else
            # ERROR: Test script does not use the standardized test framework
            if [ $SHOW_OUTPUT -eq 1 ]; then
                echo ""
                echo -e "${RED}ERROR${NC}: Test script '$TEST_NAME' does not output the required COMPLIANCE_TEST_RESULT line"
                echo "  All compliance tests MUST use the test framework and call print_summary() at the end"
                echo "  Expected format: COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:SKIPPED=Z:TOTAL=W"
                echo ""
            fi
            
            # Mark this test as failed with 0 tests run
            TEST_PASSED[$i]=0
            TEST_SKIPPED[$i]=0
            TEST_TOTAL[$i]=0
            TEST_RESULTS[$i]="FAIL"
        fi
    done
    
    # Clean up temporary output directory after processing results
    if [ -n "$TMP_OUTPUT_DIR" ] && [ -d "$TMP_OUTPUT_DIR" ]; then
        rm -rf "$TMP_OUTPUT_DIR"
    fi
else
    # Sequential execution (original behavior)
    TEST_INDEX=0
    for script in $SCRIPTS; do
        TEST_NAME=$(basename "$script" .sh)
        TEST_SCRIPTS[$TEST_INDEX]="$script"
        TEST_NAMES[$TEST_INDEX]="$TEST_NAME"
        
        # Determine OData version based on directory
        if [[ "$script" == */v4.01/* ]]; then
            VERSION_PREFIX="V4.01"
        else
            VERSION_PREFIX="V4"
        fi
        
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
            echo -e "${BLUE}Running: [$VERSION_PREFIX] $TEST_NAME${NC}"
            
            if [ $VERBOSE -eq 1 ] || [ $SKIP_REPORT -eq 1 ]; then
                # Show full output for verbose mode OR when running individual tests
                echo "─────────────────────────────────────────────────────────"
                echo "$OUTPUT"
            fi
            
            echo -e "$RESULT_MSG"
            echo ""
        fi
        
        # Extract test counts from standardized output format
        # Format: COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:SKIPPED=Z:TOTAL=W
        RESULT_LINE=$(echo "$OUTPUT" | grep "COMPLIANCE_TEST_RESULT:")
        
        if [ -n "$RESULT_LINE" ]; then
            PASSED=$(echo "$RESULT_LINE" | grep -oP 'PASSED=\K\d+' || echo "0")
            FAILED=$(echo "$RESULT_LINE" | grep -oP 'FAILED=\K\d+' || echo "0")
            SKIPPED=$(echo "$RESULT_LINE" | grep -oP 'SKIPPED=\K\d+' || echo "0")
            TOTAL=$(echo "$RESULT_LINE" | grep -oP 'TOTAL=\K\d+' || echo "0")
            
            TEST_PASSED[$TEST_INDEX]=${PASSED:-0}
            TEST_SKIPPED[$TEST_INDEX]=${SKIPPED:-0}
            TEST_TOTAL[$TEST_INDEX]=${TOTAL:-0}
        else
            # ERROR: Test script does not use the standardized test framework
            echo ""
            echo -e "${RED}ERROR${NC}: Test script '$TEST_NAME' does not output the required COMPLIANCE_TEST_RESULT line"
            echo "  All compliance tests MUST use the test framework and call print_summary() at the end"
            echo "  Expected format: COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:SKIPPED=Z:TOTAL=W"
            echo ""
            echo "  To fix this test:"
            echo "    1. Source the test framework at the top: source \"\$(dirname \"\$0\")/test_framework.sh\""
            echo "    2. Use the test_result() function for each test"
            echo "    3. Call print_summary() at the end instead of custom summary output"
            echo ""
            
            # Mark this test as failed with 0 tests run
            TEST_PASSED[$TEST_INDEX]=0
            TEST_SKIPPED[$TEST_INDEX]=0
            TEST_TOTAL[$TEST_INDEX]=0
            TEST_RESULTS[$TEST_INDEX]="FAIL"
        fi
        
        TEST_INDEX=$((TEST_INDEX + 1))
    done
fi

# Calculate overall statistics
TOTAL_PASSED=0
TOTAL_TESTS=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0
SCRIPTS_PASSED=0
SCRIPTS_TOTAL=${#TEST_SCRIPTS[@]}

for i in "${!TEST_SCRIPTS[@]}"; do
    PASSED_VAL=${TEST_PASSED[$i]:-0}
    SKIPPED_VAL=${TEST_SKIPPED[$i]:-0}
    TOTAL_VAL=${TEST_TOTAL[$i]:-0}
    TOTAL_PASSED=$((TOTAL_PASSED + PASSED_VAL))
    TOTAL_SKIPPED=$((TOTAL_SKIPPED + SKIPPED_VAL))
    TOTAL_TESTS=$((TOTAL_TESTS + TOTAL_VAL))
    TOTAL_FAILED=$((TOTAL_FAILED + TOTAL_VAL - PASSED_VAL - SKIPPED_VAL))
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
echo "  - Skipped: $TOTAL_SKIPPED"
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
**Database:** $DB_TYPE  
**OData Version:** $ODATA_VERSION  
**Overall Status:** $OVERALL_STATUS

## Summary

- **Test Scripts:** $SCRIPTS_PASSED/$SCRIPTS_TOTAL passed ($SCRIPT_PERCENTAGE%)
- **Individual Tests:** $TOTAL_TESTS total

| Metric | Count |
|--------|-------|
| Passing | $TOTAL_PASSED |
| Failing | $TOTAL_FAILED |
| Skipped | $TOTAL_SKIPPED |
| Total | $TOTAL_TESTS |

## Test Results

| Test Section | Status | Passed | Failed | Skipped | Total | Details |
|-------------|--------|--------|--------|---------|-------|---------|
EOF

# Create a temporary file for sorting
TEMP_REPORT=$(mktemp)

# Add each test to a temporary file for sorting
for i in "${!TEST_SCRIPTS[@]}"; do
    TEST_NAME="${TEST_NAMES[$i]}"
    STATUS="${TEST_RESULTS[$i]}"
    PASSED="${TEST_PASSED[$i]}"
    SKIPPED="${TEST_SKIPPED[$i]}"
    TOTAL="${TEST_TOTAL[$i]}"
    FAILED=$((TOTAL - PASSED - SKIPPED))
    
    if [ "$STATUS" = "PASS" ]; then
        STATUS_EMOJI="✅"
    else
        STATUS_EMOJI="❌"
    fi
    
    # Extract description from test script
    DESCRIPTION=$(grep -A1 "^# OData v4 Compliance Test:" "${TEST_SCRIPTS[$i]}" | tail -1 | sed 's/^# //')
    
    echo "$TEST_NAME|$STATUS_EMOJI $STATUS|$PASSED|$FAILED|$SKIPPED|$TOTAL|$DESCRIPTION" >> "$TEMP_REPORT"
done

# Sort by section number (numeric sort on the first field before underscore)
# Then format and append to report
sort -t'.' -k1,1n -k2,2n "$TEMP_REPORT" | while IFS='|' read -r name status passed failed skipped total desc; do
    echo "| $name | $status | $passed | $failed | $skipped | $total | $desc |" >> "$REPORT_FILE"
done

# Clean up temp file
rm -f "$TEMP_REPORT"

# Add footer to report
cat >> "$REPORT_FILE" << EOF

## Skipped Tests

Skipped tests indicate features from the OData v4 specification that are not yet fully implemented. These tests are marked as skipped to clearly indicate incomplete spec coverage and do not cause the compliance suite to fail.

**Note:** Tests with skipped count greater than 0 indicate areas where implementation is incomplete or pending.

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

To run all tests (both 4.0 and 4.01):
\`\`\`bash
cd compliance
./run_compliance_tests.sh
\`\`\`

To run only OData 4.0 tests:
\`\`\`bash
./run_compliance_tests.sh --version 4.0
\`\`\`

To run only OData 4.01 tests:
\`\`\`bash
./run_compliance_tests.sh --version 4.01
\`\`\`

To run specific tests:
\`\`\`bash
./run_compliance_tests.sh 8.1.1          # Run specific section
./run_compliance_tests.sh header        # Run tests matching pattern
./run_compliance_tests.sh --version 4.0 filter  # Run 4.0 filter tests
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
