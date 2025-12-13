# OData Compliance Test Suite - Issues Report

**Date:** 2025-12-13  
**Test Suite Version:** v4.0  
**Total Tests:** 666  
**Passing:** 642 (96%)  
**Failing:** 1  
**Skipped:** 23

---

## Executive Summary

This report documents issues found in the **compliance test code itself**, not the library implementation. The analysis reveals several categories of test quality issues that need to be addressed to ensure the test suite properly validates OData v4 specification compliance.

### Key Findings

1. **23 tests are incorrectly accepting 404/501 responses** as passing when they should validate actual feature behavior
2. **1 test failing** due to unexpected 500 error needs investigation
3. **Multiple tests lack proper validation** of response content beyond HTTP status codes
4. **Several tests accept non-compliant behavior** as passing instead of failing or skipping
5. **Type casting tests have incorrect error messages** claiming spec violations when features are optional
6. **Entity reference ($ref) tests are overly lenient** and don't validate actual $ref functionality
7. **Location header test is too permissive** and accepts missing headers

---

## Critical Issues (High Priority)

### Issue #1: Entity References Test Suite - Insufficient Validation
**File:** `tests/v4_0/11.2.15_entity_references.go`  
**Tests Affected:** Multiple tests in suite

**Problem:**  
The entity reference tests only validate HTTP status codes but don't properly validate that $ref actually works according to the OData specification. Several tests are marked as skipped when they receive 404 responses, but 404 means the feature is not implemented, which should be a test failure for mandatory OData features.

**OData v4 Specification Requires:**
- Entity references ($ref) are a MANDATORY feature of OData v4
- $ref must return `@odata.id` containing the entity URL
- $ref must NOT include entity properties (only the reference)
- $ref on collections must support $filter, $top, $skip, $orderby, $count
- $ref must reject $expand and $select with 400 Bad Request

**Current Test Issues:**

1. **Test: "GET $ref returns reference URL"** (Line 175-177 in skipped list)
   - Currently skipped if service returns 404
   - Should FAIL if 404 is returned since $ref is mandatory
   
2. **Test: "PUT $ref creates/updates single-valued relationship"** (Line 178 in skipped list)
   - Currently skipped if 404/405/501
   - $ref manipulation is mandatory, should fail not skip

3. **Test: "POST $ref adds to collection-valued relationship"** (Line 179 in skipped list)
   - Currently skipped if feature not implemented
   - Should fail for missing mandatory feature

4. **Test: "DELETE $ref removes relationship"** (Line 180 in skipped list)
   - Currently skipped if not implemented
   - Relationship deletion via $ref is mandatory

5. **Test: "$ref with $filter on collection"** (Lines 136-156)
   ```go
   if resp.StatusCode == 400 {
       return ctx.Skip("$ref with $filter not supported by service")
   }
   ```
   - Problem: Accepts 400 as "not supported" and skips
   - Should: Fail if 400 is returned, as this combination is required

**How to Fix:**
```go
// Change from accepting 404 as skip to:
if resp.StatusCode == 404 {
    return fmt.Errorf("$ref is a mandatory OData v4 feature but returned 404")
}

// For tests validating $ref content:
if resp.StatusCode != 200 {
    return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
}

// Then validate the actual $ref behavior:
var result map[string]interface{}
if err := json.Unmarshal(resp.Body, &result); err != nil {
    return fmt.Errorf("failed to parse $ref response: %w", err)
}

odataID, ok := result["@odata.id"].(string)
if !ok || odataID == "" {
    return fmt.Errorf("$ref response must contain @odata.id")
}
```

---

### Issue #2: Type Casting Tests - Incorrect Error Messages
**File:** `tests/v4_0/11.2.13_type_casting.go`  
**Tests Affected:** 9 tests (all skipped)

**Problem:**  
Tests return error messages claiming "specification violation" when features are actually optional or dependent on the service's type system. The tests check if derived types exist in metadata but then claim spec violations when they're not found.

**Example Issues:**

1. **Test: "Filter by type using isof function"** (Lines 27-49)
   ```go
   if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
       return fmt.Errorf("specification violation: isof function must be supported (status: %d)", resp.StatusCode)
   }
   ```
   - Problem: Claims "specification violation" but `isof` is only required if derived types exist
   - The test already checks for derived types and skips if not present
   - Should never reach this error with current skip logic

2. **Test: "Type cast in URL path"** (Lines 51-78)
   - Same issue - claims spec violation but derives types are optional

3. **Tests already correctly skip** via `skipIfDerivedTypesUnavailable()` but error messages are wrong

**OData v4 Specification:**
- Type inheritance and derived types are OPTIONAL features
- If a service declares derived types in metadata, THEN type casting must be supported
- If no derived types exist, type casting tests should skip

