#!/bin/bash

# Load Testing Script for go-odata Performance Server
# This script runs various load tests against the perfserver to measure performance

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVER_URL="${SERVER_URL:-http://localhost:9091}"
OUTPUT_DIR="${OUTPUT_DIR:-./load-test-results}"
WRK_DURATION="${WRK_DURATION:-30s}"
WRK_THREADS="${WRK_THREADS:-10}"
WRK_CONNECTIONS="${WRK_CONNECTIONS:-100}"
DB_TYPE="sqlite"           # sqlite | postgres
DB_DSN=""                  # Optional; for postgres defaults if empty
EXTERNAL_SERVER=0          # Don't start/stop server automatically
ENABLE_CPU_PROFILE=0       # Enable CPU profiling
ENABLE_SQL_TRACE=0         # Enable SQL query tracing

# Variables to track if we started the server
SERVER_PID=""
TMP_SERVER_DIR=""
CLEANUP_DONE=0

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to cleanup and stop server
cleanup() {
    if [ $CLEANUP_DONE -eq 1 ]; then
        return
    fi
    CLEANUP_DONE=1
    
    if [ -n "$SERVER_PID" ]; then
        echo ""
        echo "Stopping performance server (PID: $SERVER_PID)..."
        # Send SIGINT (Ctrl+C) instead of SIGKILL to allow graceful shutdown
        kill -INT $SERVER_PID 2>/dev/null || true
        # Wait a bit for graceful shutdown
        sleep 3
        # Force kill if still running
        kill -9 $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        echo "Server stopped."
        
        # Copy profiling/tracing files to output directory if they exist
        if [ $ENABLE_CPU_PROFILE -eq 1 ] && [ -f "/tmp/perfserver-cpu.prof" ]; then
            echo "Copying CPU profile to output directory..."
            cp /tmp/perfserver-cpu.prof "$OUTPUT_DIR/cpu.prof"
            echo -e "${GREEN}✓ CPU profile saved to: $OUTPUT_DIR/cpu.prof${NC}"
            echo ""
            echo "Analyze with:"
            echo "  go tool pprof $OUTPUT_DIR/cpu.prof"
            echo "  go tool pprof -http=:8080 $OUTPUT_DIR/cpu.prof"
        fi
        
        if [ $ENABLE_SQL_TRACE -eq 1 ] && [ -f "/tmp/perfserver-sql-trace.txt" ]; then
            echo "Copying SQL trace to output directory..."
            cp /tmp/perfserver-sql-trace.txt "$OUTPUT_DIR/sql-trace.txt"
            echo -e "${GREEN}✓ SQL trace saved to: $OUTPUT_DIR/sql-trace.txt${NC}"
        fi
    fi
    
    # Clean up temporary server directory
    if [ -n "$TMP_SERVER_DIR" ] && [ -d "$TMP_SERVER_DIR" ]; then
        rm -rf "$TMP_SERVER_DIR"
    fi
}

# Register cleanup function to run on exit
trap cleanup EXIT INT TERM

# Function to print section headers
print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Function to run wrk test
run_test() {
    local name="$1"
    local url="$2"
    local output_file="$OUTPUT_DIR/wrk_${name}.txt"
    
    echo -e "${YELLOW}Testing: ${name}${NC}"
    echo -e "URL: ${url}"
    echo -e "Duration: ${WRK_DURATION}, Threads: ${WRK_THREADS}, Connections: ${WRK_CONNECTIONS}\n"
    
    if command -v wrk &> /dev/null; then
        wrk -t"$WRK_THREADS" -c"$WRK_CONNECTIONS" -d"$WRK_DURATION" --latency "$url" > "$output_file" 2>&1
        
        # Display results
        echo -e "${GREEN}Results:${NC}"
        cat "$output_file"
        echo ""
    else
        echo -e "${RED}wrk not found. Please install wrk to run load tests.${NC}\n"
        exit 1
    fi
}

# Function to check if server is running
check_server() {
    echo -e "${YELLOW}Checking if server is running at ${SERVER_URL}...${NC}"
    if curl -s -f "${SERVER_URL}/" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Server is running${NC}\n"
        return 0
    else
        echo -e "${RED}✗ Server is not accessible at ${SERVER_URL}${NC}"
        return 1
    fi
}

