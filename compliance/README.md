# OData v4 Compliance Tests

This directory contains compliance tests for validating the go-odata library against the OData v4 specification.

## Directory Structure

Tests are organized by OData version:

- **`v4.0/`** - OData 4.0 specification compliance tests (core functionality)
- **`v4.01/`** - OData 4.01 specification compliance tests (only features new/changed in 4.01)
- **`test_framework.sh`** - Shared test framework used by all tests
- **`run_compliance_tests.sh`** - Master test runner that can run tests for specific versions or all versions

### Version Organization

- **v4.0** contains all tests for OData 4.0 core features that apply to both 4.0 and 4.01
- **v4.01** contains only tests for features that are new or different in OData 4.01:
  - `$compute` query option for computed properties
  - `$index` query option for item ordinal position
  - `$orderby` with computed properties from `$compute`

When running all tests (default), both v4.0 and v4.01 tests are executed.

## Overview

The compliance test suite consists of **106 individual test scripts** with **967 individual test cases** organized by OData v4 specification sections. Each test script validates specific aspects of the OData protocol implementation.

### Test Coverage Summary

- **3 Specification Foundation Tests** - Introduction & Overview, Conformance Requirements, Extensibility
- **19 Header & Format Tests** - HTTP request/response headers, status codes, JSON format, caching, ETags, OData-EntityId, error response consistency, Content-Type, Accept, Prefer
- **3 Metadata Tests** - Service Document, Metadata Document, Operations
- **13 URL Convention Tests** - Resource Path, Entity Addressing, Canonical URL, Property Access, Collection Operations, Metadata Levels, Delta Links, Lambda Operators, Property $value, Stream Properties, Type Casting, Singleton Operations
- **24 Query Option Tests** - $filter (with string/date/arithmetic/type/logical/comparison/geo operators), $select, $orderby, $top, $skip, $skiptoken, $count, $expand, $search, $format, $apply (including advanced transformations), $compute, $index, nested expand options, query option combinations, orderby with computed properties
- **14 Data Modification Tests** - GET, POST, PATCH, PUT, DELETE, HEAD, Conditional Requests, Relationships, Modify Relationships, Deep Insert, Batch (including error handling), Asynchronous (including async processing), Navigation Property Operations, Action/Function Parameters
- **10 Data Type Tests** - Primitive data types, Numeric edge cases, Nullable properties, Collection properties, Complex types, Enum types (including metadata validation), Temporal types, Type definitions, Navigation Properties
- **12 CSDL Tests** - EDMX elements, DataServices, Reference, Include, IncludeAnnotations, Nominal types, Structured types, Primitive types, Built-in abstract types, Navigation properties, Annotations
- **2 Operations Tests** - Actions and Functions (bound and unbound operations), operation parameter validation
- **6 Advanced Tests** - Lambda operators, filter on expanded properties, vocabulary annotations, batch error handling, advanced aggregation transformations, asynchronous request processing
- **5 String & Internationalization Tests** - String functions, Unicode and internationalization, URL encoding, edge cases

## Test Structure

Each test script:
- Is executable and can be run independently
- Makes HTTP requests to a running OData service
- Validates responses against OData v4 specification requirements
- Prints clear pass/fail/skip results with descriptions
- Can mark tests as skipped when features are not yet implemented
- Cleans up any test data it creates (non-destructive)
- Returns exit code 0 on success (including when tests are skipped), 1 on failure

### Test Status Types

- **PASS** (✓): Test passed successfully
- **FAIL** (✗): Test failed
- **SKIP** (⊘): Test skipped because the corresponding OData feature is not yet implemented

Skipped tests help track which parts of the OData v4 spec are not yet implemented without causing the test suite to fail.

## Running Tests

### Prerequisites

The compliance test script automatically starts and stops the compliance server, so no manual server setup is required.

### Run All Tests

```bash
cd compliance
./run_compliance_tests.sh
```

This will:
- Automatically start the compliance server on port 9090
- Run all test scripts from both v4.0 and v4.01 directories
- Print results to console with color coding
- Generate a markdown report (`compliance-report.md`)
- Exit with code 0 if all tests pass, 1 if any fail

