# OData 4.01 Compliance Tests

This directory contains compliance tests for **OData 4.01-specific features** - features that are new or different in OData 4.01 compared to OData 4.0.

## About These Tests

Tests in this directory validate **only the features that changed or were added in OData 4.01**. For core OData functionality that exists in both 4.0 and 4.01, see the `v4.0/` directory.

## What's Tested

OData 4.01-specific features:

### New Query Options
- **$compute** - Computed properties that can be used in $select, $filter, and $orderby
- **$index** - Zero-based ordinal position of items in a collection

### Enhanced Features
- **$orderby with computed properties** - Ordering by properties defined in $compute

## Running These Tests

From the compliance directory:

```bash
# Run only OData 4.01 tests
./run_compliance_tests.sh --version 4.01

# Run all tests (4.0 + 4.01)
./run_compliance_tests.sh --version all
```

## Adding New Tests

New tests should be added to this directory **only if** they test features that are:
- New in OData 4.01 (didn't exist in 4.0)
- Changed in OData 4.01 (different behavior than 4.0)

For all other OData features, add tests to the `v4.0/` directory instead.

When adding tests:

1. Source the test framework from parent directory:
   ```bash
   source "$SCRIPT_DIR/../test_framework.sh"
   ```

2. Follow the naming convention: `{section}_{test_name}.sh`

3. Make the test executable: `chmod +x new_test.sh`

See the main [compliance README](../README.md) for detailed testing guidelines.

## OData 4.01 Specification

Key differences between OData 4.0 and 4.01:
- **$compute** query option for defining computed properties in queries
- **$index** system query option for item position
- Delta payload format improvements
- Additional annotations and capabilities
- Enhanced type definitions

For full specification details, see:
- [OData v4.01 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
- [OData v4.01 Part 2: URL Conventions](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html)
