# OData 4.0 Compliance Tests

This directory contains compliance tests for OData 4.0 specification features.

## About These Tests

Tests in this directory validate the **core OData 4.0 functionality** that forms the foundation of the OData protocol. These features are also part of OData 4.01, so these tests apply to both versions.

## What's Tested

All OData 4.0 features including:
- HTTP headers and status codes
- Service document and metadata
- URL conventions and entity addressing
- Query options ($filter, $select, $orderby, $top, $skip, $count, $expand, etc.)
- CRUD operations (GET, POST, PATCH, PUT, DELETE)
- Data types (primitive, complex, enum, temporal)
- Built-in filter functions (string, date, arithmetic, logical, etc.)
- Batch requests
- Conditional requests (ETags)
- Relationships and navigation properties
- Annotations

## Running These Tests

From the compliance directory:

```bash
# Run only OData 4.0 tests
./run_compliance_tests.sh --version 4.0

# Run all tests (4.0 + 4.01)
./run_compliance_tests.sh --version all
```

## Adding New Tests

New tests for OData 4.0 features should be added to this directory. Make sure to:

1. Source the test framework from parent directory:
   ```bash
   source "$SCRIPT_DIR/../test_framework.sh"
   ```

2. Follow the naming convention: `{section}_{test_name}.sh`

3. Make the test executable: `chmod +x new_test.sh`

See the main [compliance README](../README.md) for detailed testing guidelines.