### Run Tests for Specific OData Version

Run only OData 4.0 tests:
```bash
./run_compliance_tests.sh --version 4.0
```

Run only OData 4.01 tests:
```bash
./run_compliance_tests.sh --version 4.01
```

Run all tests (both 4.0 and 4.01):
```bash
./run_compliance_tests.sh --version all   # or just omit the flag
```

### Run Specific Tests

Run tests matching a pattern (server auto-starts):
```bash
./run_compliance_tests.sh header          # All header tests
./run_compliance_tests.sh 8.1.1          # Specific section
./run_compliance_tests.sh query          # All query option tests
./run_compliance_tests.sh --version 4.0 filter  # Only 4.0 filter tests
```

Run a single test directly (requires server to be running):
```bash
cd v4.0
./8.1.1_header_content_type.sh
```

### Use External Server

If you want to manage the server manually:
```bash
# Terminal 1: Start the compliance server
cd cmd/complianceserver
go run .

# Terminal 2: Run tests with --external-server flag
cd compliance
./run_compliance_tests.sh --external-server
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

### Debug Mode

Enable debug mode to see complete HTTP request and response details for every test case. This is extremely useful for:
- Troubleshooting test failures
- Understanding the actual communication between client and server
- Analyzing response formats and headers
- Debugging integration issues

**Using the --debug flag:**
```bash
./run_compliance_tests.sh --debug 8.1.1        # Debug a specific test
./run_compliance_tests.sh --debug filter       # Debug all filter tests
./run_compliance_tests.sh --debug --version 4.0 query  # Debug 4.0 query tests
```

**Using the DEBUG environment variable:**
```bash
DEBUG=1 ./run_compliance_tests.sh 10.1_json_format
DEBUG=1 ./v4.0/10.1_json_format.sh             # Debug a single test script directly
```

**Debug output includes:**
- HTTP method (GET, POST, PATCH, PUT, DELETE)
- Full URL with query parameters
- Request headers (when applicable)
- Request body (when applicable, formatted as JSON if valid)
- HTTP status code
- Response body (formatted as JSON if valid, otherwise raw)

**Example debug output:**
```
╔══════════════════════════════════════════════════════╗
║ DEBUG: HTTP Request
╚══════════════════════════════════════════════════════╝

Method: GET
URL: http://localhost:9090/Products?$filter=Price gt 100

╔══════════════════════════════════════════════════════╗
║ DEBUG: HTTP Response
╚══════════════════════════════════════════════════════╝

Status Code: 200
Body:
{
    "@odata.context": "http://localhost:9090/$metadata#Products",
    "value": [
        {
            "ID": 1,
            "Name": "Laptop",
            "Price": 999.99
        }
    ]
}
```

**Note:** Debug mode works automatically with tests that use the framework's HTTP helper functions (`http_get`, `http_post`, `http_patch`, `http_put`, `http_delete`, `http_get_body`). Tests using raw `curl` commands won't show debug output.

## Test Report

The master script generates a markdown report (`compliance-report.md`) with:

- Overall pass/fail status
- Test script counts (how many test files passed/failed)
- Individual test counts (how many individual tests passed/failed/skipped)
- Detailed results for each test section sorted by OData specification section number
- Breakdown of passing vs failing vs skipped individual tests
- Information about skipped tests indicating incomplete spec coverage

Example report structure:
```markdown
## Summary

- **Test Scripts:** 23/30 passed (76%)
- **Individual Tests:** 150 total

| Metric | Count |
|--------|-------|
| Passing | 134 |
| Failing | 14 |
| Skipped | 2 |
| Total | 150 |

## Test Results

