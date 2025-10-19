# OData v4 Compliance Tests

This directory contains compliance tests for validating the go-odata library against the OData v4 specification.

## Overview

The compliance test suite consists of **59 individual test scripts** organized by OData v4 specification sections. Each test script validates specific aspects of the OData protocol implementation.

### Test Coverage Summary

- **13 Header & Format Tests** - HTTP headers, status codes, JSON format, caching, ETags, OData-EntityId
- **3 Metadata Tests** - Service Document, Metadata Document, Operations
- **11 URL Convention Tests** - Entity Addressing, Canonical URL, Property Access, Collection Operations, Metadata Levels, Delta Links, Lambda Operators, Property $value, Stream Properties, Type Casting
- **18 Query Option Tests** - $filter (with string/date/arithmetic/type/logical/comparison/geo operators), $select, $orderby, $top, $skip, $skiptoken, $count, $expand, $search, $format, $apply, $compute
- **10 Data Modification Tests** - GET, POST, PATCH, PUT, DELETE, HEAD, Conditional Requests, Relationships, Modify Relationships, Deep Insert, Batch, Asynchronous
- **5 Data Type Tests** - Primitive data types handling, Nullable properties, Collection properties, Complex types, Enum types
- **1 Annotations Test** - Instance annotations and control information

## Test Structure

Each test script:
- Is executable and can be run independently
- Makes HTTP requests to a running OData service
- Validates responses against OData v4 specification requirements
- Prints clear pass/fail results with descriptions
- Cleans up any test data it creates (non-destructive)
- Returns exit code 0 on success, 1 on failure

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

### Show Only Failures

```bash
./run_compliance_tests.sh -f
```

### Verbose Output

```bash
./run_compliance_tests.sh -v
```

## Test Report

The master script generates a markdown report (`compliance-report.md`) with:

- Overall pass/fail status
- Test script counts (how many test files passed/failed)
- Individual test counts (how many individual tests passed/failed)
- Detailed results for each test section sorted by OData specification section number
- Breakdown of passing vs failing individual tests

Example report structure:
```markdown
## Summary

- **Test Scripts:** 23/30 passed (76%)
- **Individual Tests:** 150 total

| Metric | Count |
|--------|-------|
| Passing | 134 |
| Failing | 16 |
| Total | 150 |

## Test Results

| Test Section | Status | Passed | Failed | Total | Details |
|-------------|--------|--------|--------|-------|---------|
| 8.1.1_header_content_type | ✅ PASS | 5 | 0 | 5 | Tests that... |
...
```

## Available Tests

### Primitive Types (Section 5.x)
- **5.1.1_primitive_data_types.sh** - Tests handling of OData primitive data types (String, Int32, Decimal, Boolean, DateTime, etc.)
- **5.1.2_nullable_properties.sh** - Tests handling of nullable properties, null values in filters and responses, setting properties to null
- **5.1.3_collection_properties.sh** - Tests collection-valued properties (arrays), filtering with any/all operators, and collection operations
- **5.2_complex_types.sh** - Tests complex (structured) types, nested properties, filtering, and complex type operations
- **5.3_enum_types.sh** - Tests enumeration types, enum filtering with numeric/string values, and enum operations

### Headers & Response Codes (Section 8.x)
- **8.1.1_header_content_type.sh** - Validates Content-Type headers for different response types
- **8.1.5_response_status_codes.sh** - Tests correct HTTP status codes for various operations (200, 201, 204, 400, 404, etc.)
- **8.2.1_cache_control_header.sh** - Tests Cache-Control header handling for HTTP caching
- **8.2.2_header_if_match.sh** - Tests If-Match and If-None-Match headers for optimistic concurrency control with ETags
- **8.2.3_header_odata_entityid.sh** - Tests OData-EntityId response header for entity operations
- **8.2.6_header_odata_version.sh** - Tests OData-Version header and version negotiation
- **8.2.7_header_accept.sh** - Tests Accept header content negotiation and media type handling
- **8.2.8_header_prefer.sh** - Tests Prefer header (return=minimal, return=representation, odata.maxpagesize)
- **8.2.9_header_maxversion.sh** - Tests OData-MaxVersion header for version negotiation
- **8.3_error_responses.sh** - Validates error response format and structure