# Function to start the perfserver
start_server() {
    echo "Starting performance server..."
    
    # Find the project root
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    
    # Build the perfserver into /tmp directory
    echo "Building performance server..."
    cd "$SCRIPT_DIR"
    TMP_SERVER_DIR="/tmp/perfserver-$$"
    mkdir -p "$TMP_SERVER_DIR"
    go build -o "$TMP_SERVER_DIR/perfserver" . > /tmp/perfserver-build.log 2>&1
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ Failed to build performance server${NC}"
        echo ""
        echo "Build log:"
        cat /tmp/perfserver-build.log
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

    echo "Starting performance server from $TMP_SERVER_DIR/perfserver (db=$DB_TYPE)"
    
    # Add CPU profiling and SQL tracing arguments if enabled
    SERVER_ARGS=( "${DB_ARGS[@]}" -extensive=true )
    
    if [ $ENABLE_CPU_PROFILE -eq 1 ]; then
        SERVER_ARGS+=( -cpuprofile /tmp/perfserver-cpu.prof )
        echo "📊 CPU profiling: ENABLED"
    fi
    
    if [ $ENABLE_SQL_TRACE -eq 1 ]; then
        SERVER_ARGS+=( -trace-sql -trace-sql-file /tmp/perfserver-sql-trace.txt )
        echo "🔍 SQL tracing: ENABLED"
    fi
    
    "$TMP_SERVER_DIR/perfserver" "${SERVER_ARGS[@]}" > /tmp/perfserver.log 2>&1 &
    SERVER_PID=$!
    
    echo "Performance server started (PID: $SERVER_PID)"
    echo "Waiting for server to be ready..."
    
    # Wait for server to be ready (up to 60 seconds)
    for i in {1..60}; do
        if curl -s -f -o /dev/null -w "%{http_code}" "$SERVER_URL/" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Server is ready!${NC}"
            break
        fi
        if [ $i -eq 60 ]; then
            echo -e "${RED}✗ Server failed to start within 60 seconds${NC}"
            echo ""
            echo "Server log:"
            cat /tmp/perfserver.log
            exit 1
        fi
        sleep 1
    done
    echo ""
}

# Function to display summary
display_summary() {
    print_header "Load Test Summary"
    
    echo -e "${GREEN}Test results saved to: ${OUTPUT_DIR}${NC}\n"
    
    if [ -f "$OUTPUT_DIR/summary.txt" ]; then
        cat "$OUTPUT_DIR/summary.txt"
    fi
    
    echo -e "\n${YELLOW}Analysis Tips:${NC}"
    echo -e "  • Review wrk_*.txt files for detailed metrics"
    echo -e "  • Compare results across different query patterns"
    echo -e "  • Look for high latency or low throughput"
    echo -e "  • Monitor CPU/memory usage during tests\n"
}