**How to Fix:**
```go
// The skip logic is correct, but error messages should be:
if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
    return fmt.Errorf("type casting failed but derived types declared in metadata (status: %d)", resp.StatusCode)
}

// OR even better, just expect 200:
if resp.StatusCode != 200 {
    return fmt.Errorf("type casting should work when derived types declared, got status %d", resp.StatusCode)
}
```

---

### Issue #3: Location Header Test - Too Permissive
**File:** `tests/v4_0/8.2.5_header_location.go`  
**Test:** "Location header for created entity" (Currently skipped)

**Problem:**  
Test accepts missing Location header as passing, when OData v4 spec requires it for 201 responses.

**Current Code (Lines 19-46):**
```go
if resp.StatusCode != 201 && resp.StatusCode != 200 {
    // Skip if creation fails due to validation
    if resp.StatusCode == 400 {
        return ctx.Skip("Entity creation validation error (likely schema mismatch)")
    }
    return ctx.Skip("Entity creation not supported or failed")
}

// Location header should contain URL of created entity
location := resp.Headers.Get("Location")
if location != "" && !strings.Contains(location, "Products") {
    return framework.NewError("Location should contain entity URL")
}

return nil  // PASSES if location is empty!
```

**OData v4 Specification Requires:**
- Services MUST include a Location header in 201 Created responses
- Location header MUST contain the URL to the created entity
- This is not optional for OData v4 compliance

**How to Fix:**
```go
if resp.StatusCode != 201 {
    if resp.StatusCode == 400 {
        return ctx.Skip("Entity creation validation error (likely schema mismatch)")
    }
    return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
}

location := resp.Headers.Get("Location")
if location == "" {
    return fmt.Errorf("Location header is required for 201 responses per OData v4 spec")
}

if !strings.Contains(location, "Products") {
    return fmt.Errorf("Location header should contain entity URL, got: %s", location)
}

return nil
```

---

### Issue #4: Stream Properties Test - Failing Test Investigation Needed
**File:** `tests/v4_0/11.2.12_stream_properties.go`  
**Test:** "Update media entity content" (Line 170-193)  
**Status:** Currently FAILING with status 500

**Problem:**  
Test is receiving HTTP 500 Internal Server Error, but test expects 204/200 or acceptable skip codes (404/405/501).

**Current Code:**
```go
resp, err := ctx.PUTRaw(mediaPath+"/$value", []byte("updated-binary-data"), "image/png")
if err != nil {
    return err
}

if resp.StatusCode == 204 || resp.StatusCode == 200 {
    return nil
}

if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
    return ctx.Skip("media entity update unsupported (status: %d)")
}

return fmt.Errorf("unexpected status: %d", resp.StatusCode)
```

**Issue Analysis:**
- Receiving 500 indicates a library bug (not a test issue)
- However, the test should provide better diagnostics
- Test should log response body when receiving unexpected status codes

**How to Fix (Test Improvement):**
```go
if resp.StatusCode == 204 || resp.StatusCode == 200 {
    return nil
}

if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
    return ctx.Skip("media entity update unsupported (status: %d)")
}

// Better error reporting for unexpected status
return fmt.Errorf("unexpected status %d updating media entity. Response body: %s", 
    resp.StatusCode, string(resp.Body))
```

**Note:** This is primarily a library bug (500 error), but test could be improved for better diagnostics.

---

## Medium Priority Issues

### Issue #5: Resource Path Test - Empty Segments Handling
**File:** `tests/v4_0/11.1_resource_path.go`  
**Test:** "Empty path segments should return error or redirect" (Line 382-404)

**Problem:**  
Test is skipped when server returns 200 for `/Products//` (empty segment), but this should be a test failure since the OData spec doesn't allow empty path segments.

**Current Code:**
```go
// Products// should be invalid (empty segment)
resp, err := ctx.GET("/Products//")
if err != nil {
    return err
}

// If server normalizes empty segments to a valid path and returns 200, mark as skipped
if resp.StatusCode == 200 {
    return ctx.Skip("server normalizes empty segments; behavior under review")
}
```

**OData v4 Specification:**
- Empty path segments are not valid in OData URLs
- Servers should return 404 (Not Found) or 400 (Bad Request)
- A 301 redirect to normalized path is acceptable

**How to Fix:**
```go
resp, err := ctx.GET("/Products//")
if err != nil {
    return err
}

// OData spec: empty path segments are invalid
if resp.StatusCode == 200 {
    return fmt.Errorf("server accepted invalid URL with empty path segments (should return 400 or 404)")
}

if resp.StatusCode != 404 && resp.StatusCode != 400 && resp.StatusCode != 301 {
    return fmt.Errorf("empty path segments must return 404, 400, or 301 (got %d)", resp.StatusCode)
}

return nil
```

---

