# Conditional Requests (11.5.1) - Analysis and Validation

## Executive Summary

The OData v4 compliance test for conditional requests (11.5.1) has been analyzed and validated. The library implementation is **correct and fully compliant** with the OData v4.01 specification for conditional request handling with ETags.

## Test Results

### Initial Test Results
- **Total Tests**: 6
- **Passing**: 4/6
- **Failing**: 2/6 (Tests 4 & 6)

### Root Cause of Failures
The test failures were caused by the test script using an invalid property name (`Description`) that doesn't exist on the `Product` entity. The test was attempting to update a property that is not defined in the entity schema, causing validation errors.

### After Fix
- **Total Tests**: 6  
- **Passing**: 6/6 ✅
- **Failing**: 0/6

## OData v4 Specification Compliance

According to the OData v4.01 specification (section 11.5.1 - Conditional Requests), the library correctly implements:

### 1. ETag Generation
- ✅ Generates ETags based on entity properties marked with `odata:"etag"` tag
- ✅ Uses SHA256 hash for deterministic ETag values
- ✅ Returns ETags in weak format: `W/"<hash>"`
- ✅ Includes ETag in HTTP response header
- ✅ Includes `@odata.etag` in JSON response body

### 2. If-None-Match Header (GET/HEAD Requests)
- ✅ Returns `304 Not Modified` when ETag matches
- ✅ Returns `200 OK` with full entity when ETag doesn't match
- ✅ Supports wildcard `*` matching (returns 304 for existing entities)
- ✅ Empty body for 304 responses
- ✅ Includes ETag header in 304 responses

### 3. If-Match Header (PATCH/PUT/DELETE Requests)
- ✅ Allows operation when ETag matches
- ✅ Returns `412 Precondition Failed` when ETag doesn't match
- ✅ Supports wildcard `*` (always allows operation for existing entities)
- ✅ Allows operation when If-Match header is absent (no precondition)
- ✅ Returns updated ETag after successful modification

### 4. HTTP Status Codes
- ✅ `200 OK` - Successful GET with entity data
- ✅ `204 No Content` - Successful PATCH/PUT/DELETE without return=representation
- ✅ `304 Not Modified` - If-None-Match header matches current ETag
- ✅ `412 Precondition Failed` - If-Match header doesn't match current ETag

## Implementation Details

### ETag Package (`internal/etag/`)
The ETag package provides three main functions:

1. **Generate()** - Creates ETags from entity ETag properties
   - Supports multiple field types (int, string, time.Time)
   - Handles both struct and map entities
   - Uses SHA256 for consistent hashing

2. **Match()** - Validates If-Match header
   - Compares ETags for equality
   - Handles wildcard `*` matching
   - Treats empty If-Match as "no precondition"

3. **NoneMatch()** - Validates If-None-Match header
   - Returns false if ETags match (trigger 304)
   - Returns true if ETags don't match (proceed normally)
   - Handles wildcard `*` matching

### Handler Integration (`internal/handlers/entity_crud.go`)
The entity CRUD handlers integrate ETag checking at the appropriate points:

- **GET/HEAD**: Check If-None-Match before returning entity
- **PATCH**: Check If-Match before applying updates
- **PUT**: Check If-Match before replacing entity
- **DELETE**: Check If-Match before deleting entity

## Test Coverage

### Unit Tests (`internal/etag/etag_test.go`)
- ETag generation from different field types
- Consistency testing (same input → same ETag)
- Uniqueness testing (different input → different ETag)
- Map vs struct entity handling
- Parse function testing
- Match function testing (all scenarios)
- NoneMatch function testing (all scenarios)

### Integration Tests (`test/etag_test.go`)
- GET with ETag header inclusion
- POST with ETag header inclusion
- PATCH with matching/non-matching If-Match
- PUT with matching/non-matching If-Match
- DELETE with matching/non-matching If-Match
- GET with matching/non-matching If-None-Match
- HEAD with If-None-Match
- Wildcard (*) matching for both If-Match and If-None-Match
- ETag changes after entity modification
- Prefer: return=representation with ETags

### Conditional Requests Tests (`test/conditional_requests_test.go`)
Comprehensive end-to-end test covering the complete conditional request flow:
- Complete flow from entity creation through multiple conditional operations
- If-None-Match with wildcard
- DELETE with conditional headers
- PUT with conditional headers

## Changes Made

### 1. Test Script Fix (`compliance/v4/11.5.1_conditional_requests.sh`)
**Issue**: Tests 4, 5, and 6 used invalid property name `Description`  
**Fix**: Changed to use valid property name `Name`  
**Impact**: All compliance tests now pass

### 2. New Integration Test (`test/conditional_requests_test.go`)
**Purpose**: Provide comprehensive test coverage for conditional requests  
**Coverage**: 
- Complete conditional request flow
- Edge cases and error conditions
- Proper documentation of expected behavior

## Validation Results

### Code Quality
```bash
$ golangci-lint run --timeout=5m
0 issues.
```

### Test Results
```bash
$ go test ./...
ok      github.com/nlstn/go-odata                       0.009s
ok      github.com/nlstn/go-odata/internal/etag         0.003s
ok      github.com/nlstn/go-odata/internal/handlers     0.208s
ok      github.com/nlstn/go-odata/test                  0.228s
```

### Compliance Test Results
```bash
$ bash compliance/v4/11.5.1_conditional_requests.sh
COMPLIANCE_TEST_RESULT:PASSED=6:FAILED=0:TOTAL=6
Status: PASSING
```

## Conclusion

The go-odata library correctly implements conditional request handling as specified in the OData v4.01 specification. The test failures were due to an error in the test script itself (using an invalid property name), not due to any issues with the library implementation.

### Library Strengths
1. ✅ Complete ETag generation and validation
2. ✅ Proper HTTP status code handling
3. ✅ Wildcard (*) support
4. ✅ Comprehensive test coverage
5. ✅ Clean, maintainable code
6. ✅ No linting issues

### No Implementation Changes Required
The library already had complete and correct implementation of conditional requests. Only the compliance test script needed correction.

## References

- [OData v4.01 Part 1: Protocol - Section 11.5.1 Conditional Requests](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ConditionalRequests)
- [HTTP RFC 7232: Conditional Requests](https://tools.ietf.org/html/rfc7232)
