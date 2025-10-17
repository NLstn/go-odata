# OData v4 Compliance Tests

This directory contains compliance tests for validating the go-odata library against the OData v4 specification.

## Overview

The compliance test suite consists of individual test scripts organized by OData v4 specification sections. Each test script validates specific aspects of the OData protocol implementation.

## Test Structure

Each test script:
- Is executable and can be run independently
- Makes HTTP requests to a running OData service
- Validates responses against OData v4 specification requirements
- Prints clear pass/fail results with descriptions
- Cleans up any test data it creates (non-destructive)
- Returns exit code 0 on success, 1 on failure

## Available Tests

### Headers (Section 8.x)
- **8.1.1_header_content_type.sh** - Validates Content-Type headers for different response types
- **8.2.6_header_odata_version.sh** - Tests OData-Version header and version negotiation

### Service Document (Section 9.x)
- **9.1_service_document.sh** - Validates service document structure and format

### Query Options (Section 11.2.5.x)
- **11.2.5.1_query_filter.sh** - Tests $filter query option with various operators
- **11.2.5.2_query_select_orderby.sh** - Tests $select and $orderby query options

### CRUD Operations (Section 11.4.x)
- **11.4.2_create_entity.sh** - Tests entity creation (POST) with proper headers and status codes

## Running Tests

### Prerequisites

1. Start the development server:
   ```bash
   cd cmd/devserver
   go run .
   ```

2. The server should be running on http://localhost:8080 (default)

### Run All Tests

```bash
cd compliance/v4
./run_compliance_tests.sh
```

This will:
- Run all test scripts in sequence
- Print results to console with color coding
- Generate a markdown report (`compliance-report.md`)
- Exit with code 0 if all tests pass, 1 if any fail

### Run Specific Tests

Run tests matching a pattern:
```bash
./run_compliance_tests.sh header          # All header tests
./run_compliance_tests.sh 8.1.1          # Specific section
./run_compliance_tests.sh query          # All query option tests
```

Run a single test directly:
```bash
./8.1.1_header_content_type.sh
```

### Custom Server URL

```bash
SERVER_URL=http://localhost:9090 ./run_compliance_tests.sh
```

Or:
```bash
./run_compliance_tests.sh -s http://localhost:9090
```

### Custom Report File

```bash
./run_compliance_tests.sh -o my-report.md
```

### Verbose Output

```bash
./run_compliance_tests.sh -v
```

## Test Output

### Console Output

Each test provides:
- Test section name and description
- Spec reference link
- Individual test results (✓ PASS or ✗ FAIL)
- Summary with pass/fail counts
- Overall status

Example:
```
======================================
OData v4 Compliance Test
Section: 8.1.1 Header Content-Type
======================================

Description: Validates that the service returns proper Content-Type headers
             with the correct media type and optional parameters.

Spec Reference: https://docs.oasis-open.org/odata/odata/v4.01/...

Test 1: Service Document Content-Type
  Request: GET http://localhost:8080/
✓ PASS: Service Document returns application/json with odata.metadata=minimal

...

======================================
Summary: 5/5 tests passed
Status: PASSING
```

### Markdown Report

The master test runner generates a markdown report with:
- Overall summary statistics
- Test results table with pass/fail status
- Test categories and descriptions
- Instructions for running tests

## Adding New Tests

To add a new compliance test:

1. Create a new shell script following the naming pattern: `{section}_{test_name}.sh`
   - Use the OData spec section number as prefix (e.g., `8.1.1`, `11.4.2`)

2. Make it executable:
   ```bash
   chmod +x new_test.sh
   ```

3. Follow this structure:
   ```bash
   #!/bin/bash
   
   # OData v4 Compliance Test: {Section} {Title}
   # Description of what this test validates
   # Spec: {URL to specification}
   
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   SERVER_URL="${SERVER_URL:-http://localhost:8080}"
   
   echo "======================================"
   echo "OData v4 Compliance Test"
   echo "Section: {Section Number} {Title}"
   echo "======================================"
   echo ""
   echo "Description: {What this test validates}"
   echo ""
   echo "Spec Reference: {URL}"
   echo ""
   
   PASSED=0
   FAILED=0
   TOTAL=0
   
   test_result() {
       local test_name="$1"
       local result="$2"
       local details="$3"
       
       TOTAL=$((TOTAL + 1))
       if [ "$result" = "PASS" ]; then
           PASSED=$((PASSED + 1))
           echo "✓ PASS: $test_name"
       else
           FAILED=$((FAILED + 1))
           echo "✗ FAIL: $test_name"
           if [ -n "$details" ]; then
               echo "  Details: $details"
           fi
       fi
   }
   
   # Add your tests here
   echo "Test 1: Description"
   echo "  Request: GET $SERVER_URL/endpoint"
   # ... test logic ...
   test_result "Test description" "PASS" ""
   echo ""
   
   # Summary
   echo "======================================"
   echo "Summary: $PASSED/$TOTAL tests passed"
   if [ $FAILED -gt 0 ]; then
       echo "Status: FAILING"
       exit 1
   else
       echo "Status: PASSING"
       exit 0
   fi
   ```

4. Test your script:
   ```bash
   ./new_test.sh
   ```

5. Add it to the suite - it will be automatically discovered by `run_compliance_tests.sh`

## Guidelines for Tests

### Test Independence
- Each test should be independent and not rely on other tests
- Tests should not assume specific data exists unless they create it
- Clean up any data created during the test

### Non-Destructive
- Don't delete or modify existing seeded data
- If you need to test deletion, create the entity first
- If you need to test updates, create a test entity

### Clear Output
- Use descriptive test names
- Print the actual HTTP request being made
- Provide detailed failure messages
- Include relevant response data in failures

### Comprehensive Coverage
- Test both success and failure cases
- Validate all relevant aspects (status codes, headers, body structure)
- Check edge cases when appropriate

## Specification References

- [OData v4.01 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
- [OData v4.01 Part 2: URL Conventions](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html)
- [OData v4.01 Part 3: CSDL](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html)

## Contributing

When adding new tests:
1. Reference the specific section of the OData v4 spec
2. Include the spec URL in test comments
3. Test against the development server
4. Ensure all tests pass before submitting
5. Update this README if adding new test categories

## CI/CD Integration

These tests can be integrated into CI/CD pipelines:

```bash
# Start server in background
cd cmd/devserver
go run . &
SERVER_PID=$!

# Wait for server to start
sleep 5

# Run tests
cd ../../compliance/v4
./run_compliance_tests.sh

# Capture exit code
EXIT_CODE=$?

# Stop server
kill $SERVER_PID

# Exit with test result
exit $EXIT_CODE
```

## Future Test Coverage

Additional tests to be added:
- $expand query option
- $top and $skip pagination
- $count inline and standalone
- Update operations (PATCH and PUT)
- Delete operations (DELETE)
- Batch requests ($batch)
- Actions and Functions
- ETag and If-Match headers
- Prefer header handling
- Error response format
- Metadata document validation (XML and JSON)
- Navigation properties
- Complex types
- Collection properties
- Null handling

## License

These tests are part of the go-odata project and follow the same MIT license.