| Test Section | Status | Passed | Failed | Skipped | Total | Details |
|-------------|--------|--------|--------|---------|-------|---------|
| 8.1.1_header_content_type | ✅ PASS | 5 | 0 | 0 | 5 | Tests that... |
...
```

## Available Tests

### Specification Foundation (Sections 1.x, 2.x, 6.x)
- **1.1_introduction.sh** - Tests basic service requirements from the OData v4 introduction section, including service availability and protocol version support
- **2.1_conformance.sh** - Tests service conformance to OData v4 specification requirements including proper response formats, required headers, metadata availability, and protocol compliance (MUST requirements)
- **6.1_extensibility.sh** - Tests OData v4 extensibility features including support for instance annotations, custom annotations, and proper handling of unknown elements

### CSDL Schema (Sections 3.x, 4.x)
- **3.1_edmx_element.sh** - Tests Element edmx:Edmx in metadata
- **3.2_dataservices_element.sh** - Tests Element edmx:DataServices in metadata
- **3.3_reference_element.sh** - Tests Element edmx:Reference in metadata
- **3.4_include_element.sh** - Tests Element edmx:Include in metadata
- **3.5_includeannotations_element.sh** - Tests Element edmx:IncludeAnnotations in metadata
- **4.1_nominal_types.sh** - Tests nominal types in CSDL
- **4.2_structured_types.sh** - Tests structured types (entity types, complex types) in CSDL
- **4.3_navigation_properties.sh** - Tests navigation property definitions and relationships in metadata, including relationship types, multiplicity, and partner properties
- **4.4_primitive_types.sh** - Tests primitive types in CSDL
- **4.5_builtin_abstract_types.sh** - Tests built-in abstract types
- **4.6_annotations.sh** - Tests annotations in CSDL

### Primitive Types (Section 5.x)
- **5.1.1_primitive_data_types.sh** - Tests handling of OData primitive data types (String, Int32, Decimal, Boolean, DateTime, etc.)
- **5.1.1.1_numeric_edge_cases.sh** - Tests numeric edge cases including division by zero, precision, large numbers, boundary values, and special numeric conditions
- **5.1.2_nullable_properties.sh** - Tests handling of nullable properties, null values in filters and responses, setting properties to null
- **5.1.3_collection_properties.sh** - Tests collection-valued properties (arrays), filtering with any/all operators, and collection operations
- **5.1.4_temporal_data_types.sh** - Tests temporal OData types (Edm.Date, Edm.TimeOfDay, Edm.Duration), cast/isof functions, date/time literals and comparisons
- **5.2_complex_types.sh** - Tests complex (structured) types, nested properties, filtering, and complex type operations
- **5.3_enum_types.sh** - Tests enumeration types, enum filtering with numeric/string values, and enum operations
- **5.3_enum_metadata_members.sh** - Validates enumeration metadata members and namespace configuration for registered enums
- **5.4_type_definitions.sh** - Tests custom type definitions in metadata and their usage

### Headers & Response Codes (Section 8.x)
- **8.1.1_header_content_type.sh** - Validates Content-Type headers for different response types
- **8.1.2_request_headers.sh** - Tests proper handling of OData request headers including Accept, Content-Type, OData-MaxVersion, OData-Version, and other standard HTTP request headers
- **8.1.3_response_headers.sh** - Tests that OData services return proper response headers including Content-Type, OData-Version, and other required or recommended headers
- **8.1.5_response_status_codes.sh** - Tests correct HTTP status codes for various operations (200, 201, 204, 400, 404, etc.)
- **8.2.1_cache_control_header.sh** - Tests Cache-Control header handling for HTTP caching
- **8.2.2_header_if_match.sh** - Tests If-Match and If-None-Match headers for optimistic concurrency control with ETags
- **8.2.3_header_odata_entityid.sh** - Tests OData-EntityId response header for entity operations
- **8.2.4_header_content_id.sh** - Tests Content-ID header usage in batch requests for referencing entities
- **8.2.5_header_location.sh** - Tests that Location header is properly set for resource creation
- **8.2.6_header_odata_version.sh** - Tests OData-Version header and version negotiation
- **8.2.7_header_accept.sh** - Tests Accept header content negotiation and media type handling
- **8.2.8_header_prefer.sh** - Tests Prefer header (return=minimal, return=representation, odata.maxpagesize)
- **8.2.9_header_maxversion.sh** - Tests OData-MaxVersion header for version negotiation
- **8.3_error_responses.sh** - Validates error response format and structure
- **8.4_error_response_consistency.sh** - Tests consistency of error responses across different error scenarios

### Service Document & Metadata (Section 9.x)
- **9.1_service_document.sh** - Validates service document structure and format
- **9.2_metadata_document.sh** - Tests metadata document structure, XML format, and schema elements
- **9.3_annotations_metadata.sh** - Tests vocabulary annotations in metadata document

### String Representation (Section 7.x)
- **7.1.1_unicode_strings.sh** - Tests Unicode and internationalization support including multi-byte characters, emoji, RTL text (Arabic, Hebrew), and international scripts (Chinese, Japanese, Korean, Greek, Thai, Cyrillic)

### JSON Format (Section 10.x)
- **10.1_json_format.sh** - Tests JSON format requirements (value property, @odata.context, valid structure, etc.)

### URL Conventions (Section 11.1.x, 11.2.x)
- **11.1_resource_path.sh** - Tests resource path conventions for addressing OData resources including entity sets, entities, properties, navigation paths, and system resources
- **11.2.1_addressing_entities.sh** - Tests entity addressing (entity sets, single entities, properties, $value)
- **11.2.2_canonical_url.sh** - Tests canonical URL representation in @odata.id and dereferenceability
- **11.2.3_property_access.sh** - Tests accessing individual properties and property $value
- **11.2.4_collection_operations.sh** - Tests addressing entity collections vs single entities, collection format with value wrapper
- **11.2.7_metadata_levels.sh** - Tests odata.metadata parameter (minimal, full, none)
- **11.2.8_delta_links.sh** - Tests delta link support for change tracking (optional feature)
- **11.2.9_lambda_operators.sh** - Tests lambda operators (any, all) for collection filtering
- **11.2.10_addressing_operations.sh** - Tests addressing bound and unbound actions and functions
- **11.2.11_property_value.sh** - Tests accessing raw property values using $value path segment
- **11.2.12_stream_properties.sh** - Tests media entities, stream properties, and $value access for binary content (optional feature)
- **11.2.13_type_casting.sh** - Tests derived types, type casting in URLs, isof/cast functions, and polymorphic queries (optional feature)
- **11.2.14_url_encoding.sh** - Tests proper handling of URL encoding in resource paths and query parameters
- **11.2.15_entity_references.sh** - Tests $ref for working with entity references
- **11.2.16_singleton_operations.sh** - Tests singleton entity operations (GET, PATCH, PUT) and proper error responses for invalid operations

### Query Options - Search (Section 11.2.4.x)
- **11.2.4.1_query_search.sh** - Tests $search query option for free-text search

### Query Options - System (Section 11.2.5.x)
- **11.2.5.1_query_filter.sh** - Tests $filter query option with various operators (eq, gt, contains, and/or)
- **11.2.5.2_query_select_orderby.sh** - Tests $select and $orderby query options
- **11.2.5.3_query_top_skip.sh** - Tests $top and $skip query options for paging
- **11.2.5.4_query_apply.sh** - Tests $apply query option for data aggregation (optional extension)
- **11.2.5.4.1_advanced_apply.sh** - Tests advanced $apply transformations including multiple aggregations, groupby with multiple properties, transformation pipelines, countdistinct, filter before/after aggregation, and complex combinations
- **11.2.5.5_query_count.sh** - Tests $count query option (count with filter, top, etc.)
- **11.2.5.6_query_expand.sh** - Tests $expand query option for expanding related entities
- **11.2.5.7_query_skiptoken.sh** - Tests $skiptoken for server-driven paging and continuation tokens
- **11.2.5.8_query_compute.sh** - Tests $compute query option for computed properties (OData v4.01 feature, optional)
- **11.2.5.9_nested_expand_options.sh** - Tests nested $expand with multiple levels and nested query options ($filter, $select, $orderby, $top)
- **11.2.5.10_query_option_combinations.sh** - Tests valid combinations of query options and proper error handling
- **11.2.5.11_orderby_computed_properties.sh** - Tests $orderby with computed properties from $compute (OData v4.01 feature)
- **11.2.5.13_query_index.sh** - Tests $index system query option for zero-based ordinal position of items (OData v4.01 feature, optional)

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
- **11.3.8_filter_expanded_properties.sh** - Tests filtering entities based on properties of expanded navigation entities using any() and all()

### Data Modification (Section 11.4.x)
- **11.4.1_requesting_entities.sh** - Tests various methods to request individual entities (GET, HEAD, conditional)
- **11.4.2_create_entity.sh** - Tests entity creation (POST) with proper headers and status codes
- **11.4.2.1_odata_bind.sh** - Tests @odata.bind for linking entities during creation
- **11.4.3_update_entity.sh** - Tests entity updates (PATCH) including partial updates and error cases
- **11.4.4_delete_entity.sh** - Tests entity deletion (DELETE) and verification
- **11.4.5_upsert.sh** - Tests upsert operations (PUT) for creating or replacing entities
- **11.4.6_relationships.sh** - Tests relationship management with $ref (optional feature)
- **11.4.6.1_navigation_property_operations.sh** - Tests operations on navigation properties including accessing, filtering, and query options
- **11.4.7_deep_insert.sh** - Tests creating entities with related entities in a single POST request (deep insert)
- **11.4.8_modify_relationships.sh** - Tests modifying relationships using $ref endpoints (PUT, POST, DELETE on $ref)
- **11.4.9_batch_requests.sh** - Tests batch request processing (optional feature)
- **11.4.9.1_batch_error_handling.sh** - Tests batch error handling including changeset atomicity, malformed requests, invalid methods, error response formats, and request order preservation
- **11.4.10_asynchronous_requests.sh** - Tests asynchronous request processing with Prefer: respond-async header
- **11.4.11_head_requests.sh** - Tests HEAD requests for entities and collections, validates headers without body
- **11.4.12_returning_results.sh** - Tests Prefer: return=representation and return=minimal headers
- **11.4.13_action_function_parameters.sh** - Tests parameter validation for actions and functions including required parameters and type validation
- **11.4.14_null_value_handling.sh** - Tests proper handling of null values in requests and responses
- **11.4.15_data_validation.sh** - Tests data validation for entity creation and updates

### Conditional Operations (Section 11.5.x)
- **11.5.1_conditional_requests.sh** - Tests conditional requests with ETags (If-Match, If-None-Match)

### Operations (Section 12.x)
- **12.1_operations.sh** - Tests OData operations (actions and functions) including bound and unbound operations, parameter passing, and proper invocation syntax
- **12.2_function_action_overloading.sh** - Tests function and action overloading (OData v4.01 feature)

### Asynchronous Processing (Section 13.x)
- **13.1_asynchronous_processing.sh** - Tests asynchronous request processing features including the Prefer: respond-async header, status monitor URLs, and proper async response patterns

### Conditional Operations (Section 11.5.x)
- **11.5.1_conditional_requests.sh** - Tests conditional requests with ETags (If-Match, If-None-Match)

### Annotations (Section 11.6 & 14.x)
- **11.6_annotations.sh** - Tests instance annotations, @odata control information, and custom annotations in responses
- **14.1_vocabulary_annotations.sh** - Tests vocabulary annotations including Core vocabulary (Description, LongDescription), Computed/Immutable annotations, instance annotations (@odata.type, @odata.id, @odata.etag), and optional vocabularies (Capabilities, Measures, Validation)

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

### Guide for Coding Agents (AI Contributors)

**Important:** This section provides specific guidance for AI coding agents and automated contributors on how to write compliance tests that integrate properly with the test framework.

#### Core Principles for AI Agents

1. **Always use the test framework** - Source `test_framework.sh` at the start of every test script
2. **Use framework HTTP functions** - Never use raw `curl` directly; use `http_get`, `http_post`, etc. for automatic debug logging
3. **Follow the exact template structure** - Consistency is critical for automation
4. **Return proper exit codes** - Let the framework handle this via `print_summary()`
5. **Don't manually track counters** - The framework manages PASSED, FAILED, and TOTAL automatically
6. **Clean up test data** - Always implement and register a cleanup function

#### Required Script Structure for AI Agents

Every compliance test script MUST follow this exact structure:

```bash
#!/bin/bash

