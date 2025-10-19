# Remaining Library Issues Found by Compliance Tests

This document lists the 42 failing test cases that represent legitimate library functionality gaps.

## Critical Issues (Fix Immediately)

### 1. $select Query Causes Server Panic
- **Test:** `5.3_enum_types.sh` Test 5
- **Issue:** Using `$select` query option causes server to return empty response
- **Example:** `GET /Products?$select=Name,Status`
- **Impact:** Core OData functionality broken

### 2. DateTimeOffset Literal Parsing
- **Test:** `5.1.1_primitive_data_types.sh` Test 5
- **Issue:** Cannot parse datetime literals in $filter expressions
- **Example:** `$filter=CreatedAt lt 2025-12-31T23:59:59Z`
- **Error:** "unexpected token after expression: 3 at position 17"
- **Impact:** Cannot filter by dates using literals

## High Priority Issues

### Query Options
- **$compute** (6 failures in `11.2.5.8_query_compute.sh`) - Not implemented
- **$skiptoken** (5 failures in `11.2.5.7_query_skiptoken.sh`) - Server-driven paging incomplete
- **Query combinations** (5 failures in `11.2.5.10_query_option_combinations.sh`) - Some combinations fail

### Filter Functions
- **Date/time functions** (3 failures in `11.3.2_filter_date_functions.sh`) - `date()`, `time()`, etc.
- **Type functions** (4 failures in `11.3.4_filter_type_functions.sh`) - `isof()`, `cast()`
- **String functions** (1 failure in `11.3.1_filter_string_functions.sh`) - Some functions missing

## Medium Priority Issues

### Null Handling
- **Nullable properties** (3 failures in `5.1.2_nullable_properties.sh`) - Null literal handling

### Navigation and Relationships
- **Expanded property filtering** (3 failures in `11.3.8_filter_expanded_properties.sh`)
- **Navigation operations** (1 failure in `11.4.6.1_navigation_property_operations.sh`)
- **$ref relationships** (3 failures in `11.4.8_modify_relationships.sh`)

### Advanced Features
- **Type casting** (1 failure in `11.2.13_type_casting.sh`) - Derived types
- **Action/function parameters** (2 failures in `11.4.13_action_function_parameters.sh`) - Validation
- **$orderby with $compute** (1 failure in `11.2.5.11_orderby_computed_properties.sh`)
- **Error response consistency** (1 failure in `8.4_error_response_consistency.sh`)

## Running Specific Failing Tests

To reproduce and debug specific issues:

```bash
cd compliance/v4

# Critical issues
./5.3_enum_types.sh           # $select panic
./5.1.1_primitive_data_types.sh  # DateTime parsing

# Query options
./11.2.5.8_query_compute.sh   # $compute not implemented
./11.2.5.7_query_skiptoken.sh # Server paging

# Filter functions
./11.3.2_filter_date_functions.sh  # Date functions
./11.3.4_filter_type_functions.sh  # Type functions
```

## Test Details

For detailed analysis of each failure, see `COMPLIANCE_TEST_ANALYSIS.md`.

For the latest compliance report, see `compliance-report.md`.