### Service Document & Metadata (Section 9.x)
- **9.1_service_document.sh** - Validates service document structure and format
- **9.2_metadata_document.sh** - Tests metadata document structure, XML format, and schema elements

### JSON Format (Section 10.x)
- **10.1_json_format.sh** - Tests JSON format requirements (value property, @odata.context, valid structure, etc.)

### URL Conventions (Section 11.2.x)
- **11.2.1_addressing_entities.sh** - Tests entity addressing (entity sets, single entities, properties, $value)
- **11.2.2_canonical_url.sh** - Tests canonical URL representation in @odata.id and dereferenceability
- **11.2.3_property_access.sh** - Tests accessing individual properties and property $value
- **11.2.4_collection_operations.sh** - Tests addressing entity collections vs single entities, collection format with value wrapper
- **11.2.7_metadata_levels.sh** - Tests odata.metadata parameter (minimal, full, none)
- **11.2.8_delta_links.sh** - Tests delta link support for change tracking (optional feature)
- **11.2.12_stream_properties.sh** - Tests media entities, stream properties, and $value access for binary content (optional feature)
- **11.2.13_type_casting.sh** - Tests derived types, type casting in URLs, isof/cast functions, and polymorphic queries (optional feature)
- **11.2.9_lambda_operators.sh** - Tests lambda operators (any, all) for collection filtering
- **11.2.10_addressing_operations.sh** - Tests addressing bound and unbound actions and functions
- **11.2.11_property_value.sh** - Tests accessing raw property values using $value path segment

### Query Options - Search (Section 11.2.4.x)
- **11.2.4.1_query_search.sh** - Tests $search query option for free-text search

### Query Options - System (Section 11.2.5.x)
- **11.2.5.1_query_filter.sh** - Tests $filter query option with various operators (eq, gt, contains, and/or)
- **11.2.5.2_query_select_orderby.sh** - Tests $select and $orderby query options
- **11.2.5.3_query_top_skip.sh** - Tests $top and $skip query options for paging
- **11.2.5.4_query_apply.sh** - Tests $apply query option for data aggregation (optional extension)
- **11.2.5.5_query_count.sh** - Tests $count query option (count with filter, top, etc.)
- **11.2.5.6_query_expand.sh** - Tests $expand query option for expanding related entities
- **11.2.5.7_query_skiptoken.sh** - Tests $skiptoken for server-driven paging and continuation tokens
- **11.2.5.8_query_compute.sh** - Tests $compute query option for computed properties (OData v4.01 feature, optional)

### Query Options - Format (Section 11.2.6)
- **11.2.6_query_format.sh** - Tests $format query option for specifying response format

### Built-in Filter Functions (Section 11.3.x)
- **11.3.1_filter_string_functions.sh** - Tests string functions (contains, startswith, endswith, length, indexof, substring, tolower, toupper, trim, concat)
- **11.3.2_filter_date_functions.sh** - Tests date/time functions (year, month, day, hour, minute, second, date, time, now)
- **11.3.3_filter_arithmetic_functions.sh** - Tests arithmetic operators and math functions (add, sub, mul, div, mod, ceiling, floor, round)
- **11.3.4_filter_type_functions.sh** - Tests type checking and casting functions (isof, cast)
- **11.3.5_filter_logical_operators.sh** - Tests logical operators (and, or, not) and operator precedence with parentheses
- **11.3.6_filter_comparison_operators.sh** - Tests all comparison operators (eq, ne, gt, ge, lt, le) with various data types
- **11.3.7_filter_geo_functions.sh** - Tests geospatial functions (geo.distance, geo.length, geo.intersects) for geographic queries (optional feature)