# OData v4 Compliance Test: <Section Number> <Title>
# <Brief description of what this test validates>
# Spec: <URL to OData v4 specification section>

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: <Section Number> <Title>"
echo "======================================"
echo ""
echo "Description: <Detailed multi-line description>"
echo ""
echo "Spec Reference: <URL>"
echo ""

# [Optional] Define cleanup function if you create test data
CREATED_IDS=()

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        curl -s -X DELETE "$SERVER_URL/EntitySet($id)" > /dev/null 2>&1
    done
}

# [Optional] Register cleanup if you defined it
register_cleanup

# Define test functions (one per test case)
test_something() {
    # Test implementation
    # Return 0 for success, 1 for failure
    local HTTP_CODE=$(http_get "$SERVER_URL/EntitySet")
    check_status "$HTTP_CODE" "200"
}

test_another_thing() {
    local RESPONSE=$(http_get_body "$SERVER_URL/EntitySet(1)")
    check_contains "$RESPONSE" "expectedValue"
}

# Run tests using run_test function
run_test "Description of what test 1 validates" test_something
run_test "Description of what test 2 validates" test_another_thing

# REQUIRED: Call print_summary at the end
print_summary
```

#### Framework Functions for AI Agents

**HTTP Request Functions (ALWAYS use these instead of raw curl):**

- `http_get URL [headers...]` - GET request, returns HTTP status code only
- `http_get_body URL [headers...]` - GET request, returns response body
- `http_post URL data [headers...]` - POST request with data
- `http_patch URL data [headers...]` - PATCH request with data
- `http_put URL data [headers...]` - PUT request with data
- `http_delete URL [headers...]` - DELETE request

**Validation Functions:**

- `check_status actual expected` - Verify HTTP status code matches expected
- `check_contains "$response" "substring"` - Verify response contains substring
- `check_json_field "$response" "fieldName"` - Verify JSON response has field

**Test Execution:**

- `run_test "description" test_function` - Execute a test and track results
- `skip_test "description" "reason"` - Mark a test as skipped (for unimplemented features)
- `print_summary` - Print final summary and exit (REQUIRED at end of script)

**Cleanup:**

- `register_cleanup` - Register the `cleanup()` function to run on exit

#### AI Agent Examples

**Example 1: Simple GET request test**
```bash
test_get_entity() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products(1)")
    check_status "$HTTP_CODE" "200"
}