### Issue #6: Geospatial Functions - All Tests Skipped But Should Validate Properly
**File:** `tests/v4_0/11.3.7_filter_geo_functions.go`  
**Tests Affected:** 6 optional feature tests (currently skipped)

**Problem:**  
Tests correctly skip when geospatial functions are not implemented, but they don't properly validate the functions when they ARE implemented.

**Current Pattern (Lines 72-88):**
```go
resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
if err != nil {
    return err
}

// 200 OK = supported, 400/404/501 = not implemented (acceptable)
switch resp.StatusCode {
case 200:
    return nil  // Just accepts 200 without validating results!
case 400, 404, 501:
    return ctx.Skip("geo.distance not implemented (optional feature)")
default:
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
```

**Problem:**  
When geospatial functions ARE supported (200 response), test should validate:
1. Response contains valid results
2. Filter actually applied correctly
3. Results match the geospatial query

**How to Fix:**
```go
case 200:
    // Validate that geo filter actually worked
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }
    
    value, ok := result["value"].([]interface{})
    if !ok {
        return fmt.Errorf("response missing 'value' array")
    }
    
    // Geospatial filter should return results or empty array
    // Just validate structure is correct
    ctx.Log("Geospatial filter returned %d results", len(value))
    return nil
```

---

### Issue #7: Query Filter Tests - Should Validate Filter Behavior
**File:** `tests/v4_0/11.2.5.1_query_filter.go`  
**Observation:** Good validation pattern, but could be applied to more tests

**Good Example (Lines 20-72):**
The eq filter test properly validates:
- Status code is 200
- Response has correct structure
- Filter actually returned matching entities
- All entities match the filter condition

**Problem:**  
Many other tests just check status 200 without validating the filter worked correctly.

**Recommendation:**  
Apply the same validation pattern from the `test_filter_eq` test to other filter tests throughout the suite.

---

### Issue #8: Select/OrderBy Tests - Good Validation But Edge Cases Missing
**File:** `tests/v4_0/11.2.5.2_query_select_orderby.go`  
**Tests:** $select and $orderby tests

**Good Aspects:**
- Properly validates selected fields are present
- Validates non-selected fields are absent
- Validates ordering is correct

**Missing Edge Cases:**
1. No test for `$select=*` (select all)
2. No test for `$select` with navigation properties
3. No test for `$orderby` with null values
4. No test for `$orderby` with case-insensitive sorting

**Recommendation:**  
Add tests for these edge cases to ensure comprehensive coverage.

---

### Issue #9: Expand Tests - Should Validate Expansion Content
**File:** `tests/v4_0/11.2.5.6_query_expand.go`  
**Lines:** 20-66

**Current Validation:**
- Checks status 200
- Verifies expanded field exists
- Verifies it's an array

**Missing Validation:**
- Should verify expanded entities have proper structure
- Should verify expanded entities contain expected fields
- Should verify expansion depth is correct
- Should check for @odata.context in expanded content

**How to Improve:**
```go
// After verifying descriptions is an array:
if len(descArray) > 0 {
    desc, ok := descArray[0].(map[string]interface{})
    if !ok {
        return fmt.Errorf("first description is not an object")
    }
    
    // Validate expanded entity has required fields
    if _, ok := desc["ID"]; !ok {
        return fmt.Errorf("expanded entity missing ID field")
    }
    if _, ok := desc["Description"]; !ok {
        return fmt.Errorf("expanded entity missing Description field")
    }
}
```

---

## Low Priority Issues

### Issue #10: Singleton Operations Tests
**File:** `tests/v4_0/11.2.16_singleton_operations.go`  
**Issue:** Test claims 404 is "non-compliant" but singletons are optional

**Lines around 98-106:**
```go
if resp.StatusCode == 404 {
    return fmt.Errorf("status code 404 indicates singleton property access is not implemented and is non-compliant")
}
```

**Problem:**  
Singletons are an OPTIONAL OData feature. Test should skip if not implemented, not claim non-compliance.

**How to Fix:**
```go
if resp.StatusCode == 404 {
    return ctx.Skip("singleton properties not implemented (optional feature)")
}
```

---

### Issue #11: Navigation Property Operations - Relationship Tests
**File:** `tests/v4_0/11.4.6.1_navigation_property_operations.go`  
**Issue:** Tests skip on 404 but should validate when feature is present

Similar to Issue #1, relationship manipulation tests are too lenient in accepting 404 as "not implemented" when these are core OData features.

---

### Issue #12: Addressing Operations Tests
**File:** `tests/v4_0/11.2.10_addressing_operations.go`  
**Issue:** Error message confusing

**Current (around line 40):**
```go
if resp.StatusCode == 404 || resp.StatusCode == 501 {
    return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
}
```