### Data Modification (Section 11.4.x)
- **11.4.1_requesting_entities.sh** - Tests various methods to request individual entities (GET, HEAD, conditional)
- **11.4.2_create_entity.sh** - Tests entity creation (POST) with proper headers and status codes
- **11.4.3_update_entity.sh** - Tests entity updates (PATCH) including partial updates and error cases
- **11.4.4_delete_entity.sh** - Tests entity deletion (DELETE) and verification
- **11.4.5_upsert.sh** - Tests upsert operations (PUT) for creating or replacing entities
- **11.4.6_relationships.sh** - Tests relationship management with $ref (optional feature)
- **11.4.7_deep_insert.sh** - Tests creating entities with related entities in a single POST request (deep insert)
- **11.4.8_modify_relationships.sh** - Tests modifying relationships using $ref endpoints (PUT, POST, DELETE on $ref)
- **11.4.9_batch_requests.sh** - Tests batch request processing (optional feature)
- **11.4.10_asynchronous_requests.sh** - Tests asynchronous request processing with Prefer: respond-async header
- **11.4.11_head_requests.sh** - Tests HEAD requests for entities and collections, validates headers without body

### Conditional Operations (Section 11.5.x)
- **11.5.1_conditional_requests.sh** - Tests conditional requests with ETags (If-Match, If-None-Match)

### Annotations (Section 11.6)
- **11.6_annotations.sh** - Tests instance annotations, @odata control information, and custom annotations in responses

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

## Writing Compliance Tests

### Using the Test Framework

All new compliance tests **MUST** use the standardized test framework (`test_framework.sh`) to ensure consistent reporting and integration with the master test runner.

#### Quick Start

1. Create a new shell script following the naming pattern: `{section}_{test_name}.sh`
   - Use the OData spec section number as prefix (e.g., `8.1.1`, `11.4.2`)

2. Make it executable:
   ```bash
   chmod +x new_test.sh
   ```

3. Use the framework template:
   ```bash
   #!/bin/bash
   
   # OData v4 Compliance Test: <Section Number> <Title>
   # <Brief description>
   # Spec: <URL to OData v4 specification section>
   
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   source "$SCRIPT_DIR/test_framework.sh"
   
   echo "======================================"
   echo "OData v4 Compliance Test"
   echo "Section: <Section Number> <Title>"
   echo "======================================"
   echo ""
   echo "Description: <Detailed description>"
   echo ""
   echo "Spec Reference: <URL>"
   echo ""
   
   # Define cleanup function if you create test data
   CREATED_IDS=()
   
   cleanup() {
       for id in "${CREATED_IDS[@]}"; do
           curl -s -X DELETE "$SERVER_URL/YourEntity($id)" > /dev/null 2>&1
       done
   }
   
   # Register cleanup to run on exit
   register_cleanup
   
   # Define test functions
   test_something() {
       local HTTP_CODE=$(http_get "$SERVER_URL/YourEntity")
       check_status "$HTTP_CODE" "200"
   }
   
   test_another_thing() {
       local RESPONSE=$(http_get_body "$SERVER_URL/YourEntity")
       check_contains "$RESPONSE" "expectedValue"
   }
   
   # Run tests using run_test function
   run_test "Test description 1" test_something
   run_test "Test description 2" test_another_thing
   
   # The framework will automatically print the summary and exit
   ```

4. Test your script:
   ```bash
   ./new_test.sh
   ```

5. Add it to the suite - it will be automatically discovered by `run_compliance_tests.sh`

### Test Framework Functions

#### HTTP Request Functions

- `http_get URL [headers...]` - GET request, returns status code
- `http_get_body URL [headers...]` - GET request, returns response body
- `http_post URL data [headers...]` - POST request
- `http_patch URL data [headers...]` - PATCH request
- `http_put URL data [headers...]` - PUT request
- `http_delete URL [headers...]` - DELETE request

Example:
```bash
HTTP_CODE=$(http_get "$SERVER_URL/Products")
RESPONSE=$(http_get_body "$SERVER_URL/Products(1)" -H "Accept: application/json")
```

#### Validation Functions

- `check_status actual expected` - Verify HTTP status code
- `check_contains "$response" "value"` - Check if response contains value
- `check_json_field "$response" "fieldName"` - Check if JSON has field

Example:
```bash
test_status_code() {
    local CODE=$(http_get "$SERVER_URL/Products")
    check_status "$CODE" "200"
}

test_json_content() {
    local BODY=$(http_get_body "$SERVER_URL/Products(1)")
    check_json_field "$BODY" "Name"
}
```

#### Test Execution

- `run_test "description" test_function` - Run a test and track results
- `register_cleanup` - Register cleanup function to run on exit