run_test "GET request returns 200 OK" test_get_entity
```

**Example 2: POST request with validation**
```bash
test_create_entity() {
    local RESPONSE=$(http_post "$SERVER_URL/Products" \
        '{"Name":"Test Product","Price":99.99}' \
        -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if check_status "$HTTP_CODE" "201"; then
        # Extract ID for cleanup
        local BODY=$(echo "$RESPONSE" | sed '$d')
        local ID=$(echo "$BODY" | grep -oP '"ID":\s*\K\d+')
        CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        http_delete "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup
run_test "POST creates new entity" test_create_entity
```

**Example 3: Testing with query options**
```bash
test_filter_query() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products?\$filter=Price gt 100")
    
    # Check response contains expected field
    if check_json_field "$RESPONSE" "value"; then
        # Additional validation
        if echo "$RESPONSE" | grep -q '"Price"'; then
            return 0
        fi
    fi
    return 1
}

run_test "Filter query returns filtered results" test_filter_query
```

**Example 4: Skipping a test for unimplemented features**
```bash
# Skip a test when the corresponding OData feature is not yet implemented
skip_test "Delta token support" "Delta token feature is not yet implemented in go-odata"

# Skip with default reason (defaults to "Feature not yet implemented")
skip_test "Stream property support"

# You can also conditionally skip tests based on your own checks
# For example, check if a feature flag or capability is available
if [ -z "$BATCH_REQUESTS_ENABLED" ]; then
    skip_test "Batch request processing" "Batch request feature pending implementation"
else
    run_test "Batch request creates multiple entities" test_batch_create
fi
```

#### Common Pitfalls for AI Agents (AVOID THESE)

❌ **DON'T use raw curl:**
```bash
# WRONG - bypasses debug logging and framework
curl -s "$SERVER_URL/Products"
```

✅ **DO use framework functions:**
```bash
# CORRECT - enables debug logging
http_get "$SERVER_URL/Products"
```

❌ **DON'T manually manage counters:**
```bash
# WRONG
PASSED=0
FAILED=0
if [ condition ]; then
    PASSED=$((PASSED + 1))
fi
```

✅ **DO use run_test:**
```bash
# CORRECT
run_test "Test description" test_function
```

❌ **DON'T manually print summary:**
```bash
# WRONG
echo "Summary: $PASSED/$TOTAL tests passed"
exit $FAILED
```

✅ **DO call print_summary:**
```bash
# CORRECT
print_summary  # Handles everything automatically
```

#### Debugging Your Tests as an AI Agent

When your test fails, use debug mode to see the actual HTTP traffic:

```bash
DEBUG=1 ./v4.0/your_test.sh
```

This will show:
- Exact request URLs and methods
- Request headers and bodies
- Response status codes
- Response bodies (formatted)

Use this information to understand why a test is failing and adjust your test logic accordingly.

#### Quick Start

1. Create a new shell script in the appropriate version directory following the naming pattern: `{section}_{test_name}.sh`
   - Use the OData spec section number as prefix (e.g., `8.1.1`, `11.4.2`)
   - Place in `v4.0/` for OData 4.0 features
   - Place in `v4.01/` for OData 4.01-specific features

2. Make it executable:
   ```bash
   chmod +x v4.0/new_test.sh
   ```

3. Use the framework template:
   ```bash
   #!/bin/bash
   
   # OData v4 Compliance Test: <Section Number> <Title>
   # <Brief description>
   # Spec: <URL to OData v4 specification section>
   
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   source "$SCRIPT_DIR/../test_framework.sh"
   
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
   cd v4.0  # or v4.01
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
- `skip_test "description" ["reason"]` - Mark a test as skipped (optional reason defaults to "Feature not yet implemented")
- `register_cleanup` - Register cleanup function to run on exit

Example:
```bash
run_test "Service returns 200 OK" test_status_code
run_test "Response contains Name field" test_json_content

# Skip a test for an unimplemented feature
skip_test "Delta token support" "Delta tokens are not yet implemented"
```

### Standardized Output Format

The test framework automatically outputs results in a standardized, machine-parsable format:

```
COMPLIANCE_TEST_RESULT:PASSED=X:FAILED=Y:SKIPPED=Z:TOTAL=W
```

Where:
- **X** = Number of tests that passed
- **Y** = Number of tests that failed
- **Z** = Number of tests that were skipped
- **W** = Total number of tests (W = X + Y + Z)

**DO NOT** manually print this line or manipulate test counters - the framework handles everything automatically.

### Exit Codes

- Exit `0`: All tests passed (skipped tests do not cause failure)
- Exit `1`: One or more tests failed

The framework handles exit codes automatically based on test results. Skipped tests are tracked separately and do not cause the suite to fail.

### Debugging Test Failures

When a test fails, follow these steps to diagnose and fix the issue:

#### Step 1: Enable Debug Mode

Run the failing test with debug mode enabled to see full HTTP request/response details:

```bash
# Using the test runner
./run_compliance_tests.sh --debug test_name

# Or directly with environment variable
DEBUG=1 ./v4.0/test_name.sh
```

#### Step 2: Analyze the Debug Output

The debug output shows:
- **Request details**: Method, URL, headers, body
- **Response details**: Status code, body (formatted JSON when applicable)

Look for:
- Incorrect URLs or query parameters
- Missing or wrong headers
- Unexpected status codes
- Malformed request/response bodies
- Missing or incorrect JSON fields

**Example debug output analysis:**

```
╔══════════════════════════════════════════════════════╗
║ DEBUG: HTTP Request
╚══════════════════════════════════════════════════════╝

Method: GET
URL: http://localhost:9090/Products?$filter=Price gt 100
                                    ^^^^^^^^
                                    Check: Is the filter syntax correct?

╔══════════════════════════════════════════════════════╗
║ DEBUG: HTTP Response
╚══════════════════════════════════════════════════════╝

Status Code: 400
             ^^^
             Expected: 200, Got: 400 (Bad Request)
Body:
{
    "error": {
        "code": "BadRequest",
        "message": "Invalid filter syntax"
                    ^^^^^^^^^^^^^^^^^^^^^^
                    Root cause identified!
    }
}
```

#### Step 3: Check Common Issues

**URL encoding problems:**
- Query parameters with spaces should be properly encoded
- The framework handles basic URL encoding, but complex queries may need attention

**Status code mismatches:**
- Verify the expected status code is correct per OData spec
- Check if the server implementation differs from spec (may need spec reference)

**JSON validation:**
- Use `jq` or `python -m json.tool` to validate JSON structure
- Check for required OData fields: `@odata.context`, `value`, etc.

**Test data dependencies:**
- Ensure test data exists (use seeded data or create it in the test)
- Check if test data was cleaned up by a previous test

#### Step 4: Test Locally with Manual Requests

Start the compliance server manually and test with curl:

```bash
# Terminal 1: Start server
cd cmd/complianceserver
go run .

# Terminal 2: Test manually
curl -i http://localhost:9090/Products
curl -i "http://localhost:9090/Products?\$filter=Price gt 100"
curl -i -X POST http://localhost:9090/Products \
  -H "Content-Type: application/json" \
  -d '{"Name":"Test","Price":99.99}'
```

#### Step 5: Verify Test Logic

Check your test function logic:

```bash
test_example() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    
    # Add temporary debug output
    echo "  DEBUG: Got status code: $HTTP_CODE"
    
    check_status "$HTTP_CODE" "200"
}
```

#### Step 6: Common Test Patterns

**Pattern 1: Test a GET request**
```bash
test_get_entity() {
    local RESPONSE=$(http_get_body "$SERVER_URL/Products(1)")
    check_json_field "$RESPONSE" "Name"
}
```

**Pattern 2: Test status code only**
```bash
test_status() {
    local HTTP_CODE=$(http_get "$SERVER_URL/Products")
    check_status "$HTTP_CODE" "200"
}
```

**Pattern 3: Test with POST and cleanup**
```bash
CREATED_IDS=()

test_create() {
    local RESPONSE=$(http_post "$SERVER_URL/Products" \
        '{"Name":"Test","Price":99.99}' \
        -H "Content-Type: application/json")
    local HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" = "201" ]; then
        local ID=$(echo "$RESPONSE" | sed '$d' | grep -oP '"ID":\s*\K\d+')
        CREATED_IDS+=("$ID")
        return 0
    fi
    return 1
}

cleanup() {
    for id in "${CREATED_IDS[@]}"; do
        http_delete "$SERVER_URL/Products($id)" > /dev/null 2>&1
    done
}

register_cleanup
```

#### Step 7: Validate Against OData Spec

Always cross-reference with the OData specification:
- [OData v4.01 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
- [OData v4.01 Part 2: URL Conventions](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html)

Ensure your test expectations match the spec requirements.

### Example Test

See `v4.0/11.4.3_update_entity.sh` for a complete example using the test framework.

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
source "$SCRIPT_DIR/../test_framework.sh"

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
1. **Choose the right version directory**:
   - Add to `v4.0/` for tests that apply to OData 4.0 (and by extension 4.01)
   - Add to `v4.01/` only for features that are new or different in OData 4.01
2. **Use the test framework** - Source `test_framework.sh` from parent directory in all new tests
3. **Reference the spec** - Include the specific section number and URL
4. **Test thoroughly** - Run your test multiple times to ensure consistency
5. **Follow best practices** - See the Best Practices section above
6. **Ensure all tests pass** - Don't break existing tests
7. **Update documentation** - Add your test to the Available Tests section

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