# Main execution
main() {
    print_header "go-odata Performance Load Tests"
    
    echo -e "Configuration:"
    echo -e "  Server URL: ${GREEN}${SERVER_URL}${NC}"
    echo -e "  Output Directory: ${GREEN}${OUTPUT_DIR}${NC}"
    echo -e "  Duration: ${GREEN}${WRK_DURATION}${NC}"
    echo -e "  Threads: ${GREEN}${WRK_THREADS}${NC}"
    echo -e "  Connections: ${GREEN}${WRK_CONNECTIONS}${NC}"
    echo -e "  Database: ${GREEN}${DB_TYPE}${DB_DSN:+ (dsn provided)}${NC}"
    echo ""
    
    # Start server if not using external server
    if [ $EXTERNAL_SERVER -eq 0 ]; then
        start_server
    else
        # Check if external server is accessible
        echo -n "Checking external server connectivity... "
        if check_server; then
            echo -e "${GREEN}✓ Connected${NC}"
        else
            echo -e "${RED}✗ Failed${NC}"
            echo ""
            echo "Error: Cannot connect to server at $SERVER_URL"
            echo "Please ensure the perfserver is running:"
            echo "  cd cmd/perfserver"
            echo "  go run . -db sqlite -dsn :memory:"
            exit 1
        fi
        echo ""
    fi
    
    # Start timestamp
    START_TIME=$(date +%s)
    echo "Test started at: $(date)" > "$OUTPUT_DIR/summary.txt"
    
    # Test 1: Service Document
    print_header "Test 1: Service Document"
    run_test "service_document" "${SERVER_URL}/"
    
    # Test 2: Metadata Document
    print_header "Test 2: Metadata Document"
    run_test "metadata" "${SERVER_URL}/\$metadata"
    
    # Test 3: Simple Entity Collection
    print_header "Test 3: Categories (Simple Collection)"
    run_test "categories" "${SERVER_URL}/Categories"
    
    # Test 4: Large Entity Collection
    print_header "Test 4: Products (Large Collection)"
    run_test "products" "${SERVER_URL}/Products"
    
    # Test 5: Filtering
    print_header "Test 5: Filter Query"
    run_test "filter" "${SERVER_URL}/Products?\$filter=Price%20gt%20500"
    
    # Test 6: Ordering
    print_header "Test 6: OrderBy Query"
    run_test "orderby" "${SERVER_URL}/Products?\$orderby=Price%20desc"
    
    # Test 7: Pagination
    print_header "Test 7: Top and Skip"
    run_test "pagination" "${SERVER_URL}/Products?\$top=100&\$skip=1000"
    
    # Test 8: Select
    print_header "Test 8: Select Specific Fields"
    run_test "select" "${SERVER_URL}/Products?\$select=ID,Name,Price"
    
    # Test 9: Expand (Relationship)
    print_header "Test 9: Expand Navigation Property"
    run_test "expand" "${SERVER_URL}/Products?\$expand=Category"
    
    # Test 10: Complex Query
    print_header "Test 10: Complex Query (Filter + OrderBy + Top + Expand)"
    run_test "complex" "${SERVER_URL}/Products?\$filter=Price%20gt%20100&\$orderby=Price%20desc&\$top=50&\$expand=Category"
    
    # Test 11: Count
    print_header "Test 11: Count with Filter"
    run_test "count" "${SERVER_URL}/Products/\$count?\$filter=Price%20lt%20200"
    
    # Test 12: Single Entity by Key
    print_header "Test 12: Single Entity Lookup"
    run_test "single_entity" "${SERVER_URL}/Products(1)"
    
    # Test 13: Singleton
    print_header "Test 13: Singleton Access"
    run_test "singleton" "${SERVER_URL}/Company"
    
    # Test 14: Apply/Aggregation
    print_header "Test 14: Apply with GroupBy and Aggregate"
    run_test "apply" "${SERVER_URL}/Products?\$apply=groupby((CategoryID),aggregate(Price%20with%20average%20as%20AvgPrice))"
    
    # End timestamp
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo "Test completed at: $(date)" >> "$OUTPUT_DIR/summary.txt"
    echo "Total duration: ${DURATION} seconds" >> "$OUTPUT_DIR/summary.txt"
    
    display_summary
    
    echo -e "${GREEN}✓ All load tests completed!${NC}\n"
}

# Show help
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Run load tests against the go-odata performance server using wrk.

OPTIONS:
    -h, --help              Show this help message
    -u, --url URL           Server URL (default: http://localhost:9091)
    -o, --output DIR        Output directory for results (default: ./load-test-results)
    -d, --duration TIME     Duration for wrk tests (default: 30s)
    -t, --threads NUM       Number of threads for wrk (default: 10)
    -C, --connections NUM   Number of connections for wrk (default: 100)
    --db TYPE               Database type to use: sqlite | postgres (default: sqlite)
    --dsn DSN              Database DSN/connection string (required for postgres)
    --external-server      Use an external server (don't start/stop the perfserver)
    --cpu-profile          Enable CPU profiling (saves to OUTPUT_DIR/cpu.prof)
    --sql-trace            Enable SQL query tracing (saves to OUTPUT_DIR/sql-trace.txt)

EXAMPLES:
    # Run with default settings - auto-starts perfserver
    $0

    # Run with custom parameters
    $0 -d 60s -t 12 -C 200
    
    # Enable CPU profiling and SQL tracing
    $0 --cpu-profile --sql-trace

    # Custom server URL and output directory (external server)
    $0 --external-server -u http://localhost:8080 -o ./my-results

    # Use PostgreSQL database with profiling
    $0 --db postgres --dsn "postgresql://user:pass@localhost:5432/dbname" --cpu-profile

PREREQUISITES:
    - wrk must be installed
    - Go must be installed (to build the perfserver)

    Install wrk:
        sudo apt-get install wrk            # Debian/Ubuntu
        brew install wrk                    # macOS

NOTE:
    The script automatically starts and stops the performance server.
    Use --external-server if you want to manage the server yourself.

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -u|--url)
            SERVER_URL="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -d|--duration)
            WRK_DURATION="$2"
            shift 2
            ;;
        -t|--threads)
            WRK_THREADS="$2"
            shift 2
            ;;
        -C|--connections)
            WRK_CONNECTIONS="$2"
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
        --external-server)
            EXTERNAL_SERVER=1
            shift
            ;;
        --cpu-profile)
            ENABLE_CPU_PROFILE=1
            shift
            ;;
        --sql-trace)
            ENABLE_SQL_TRACE=1
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Run main
main
