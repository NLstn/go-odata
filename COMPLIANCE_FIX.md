# OData v4 Compliance Fix: Enum Types with $select

## Issue Summary

The go-odata library was failing compliance test 5.3 (Enumeration Types), specifically test #5: "Select enum property". The issue manifested as an HTTP 000 response (empty reply from server), indicating a server panic.

## Root Cause

When the `$select` query option is used (e.g., `GET /Products?$select=Name,Status`), the OData library converts query results from struct instances to `map[string]interface{}` for efficient field filtering. However, the ETag generation code in `internal/etag/etag.go:28` was calling `reflect.Value.FieldByName()` on these map values, which caused a panic:

```
panic: reflect: call of reflect.Value.FieldByName on map Value
```

This panic occurred because:
1. `$select` converts entities to maps for selective field inclusion
2. ETag generation attempted to use struct reflection methods on maps
3. The panic happened during response serialization, resulting in an empty HTTP response

## Solution

Modified the `etag.Generate()` function to handle both struct and map entities:

### Changes to `internal/etag/etag.go`

1. **Added map detection**: Check if the entity value is a map before attempting struct field access
2. **Created `generateFromMap()` helper**: Handles ETag generation from `map[string]interface{}`
3. **Extracted conversion logic**: Created reusable helper functions for converting values to ETag sources:
   - `convertToETagSource()`: For `reflect.Value` (structs)
   - `convertInterfaceToETagSource()`: For `interface{}` (maps)
4. **Maintained consistency**: Ensured identical ETags are generated for structs and maps with the same values

### Key Code Changes

```go
// Before (crashed on maps)
func Generate(entity interface{}, meta *metadata.EntityMetadata) string {
    entityValue := reflect.ValueOf(entity)
    fieldValue := entityValue.FieldByName(meta.ETagProperty.FieldName) // PANIC on maps!
    // ...
}

// After (handles both structs and maps)
func Generate(entity interface{}, meta *metadata.EntityMetadata) string {
    entityValue := reflect.ValueOf(entity)
    
    // Handle map[string]interface{} (from $select queries)
    if entityValue.Kind() == reflect.Map {
        entityMap, ok := entity.(map[string]interface{})
        if !ok {
            return ""
        }
        return generateFromMap(entityMap, meta.ETagProperty)
    }
    
    // Original struct handling
    fieldValue := entityValue.FieldByName(meta.ETagProperty.FieldName)
    // ...
}
```

## Test Coverage

### Unit Tests (`internal/etag/etag_test.go`)

Added comprehensive tests for map-based ETag generation:

- `TestGenerate_WithMap`: Tests ETag generation from maps with various data types
- `TestGenerate_MapConsistency`: Verifies same input produces same ETag
- `TestGenerate_MapVsStruct`: Ensures struct and map produce identical ETags
- `TestGenerate_WithEnumInMap`: Tests enum types in maps
- `TestGenerate_WithEnumAsETag`: Tests using enum values as ETag fields

### Integration Tests (`test/enum_select_test.go`)

Created comprehensive integration tests covering:

- `$select` with enum properties and ETag properties
- `$select` with only enum properties (no ETag)
- `$orderby` with enum properties
- `$filter` with enum numeric values
- Combinations of `$select` and `$filter` with enums
- Single entity requests with `$select` and enums

## Validation

### Compliance Test Results

```
OData v4 Compliance Test - Section: 5.3 Enumeration Types
COMPLIANCE_TEST_RESULT:PASSED=12:FAILED=0:TOTAL=12
Status: PASSING
```

All 12 enum type compliance tests now pass, including:
- Test 5: Select enum property (previously failing) ✅
- All other enum-related tests ✅

### Code Quality

- **golangci-lint**: 0 issues
- **All unit tests**: 291 tests passing
- **All integration tests**: Passing
- **No breaking changes**: Existing functionality preserved

## Impact

This fix enables:
1. ✅ Using `$select` with enum properties
2. ✅ Proper ETag generation for all query scenarios
3. ✅ Full OData v4 compliance for enum types
4. ✅ Correct handling of map-based entities throughout the codebase

## Files Modified

1. `internal/etag/etag.go`: Core fix for map handling
2. `internal/etag/etag_test.go`: Unit tests for map-based ETag generation
3. `test/enum_select_test.go`: Integration tests for enum types with query options (NEW)

## Backwards Compatibility

✅ This fix is fully backwards compatible:
- No API changes
- No breaking changes to existing functionality
- All existing tests continue to pass
- Enhancement only - fixes a crash scenario

## OData v4 Specification Compliance

This fix ensures compliance with:
- [OData v4.01 Part 1: Protocol - Section 5.3 Enumeration Types](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_EnumerationType)
- OData v4 requirements for `$select` query option
- OData v4 requirements for ETag handling