**Problem:**  
Returns an error claiming compliance failure, which is correct, but the message format is confusing because it's inside an error return.

**How to Fix:**
Message is actually correct - just clarify:
```go
if resp.StatusCode == 404 || resp.StatusCode == 501 {
    return fmt.Errorf("operation addressing is required by OData v4 but got status %d", resp.StatusCode)
}
```

---

### Issue #13: Deep Insert Tests - Limited Validation
**File:** `tests/v4_0/11.4.7_deep_insert.go`  
**Issue:** Tests accept 201 but don't validate the deep insert actually worked

**Recommendation:**  
After creating entities via deep insert:
1. Fetch the created entity
2. Verify related entities were also created
3. Verify relationships are properly established

---

### Issue #14: Batch Request Tests - Error Handling Could Be Stricter
**File:** `tests/v4_0/11.4.9_batch_requests.go` and `11.4.9.1_batch_error_handling.go`  
**Issue:** Could validate batch response format more strictly

**Recommendations:**
1. Validate Content-Type is `multipart/mixed`
2. Validate batch boundaries are correct
3. Validate each part has proper headers
4. Validate atomicity groups work correctly

---

## Test Categorization

### Tests That Need Fixing:

1. **Must Fail Instead of Skip** (Critical):
   - All 6 $ref manipulation tests in `11.2.15_entity_references.go`
   - Location header test in `8.2.5_header_location.go`
   - Empty path segments test in `11.1_resource_path.go`

2. **Error Messages Need Correction** (Medium):
   - All 9 type casting tests in `11.2.13_type_casting.go`
   - Singleton test in `11.2.16_singleton_operations.go`

3. **Need Better Validation** (Medium):
   - Geospatial function tests (6 tests)
   - Expand tests (need to validate expansion content)
   - Deep insert tests (need to verify results)

4. **Need Enhanced Diagnostics** (Low):
   - Stream properties update test (currently failing)
   - Batch request tests

---

## Summary of Test Quality by Category

### Excellent Tests (Good Examples):
- `11.2.5.1_query_filter.go` - Properly validates filter behavior
- `11.2.5.2_query_select_orderby.go` - Good validation of select/orderby results
- `11.4.2_create_entity.go` - Comprehensive entity creation validation

### Tests Needing Major Fixes:
- `11.2.15_entity_references.go` - Too lenient, skips mandatory features
- `11.2.13_type_casting.go` - Wrong error messages
- `8.2.5_header_location.go` - Accepts missing required header

### Tests Needing Minor Improvements:
- `11.3.7_filter_geo_functions.go` - Should validate results when supported
- `11.2.5.6_query_expand.go` - Should validate expansion content
- `11.4.7_deep_insert.go` - Should verify deep relationships

---

## Recommended Actions

### Immediate (High Priority):
1. Fix entity reference tests to fail when $ref is not implemented
2. Fix location header test to require header in 201 responses
3. Correct type casting error messages
4. Fix empty path segments test to fail on 200 response

### Short Term (Medium Priority):
5. Add content validation to geospatial tests when features are supported
6. Enhance expand tests to validate expanded content structure
7. Add diagnostic improvements to failing media stream test
8. Review all tests that skip on 404 to determine if feature is mandatory

### Long Term (Low Priority):
9. Add edge case tests for $select and $orderby
10. Enhance batch request validation
11. Add deep insert result verification
12. Create test documentation explaining mandatory vs optional features

---

## Testing Philosophy Recommendations

### Current Issues:
- Too many tests accept 404/501 as "not implemented" and skip
- Tests don't distinguish between mandatory and optional OData features
- Many tests only validate HTTP status without checking actual behavior
- Error messages sometimes claim spec violations for optional features

### Recommended Approach:
1. **Mandatory Features:** Should FAIL if not implemented (404/501)
2. **Optional Features:** Should SKIP if not implemented (404/501)
3. **All Features:** Should thoroughly validate behavior when implemented (not just 200 OK)
4. **Error Messages:** Should be clear about whether feature is mandatory or optional

### Test Strictness Levels:
- **Level 1 (Current):** Check HTTP status code only
- **Level 2 (Needed):** Check status + response structure
- **Level 3 (Goal):** Check status + structure + actual behavior/data

Most tests are at Level 1 or 2. Goal should be Level 3 for comprehensive validation.

---

## Conclusion

The compliance test suite has a solid foundation with 666 tests covering most OData v4 features. However, many tests are too lenient and don't properly validate that features actually work according to the specification. 

**Key improvements needed:**
1. Distinguish mandatory vs optional features in test behavior
2. Add behavior validation beyond HTTP status codes
3. Fix tests that accept non-compliant behavior as passing
4. Improve error messages and diagnostics

Implementing these fixes will significantly improve the test suite's ability to catch OData v4 compliance issues in the go-odata library.