Example:
```bash
run_test "Service returns 200 OK" test_status_code
run_test "Response contains Name field" test_json_content
```

### Standardized Output Format

The test framework automatically outputs results in a standardized, machine-parsable format:

```
COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:TOTAL=Z
```

**DO NOT** manually print this line or manipulate test counters - the framework handles everything automatically.

### Exit Codes

- Exit `0`: All tests passed
- Exit `1`: One or more tests failed

The framework handles exit codes automatically based on test results.

### Example Test

See `11.4.3_update_entity.sh` for a complete example using the test framework.

## Migrating Existing Tests

If you have existing compliance tests using the old format, migrate them to use the test framework:

### Before (Old Format)

```bash
PASSED=0
FAILED=0
TOTAL=0

# Test 1
TOTAL=$((TOTAL + 1))
if [ condition ]; then
    PASSED=$((PASSED + 1))
    echo "✓ PASS: Test name"
else
    FAILED=$((FAILED + 1))
    echo "✗ FAIL: Test name"
fi

# Print summary
echo "Summary: $PASSED/$TOTAL tests passed"
if [ $FAILED -eq 0 ]; then
    exit 0
else
    exit 1
fi
```

### After (New Framework)

```bash
source "$SCRIPT_DIR/test_framework.sh"

test_something() {
    # Test logic
    if [ condition ]; then
        return 0
    else
        return 1
    fi
}

run_test "Test name" test_something

# Framework handles summary and exit automatically
```

## Best Practices

### Test Independence
- Each test should be independent and not rely on other tests
- Tests should not assume specific data exists unless they create it
- Clean up any data created during the test using the `cleanup()` function

### Non-Destructive Testing
- Don't delete or modify existing seeded data
- If you need to test deletion, create the entity first
- If you need to test updates, create a test entity
- Always register cleanup with `register_cleanup`

### Clear Output
- Use descriptive test names that explain what is being validated
- Print the actual HTTP request being made for debugging
- Let the framework handle pass/fail messages
- Framework automatically provides detailed failure messages

### Comprehensive Coverage
- Test both success and failure cases
- Validate all relevant aspects (status codes, headers, body structure)
- Check edge cases when appropriate
- Reference the specific OData v4 specification section

### Use Framework Functions
- **DO** use `http_get`, `http_post`, etc. for HTTP requests
- **DO** use `check_status`, `check_contains` for validations
- **DO** use `run_test` to execute tests
- **DON'T** bypass the framework with raw curl and manual counters

## Specification References

- [OData v4.01 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
- [OData v4.01 Part 2: URL Conventions](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html)
- [OData v4.01 Part 3: CSDL](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html)

## CI/CD Integration

The compliance tests can be integrated into CI/CD pipelines. The master script returns exit code 0 on success and 1 on failure:

```bash
#!/bin/bash

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

For GitHub Actions or similar CI systems:

```yaml
- name: Start OData Server
  run: |
    cd cmd/devserver
    go run . &
    sleep 5

- name: Run Compliance Tests
  run: |
    cd compliance/v4
    ./run_compliance_tests.sh
    
- name: Upload Test Report
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: compliance-report
    path: compliance/v4/compliance-report.md
```

## Contributing

When adding new tests:
1. **Use the test framework** - Source `test_framework.sh` in all new tests
2. **Reference the spec** - Include the specific section number and URL
3. **Test thoroughly** - Run your test multiple times to ensure consistency
4. **Follow best practices** - See the Best Practices section above
5. **Ensure all tests pass** - Don't break existing tests
6. **Update documentation** - Add your test to the Available Tests section

## Future Enhancements

Potential improvements to the test framework:
- JSON schema validation for responses
- Performance/load testing capabilities
- Automated comparison with reference implementations
- Support for testing with different database backends
- Parallel test execution
- Test coverage metrics

Additional OData features to test:
- Derived types and type casting (expanded coverage)
- Enum types edge cases
- Stream properties and media entities
- Annotations in responses
- Geographic functions (geo.distance, geo.intersects, etc.)
- Advanced aggregation transformations
- Complex type nested properties
- Collection-valued complex properties

## License

These tests are part of the go-odata project and follow the same MIT license.
